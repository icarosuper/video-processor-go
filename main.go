package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"video-processor/config"
	"video-processor/internal/processor"
	"video-processor/minio"
	"video-processor/queue"
)

func main() {
	cfg := config.LoadConfig()

	initClients(cfg)

	numWorkers := cfg.WorkerCount
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU()
	}

	fmt.Printf("Número de workers: %d\n", numWorkers)

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
					fmt.Printf("Worker %d: Finalizando graciosamente...\n", workerID)
					return
				default:
					if err := processNextMessage(ctx, workerID); err != nil {
						if err != context.Canceled {
							fmt.Printf("Worker %d: Erro: %v\n", workerID, err)
						}
					}
				}
			}
		}(i + 1)
	}

	// Aguarda sinal de interrupção
	<-sigChan
	fmt.Println("\nSinal de desligamento recebido. Iniciando shutdown gracioso...")

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
		fmt.Println("Todos os workers encerraram normalmente.")
	case <-time.After(30 * time.Second):
		fmt.Println("Timeout atingido. Forçando encerramento dos workers restantes.")
	}

	fmt.Println("Programa encerrado.")
}

func initClients(cfg *config.Config) {
	queue.InitRedisClient(cfg)
	minio.InitMinioClient(cfg)
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
		fmt.Printf("[Worker %d] Processando vídeo: %s\n", workerID, videoID)

		// Usa os.TempDir() para compatibilidade com Windows
		inputPath := filepath.Join(os.TempDir(), videoID+"_input.mp4")
		outputPath := filepath.Join(os.TempDir(), videoID+"_output.mp4")

		// Limpeza dos arquivos temporários ao finalizar
		defer func() {
			os.Remove(inputPath) // todo: Handle these
			os.Remove(outputPath)
		}()

		if err := minio.DownloadVideo(minio.VideoTypeRaw, videoID, inputPath); err != nil {
			done <- fmt.Errorf("erro ao baixar vídeo: %v", err)
			return
		}

		if err := processor.ProcessVideo(inputPath, outputPath); err != nil {
			done <- fmt.Errorf("erro ao processar vídeo: %v", err)
			return
		}

		processedID := videoID + "_processed"
		if err := minio.UploadVideo(outputPath, minio.VideoTypeProcessed, processedID); err != nil {
			done <- fmt.Errorf("erro ao fazer upload do vídeo: %v", err)
			return
		}

		if err := queue.PublishSuccessMessage(processedID); err != nil {
			done <- fmt.Errorf("erro ao publicar mensagem de sucesso: %v", err)
			return
		}

		fmt.Printf("[Worker %d] Vídeo %s processado com sucesso\n", workerID, videoID)
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
