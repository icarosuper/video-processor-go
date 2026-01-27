package processor_steps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SegmentForStreaming cria segmentos HLS para streaming adaptativo.
func SegmentForStreaming(inputPath, outputDir string) error {
	// Criar diretório de saída se não existir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório: %w", err)
	}

	playlistPath := filepath.Join(outputDir, "playlist.m3u8")
	segmentPath := filepath.Join(outputDir, "segment_%03d.ts")

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264", // Codec de vídeo H.264
		"-c:a", "aac", // Codec de áudio AAC
		"-f", "hls", // Formato HLS
		"-hls_time", "6", // 6 segundos por segmento
		"-hls_list_size", "0", // Manter todos os segmentos na playlist
		"-hls_segment_filename", segmentPath,
		"-y", // Sobrescrever
		playlistPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na segmentação: %w, output: %s", err, string(output))
	}

	return nil
}
