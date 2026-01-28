package processor_steps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVideo_ValidVideo(t *testing.T) {
	// Gerar vídeo de teste válido
	videoPath := GenerateTestVideo(t, 5)

	// Validar
	err := ValidateVideo(videoPath)
	if err != nil {
		t.Errorf("ValidateVideo() deveria ter sucesso com vídeo válido, mas retornou erro: %v", err)
	}
}

func TestValidateVideo_InvalidVideo(t *testing.T) {
	// Criar arquivo inválido
	invalidPath := CreateInvalidFile(t)

	// Validar
	err := ValidateVideo(invalidPath)
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo inválido, mas teve sucesso")
	}
}

func TestValidateVideo_NonExistentFile(t *testing.T) {
	// Testar com arquivo que não existe
	err := ValidateVideo("/caminho/que/nao/existe.mp4")
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo inexistente, mas teve sucesso")
	}
}

func TestValidateVideo_EmptyFile(t *testing.T) {
	// Criar arquivo vazio
	tempDir := t.TempDir()
	emptyPath := filepath.Join(tempDir, "empty.mp4")

	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("Falha ao criar arquivo vazio: %v", err)
	}

	// Validar
	err := ValidateVideo(emptyPath)
	if err == nil {
		t.Error("ValidateVideo() deveria falhar com arquivo vazio, mas teve sucesso")
	}
}
