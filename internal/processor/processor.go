package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	processor_steps "video-processor/internal/processor/processor-steps"
	"video-processor/metrics"
)

// Timeouts individuais por etapa do pipeline.
// Etapas críticas têm limites maiores; etapas rápidas têm limites curtos.
const (
	stepTimeoutValidate   = 30 * time.Second
	stepTimeoutAnalyze    = 30 * time.Second
	stepTimeoutTranscode  = 3 * time.Minute
	stepTimeoutThumbnails = 60 * time.Second
	stepTimeoutAudio      = 2 * time.Minute
	stepTimeoutPreview    = 2 * time.Minute
	stepTimeoutStreaming   = 4 * time.Minute
)

// ProcessingResult contém os caminhos dos artefatos gerados pelo pipeline.
// TempDir deve ser removido pelo chamador após os uploads.
type ProcessingResult struct {
	TempDir       string
	ThumbnailsDir string // vazio se a etapa falhou
	AudioPath     string
	PreviewPath   string
	StreamingDir  string
	Metadata      *processor_steps.VideoMetadata // nil se a análise falhou
}

// ProcessVideo executa todas as etapas do pipeline de processamento de vídeo.
// Retorna ProcessingResult mesmo em caso de erro, para que o chamador possa limpar TempDir.
// O ctx pai controla o cancelamento global; cada etapa recebe seu próprio sub-contexto com timeout.
func ProcessVideo(ctx context.Context, inputPath, outputPath string) (*ProcessingResult, error) {
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
	stepCtx, cancel := context.WithTimeout(ctx, stepTimeoutValidate)
	err := processor_steps.ValidateVideo(stepCtx, inputPath)
	cancel()
	if err != nil {
		return result, fmt.Errorf("validação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("validate").Observe(time.Since(start).Seconds())

	// 2. Análise de conteúdo
	log.Info().Msg("Etapa 2/7: Analisando conteúdo")
	start = time.Now()
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutAnalyze)
	metadata, err := processor_steps.AnalyzeContent(stepCtx, inputPath)
	cancel()
	if err != nil {
		log.Warn().Err(err).Msg("Falha na análise de conteúdo")
	} else {
		result.Metadata = metadata
	}
	metrics.ProcessingStepDuration.WithLabelValues("analyze").Observe(time.Since(start).Seconds())

	// 3. Transcodificação (etapa crítica)
	log.Info().Msg("Etapa 3/7: Transcodificando vídeo")
	start = time.Now()
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutTranscode)
	err = processor_steps.TranscodeVideo(stepCtx, inputPath, outputPath)
	cancel()
	if err != nil {
		return result, fmt.Errorf("transcodificação falhou: %w", err)
	}
	metrics.ProcessingStepDuration.WithLabelValues("transcode").Observe(time.Since(start).Seconds())

	transcodedPath := outputPath

	// 4. Geração de thumbnails
	log.Info().Msg("Etapa 4/7: Gerando thumbnails")
	start = time.Now()
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutThumbnails)
	err = processor_steps.GenerateThumbnails(stepCtx, transcodedPath, thumbnailsDir)
	cancel()
	if err != nil {
		log.Warn().Err(err).Msg("Falha ao gerar thumbnails")
	} else {
		result.ThumbnailsDir = thumbnailsDir
	}
	metrics.ProcessingStepDuration.WithLabelValues("thumbnails").Observe(time.Since(start).Seconds())

	// 5. Extração de áudio
	log.Info().Msg("Etapa 5/7: Extraindo áudio")
	start = time.Now()
	audioPath := filepath.Join(tempDir, "audio.mp3")
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutAudio)
	err = processor_steps.ExtractAudio(stepCtx, transcodedPath, audioPath)
	cancel()
	if err != nil {
		log.Warn().Err(err).Msg("Falha na extração de áudio")
	} else {
		result.AudioPath = audioPath
	}
	metrics.ProcessingStepDuration.WithLabelValues("audio").Observe(time.Since(start).Seconds())

	// 6. Geração de preview
	log.Info().Msg("Etapa 6/7: Gerando preview")
	start = time.Now()
	previewPath := filepath.Join(tempDir, "preview.mp4")
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutPreview)
	err = processor_steps.GeneratePreview(stepCtx, transcodedPath, previewPath)
	cancel()
	if err != nil {
		log.Warn().Err(err).Msg("Falha na geração de preview")
	} else {
		result.PreviewPath = previewPath
	}
	metrics.ProcessingStepDuration.WithLabelValues("preview").Observe(time.Since(start).Seconds())

	// 7. Segmentação para streaming (usa o input original para evitar dupla transcodificação)
	log.Info().Msg("Etapa 7/7: Segmentando para streaming")
	start = time.Now()
	streamingDir := filepath.Join(tempDir, "streaming")
	stepCtx, cancel = context.WithTimeout(ctx, stepTimeoutStreaming)
	err = processor_steps.SegmentForStreaming(stepCtx, inputPath, streamingDir)
	cancel()
	if err != nil {
		log.Warn().Err(err).Msg("Falha na segmentação para streaming")
	} else {
		result.StreamingDir = streamingDir
	}
	metrics.ProcessingStepDuration.WithLabelValues("streaming").Observe(time.Since(start).Seconds())

	log.Info().Msg("Pipeline de processamento concluído com sucesso")
	return result, nil
}
