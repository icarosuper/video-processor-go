package processor_steps

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// GenerateTestVideo cria um vídeo de teste usando FFmpeg para uso em testes
func GenerateTestVideo(t *testing.T, duration int) string {
	t.Helper()

	// Criar diretório temporário
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "test_video.mp4")

	// Verificar se FFmpeg está disponível
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg não está disponível - pulando teste")
	}

	// Gerar vídeo de teste com FFmpeg
	// testsrc: gera padrão de teste visual
	// sine: gera áudio de teste
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "testsrc=duration=5:size=640x480:rate=30",
		"-f", "lavfi",
		"-i", "sine=frequency=1000:duration=5",
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-y",
		videoPath,
	)

	// Executar comando silenciosamente
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		t.Fatalf("Falha ao gerar vídeo de teste: %v", err)
	}

	return videoPath
}

// CreateInvalidFile cria um arquivo inválido para testes de erro
func CreateInvalidFile(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	invalidPath := filepath.Join(tempDir, "invalid.mp4")

	// Criar arquivo com conteúdo inválido
	if err := os.WriteFile(invalidPath, []byte("not a valid video"), 0644); err != nil {
		t.Fatalf("Falha ao criar arquivo inválido: %v", err)
	}

	return invalidPath
}
