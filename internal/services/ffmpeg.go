package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"chisa_bot/internal/config"
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

// AddTextToWebP overlays meme-style bottom text onto a WebP sticker.
// Text is rendered in white with a black border, centered at the bottom.
func (f *FFmpegService) AddTextToWebP(inputData []byte, text string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-ts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.webp")
	outputPath := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	// Escape special characters for FFmpeg drawtext.
	safeText := escapeFfmpegText(text)

	// drawtext: white text, black border, centered at bottom.
	drawFilter := fmt.Sprintf(
		"drawtext=fontfile=%s:text='%s':fontcolor=white:fontsize=72:bordercolor=black:borderw=6:x=(w-text_w)/2:y=h-text_h-20",
		config.MemeFontPath, safeText,
	)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", drawFilter,
		"-c:v", "libwebp",
		"-preset", "default",
		"-quality", "80",
		"-loop", "0",
		"-an", "-vsync", "0",
		"-y", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg drawtext failed: %w\nOutput: %s", err, string(output))
	}

	return os.ReadFile(outputPath)
}

// escapeFfmpegText escapes characters that are special in FFmpeg drawtext.
func escapeFfmpegText(s string) string {
	// Order matters: escape backslash first.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "'", `\'`)
	s = strings.ReplaceAll(s, ":", `\:`)
	s = strings.ReplaceAll(s, "[", `\[`)
	s = strings.ReplaceAll(s, "]", `\]`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, "%", "%%")
	return s
}

// GenerateBratSticker generates a brat-style sticker (white background, black Arial/sans text, auto-wrapped).
func (f *FFmpegService) GenerateBratSticker(text string) ([]byte, error) {
	// Create a temporary output file for the result
	tmpDir, err := os.MkdirTemp("", "chisabot-brat-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "brat.webp")

	// The user requested white background with black text matching the photo.
	// ImageMagick 'caption:' auto-wraps and scales text to fit the box.
	args := []string{
		"-background", "white",
		"-fill", "black",
		"-font", "DejaVu-Sans", // Fallback font that works globally
		"-size", "512x512",
		"-gravity", "center",
		fmt.Sprintf(`caption:%s`, text),
		"-filter", "box",
		"-blur", "0x1",
		"-trim",       // Trim exact text boundary
		"-bordercolor", "white", 
		"-border", "50", // Add 50px border margin
		"-resize", "512x512>", // Resize while preserving aspect ratio inside a 512x512 box
		"-gravity", "center",
		"-background", "white",
		"-extent", "512x512", // Pad to a perfect 512x512 square
		outputPath,
	}

	// Try magick first, fallback to convert
	bin := "magick"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "convert"
	}

	cmd := exec.Command(bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("imagemagick brat generation failed: %w\nOutput: %s", err, string(output))
	}

	return os.ReadFile(outputPath)
}
