package processor_steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVideo_ValidVideo(t *testing.T) {
	videoPath := GenerateTestVideo(t, 5)

	err := ValidateVideo(context.Background(), videoPath)
	if err != nil {
		t.Errorf("ValidateVideo() deveria ter sucesso com vídeo válido, mas retornou erro: %v", err)
	}
}

func TestValidateVideo_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)

	err := ValidateVideo(context.Background(), invalidPath)
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo inválido, mas teve sucesso")
	}
}

func TestValidateVideo_NonExistentFile(t *testing.T) {
	err := ValidateVideo(context.Background(), "/caminho/que/nao/existe.mp4")
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo inexistente, mas teve sucesso")
	}
}

func TestValidateVideo_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	emptyPath := filepath.Join(tempDir, "empty.mp4")

	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("Falha ao criar arquivo vazio: %v", err)
	}

	err := ValidateVideo(context.Background(), emptyPath)
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo vazio, mas teve sucesso")
	}
}
