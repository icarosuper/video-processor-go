package processor_steps

import (
	"context"
	"testing"
)

func TestAnalyzeContent_ValidVideo(t *testing.T) {
	videoPath := GenerateTestVideo(t, 5)

	metadata, err := AnalyzeContent(context.Background(), videoPath)
	if err != nil {
		t.Fatalf("AnalyzeContent() falhou: %v", err)
	}
	if metadata == nil {
		t.Fatal("AnalyzeContent() retornou metadata nil")
	}
	if metadata.Duration <= 0 {
		t.Errorf("AnalyzeContent() retornou duração inválida: %v", metadata.Duration)
	}
	if metadata.Width == 0 || metadata.Height == 0 {
		t.Errorf("AnalyzeContent() retornou resolução inválida: %dx%d", metadata.Width, metadata.Height)
	}
	if metadata.VideoCodec == "" {
		t.Error("AnalyzeContent() não retornou codec de vídeo")
	}
}

func TestAnalyzeContent_InvalidVideo(t *testing.T) {
	invalidPath := CreateInvalidFile(t)

	metadata, err := AnalyzeContent(context.Background(), invalidPath)
	if err == nil {
		t.Error("AnalyzeContent() deveria falhar com vídeo inválido")
	}
	if metadata != nil {
		t.Error("AnalyzeContent() deveria retornar nil para vídeo inválido")
	}
}

func TestAnalyzeContent_NonExistentVideo(t *testing.T) {
	metadata, err := AnalyzeContent(context.Background(), "/caminho/que/nao/existe.mp4")
	if err == nil {
		t.Error("AnalyzeContent() deveria falhar com arquivo inexistente")
	}
	if metadata != nil {
		t.Error("AnalyzeContent() deveria retornar nil para arquivo inexistente")
	}
}
