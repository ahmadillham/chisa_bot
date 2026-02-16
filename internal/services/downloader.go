package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// MediaResult holds the result of a download operation.
type MediaResult struct {
	Title    string
	Type     string // "video", "audio"
	Mimetype string
	Data     []byte
	FilePath string
}

// YtDlpService wraps yt-dlp CLI for downloading media from various platforms.
type YtDlpService struct {
	bin string // absolute path to yt-dlp binary
}

// NewYtDlpService creates a new YtDlpService, auto-detecting the yt-dlp binary path.
func NewYtDlpService() *YtDlpService {
	bin := findYtDlp()
	return &YtDlpService{bin: bin}
}

// findYtDlp looks for yt-dlp in PATH and common install locations.
func findYtDlp() string {
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
	}
	// Check common locations
	home, _ := os.UserHomeDir()
	candidates := []string{
		home + "/.local/bin/yt-dlp",
		"/usr/local/bin/yt-dlp",
		"/usr/bin/yt-dlp",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "yt-dlp" // fallback, hope it's in PATH
}

// ytDlpInfo holds metadata from yt-dlp --dump-json.
type ytDlpInfo struct {
	Title string `json:"title"`
}

// DownloadAny automatically detects the platform and downloads the best video.
func (s *YtDlpService) DownloadAny(sourceURL string) (*MediaResult, error) {
	// Simple routing based on domain, though yt-dlp handles most automatically.
	// We route physically only if we need specific flags (like tiktok watermark removal).
	if strings.Contains(sourceURL, "tiktok.com") {
		return s.DownloadTikTok(sourceURL)
	}
	if strings.Contains(sourceURL, "instagram.com") {
		return s.DownloadInstagram(sourceURL)
	}
	// For most others (YouTube, FB, Twitter), standard download works best.
	return s.downloadGeneric(sourceURL)
}

// DownloadInstagram downloads IG content (Video or Image).
func (s *YtDlpService) DownloadInstagram(sourceURL string) (*MediaResult, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-dl-ig-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	// We don't defer remove here immediately because if we succeed we read file then remove.
	// But it's better to remove in all cases. We will handle cleanup carefully.

	// Output template without extension, yt-dlp adds it.
	outputTemplate := filepath.Join(tmpDir, "media.%(ext)s")

	// Use "best" to allow images (jpg/webp) or video
	args := []string{
		"-f", "best",
		"--max-filesize", "100M",
		"--no-playlist",
		"--no-warnings",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"-o", outputTemplate,
		sourceURL,
	}

	cmd := exec.Command(s.bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "no video") {
			// Fallback: Try to scrape og:image
			if fallbackRes, fbErr := s.scrapeIGImage(sourceURL); fbErr == nil {
				os.RemoveAll(tmpDir)
				return fallbackRes, nil
			}
		}
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("ig download failed: %w\nOutput: %s", err, outputStr)
	}

	// Find the downloaded file
	files, err := os.ReadDir(tmpDir)
	if err != nil || len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no file downloaded")
	}

	filename := files[0].Name()
	filePath := filepath.Join(tmpDir, filename)
	data, err := os.ReadFile(filePath)
	os.RemoveAll(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine type based on extension
	ext := strings.ToLower(filepath.Ext(filename))
	mimetype := "application/octet-stream"
	msgType := "document"

	switch ext {
	case ".mp4":
		mimetype = "video/mp4"
		msgType = "video"
	case ".jpg", ".jpeg":
		mimetype = "image/jpeg"
		msgType = "image"
	case ".png":
		mimetype = "image/png"
		msgType = "image"
	case ".webp":
		mimetype = "image/webp"
		msgType = "image"
	}

	title := s.getTitle(sourceURL)

	return &MediaResult{
		Title:    title,
		Type:     msgType,
		Mimetype: mimetype,
		Data:     data,
	}, nil
}

