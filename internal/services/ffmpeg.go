package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FFmpegService provides methods to convert media using ffmpeg.
type FFmpegService struct{}

// NewFFmpegService creates a new FFmpegService.
func NewFFmpegService() *FFmpegService {
	return &FFmpegService{}
}

// ImageToWebP converts an image (JPEG/PNG) to a static WebP sticker (512x512 max).
func (f *FFmpegService) ImageToWebP(inputData []byte) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input")
	outputPath := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", "scale='if(gt(iw,ih),512,-1)':'if(gt(iw,ih),-1,512)',format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
		"-c:v", "libwebp",
		"-preset", "default",
		"-loop", "0",
		"-an", "-vsync", "0",
		"-quality", "80",
		"-y", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg image->webp failed: %w\nOutput: %s", err, string(output))
	}

	return os.ReadFile(outputPath)
}

// VideoToWebP converts a video/GIF to an animated WebP sticker.
func (f *FFmpegService) VideoToWebP(inputData []byte, ext string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if ext == "" {
		ext = ".mp4"
	}
	inputPath := filepath.Join(tmpDir, "input"+ext)
	outputPath := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Limit to 8 seconds max, scale to 512x512 max, 15 fps for smaller size.
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-t", "8",
		"-vf", "scale='if(gt(iw,ih),512,-1)':'if(gt(iw,ih),-1,512)',fps=15,format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
		"-c:v", "libwebp",
		"-preset", "default",
		"-loop", "0",
		"-an", "-vsync", "0",
		"-quality", "50",
		"-compression_level", "6",
		"-y", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg video->webp failed: %w\nOutput: %s", err, string(output))
	}

	return os.ReadFile(outputPath)
}

// WebPToImage converts a WebP file to PNG.
func (f *FFmpegService) WebPToImage(inputData []byte) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.webp")
	outputPath := filepath.Join(tmpDir, "output.png")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-frames:v", "1",
		"-y", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg webp->png failed: %w\nOutput: %s", err, string(output))
	}

	return os.ReadFile(outputPath)
}
