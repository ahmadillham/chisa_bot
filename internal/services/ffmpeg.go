package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"chisa_bot/internal/config"
)

// FFmpegService provides methods to convert media using ffmpeg.
type FFmpegService struct {
	fontPath string
}

// NewFFmpegService creates a new FFmpegService and auto-discovers a suitable font.
func NewFFmpegService() *FFmpegService {
	f := &FFmpegService{}
	for _, path := range config.MemeFontCandidates {
		if _, err := os.Stat(path); err == nil {
			f.fontPath = path
			break
		}
	}
	if f.fontPath == "" {
		// Fallback to the default hardcoded one if all else fails,
		// though it will likely fail during execution too.
		f.fontPath = config.MemeFontPath
	}
	return f
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-vf", "scale='if(gt(iw,ih),510,-2)':'if(gt(iw,ih),-2,510)',format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-t", "8",
		"-vf", "scale='if(gt(iw,ih),510,-2)':'if(gt(iw,ih),-2,510)',fps=15,format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000",
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
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

// wrapText inserts line breaks so text fits approximately within maxChars per line.
func wrapText(text string, maxChars int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= maxChars {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return strings.Join(lines, "\n")
}

// AddTextToWebP overlays meme-style bottom text onto a WebP sticker.
// It supports both static and animated inputs (GIF/Video/WebP).
func (f *FFmpegService) AddTextToWebP(inputData []byte, text string, ext string, isAnimated bool) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-ts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if ext == "" {
		ext = ".webp"
	}
	inputPath := filepath.Join(tmpDir, "input"+ext)
	outputPath := filepath.Join(tmpDir, "output.webp")

	if err := os.WriteFile(inputPath, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	var bottomText, topText string
	// Support both comma and pipe as separators
	normalizedText := strings.ReplaceAll(text, ",", "|")
	parts := strings.Split(normalizedText, "|")

	if len(parts) > 1 {
		topText = strings.TrimSpace(parts[0])
		bottomText = strings.TrimSpace(parts[1])
	} else {
		bottomText = strings.TrimSpace(parts[0])
	}

	// Base scaling and padding for sticker format. We use format=bgra to ensure alpha channel support.
	drawFilter := "scale='if(gt(iw,ih),510,-2)':'if(gt(iw,ih),-2,510)',format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000"

	// If animated, limit FPS and add fps filter early
	if isAnimated {
		drawFilter += ",fps=15"
	}

	// Double-escape backslashes for FFmpeg's drawtext fontfile path
	fontPath := strings.ReplaceAll(f.fontPath, `\`, `\\`)
	fontPath = strings.ReplaceAll(fontPath, `:`, `\:`)

	if topText != "" {
		safeTop := escapeFfmpegText(wrapText(topText, 13))
		drawFilter += fmt.Sprintf(",drawtext=fontfile='%s':text='%s':fontcolor=white:fontsize=72:bordercolor=black:borderw=6:x=(w-text_w)/2:y=20:line_spacing=5:text_align=C", fontPath, safeTop)
	}

	if bottomText != "" {
		safeBottom := escapeFfmpegText(wrapText(bottomText, 13))
		drawFilter += fmt.Sprintf(",drawtext=fontfile='%s':text='%s':fontcolor=white:fontsize=72:bordercolor=black:borderw=6:x=(w-text_w)/2:y=h-text_h-20:line_spacing=5:text_align=C", fontPath, safeBottom)
	}

	ffmpegArgs := []string{"-i", inputPath}

	if isAnimated {
		ffmpegArgs = append(ffmpegArgs, "-t", "8")
	}

	ffmpegArgs = append(ffmpegArgs,
		"-vf", drawFilter,
		"-c:v", "libwebp",
		"-preset", "default",
	)

	if isAnimated {
		ffmpegArgs = append(ffmpegArgs, "-quality", "50", "-compression_level", "6")
	} else {
		ffmpegArgs = append(ffmpegArgs, "-quality", "80")
	}

	ffmpegArgs = append(ffmpegArgs,
		"-loop", "0",
		"-an", "-vsync", "0",
		"-y", outputPath,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Info("Filter chain", "val", drawFilter)
		return nil, fmt.Errorf("ffmpeg ts failed: %w\nOutput: %s", err, string(output))
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
		"-font", "DejaVu-Sans", // More robust font handling across different Linux bounds
		"-size", "512x512", // Stretch box to absolute boundary to eliminate edge margins
		"-gravity", "West", // Left aligned, vertically centered
		fmt.Sprintf(`caption:%s`, sanitizeMagickText(text)),
		"-filter", "box",
		"-blur", "0x2.5", // More blur as requested ("agak blur")
		"-resize", "512x512!", // Ensure exact 512x512 sticker size
		"-strip", // STRIP ALL METADATA/ICC PROFILES to prevent WhatsApp Mobile crash!
		outputPath,
	}

	// Try magick first, fallback to convert
	bin := "magick"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "convert"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
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

// sanitizeMagickText prevents ImageMagick @file syntax injections.
func sanitizeMagickText(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "@") || strings.HasPrefix(s, "-") {
		return " " + s
	}
	return s
}
