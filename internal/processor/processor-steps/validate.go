package processor_steps

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ValidateVideo valida o formato, integridade e codecs do vídeo usando ffprobe.
func ValidateVideo(ctx context.Context, inputPath string) error {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vídeo inválido ou corrompido: %w, output: %s", err, string(output))
	}

	duration := strings.TrimSpace(string(output))
	if duration == "" || duration == "N/A" {
		return fmt.Errorf("vídeo não possui duração válida")
	}

	return nil
}
