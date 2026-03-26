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

// ProcessingResult contém os caminhos dos artefatos gerados pelo pipeline.
// TempDir deve ser removido pelo chamador após os uploads.
type ProcessingResult struct {
	TempDir       string
	ThumbnailsDir string // vazio se a etapa falhou
	AudioPath     string
	PreviewPath   string
	StreamingDir  string
}

// ProcessVideo executa todas as etapas do pipeline de processamento de vídeo.
// Retorna ProcessingResult mesmo em caso de erro, para que o chamador possa limpar TempDir.
func ProcessVideo(inputPath, outputPath string) (*ProcessingResult, error) {
	baseDir := filepath.Dir(outputPath)
	videoBaseName := filepath.Base(inputPath)
	videoBaseName = videoBaseName[:len(videoBaseName)-len(filepath.Ext(videoBaseName))]

	tempDir := filepath.Join(baseDir, videoBaseName+"_temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}

	result := &ProcessingResult{TempDir: tempDir}

	// 1. Validação
	log.Info().Msg("Etapa 1/7: Validando vídeo")
	start := time.Now()
	if err := processor_steps.ValidateVideo(inputPath); err != nil {
		return result, fmt.Errorf("validação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("validate").Observe(time.Since(start).Seconds())

	// 2. Análise de conteúdo
	log.Info().Msg("Etapa 2/7: Analisando conteúdo")
	start = time.Now()
	if err := processor_steps.AnalyzeContent(inputPath); err != nil {
		log.Warn().Err(err).Msg("Falha na análise de conteúdo")
	}
	metrics.ProcessingStepDuration.WithLabelValues("analyze").Observe(time.Since(start).Seconds())

	// 3. Transcodificação (etapa crítica)
	log.Info().Msg("Etapa 3/7: Transcodificando vídeo")
	start = time.Now()
	if err := processor_steps.TranscodeVideo(inputPath, outputPath); err != nil {
		return result, fmt.Errorf("transcodificação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("transcode").Observe(time.Since(start).Seconds())

	transcodedPath := outputPath

	// 4. Geração de thumbnails
	log.Info().Msg("Etapa 4/7: Gerando thumbnails")
	start = time.Now()
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := processor_steps.GenerateThumbnails(transcodedPath, thumbnailsDir); err != nil {
		log.Warn().Err(err).Msg("Falha ao gerar thumbnails")
	} else {
		result.ThumbnailsDir = thumbnailsDir
	}
	metrics.ProcessingStepDuration.WithLabelValues("thumbnails").Observe(time.Since(start).Seconds())

	// 5. Extração de áudio
	log.Info().Msg("Etapa 5/7: Extraindo áudio")
	start = time.Now()
	audioPath := filepath.Join(tempDir, "audio.mp3")
	if err := processor_steps.ExtractAudio(transcodedPath, audioPath); err != nil {
		log.Warn().Err(err).Msg("Falha na extração de áudio")
	} else {
		result.AudioPath = audioPath
	}
	metrics.ProcessingStepDuration.WithLabelValues("audio").Observe(time.Since(start).Seconds())

	// 6. Geração de preview
	log.Info().Msg("Etapa 6/7: Gerando preview")
	start = time.Now()
	previewPath := filepath.Join(tempDir, "preview.mp4")
	if err := processor_steps.GeneratePreview(transcodedPath, previewPath); err != nil {
		log.Warn().Err(err).Msg("Falha na geração de preview")
	} else {
		result.PreviewPath = previewPath
	}
	metrics.ProcessingStepDuration.WithLabelValues("preview").Observe(time.Since(start).Seconds())

	// 7. Segmentação para streaming
	log.Info().Msg("Etapa 7/7: Segmentando para streaming")
	start = time.Now()
	streamingDir := filepath.Join(tempDir, "streaming")
	if err := processor_steps.SegmentForStreaming(transcodedPath, streamingDir); err != nil {
		log.Warn().Err(err).Msg("Falha na segmentação para streaming")
	} else {
		result.StreamingDir = streamingDir
	}
	metrics.ProcessingStepDuration.WithLabelValues("streaming").Observe(time.Since(start).Seconds())

	log.Info().Msg("Pipeline de processamento concluído com sucesso")
	return result, nil
}
