package monitor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/deps"
	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// ListMonitors returns all available monitors
func ListMonitors() ([]models.Monitor, error) {
	currentOS := deps.DetectOS()

	switch currentOS {
	case deps.OSWindows:
		return listMonitorsWindows()
	case deps.OSDarwin:
		return listMonitorsMacOS()
	case deps.OSLinux:
		switch deps.DetectDisplayServer() {
		case deps.DisplayServerWayland:
			return listMonitorsWayland()
		case deps.DisplayServerX11:
			return listMonitorsX11()
		default:
			// Try Wayland first, then X11
			monitors, err := listMonitorsWayland()
			if err == nil {
				return monitors, nil
			}
			return listMonitorsX11()
		}
	default:
		// Unknown OS - try Linux Wayland/X11
		monitors, err := listMonitorsWayland()
		if err == nil {
			return monitors, nil
		}
		return listMonitorsX11()
	}
}

// listMonitorsWayland returns monitors using hyprctl (Wayland/Hyprland)
func listMonitorsWayland() ([]models.Monitor, error) {
	cmd := exec.Command("hyprctl", "monitors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run hyprctl monitors: %w", err)
	}

	var monitors []models.Monitor
	if err := json.Unmarshal(output, &monitors); err != nil {
		return nil, fmt.Errorf("failed to parse monitors JSON: %w", err)
	}

	return monitors, nil
}

// listMonitorsX11 returns monitors using xrandr (X11)
func listMonitorsX11() ([]models.Monitor, error) {
	cmd := exec.Command("xrandr", "--query")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run xrandr: %w", err)
	}

	var monitors []models.Monitor
	lines := strings.Split(string(output), "\n")

	// Pattern: "DP-0 connected primary 1920x1080+0+0 ..."
	// or: "HDMI-0 connected 1920x1080+1920+0 ..."
	re := regexp.MustCompile(`^(\S+)\s+connected\s+(primary\s+)?(\d+)x(\d+)\+(\d+)\+(\d+)`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		isPrimary := matches[2] != ""
		width, _ := strconv.Atoi(matches[3])
		height, _ := strconv.Atoi(matches[4])
		x, _ := strconv.Atoi(matches[5])
		y, _ := strconv.Atoi(matches[6])

		monitors = append(monitors, models.Monitor{
			Name:    name,
			Width:   width,
			Height:  height,
			X:       x,
			Y:       y,
			Focused: isPrimary, // Use primary as "focused" for X11
		})
	}

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no monitors found via xrandr")
	}

	return monitors, nil
}

// listMonitorsWindows returns monitors on Windows (using a default/generic approach)
func listMonitorsWindows() ([]models.Monitor, error) {
	// On Windows, we return a generic "desktop" monitor placeholder
	// Actual screen resolution is handled by ffmpeg's gdigrab at runtime
	// The 1920x1080 resolution here is a placeholder and doesn't affect actual recording
	monitors := []models.Monitor{
		{
			Name:    "desktop",
			Width:   1920, // Placeholder resolution
			Height:  1080, // Placeholder resolution
			X:       0,
			Y:       0,
			Focused: true,
		},
	}
	return monitors, nil
}

// listMonitorsMacOS returns monitors on macOS (using a default/generic approach)
func listMonitorsMacOS() ([]models.Monitor, error) {
	// On macOS, we return a generic screen capture placeholder
	// Actual screen resolution is handled by ffmpeg's avfoundation at runtime
	// The 1920x1080 resolution here is a placeholder and doesn't affect actual recording
	monitors := []models.Monitor{
		{
			Name:    "screen-1",
			Width:   1920, // Placeholder resolution
			Height:  1080, // Placeholder resolution
			X:       0,
			Y:       0,
			Focused: true,
		},
	}
	return monitors, nil
}

