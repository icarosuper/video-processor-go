package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"

	"video-processor/config"
	"video-processor/internal/processor"
	"video-processor/minio"
	"video-processor/queue"

	"github.com/joho/godotenv"
)

func getWorkerCount() int {
	val := os.Getenv("WORKER_COUNT")
	if val != "" {
		n, err := strconv.Atoi(val)
		if err == nil && n > 0 {
			return n
		}
	}
	return runtime.NumCPU()
}

func main() {
	// Carregar variáveis do .env, se existir
	_ = godotenv.Load()

	fmt.Println("Iniciando o worker de processamento de vídeo...")

	_, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Erro ao carregar configurações: %v\n", err)
		return
	}

	numWorkers := getWorkerCount()
	fmt.Printf("Número de workers: %d\n", numWorkers)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					fmt.Printf("Worker %d encerrando...\n", workerID)
					return
				default:
					processNextMessage(workerID)
				}
			}
		}(i + 1)
	}

	<-sigChan
	fmt.Println("Sinal de desligamento recebido. Encerrando workers...")
	close(stopChan)
	wg.Wait()
	fmt.Println("Worker encerrado.")
}

// processNextMessage consome e processa uma mensagem da fila
func processNextMessage(workerID int) {
	msg, err := queue.ConsumeMessage()
	if err != nil {
		fmt.Printf("[Worker %d] Erro ao consumir mensagem: %v\n", workerID, err)
		return
	}
	if msg == nil {
		return
	}

	videoID := msg.VideoID
	fmt.Printf("[Worker %d] Mensagem recebida. VideoID: %s\n", workerID, videoID)

	// Baixar vídeo do MinIO
	inputPath := "/tmp/" + videoID + "_input.mp4"
	if err := minio.DownloadVideo(videoID, inputPath); err != nil {
		fmt.Printf("[Worker %d] Erro ao baixar vídeo do MinIO: %v\n", workerID, err)
		return
	}

	// Processar vídeo
	outputPath := "/tmp/" + videoID + "_output.mp4"
	if err := processor.ProcessVideo(inputPath, outputPath); err != nil {
		fmt.Printf("[Worker %d] Erro ao processar vídeo: %v\n", workerID, err)
		return
	}

	// Fazer upload do vídeo processado para o MinIO
	processedID := videoID + "_processed"
	if err := minio.UploadVideo(outputPath, processedID); err != nil {
		fmt.Printf("[Worker %d] Erro ao fazer upload do vídeo processado: %v\n", workerID, err)
		return
	}

	// Publicar mensagem de sucesso na fila
	if err := queue.PublishSuccessMessage(processedID); err != nil {
		fmt.Printf("[Worker %d] Erro ao publicar mensagem de sucesso: %v\n", workerID, err)
		return
	}

	fmt.Printf("[Worker %d] Processamento concluído com sucesso!\n", workerID)
}
