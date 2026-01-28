package processor_steps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscodeVideo_ValidVideo(t *testing.T) {
	// Gerar vídeo de teste
	inputPath := GenerateTestVideo(t, 5)

	// Definir caminho de saída
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	// Transcodificar
	err := TranscodeVideo(inputPath, outputPath)
	if err != nil {
		t.Fatalf("TranscodeVideo() falhou: %v", err)
	}

	// Verificar se o arquivo de saída foi criado
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("TranscodeVideo() não criou o arquivo de saída")
	}

	// Verificar se o arquivo de saída tem tamanho > 0
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Erro ao verificar arquivo de saída: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("TranscodeVideo() criou arquivo de saída vazio")
	}

	// Validar que o arquivo de saída é um vídeo válido
	if err := ValidateVideo(outputPath); err != nil {
		t.Errorf("TranscodeVideo() produziu vídeo inválido: %v", err)
	}
}

func TestTranscodeVideo_InvalidInput(t *testing.T) {
	// Criar arquivo inválido
	invalidPath := CreateInvalidFile(t)
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	// Transcodificar
	err := TranscodeVideo(invalidPath, outputPath)
	if err == nil {
		t.Error("TranscodeVideo() deveria falhar com entrada inválida")
	}
}

func TestTranscodeVideo_NonExistentInput(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "output.mp4")

	// Transcodificar arquivo inexistente
	err := TranscodeVideo("/caminho/que/nao/existe.mp4", outputPath)
	if err == nil {
		t.Error("TranscodeVideo() deveria falhar com arquivo inexistente")
	}
}
