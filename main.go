package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"video-processor/config"
	"video-processor/internal/processor"
	"video-processor/minio"
	"video-processor/queue"

	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis do .env, se existir
	_ = godotenv.Load()

	fmt.Println("Iniciando o worker de processamento de vídeo...")

	_, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Erro ao carregar configurações: %v\n", err)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	workerLoop(sigChan)

	fmt.Println("Worker encerrado.")
}

// workerLoop executa o loop principal do worker, processando mensagens da fila
func workerLoop(sigChan chan os.Signal) {
	for {
		select {
		case <-sigChan:
			fmt.Println("Sinal de desligamento recebido. Encerrando worker...")
			return
		default:
			processNextMessage()
		}
	}
}

// processNextMessage consome e processa uma mensagem da fila
func processNextMessage() {
	msg, err := queue.ConsumeMessage()
	if err != nil {
		fmt.Printf("Erro ao consumir mensagem: %v\n", err)
		return
	}
	if msg == nil {
		return
	}

	videoID := msg.VideoID
	fmt.Printf("Mensagem recebida. VideoID: %s\n", videoID)

	// Baixar vídeo do MinIO
	inputPath := "/tmp/" + videoID + "_input.mp4"
	if err := minio.DownloadVideo(videoID, inputPath); err != nil {
		fmt.Printf("Erro ao baixar vídeo do MinIO: %v\n", err)
		return
	}

	// Processar vídeo
	outputPath := "/tmp/" + videoID + "_output.mp4"
	if err := processor.ProcessVideo(inputPath, outputPath); err != nil {
		fmt.Printf("Erro ao processar vídeo: %v\n", err)
		return
	}

	// Fazer upload do vídeo processado para o MinIO
	processedID := videoID + "_processed"
	if err := minio.UploadVideo(outputPath, processedID); err != nil {
		fmt.Printf("Erro ao fazer upload do vídeo processado: %v\n", err)
		return
	}

	// Publicar mensagem de sucesso na fila
	if err := queue.PublishSuccessMessage(processedID); err != nil {
		fmt.Printf("Erro ao publicar mensagem de sucesso: %v\n", err)
		return
	}

	fmt.Println("Processamento concluído com sucesso!")
}
