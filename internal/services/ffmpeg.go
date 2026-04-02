package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	cmd := exec.Command("ffmpeg",
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

	// Wrap text so it doesn't overflow the 512px sticker width (approx 13 chars max at fontsize 72)
	wrappedText := wrapText(text, 13)

	// Escape special characters for FFmpeg drawtext.
	safeText := escapeFfmpegText(wrappedText)

	// Ensure 512x512 bounded container with 1px transparent padding to force a valid VP8X 
	// alpha chunk, preventing "corrupt sticker" errors on WhatsApp Mobile, then draw the text.
	drawFilter := fmt.Sprintf(
		"scale='if(gt(iw,ih),510,-2)':'if(gt(iw,ih),-2,510)',format=bgra,pad=512:512:(512-iw)/2:(512-ih)/2:color=0x00000000,drawtext=fontfile=%s:text='%s':fontcolor=white:fontsize=72:bordercolor=black:borderw=4:x=(w-text_w)/2:y=h-text_h-20:line_spacing=5:text_align=C",
		"/usr/share/fonts/TTF/DejaVuSans-Bold.ttf", safeText,
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

	outputPath := filepath.Join(tmpDir, "brat.png") // Output as PNG first

	// The user requested white background with black text matching the photo.
	// ImageMagick 'caption:' auto-wraps and scales text to fit the box.
	args := []string{
		"-background", "white",
		"-fill", "black",
		"-font", "DejaVu-Sans", // More robust font handling across different Linux bounds
		"-size", "512x512",     // Stretch box to absolute boundary to eliminate edge margins
		"-gravity", "West",     // Left aligned, vertically centered
		fmt.Sprintf(`caption:%s`, text),
		"-filter", "box",
		"-blur", "0x2.5",       // More blur as requested ("agak blur")
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
