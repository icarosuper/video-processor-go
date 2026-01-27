package processor_steps

import (
	"fmt"
	"os/exec"
)

// ExtractAudio extrai a faixa de áudio do vídeo em formato MP3.
func ExtractAudio(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vn",                   // No video
		"-acodec", "libmp3lame", // Codec MP3
		"-ab", "192k", // Bitrate de áudio
		"-y", // Sobrescrever
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na extração de áudio: %w, output: %s", err, string(output))
	}

	return nil
}
