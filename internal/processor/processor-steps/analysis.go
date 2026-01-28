package processor_steps

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// VideoMetadata contém metadados extraídos do vídeo.
type VideoMetadata struct {
	Duration   float64 `json:"duration"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	VideoCodec string  `json:"video_codec"`
	AudioCodec string  `json:"audio_codec"`
	FPS        float64 `json:"fps"`
	Bitrate    int64   `json:"bitrate"`
	Size       int64   `json:"size"`
}

// AnalyzeContent extrai metadados e informações técnicas do vídeo.
func AnalyzeContent(inputPath string) error {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("falha na análise: %w", err)
	}

	// Parse JSON output
	var probeData struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return fmt.Errorf("falha ao parsear JSON: %w", err)
	}

	// Extrair metadados
	metadata := &VideoMetadata{}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "video" {
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			metadata.VideoCodec = stream.CodecName

			// Parse FPS (format: "30000/1001")
			if parts := strings.Split(stream.RFrameRate, "/"); len(parts) == 2 {
				numerator, _ := strconv.ParseFloat(parts[0], 64)
				denominator, _ := strconv.ParseFloat(parts[1], 64)
				if denominator != 0 {
					metadata.FPS = numerator / denominator
				}
			}
		} else if stream.CodecType == "audio" {
			metadata.AudioCodec = stream.CodecName
		}
	}

	// Parse duration, size, bitrate
	metadata.Duration, _ = strconv.ParseFloat(probeData.Format.Duration, 64)
	metadata.Size, _ = strconv.ParseInt(probeData.Format.Size, 10, 64)
	metadata.Bitrate, _ = strconv.ParseInt(probeData.Format.BitRate, 10, 64)

	// Log metadados para debug
	log.Info().
		Float64("duration", metadata.Duration).
		Int("width", metadata.Width).
		Int("height", metadata.Height).
		Str("videoCodec", metadata.VideoCodec).
		Str("audioCodec", metadata.AudioCodec).
		Float64("fps", metadata.FPS).
		Int64("bitrate", metadata.Bitrate).
		Int64("size", metadata.Size).
		Msg("Metadados do vídeo extraídos")

	return nil
}
