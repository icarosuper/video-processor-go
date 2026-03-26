package processor_steps

import (
	"context"
	"fmt"
	"os/exec"
)

// ExtractAudio extrai a faixa de áudio do vídeo em formato MP3.
func ExtractAudio(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-ab", "192k",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na extração de áudio: %w, output: %s", err, string(output))
	}

	return nil
}
