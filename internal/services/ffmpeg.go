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
		"-vf", "scale='if(gt(iw,ih),510,-1)':'if(gt(iw,ih),-1,510)',format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
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
		"-vf", "scale='if(gt(iw,ih),510,-1)':'if(gt(iw,ih),-1,510)',fps=15,format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
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
// Uses ImageMagick for flawless word-wrapping and auto-scaling, then FFmpeg for animated WebP overlay.
func (f *FFmpegService) AddTextToWebP(inputData []byte, text string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-ts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.webp")
	textOverlayPath := filepath.Join(tmpDir, "text.png")
	outputPath := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	// 1. Generate text overlay image with ImageMagick (handles word-wrap and auto-scale).
	// We use 480x180 to leave a 16px margin on 512x512 stickers.
	imBin := "magick"
	if _, err := exec.LookPath(imBin); err != nil {
		imBin = "convert"
	}
	imArgs := []string{
		"-background", "none",
		"-fill", "white",
		"-stroke", "black",
		"-strokewidth", "3",
		"-font", "DejaVu-Sans-Bold", // System default bold fallback
		"-gravity", "center",
		"-size", "480x180",
		fmt.Sprintf("caption:%s", text),
		textOverlayPath,
	}
	imCmd := exec.Command(imBin, imArgs...)
	if out, err := imCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("imagemagick text generation failed: %w\nOutput: %s", err, string(out))
	}

	// 2. Overlay the text image onto the WebP using FFmpeg.
	// We place it at the bottom-center: x=(W-w)/2, y=H-h-16
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-i", textOverlayPath,
		"-filter_complex", "[0:v][1:v]overlay=(W-w)/2:H-h-16",
		"-c:v", "libwebp",
		"-preset", "default",
		"-quality", "80",
		"-loop", "0",
		"-an", "-vsync", "0",
		"-y", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg overlay failed: %w\nOutput: %s", err, string(output))
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

	outputPath := filepath.Join(tmpDir, "brat.png") // Output as PNG first

	// The user requested white background with black text matching the photo.
	// ImageMagick 'caption:' auto-wraps and scales text to fit the box.
	args := []string{
		"-background", "white",
		"-fill", "black",
		"-font", "Arial",       // Try Arial first
		"-size", "450x450",     // Tighter box so it doesn't touch edges
		"-gravity", "West",     // Left aligned, vertically centered
		fmt.Sprintf(`caption:%s`, text),
		"-filter", "box",
		"-blur", "0x2.5",       // More blur as requested ("agak blur")
		"-bordercolor", "white", 
		"-border", "31",        // 450 + 31*2 = 512
		"-resize", "512x512!",  // Ensure exact 512x512 sticker size
		"-strip",               // STRIP ALL METADATA/ICC PROFILES to prevent WhatsApp Mobile crash!
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

	pngData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw brat image: %w", err)
	}

	// Transcode with FFmpeg to guarantee mobile WhatsApp WebP compatibility
	return f.ImageToWebP(pngData)
}
