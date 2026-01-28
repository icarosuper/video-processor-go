package processor_steps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateThumbnails_ValidVideo(t *testing.T) {
	// Gerar vídeo de teste
	videoPath := GenerateTestVideo(t, 5)

	// Definir diretório de saída
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	// Gerar thumbnails
	err := GenerateThumbnails(videoPath, outputDir)
	if err != nil {
		t.Fatalf("GenerateThumbnails() falhou: %v", err)
	}

	// Verificar se o diretório foi criado
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("GenerateThumbnails() não criou o diretório de saída")
	}

	// Verificar se os thumbnails foram criados (esperamos 5 thumbnails)
	expectedThumbnails := []string{
		"thumb_001.jpg",
		"thumb_002.jpg",
		"thumb_003.jpg",
		"thumb_004.jpg",
		"thumb_005.jpg",
	}

	for _, filename := range expectedThumbnails {
		thumbPath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
			t.Errorf("GenerateThumbnails() não criou %s", filename)
		}

		// Verificar se o arquivo tem tamanho > 0
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
	// Criar arquivo inválido
	invalidPath := CreateInvalidFile(t)
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	// Tentar gerar thumbnails
	err := GenerateThumbnails(invalidPath, outputDir)
	if err == nil {
		t.Error("GenerateThumbnails() deveria falhar com vídeo inválido")
	}
}

func TestGenerateThumbnails_NonExistentVideo(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "thumbnails")

	// Tentar gerar thumbnails de arquivo inexistente
	err := GenerateThumbnails("/caminho/que/nao/existe.mp4", outputDir)
	if err == nil {
		t.Error("GenerateThumbnails() deveria falhar com arquivo inexistente")
	}
}
