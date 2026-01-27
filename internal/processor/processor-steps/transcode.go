package processor_steps

import (
	"fmt"
	"os/exec"
)

// TranscodeVideo converte o vídeo para formatos padronizados (MP4, H.264, AAC).
func TranscodeVideo(inputPath, outputPath string) error {
	// Converter para MP4 com H.264 video e AAC audio
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264", // Codec de vídeo H.264
		"-preset", "medium", // Balance entre velocidade e compressão
		"-crf", "23", // Qualidade (lower = better, 18-28 é bom)
		"-c:a", "aac", // Codec de áudio
		"-b:a", "128k", // Bitrate de áudio
		"-movflags", "+faststart", // Otimização para streaming
		"-y", // Sobrescrever arquivo de saída
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na transcodificação: %w, output: %s", err, string(output))
	}

	return nil
}
