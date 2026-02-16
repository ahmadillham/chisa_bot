package handlers

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/pkg/utils"
)

// SystemHandler handles system information commands.
type SystemHandler struct {
	startTime time.Time
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		startTime: time.Now(),
	}
}

// HandleStats sends server stats (CPU, RAM, Uptime).
func (h *SystemHandler) HandleStats(client *whatsmeow.Client, evt *events.Message) {
	// CPU info
	cpuModel := getCPUModel()
	cpuCores := runtime.NumCPU()

	// RAM info
	totalRAM, usedRAM, _ := getMemoryInfo()

	// System uptime
	sysUptime := getSystemUptime()
	distro := getDistroName()

	// Go runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	botMemMB := float64(memStats.Alloc) / 1024 / 1024

	stats := fmt.Sprintf(`📊 *Server Stats*

🖥️ *System*
• Distro: %s
• CPU: %s (%d Cores)
• RAM: %s / %s
• Uptime: %s

🤖 *Bot*
• Memory: %.2f MB`,
		distro,
		cpuModel, cpuCores,
		usedRAM, totalRAM,
		sysUptime,
		botMemMB,
	)

	utils.ReplyText(client, evt, stats)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func getDistroName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(name, "\"")
		}
	}
	return "Unknown"
}

func getCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func getMemoryInfo() (total, used, free string) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A", "N/A", "N/A"
	}

	var totalKB, availKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &availKB)
		}
	}

	usedKB := totalKB - availKB
	return formatBytes(totalKB * 1024), formatBytes(usedKB * 1024), formatBytes(availKB * 1024)
}

func getSystemUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}

	var seconds float64
	fmt.Sscanf(string(data), "%f", &seconds)
	return formatDuration(time.Duration(seconds) * time.Second)
}

func formatBytes(bytes uint64) string {
	const (
		MB = 1024 * 1024
		GB = 1024 * MB
	)
	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
}
