package processor_steps

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GeneratePreview gera um preview de baixa qualidade do vídeo (primeiros 30 segundos ou 10% do vídeo).
func GeneratePreview(inputPath, outputPath string) error {
	// Obter duração do vídeo
	durationCmd := exec.Command("ffprobe",
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

	// Criar preview de 30 segundos ou duração total (o que for menor)
	previewDuration := duration
	if previewDuration > 30 {
		previewDuration = 30
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-t", strconv.FormatFloat(previewDuration, 'f', 0, 64), // Duração
		"-vf", "scale=640:-2", // Escalar para largura 640
		"-b:v", "500k", // Bitrate baixo
		"-c:a", "aac", // Áudio AAC
		"-b:a", "64k", // Bitrate de áudio baixo
		"-preset", "veryfast", // Processamento rápido
		"-y", // Sobrescrever
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na geração de preview: %w, output: %s", err, string(output))
	}

	return nil
}
