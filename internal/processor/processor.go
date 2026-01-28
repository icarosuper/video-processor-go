package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	processor_steps "video-processor/internal/processor/processor-steps"
	"video-processor/metrics"
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
	log.Info().Msg("Etapa 1/7: Validando vídeo")
	start := time.Now()
	if err := processor_steps.ValidateVideo(inputPath); err != nil {
		return fmt.Errorf("validação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("validate").Observe(time.Since(start).Seconds())

	// 2. Análise de conteúdo (antes de transcodificar para obter metadados originais)
	log.Info().Msg("Etapa 2/7: Analisando conteúdo")
	start = time.Now()
	if err := processor_steps.AnalyzeContent(inputPath); err != nil {
		log.Warn().Err(err).Msg("Falha na análise de conteúdo")
		// Não retorna erro - análise é informativa
	}
	metrics.ProcessingStepDuration.WithLabelValues("analyze").Observe(time.Since(start).Seconds())

	// 3. Transcodificação (etapa crítica)
	log.Info().Msg("Etapa 3/7: Transcodificando vídeo")
	start = time.Now()
	if err := processor_steps.TranscodeVideo(inputPath, outputPath); err != nil {
		return fmt.Errorf("transcodificação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("transcode").Observe(time.Since(start).Seconds())

	// As próximas etapas usam o vídeo transcodificado
	transcodedPath := outputPath

	// 4. Geração de thumbnails
	log.Info().Msg("Etapa 4/7: Gerando thumbnails")
	start = time.Now()
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := processor_steps.GenerateThumbnails(transcodedPath, thumbnailsDir); err != nil {
		log.Warn().Err(err).Msg("Falha ao gerar thumbnails")
		// Não retorna erro - thumbnails são opcionais
	}
	metrics.ProcessingStepDuration.WithLabelValues("thumbnails").Observe(time.Since(start).Seconds())

	// 5. Extração de áudio
	log.Info().Msg("Etapa 5/7: Extraindo áudio")
	start = time.Now()
	audioPath := filepath.Join(tempDir, "audio.mp3")
	if err := processor_steps.ExtractAudio(transcodedPath, audioPath); err != nil {
		log.Warn().Err(err).Msg("Falha na extração de áudio")
		// Não retorna erro - áudio separado é opcional
	}
	metrics.ProcessingStepDuration.WithLabelValues("audio").Observe(time.Since(start).Seconds())

	// 6. Geração de preview
	log.Info().Msg("Etapa 6/7: Gerando preview")
	start = time.Now()
	previewPath := filepath.Join(tempDir, "preview.mp4")
	if err := processor_steps.GeneratePreview(transcodedPath, previewPath); err != nil {
		log.Warn().Err(err).Msg("Falha na geração de preview")
		// Não retorna erro - preview é opcional
	}
	metrics.ProcessingStepDuration.WithLabelValues("preview").Observe(time.Since(start).Seconds())

	// 7. Segmentação para streaming
	log.Info().Msg("Etapa 7/7: Segmentando para streaming")
	start = time.Now()
	streamingDir := filepath.Join(tempDir, "streaming")
	if err := processor_steps.SegmentForStreaming(transcodedPath, streamingDir); err != nil {
		log.Warn().Err(err).Msg("Falha na segmentação para streaming")
		// Não retorna erro - streaming é opcional
	}
	metrics.ProcessingStepDuration.WithLabelValues("streaming").Observe(time.Since(start).Seconds())

	log.Info().Msg("Pipeline de processamento concluído com sucesso")
	return nil
}
