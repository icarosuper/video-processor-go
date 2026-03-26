package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSegmentForStreaming_ValidVideo(t *testing.T) {
	inputPath := GenerateTestVideo(t, 5)
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming(context.Background(), inputPath, outputDir); err != nil {
		t.Fatalf("SegmentForStreaming() falhou: %v", err)
	}

	masterPath := filepath.Join(outputDir, "master.m3u8")
	if _, err := os.Stat(masterPath); os.IsNotExist(err) {
		t.Fatal("SegmentForStreaming() não criou master.m3u8")
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("erro ao ler diretório de saída: %v", err)
	}

	variantFound := false
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		varDir := filepath.Join(outputDir, e.Name())
		playlistPath := filepath.Join(varDir, "playlist.m3u8")
		if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
			t.Errorf("variante %s não tem playlist.m3u8", e.Name())
			continue
		}
		segs, _ := os.ReadDir(varDir)
		tsCount := 0
		for _, s := range segs {
			if strings.HasSuffix(s.Name(), ".ts") {
				tsCount++
			}
		}
		if tsCount == 0 {
			t.Errorf("variante %s não tem segmentos .ts", e.Name())
			continue
		}
		variantFound = true
	}
	if !variantFound {
		t.Error("SegmentForStreaming() não gerou nenhuma variante de resolução")
	}
}

func TestSegmentForStreaming_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming(context.Background(), invalidPath, outputDir); err == nil {
		t.Error("SegmentForStreaming() deveria falhar com entrada inválida")
	}
}

func TestSegmentForStreaming_NonExistentVideo(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "hls")

	if err := SegmentForStreaming(context.Background(), "/caminho/que/nao/existe.mp4", outputDir); err == nil {
		t.Error("SegmentForStreaming() deveria falhar com arquivo inexistente")
	}
}
