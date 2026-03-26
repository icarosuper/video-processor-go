package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateThumbnails_ValidVideo(t *testing.T) {
	videoPath := GenerateTestVideo(t, 5)
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	err := GenerateThumbnails(context.Background(), videoPath, outputDir)
	if err != nil {
		t.Fatalf("GenerateThumbnails() falhou: %v", err)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("GenerateThumbnails() não criou o diretório de saída")
	}

	expectedThumbnails := []string{
		"thumb_001.jpg", "thumb_002.jpg", "thumb_003.jpg",
		"thumb_004.jpg", "thumb_005.jpg",
	}

	for _, filename := range expectedThumbnails {
		thumbPath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
			t.Errorf("GenerateThumbnails() não criou %s", filename)
		}
		fileInfo, err := os.Stat(thumbPath)
		if err != nil {
			t.Fatalf("Erro ao verificar thumbnail %s: %v", filename, err)
		}
		if fileInfo.Size() == 0 {
			t.Errorf("Thumbnail %s está vazio", filename)
		}
	}
}

func TestGenerateThumbnails_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	err := GenerateThumbnails(context.Background(), invalidPath, outputDir)
	if err == nil {
		t.Error("GenerateThumbnails() deveria falhar com vídeo inválido")
	}
}

func TestGenerateThumbnails_NonExistentVideo(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	err := GenerateThumbnails(context.Background(), "/caminho/que/nao/existe.mp4", outputDir)
	if err == nil {
		t.Error("GenerateThumbnails() deveria falhar com arquivo inexistente")
	}
}
