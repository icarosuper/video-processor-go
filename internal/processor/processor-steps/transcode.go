package processor_steps

import (
	"context"
	"fmt"
	"os/exec"
)

// TranscodeVideo converts the video to standardized formats (MP4, H.264, AAC).
func TranscodeVideo(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("transcoding failed: %w, output: %s", err, string(output))
	}

	return nil
}
