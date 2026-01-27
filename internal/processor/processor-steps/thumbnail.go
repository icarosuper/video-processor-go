package processor_steps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ThumbnailConfig define as configurações para geração de thumbnails.
type ThumbnailConfig struct {
	Count  int // Número de thumbnails (ex: 5)
	Width  int // Largura em pixels (ex: 320)
	Height int // Altura em pixels (ex: 180)
}

// GenerateThumbnails gera thumbnails do vídeo em múltiplos timestamps.
func GenerateThumbnails(inputPath, outputDir string) error {
	// Configuração padrão
	config := ThumbnailConfig{
		Count:  5,
		Width:  320,
		Height: 180,
	}

	// Criar diretório de saída se não existir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório de thumbnails: %w", err)
	}

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

	// Gerar thumbnails em intervalos regulares
	interval := duration / float64(config.Count+1)

	for i := 1; i <= config.Count; i++ {
		timestamp := interval * float64(i)
		thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%03d.jpg", i))

		cmd := exec.Command("ffmpeg",
			"-ss", strconv.FormatFloat(timestamp, 'f', 2, 64), // Timestamp
			"-i", inputPath,
			"-vframes", "1", // Um único frame
			"-vf", fmt.Sprintf("scale=%d:%d", config.Width, config.Height),
			"-y", // Sobrescrever
			thumbnailPath,
		)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("falha ao gerar thumbnail %d: %w", i, err)
		}
	}

	return nil
}