// GetCursorPosition returns the current cursor position
func GetCursorPosition() (models.CursorPosition, error) {
	currentOS := deps.DetectOS()

	switch currentOS {
	case deps.OSWindows:
		return getCursorPositionWindows()
	case deps.OSDarwin:
		return getCursorPositionMacOS()
	case deps.OSLinux:
		switch deps.DetectDisplayServer() {
		case deps.DisplayServerWayland:
			return getCursorPositionWayland()
		case deps.DisplayServerX11:
			return getCursorPositionX11()
		default:
			// Try Wayland first
			pos, err := getCursorPositionWayland()
			if err == nil {
				return pos, nil
			}
			return getCursorPositionX11()
		}
	default:
		// Unknown OS - try Linux
		pos, err := getCursorPositionWayland()
		if err == nil {
			return pos, nil
		}
		return getCursorPositionX11()
	}
}

// getCursorPositionWayland gets cursor position using hyprctl
func getCursorPositionWayland() (models.CursorPosition, error) {
	cmd := exec.Command("hyprctl", "cursorpos")
	output, err := cmd.Output()
	if err != nil {
		return models.CursorPosition{}, fmt.Errorf("failed to get cursor position: %w", err)
	}

	// Parse format: "x, y"
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) != 2 {
		return models.CursorPosition{}, fmt.Errorf("unexpected cursor position format: %s", output)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return models.CursorPosition{}, fmt.Errorf("failed to parse cursor X: %w", err)
	}

	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return models.CursorPosition{}, fmt.Errorf("failed to parse cursor Y: %w", err)
	}

	return models.CursorPosition{X: x, Y: y}, nil
}

// getCursorPositionX11 gets cursor position using xdotool
func getCursorPositionX11() (models.CursorPosition, error) {
	cmd := exec.Command("xdotool", "getmouselocation", "--shell")
	output, err := cmd.Output()
	if err != nil {
		return models.CursorPosition{}, fmt.Errorf("failed to get cursor position: %w", err)
	}

	// Parse format: "X=123\nY=456\nSCREEN=0\nWINDOW=..."
	var x, y int
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "X=") {
			x, _ = strconv.Atoi(strings.TrimPrefix(line, "X="))
		} else if strings.HasPrefix(line, "Y=") {
			y, _ = strconv.Atoi(strings.TrimPrefix(line, "Y="))
		}
	}

	return models.CursorPosition{X: x, Y: y}, nil
}

// getCursorPositionWindows gets cursor position on Windows (returns default)
func getCursorPositionWindows() (models.CursorPosition, error) {
	// Return center of default screen as a placeholder
	// For precise cursor position, we'd need Windows-specific APIs
	return models.CursorPosition{X: 960, Y: 540}, nil
}

// getCursorPositionMacOS gets cursor position on macOS (returns default)
func getCursorPositionMacOS() (models.CursorPosition, error) {
	// Return center of default screen as a placeholder
	// For precise cursor position, we'd need macOS-specific APIs
	return models.CursorPosition{X: 960, Y: 540}, nil
}

// GetMouseMonitor returns the name of the monitor containing the mouse cursor
func GetMouseMonitor() (string, error) {
	pos, err := GetCursorPosition()
	if err != nil {
		// Fallback to focused monitor
		return GetFocusedMonitor()
	}

	monitors, err := ListMonitors()
	if err != nil {
		return "", err
	}

	for _, m := range monitors {
		if m.ContainsCursor(pos) {
			return m.Name, nil
		}
	}

	// If no monitor found, return focused monitor
	return GetFocusedMonitor()
}

// GetFocusedMonitor returns the name of the currently focused monitor
func GetFocusedMonitor() (string, error) {
	monitors, err := ListMonitors()
	if err != nil {
		return "", err
	}

	for _, m := range monitors {
		if m.Focused {
			return m.Name, nil
		}
	}

	// Return first monitor if none focused
	if len(monitors) > 0 {
		return monitors[0].Name, nil
	}

	return "", fmt.Errorf("no monitors found")
}

// GetMonitorByName returns the monitor with the given name
func GetMonitorByName(name string) (*models.Monitor, error) {
	monitors, err := ListMonitors()
	if err != nil {
		return nil, err
	}

	for _, m := range monitors {
		if m.Name == name {
			return &m, nil
		}
	}

	return nil, fmt.Errorf("monitor not found: %s", name)
}
