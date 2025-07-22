package processor

import (
	"fmt"
	processor_steps "video-processor/internal/processor/processor-steps"
)

// ProcessVideo executa todas as etapas do pipeline de processamento de vídeo.
func ProcessVideo(inputPath, outputPath string) error {
	// 1. Validação
	if err := processor_steps.ValidateVideo(inputPath); err != nil {
		return fmt.Errorf("validação falhou: %w", err)
	}

	// 2. Transcodificação
	if err := processor_steps.TranscodeVideo(inputPath, outputPath); err != nil {
		return fmt.Errorf("transcodificação falhou: %w", err)
	}

	// 3. Geração de thumbnails
	if err := processor_steps.GenerateThumbnails(inputPath, "/tmp/thumbnails"); err != nil {
		return fmt.Errorf("thumbnails falhou: %w", err)
	}

	// 4. Extração de áudio
	if err := processor_steps.ExtractAudio(inputPath, "/tmp/audio.mp3"); err != nil {
		return fmt.Errorf("extração de áudio falhou: %w", err)
	}

	// 5. Geração de pré-visualização
	if err := processor_steps.GeneratePreview(inputPath, "/tmp/preview.jpg"); err != nil {
		return fmt.Errorf("preview falhou: %w", err)
	}

	// 6. Análise de conteúdo
	if err := processor_steps.AnalyzeContent(inputPath); err != nil {
		return fmt.Errorf("análise de conteúdo falhou: %w", err)
	}

	// 7. Segmentação para streaming
	if err := processor_steps.SegmentForStreaming(inputPath, "/tmp/streaming"); err != nil {
		return fmt.Errorf("segmentação para streaming falhou: %w", err)
	}

	return nil
}
