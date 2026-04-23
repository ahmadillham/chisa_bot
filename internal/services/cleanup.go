package services

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartTempCleaner runs a background goroutine that periodically scans the OS temp directory
// and removes any 'chisabot-*' directories/files that are older than maxAge.
func StartTempCleaner(interval time.Duration, maxAge time.Duration) {
	tempDir := os.TempDir()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			entries, err := os.ReadDir(tempDir)
			if err != nil {
				slog.Error("failed to read temp dir", "error", err)
				continue
			}

			now := time.Now()
			cleanedCount := 0

			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, "chisabot-") {
					fullPath := filepath.Join(tempDir, name)

					info, err := entry.Info()
					if err != nil {
						continue
					}

					// If the file/dir is older than maxAge, delete it
					if now.Sub(info.ModTime()) > maxAge {
						if err := os.RemoveAll(fullPath); err == nil {
							cleanedCount++
						} else {
							slog.Error("failed to remove", "val", fullPath, "error", err)
						}
					}
				}
			}

			if cleanedCount > 0 {
				slog.Info("Deleted %d old temporary files/directories.", "val", cleanedCount)
			}
		}
	}()
}
