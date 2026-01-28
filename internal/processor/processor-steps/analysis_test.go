package processor_steps

import (
	"testing"
)

func TestAnalyzeContent_ValidVideo(t *testing.T) {
	// Gerar vídeo de teste
	videoPath := GenerateTestVideo(t, 5)

	// Analisar conteúdo
	err := AnalyzeContent(videoPath)
	if err != nil {
		t.Fatalf("AnalyzeContent() falhou: %v", err)
	}

	// A função atualmente apenas loga os metadados, não retorna nada
	// Este teste verifica que a análise não falha com vídeo válido
}

func TestAnalyzeContent_InvalidVideo(t *testing.T) {
	// Criar arquivo inválido
	invalidPath := CreateInvalidFile(t)

	// Analisar conteúdo
	err := AnalyzeContent(invalidPath)
	if err == nil {
		t.Error("AnalyzeContent() deveria falhar com vídeo inválido")
	}
}

func TestAnalyzeContent_NonExistentVideo(t *testing.T) {
	// Tentar analisar arquivo inexistente
	err := AnalyzeContent("/caminho/que/nao/existe.mp4")
	if err == nil {
		t.Error("AnalyzeContent() deveria falhar com arquivo inexistente")
	}
}