// scrapeIGImage tries to fetch the og:image from the public IG page.
func (s *YtDlpService) scrapeIGImage(url string) (*MediaResult, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Use a crawler UA to hopefully get the SSR info
	req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Try og:image first (usually standard)
	var imageURL string
	reOG := regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`)
	matchesOG := reOG.FindStringSubmatch(html)
	if len(matchesOG) > 1 {
		imageURL = matchesOG[1]
	} else {
		// Fallback to twitter:image (sometimes better quality/aspect)
		reTw := regexp.MustCompile(`<meta name="twitter:image" content="([^"]+)"`)
		matchesTw := reTw.FindStringSubmatch(html)
		if len(matchesTw) > 1 {
			imageURL = matchesTw[1]
		}
	}

	if imageURL == "" {
		return nil, fmt.Errorf("no image found (checked og:image and twitter:image)")
	}

	imageURL = strings.ReplaceAll(imageURL, "&amp;", "&")

	// Download the image
	imgResp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer imgResp.Body.Close()

	imgData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, err
	}

	// Determine title
	title := "Instagram Photo"
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	if tMatches := titleRe.FindStringSubmatch(html); len(tMatches) > 1 {
		title = tMatches[1]
	}

	return &MediaResult{
		Title:    title,
		Type:     "image",
		Mimetype: "image/jpeg", // Default to jpeg for og:image
		Data:     imgData,
	}, nil
}

// downloadGeneric is a robust fallback for YouTube, FB, Twitter, etc.
func (s *YtDlpService) downloadGeneric(sourceURL string) (*MediaResult, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-dl-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "video.mp4")

	// Best compatible video format (mp4+aac).
	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"--max-filesize", "100M",
		"--no-playlist",
		"--no-warnings",
		"-o", outputPath,
		sourceURL,
	}

	cmd := exec.Command(s.bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("download failed: %w\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(outputPath)
	os.RemoveAll(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	title := s.getTitle(sourceURL)

	return &MediaResult{
		Title:    title,
		Type:     "video",
		Mimetype: "video/mp4",
		Data:     data,
	}, nil
}

// DownloadAudio downloads audio from a given URL using yt-dlp.
func (s *YtDlpService) DownloadAudio(sourceURL string) (*MediaResult, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-dl-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "audio.mp3")

	cmd := exec.Command(s.bin,
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--max-filesize", "50M",
		"--no-playlist",
		"--no-warnings",
		"-o", outputPath,
		sourceURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp audio failed: %w\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(outputPath)
	os.RemoveAll(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded file: %w", err)
	}

	title := s.getTitle(sourceURL)

	return &MediaResult{
		Title:    title,
		Type:     "audio",
		Mimetype: "audio/mpeg",
		Data:     data,
	}, nil
}

// DownloadTikTok downloads a TikTok video without watermark.
func (s *YtDlpService) DownloadTikTok(sourceURL string) (*MediaResult, error) {
	tmpDir, err := os.MkdirTemp("", "chisabot-dl-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "tiktok.mp4")

	cmd := exec.Command(s.bin,
		"-f", "best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"--max-filesize", "100M",
		"--no-playlist",
		"--no-warnings",
		"-o", outputPath,
		sourceURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp tiktok failed: %w\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(outputPath)
	os.RemoveAll(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded file: %w", err)
	}

	title := s.getTitle(sourceURL)

	return &MediaResult{
		Title:    title,
		Type:     "video",
		Mimetype: "video/mp4",
		Data:     data,
	}, nil
}

// getTitle fetches the title of a URL using yt-dlp --dump-json.
func (s *YtDlpService) getTitle(sourceURL string) string {
	cmd := exec.Command(s.bin,
		"--dump-json",
		"--no-download",
		"--no-warnings",
		"--no-playlist",
		sourceURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return "Downloaded Media"
	}

	var info ytDlpInfo
	if err := json.Unmarshal(output, &info); err != nil || info.Title == "" {
		return "Downloaded Media"
	}
	return info.Title
}
