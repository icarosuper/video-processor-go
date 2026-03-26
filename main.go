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

	ctx, cancel := context.WithCancel(context.Background())

	// Goroutine que recoloca jobs órfãos (crash durante processamento)
	go queue.StartRecovery(ctx, 10*time.Minute)

	// Goroutine que atualiza a métrica de tamanho da fila a cada 30 segundos
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if size, err := queue.GetQueueSize(); err == nil {
					metrics.QueueSize.Set(float64(size))
				}
			}
		}
	}()
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
	// Bloqueia até receber mensagem ou ctx ser cancelado (shutdown).
	// BRPOPLPUSH move o job atomicamente para a fila de processamento.
	msg, err := queue.ConsumeMessage(ctx)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}

	videoID := msg.VideoID
	log.Info().Int("workerID", workerID).Str("videoID", videoID).Msg("Processando vídeo")

	if err := queue.SetJobProcessing(videoID); err != nil {
		log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao atualizar estado do job para processing")
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		// jobErr rastreia o erro final para o defer abaixo.
		var jobErr error

		metrics.ActiveWorkers.Inc()
		defer metrics.ActiveWorkers.Dec()

		defer func() {
			if jobErr != nil {
				state, err := queue.SetJobFailed(videoID, jobErr)
				if err != nil {
					log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao atualizar estado do job para failed")
				}
				if state != nil && state.RetryCount <= queue.MaxJobRetries {
					if err := queue.RequeueJob(videoID); err != nil {
						log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao recolocar job na fila")
					} else {
						log.Warn().Str("videoID", videoID).Int("tentativa", state.RetryCount).Int("max", queue.MaxJobRetries).Msg("Job agendado para retry")
					}
				} else {
					if err := queue.MoveToDLQ(videoID); err != nil {
						log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao mover job para dead letter queue")
					} else {
						log.Error().Str("videoID", videoID).Str("erro", jobErr.Error()).Msg("Job movido para dead letter queue após esgotar tentativas")
					}
				}
			}
			if err := queue.AcknowledgeMessage(videoID); err != nil {
				log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao confirmar processamento do job")
			}
		}()

		startTime := time.Now()

		inputPath := filepath.Join(os.TempDir(), videoID+"_input.mp4")
		outputPath := filepath.Join(os.TempDir(), videoID+"_output.mp4")

		defer func() {
			os.Remove(inputPath)
			os.Remove(outputPath)
		}()

		if err := minio.DownloadVideo(minio.VideoTypeRaw, videoID, inputPath); err != nil {
			jobErr = fmt.Errorf("erro ao baixar vídeo: %v", err)
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- jobErr
			return
		}
		if info, err := os.Stat(inputPath); err == nil {
			metrics.VideoSizeBytes.Observe(float64(info.Size()))
		}

		result, err := processor.ProcessVideo(inputPath, outputPath)
		if result != nil {
			defer os.RemoveAll(result.TempDir)
		}
		if err != nil {
			jobErr = fmt.Errorf("erro ao processar vídeo: %v", err)
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- jobErr
			return
		}

		processedID := videoID + "_processed"
		if err := minio.UploadVideo(outputPath, minio.VideoTypeProcessed, processedID); err != nil {
			jobErr = fmt.Errorf("erro ao fazer upload do vídeo: %v", err)
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- jobErr
			return
		}

		// Upload dos artefatos opcionais gerados pelo pipeline
		if result.ThumbnailsDir != "" {
			if err := minio.UploadDirectory(result.ThumbnailsDir, "thumbnails/"+videoID); err != nil {
				log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao fazer upload dos thumbnails")
			}
		}
		if result.AudioPath != "" {
			if err := minio.UploadFile(result.AudioPath, "audio/"+videoID+".mp3"); err != nil {
				log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao fazer upload do áudio")
			}
		}
		if result.PreviewPath != "" {
			if err := minio.UploadFile(result.PreviewPath, "preview/"+videoID+"_preview.mp4"); err != nil {
				log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao fazer upload do preview")
			}
		}
		if result.StreamingDir != "" {
			if err := minio.UploadDirectory(result.StreamingDir, "hls/"+videoID); err != nil {
				log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao fazer upload dos segmentos HLS")
			}
		}

		if err := queue.PublishSuccessMessage(processedID); err != nil {
			jobErr = fmt.Errorf("erro ao publicar mensagem de sucesso: %v", err)
			metrics.VideosProcessedTotal.WithLabelValues("error").Inc()
			done <- jobErr
			return
		}

		// Registra estado final e métricas de sucesso
		artifacts := buildJobArtifacts(videoID, processedID, result)
		metadata := toJobMetadata(result)
		if err := queue.SetJobDone(videoID, artifacts, metadata); err != nil {
			log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao atualizar estado do job para done")
		}

		duration := time.Since(startTime).Seconds()
		metrics.ProcessingDuration.Observe(duration)
		metrics.VideosProcessedTotal.WithLabelValues("success").Inc()

		log.Info().Int("workerID", workerID).Str("videoID", videoID).Float64("duration_seconds", duration).Msg("Vídeo processado com sucesso")
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-processCtx.Done():
		return fmt.Errorf("operação cancelada: %v", processCtx.Err())
	}
}

// toJobMetadata converte os metadados do pipeline para o tipo do pacote queue.
func toJobMetadata(result *processor.ProcessingResult) *queue.VideoMetadata {
	if result.Metadata == nil {
		return nil
	}
	return &queue.VideoMetadata{
		Duration:   result.Metadata.Duration,
		Width:      result.Metadata.Width,
		Height:     result.Metadata.Height,
		VideoCodec: result.Metadata.VideoCodec,
		AudioCodec: result.Metadata.AudioCodec,
		FPS:        result.Metadata.FPS,
		Bitrate:    result.Metadata.Bitrate,
		Size:       result.Metadata.Size,
	}
}

// buildJobArtifacts monta o objeto de artefatos com os paths no MinIO
// a partir do resultado do pipeline. Só inclui artefatos que foram gerados.
func buildJobArtifacts(videoID, processedID string, result *processor.ProcessingResult) queue.JobArtifacts {
	artifacts := queue.JobArtifacts{
		Video: "processed/" + processedID,
	}
	if result.ThumbnailsDir != "" {
		artifacts.Thumbnails = "thumbnails/" + videoID
	}
	if result.AudioPath != "" {
		artifacts.Audio = "audio/" + videoID + ".mp3"
	}
	if result.PreviewPath != "" {
		artifacts.Preview = "preview/" + videoID + "_preview.mp4"
	}
	if result.StreamingDir != "" {
		artifacts.HLS = "hls/" + videoID
	}
	return artifacts
}
