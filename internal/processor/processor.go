package processor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	processor_steps "video-processor/internal/processor/processor-steps"
)

// ProcessVideo executa todas as etapas do pipeline de processamento de vídeo.
func ProcessVideo(inputPath, outputPath string) error {
	// Criar diretório de output temporário para os arquivos gerados
	baseDir := filepath.Dir(outputPath)
	videoBaseName := filepath.Base(inputPath)
	videoBaseName = videoBaseName[:len(videoBaseName)-len(filepath.Ext(videoBaseName))]

	tempDir := filepath.Join(baseDir, videoBaseName+"_temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}
	defer os.RemoveAll(tempDir) // Limpar diretório temporário ao finalizar

	// 1. Validação
	log.Printf("Etapa 1/7: Validando vídeo...")
	if err := processor_steps.ValidateVideo(inputPath); err != nil {
		return fmt.Errorf("validação falhou: %w", err)
	}

	// 2. Análise de conteúdo (antes de transcodificar para obter metadados originais)
	log.Printf("Etapa 2/7: Analisando conteúdo...")
	if err := processor_steps.AnalyzeContent(inputPath); err != nil {
		log.Printf("Aviso: falha na análise de conteúdo: %v", err)
		// Não retorna erro - análise é informativa
	}

	// 3. Transcodificação (etapa crítica)
	log.Printf("Etapa 3/7: Transcodificando vídeo...")
	if err := processor_steps.TranscodeVideo(inputPath, outputPath); err != nil {
		return fmt.Errorf("transcodificação falhou: %w", err)
	}

	// As próximas etapas usam o vídeo transcodificado
	transcodedPath := outputPath

	// 4. Geração de thumbnails
	log.Printf("Etapa 4/7: Gerando thumbnails...")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := processor_steps.GenerateThumbnails(transcodedPath, thumbnailsDir); err != nil {
		log.Printf("Aviso: falha ao gerar thumbnails: %v", err)
		// Não retorna erro - thumbnails são opcionais
	}

	// 5. Extração de áudio
	log.Printf("Etapa 5/7: Extraindo áudio...")
	audioPath := filepath.Join(tempDir, "audio.mp3")
	if err := processor_steps.ExtractAudio(transcodedPath, audioPath); err != nil {
		log.Printf("Aviso: falha na extração de áudio: %v", err)
		// Não retorna erro - áudio separado é opcional
	}

	// 6. Geração de preview
	log.Printf("Etapa 6/7: Gerando preview...")
	previewPath := filepath.Join(tempDir, "preview.mp4")
	if err := processor_steps.GeneratePreview(transcodedPath, previewPath); err != nil {
		log.Printf("Aviso: falha na geração de preview: %v", err)
		// Não retorna erro - preview é opcional
	}

	// 7. Segmentação para streaming
	log.Printf("Etapa 7/7: Segmentando para streaming...")
	streamingDir := filepath.Join(tempDir, "streaming")
	if err := processor_steps.SegmentForStreaming(transcodedPath, streamingDir); err != nil {
		log.Printf("Aviso: falha na segmentação para streaming: %v", err)
		// Não retorna erro - streaming é opcional
	}

	log.Printf("Pipeline de processamento concluído com sucesso!")
	return nil
}
