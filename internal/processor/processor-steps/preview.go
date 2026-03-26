package processor_steps

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GeneratePreview gera um preview de baixa qualidade do vídeo (primeiros 30 segundos ou 10% do vídeo).
func GeneratePreview(ctx context.Context, inputPath, outputPath string) error {
	durationCmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	durationOutput, err := durationCmd.Output()
	if err != nil {
		return fmt.Errorf("falha ao obter duração: %w", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil {
		return fmt.Errorf("duração inválida: %w", err)
	}

	previewDuration := duration
	if previewDuration > 30 {
		previewDuration = 30
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-t", strconv.FormatFloat(previewDuration, 'f', 0, 64),
		"-vf", "scale=640:-2",
		"-b:v", "500k",
		"-c:a", "aac",
		"-b:a", "64k",
		"-preset", "veryfast",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na geração de preview: %w, output: %s", err, string(output))
	}

	return nil
}
