package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePreview_ValidVideo(t *testing.T) {
	inputPath := GenerateTestVideo(t, 5)
	outputPath := filepath.Join(t.TempDir(), "preview.mp4")

	if err := GeneratePreview(context.Background(), inputPath, outputPath); err != nil {
		t.Fatalf("GeneratePreview() falhou: %v", err)
	}

	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		t.Fatal("GeneratePreview() não criou o arquivo de saída")
	}
	if info.Size() == 0 {
		t.Error("GeneratePreview() criou arquivo de preview vazio")
	}

	if err := ValidateVideo(context.Background(), outputPath); err != nil {
		t.Errorf("GeneratePreview() produziu vídeo inválido: %v", err)
	}
}

func TestGeneratePreview_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputPath := filepath.Join(t.TempDir(), "preview.mp4")

	if err := GeneratePreview(context.Background(), invalidPath, outputPath); err == nil {
		t.Error("GeneratePreview() deveria falhar com entrada inválida")
	}
}

func TestGeneratePreview_NonExistentVideo(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "preview.mp4")

	if err := GeneratePreview(context.Background(), "/caminho/que/nao/existe.mp4", outputPath); err == nil {
		t.Error("GeneratePreview() deveria falhar com arquivo inexistente")
	}
}
