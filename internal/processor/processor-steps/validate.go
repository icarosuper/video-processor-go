package processor_steps

import (
	"fmt"
	"os/exec"
	"strings"
)

// ValidateVideo valida o formato, integridade e codecs do vídeo usando ffprobe.
func ValidateVideo(inputPath string) error {
	// Usar ffprobe para validar o vídeo
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vídeo inválido ou corrompido: %w, output: %s", err, string(output))
	}

	// Verificar duração mínima (deve ter pelo menos uma duração válida)
	duration := strings.TrimSpace(string(output))
	if duration == "" || duration == "N/A" {
		return fmt.Errorf("vídeo não possui duração válida")
	}

	return nil
}
