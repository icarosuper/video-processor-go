package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAudio_ValidVideo(t *testing.T) {
	inputPath := GenerateTestVideo(t, 5)
	outputPath := filepath.Join(t.TempDir(), "audio.mp3")

	if err := ExtractAudio(context.Background(), inputPath, outputPath); err != nil {
		t.Fatalf("ExtractAudio() falhou: %v", err)
	}

	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		t.Fatal("ExtractAudio() não criou o arquivo de saída")
	}
	if info.Size() == 0 {
		t.Error("ExtractAudio() criou arquivo de áudio vazio")
	}
}

func TestExtractAudio_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputPath := filepath.Join(t.TempDir(), "audio.mp3")

	if err := ExtractAudio(context.Background(), invalidPath, outputPath); err == nil {
		t.Error("ExtractAudio() deveria falhar com entrada inválida")
	}
}

func TestExtractAudio_NonExistentVideo(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "audio.mp3")

	if err := ExtractAudio(context.Background(), "/caminho/que/nao/existe.mp4", outputPath); err == nil {
		t.Error("ExtractAudio() deveria falhar com arquivo inexistente")
	}
}
