package processor_steps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// hlsVariant define uma variante de qualidade para o streaming adaptativo.
type hlsVariant struct {
	Name         string
	Height       int
	VideoBitrate string
	AudioBitrate string
	Bandwidth    int // bits/s para o EXT-X-STREAM-INF
}

// hlsVariants lista as resoluções suportadas em ordem crescente.
// Só são geradas as variantes com Height <= altura do vídeo original.
var hlsVariants = []hlsVariant{
	{"240p", 240, "400k", "64k", 464000},
	{"360p", 360, "800k", "96k", 896000},
	{"480p", 480, "1400k", "128k", 1528000},
	{"720p", 720, "2800k", "128k", 2928000},
	{"1080p", 1080, "5000k", "192k", 5192000},
}

// SegmentForStreaming gera segmentos HLS adaptativos para múltiplas resoluções.
// A estrutura de saída em outputDir é:
//
//	master.m3u8          — playlist mestre referenciando as variantes
//	{resolução}/
//	  playlist.m3u8      — playlist de cada variante
//	  seg_000.ts, ...    — segmentos de vídeo
func SegmentForStreaming(ctx context.Context, inputPath, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório: %w", err)
	}

	sourceHeight := probeSourceHeight(ctx, inputPath)

	var selected []hlsVariant
	for _, v := range hlsVariants {
		if sourceHeight == 0 || v.Height <= sourceHeight {
			selected = append(selected, v)
		}
	}
	if len(selected) == 0 {
		selected = hlsVariants[:1]
	}

	for _, v := range selected {
		varDir := filepath.Join(outputDir, v.Name)
		if err := os.MkdirAll(varDir, 0755); err != nil {
			return fmt.Errorf("falha ao criar diretório %s: %w", v.Name, err)
		}
		if err := transcodeHLSVariant(ctx, inputPath, varDir, v); err != nil {
			return err
		}
	}

	return writeMasterPlaylist(outputDir, selected)
}

func transcodeHLSVariant(ctx context.Context, inputPath, varDir string, v hlsVariant) error {
	playlistPath := filepath.Join(varDir, "playlist.m3u8")
	segmentPath := filepath.Join(varDir, "seg_%03d.ts")

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-vf", fmt.Sprintf("scale=-2:%d", v.Height),
		"-b:v", v.VideoBitrate,
		"-c:a", "aac",
		"-b:a", v.AudioBitrate,
		"-f", "hls",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", segmentPath,
		"-y",
		playlistPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha na segmentação %s: %w, output: %s", v.Name, err, string(output))
	}
	return nil
}

func writeMasterPlaylist(outputDir string, variants []hlsVariant) error {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n\n")
	for _, v := range variants {
		fmt.Fprintf(&sb, "#EXT-X-STREAM-INF:BANDWIDTH=%d\n%s/playlist.m3u8\n", v.Bandwidth, v.Name)
	}
	masterPath := filepath.Join(outputDir, "master.m3u8")
	return os.WriteFile(masterPath, []byte(sb.String()), 0644)
}

func probeSourceHeight(ctx context.Context, inputPath string) int {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-select_streams", "v:0",
		"-show_entries", "stream=height",
		"-of", "csv=p=0",
		inputPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	h, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return h
}
