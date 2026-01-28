package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"video-processor/config"
	"video-processor/internal/processor"
	"video-processor/metrics"
	"video-processor/minio"
	"video-processor/queue"
)

func main() {
	// Configurar zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()

	initClients(cfg)

	// Iniciar servidor HTTP com métricas e health check
	startHTTPServer()

	numWorkers := cfg.WorkerCount
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU()
	}

	log.Info().Int("workers", numWorkers).Msg("Iniciando video-processor")

	// Inicializar métrica de workers ativos
	metrics.ActiveWorkers.Set(float64(numWorkers))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Inicia os workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					log.Info().Int("workerID", workerID).Msg("Finalizando worker graciosamente")
					return
				default:
					if err := processNextMessage(ctx, workerID); err != nil {
						if err != context.Canceled {
							log.Error().Err(err).Int("workerID", workerID).Msg("Erro ao processar mensagem")
						}
					}
				}
			}
		}(i + 1)
	}

	// Aguarda sinal de interrupção
	<-sigChan
	log.Warn().Msg("Sinal de desligamento recebido. Iniciando shutdown gracioso")

	// Cancela o contexto para iniciar o shutdown
	cancel()

	// Aguarda os workers com timeout
	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownComplete)
	}()

	// Define um timeout para o shutdown (30 segundos)
	select {
	case <-shutdownComplete:
		log.Info().Msg("Todos os workers encerraram normalmente")
	case <-time.After(30 * time.Second):
		log.Warn().Msg("Timeout atingido. Forçando encerramento dos workers restantes")
	}

	log.Info().Msg("Programa encerrado")
}

func initClients(cfg *config.Config) {
	queue.InitRedisClient(cfg)
	minio.InitMinioClient(cfg)
}

func startHTTPServer() {
	// Configurar rotas
	http.HandleFunc("/health", healthCheckHandler)
	http.Handle("/metrics", promhttp.Handler())

	// Iniciar servidor em goroutine
	go func() {
		log.Info().Str("address", ":8080").Msg("Servidor HTTP iniciado")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal().Err(err).Msg("Erro ao iniciar servidor HTTP")
		}
	}()
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Verificar Redis
	if err := queue.HealthCheck(); err != nil {
		log.Error().Err(err).Msg("Health check Redis falhou")
		http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
		return
	}

	// Verificar MinIO
	if err := minio.HealthCheck(); err != nil {
		log.Error().Err(err).Msg("Health check MinIO falhou")
		http.Error(w, "MinIO unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func processNextMessage(ctx context.Context, workerID int) error {
	// Timeout para processar cada mensagem (5 minutos)
	processCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Canal para controle da operação
	done := make(chan error, 1)

	go func() {
		msg, err := queue.ConsumeMessage()
		if err != nil {
			done <- fmt.Errorf("erro ao consumir mensagem: %v", err)
			return
		}
		if msg == nil {
			done <- nil
			return
		}

		videoID := msg.VideoID
		log.Info().Int("workerID", workerID).Str("videoID", videoID).Msg("Processando vídeo")

		// Medir tempo de processamento
		startTime := time.Now()

		// Usa os.TempDir() para compatibilidade com Windows
		inputPath := filepath.Join(os.TempDir(), videoID+"_input.mp4")
		outputPath := filepath.Join(os.TempDir(), videoID+"_output.mp4")

		// Limpeza dos arquivos temporários ao finalizar
		defer func() {
			os.Remove(inputPath) // todo: Handle these
			os.Remove(outputPath)
		}()

		if err := minio.DownloadVideo(minio.VideoTypeRaw, videoID, inputPath); err != nil {
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- fmt.Errorf("erro ao baixar vídeo: %v", err)
			return
		}

		if err := processor.ProcessVideo(inputPath, outputPath); err != nil {
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- fmt.Errorf("erro ao processar vídeo: %v", err)
			return
		}

		processedID := videoID + "_processed"
		if err := minio.UploadVideo(outputPath, minio.VideoTypeProcessed, processedID); err != nil {
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- fmt.Errorf("erro ao fazer upload do vídeo: %v", err)
			return
		}

		if err := queue.PublishSuccessMessage(processedID); err != nil {
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- fmt.Errorf("erro ao publicar mensagem de sucesso: %v", err)
			return
		}

		// Registrar métricas de sucesso
		duration := time.Since(startTime).Seconds()
		metrics.ProcessingDuration.Observe(duration)
		metrics.VideosProcessedTotal.WithLabelValues("success").Inc()

		log.Info().Int("workerID", workerID).Str("videoID", videoID).Float64("duration_seconds", duration).Msg("Vídeo processado com sucesso")
		done <- nil
	}()

	// Aguarda a conclusão ou cancelamento
	select {
	case err := <-done:
		return err
	case <-processCtx.Done():
		return fmt.Errorf("operação cancelada: %v", processCtx.Err())
	}
}
