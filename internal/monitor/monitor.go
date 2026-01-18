package monitor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// ListMonitors returns all available monitors from Hyprland
func ListMonitors() ([]models.Monitor, error) {
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

// GetCursorPosition returns the current cursor position
func GetCursorPosition() (models.CursorPosition, error) {
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
