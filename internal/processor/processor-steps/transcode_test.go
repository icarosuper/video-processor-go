package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscodeVideo_ValidVideo(t *testing.T) {
	inputPath := GenerateTestVideo(t, 5)
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	err := TranscodeVideo(context.Background(), inputPath, outputPath)
	if err != nil {
		t.Fatalf("TranscodeVideo() falhou: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("TranscodeVideo() não criou o arquivo de saída")
	}

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Erro ao verificar arquivo de saída: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("TranscodeVideo() criou arquivo de saída vazio")
	}

	if err := ValidateVideo(context.Background(), outputPath); err != nil {
		t.Errorf("TranscodeVideo() produziu vídeo inválido: %v", err)
	}
}

func TestTranscodeVideo_InvalidInput(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	err := TranscodeVideo(context.Background(), invalidPath, outputPath)
	if err == nil {
		t.Error("TranscodeVideo() deveria falhar com entrada inválida")
	}
}

func TestTranscodeVideo_NonExistentInput(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	err := TranscodeVideo(context.Background(), "/caminho/que/nao/existe.mp4", outputPath)
	if err == nil {
		t.Error("TranscodeVideo() deveria falhar com arquivo inexistente")
	}
}
