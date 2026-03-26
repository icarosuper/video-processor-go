package processor_steps

import (
	"context"
	"fmt"
	"os/exec"
)

// TranscodeVideo converte o vídeo para formatos padronizados (MP4, H.264, AAC).
func TranscodeVideo(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na transcodificação: %w, output: %s", err, string(output))
	}

	return nil
}
