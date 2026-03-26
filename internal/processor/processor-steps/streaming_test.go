package processor_steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSegmentForStreaming_ValidVideo(t *testing.T) {
	inputPath := GenerateTestVideo(t, 5)
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming(inputPath, outputDir); err != nil {
		t.Fatalf("SegmentForStreaming() falhou: %v", err)
	}

	// Verificar playlist
	playlistPath := filepath.Join(outputDir, "playlist.m3u8")
	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		t.Fatal("SegmentForStreaming() não criou playlist.m3u8")
	}

	// Verificar se há pelo menos um segmento .ts
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("erro ao ler diretório de saída: %v", err)
	}
	tsCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ts") {
			tsCount++
		}
	}
	if tsCount == 0 {
		t.Error("SegmentForStreaming() não gerou nenhum segmento .ts")
	}
}

func TestSegmentForStreaming_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming(invalidPath, outputDir); err == nil {
		t.Error("SegmentForStreaming() deveria falhar com entrada inválida")
	}
}

func TestSegmentForStreaming_NonExistentVideo(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming("/caminho/que/nao/existe.mp4", outputDir); err == nil {
		t.Error("SegmentForStreaming() deveria falhar com arquivo inexistente")
	}
}
