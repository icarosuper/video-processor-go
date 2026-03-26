package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"github.com/rs/zerolog/log"

	processor_steps "video-processor/internal/processor/processor-steps"
	"video-processor/internal/telemetry"
	"video-processor/metrics"
)

// Timeouts individuais por etapa do pipeline.
const (
	stepTimeoutValidate   = 30 * time.Second
	stepTimeoutAnalyze    = 30 * time.Second
	stepTimeoutTranscode  = 3 * time.Minute
	stepTimeoutThumbnails = 60 * time.Second
	stepTimeoutAudio      = 2 * time.Minute
	stepTimeoutPreview    = 2 * time.Minute
	stepTimeoutStreaming  = 4 * time.Minute
)

// ProcessingResult contém os caminhos dos artefatos gerados pelo pipeline.
// TempDir deve ser removido pelo chamador após os uploads.
type ProcessingResult struct {
	TempDir       string
	ThumbnailsDir string
	AudioPath     string
	PreviewPath   string
	StreamingDir  string
	Metadata      *processor_steps.VideoMetadata
}

// ProcessVideo executa todas as etapas do pipeline de processamento de vídeo.
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
	if err := runStep(ctx, "validate", stepTimeoutValidate, func(stepCtx context.Context) error {
		return processor_steps.ValidateVideo(stepCtx, inputPath)
	}); err != nil {
		return result, fmt.Errorf("validação falhou: %w", err)
	}

	// 2. Análise de conteúdo
	log.Info().Msg("Etapa 2/7: Analisando conteúdo")
	_ = runStep(ctx, "analyze", stepTimeoutAnalyze, func(stepCtx context.Context) error {
		metadata, err := processor_steps.AnalyzeContent(stepCtx, inputPath)
		if err != nil {
			log.Warn().Err(err).Msg("Falha na análise de conteúdo")
			return err
		}
		result.Metadata = metadata
		return nil
	})

	// 3. Transcodificação (etapa crítica)
	log.Info().Msg("Etapa 3/7: Transcodificando vídeo")
	if err := runStep(ctx, "transcode", stepTimeoutTranscode, func(stepCtx context.Context) error {
		return processor_steps.TranscodeVideo(stepCtx, inputPath, outputPath)
	}); err != nil {
		return result, fmt.Errorf("transcodificação falhou: %w", err)
	}

	transcodedPath := outputPath

	// 4. Geração de thumbnails
	log.Info().Msg("Etapa 4/7: Gerando thumbnails")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := runStep(ctx, "thumbnails", stepTimeoutThumbnails, func(stepCtx context.Context) error {
		return processor_steps.GenerateThumbnails(stepCtx, transcodedPath, thumbnailsDir)
	}); err != nil {
		log.Warn().Err(err).Msg("Falha ao gerar thumbnails")
	} else {
		result.ThumbnailsDir = thumbnailsDir
	}

	// 5. Extração de áudio
	log.Info().Msg("Etapa 5/7: Extraindo áudio")
	audioPath := filepath.Join(tempDir, "audio.mp3")
	if err := runStep(ctx, "audio", stepTimeoutAudio, func(stepCtx context.Context) error {
		return processor_steps.ExtractAudio(stepCtx, transcodedPath, audioPath)
	}); err != nil {
		log.Warn().Err(err).Msg("Falha na extração de áudio")
	} else {
		result.AudioPath = audioPath
	}

	// 6. Geração de preview
	log.Info().Msg("Etapa 6/7: Gerando preview")
	previewPath := filepath.Join(tempDir, "preview.mp4")
	if err := runStep(ctx, "preview", stepTimeoutPreview, func(stepCtx context.Context) error {
		return processor_steps.GeneratePreview(stepCtx, transcodedPath, previewPath)
	}); err != nil {
		log.Warn().Err(err).Msg("Falha na geração de preview")
	} else {
		result.PreviewPath = previewPath
	}

	// 7. Segmentação para streaming
	log.Info().Msg("Etapa 7/7: Segmentando para streaming")
	streamingDir := filepath.Join(tempDir, "streaming")
	if err := runStep(ctx, "streaming", stepTimeoutStreaming, func(stepCtx context.Context) error {
		return processor_steps.SegmentForStreaming(stepCtx, inputPath, streamingDir)
	}); err != nil {
		log.Warn().Err(err).Msg("Falha na segmentação para streaming")
	} else {
		result.StreamingDir = streamingDir
	}

	log.Info().Msg("Pipeline de processamento concluído com sucesso")
	return result, nil
}

// runStep executa uma etapa do pipeline dentro de um span OTel e registra duração via Prometheus.
// O span é marcado como erro se a etapa falhar.
func runStep(ctx context.Context, name string, timeout time.Duration, fn func(context.Context) error) error {
	stepCtx, span := telemetry.Tracer().Start(ctx, "step/"+name,
		oteltrace.WithAttributes(attribute.String("step.name", name)),
	)
	defer span.End()

	stepCtx, cancel := context.WithTimeout(stepCtx, timeout)
	defer cancel()

	start := time.Now()
	err := fn(stepCtx)
	metrics.ProcessingStepDuration.WithLabelValues(name).Observe(time.Since(start).Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
