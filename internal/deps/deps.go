package deps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DisplayServer represents the type of display server in use
type DisplayServer string

const (
	DisplayServerWayland DisplayServer = "wayland"
	DisplayServerX11     DisplayServer = "x11"
	DisplayServerUnknown DisplayServer = "unknown"
)

// Dependency represents a required external dependency
type Dependency struct {
	Name        string // Command name (e.g., "ffmpeg")
	Description string // Human-readable description
	Required    bool   // If true, app cannot run without it
	CheckCmd    string // Optional: specific command to check (defaults to "which <name>")
}

// CheckResult contains the result of checking a dependency
type CheckResult struct {
	Dependency Dependency
	Available  bool
	Path       string // Path to the executable if found
	Error      error  // Error if check failed
}

// DetectDisplayServer determines if running on Wayland or X11
func DetectDisplayServer() DisplayServer {
	// Check for Wayland first
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return DisplayServerWayland
	}
	// Check for X11
	if os.Getenv("DISPLAY") != "" {
		return DisplayServerX11
	}
	return DisplayServerUnknown
}

// GetDisplayServerName returns a human-readable name for the display server
func GetDisplayServerName() string {
	switch DetectDisplayServer() {
	case DisplayServerWayland:
		return "Wayland"
	case DisplayServerX11:
		return "X11"
	default:
		return "Unknown"
	}
}

// BaseDeps lists dependencies required regardless of display server
var BaseDeps = []Dependency{
	{
		Name:        "ffmpeg",
		Description: "Video/audio processing and merging",
		Required:    true,
	},
	{
		Name:        "ffprobe",
		Description: "Video metadata extraction",
		Required:    true,
	},
	{
		Name:        "pw-record",
		Description: "PipeWire audio recording",
		Required:    true,
	},
}

// WaylandDeps lists dependencies specific to Wayland
var WaylandDeps = []Dependency{
	{
		Name:        "wl-screenrec",
		Description: "Wayland screen recording",
		Required:    true,
	},
}

// X11Deps lists dependencies specific to X11
// Note: X11 screen recording uses ffmpeg with x11grab, so no additional dep needed
var X11Deps = []Dependency{
	// ffmpeg with x11grab is used for X11 screen recording
	// No additional binary needed beyond ffmpeg
}

// OptionalDeps lists optional dependencies that enhance functionality
var OptionalDeps = []Dependency{
	{
		Name:        "notify-send",
		Description: "Desktop notifications",
		Required:    false,
	},
	{
		Name:        "paplay",
		Description: "Audio playback for countdown beeps",
		Required:    false,
	},
	{
		Name:        "speaker-test",
		Description: "Alternative audio playback for countdown",
		Required:    false,
	},
}

// GetRequiredDeps returns the required dependencies based on current display server
func GetRequiredDeps() []Dependency {
	deps := make([]Dependency, len(BaseDeps))
	copy(deps, BaseDeps)

	switch DetectDisplayServer() {
	case DisplayServerWayland:
		deps = append(deps, WaylandDeps...)
	case DisplayServerX11:
		deps = append(deps, X11Deps...)
	default:
		// Unknown - require Wayland deps as default
		deps = append(deps, WaylandDeps...)
	}

	return deps
}

// RequiredDeps is kept for backward compatibility but now returns display-server-specific deps
var RequiredDeps = GetRequiredDeps()

// Check verifies if a single dependency is available
func Check(dep Dependency) CheckResult {
	result := CheckResult{Dependency: dep}

	path, err := exec.LookPath(dep.Name)
	if err != nil {
		result.Available = false
		result.Error = err
	} else {
		result.Available = true
		result.Path = path
	}

	return result
}

// CheckAll verifies all required and optional dependencies
func CheckAll() (required []CheckResult, optional []CheckResult) {
	for _, dep := range GetRequiredDeps() {
		required = append(required, Check(dep))
	}
	for _, dep := range OptionalDeps {
		optional = append(optional, Check(dep))
	}
	return required, optional
}

// CheckRequired verifies only required dependencies
func CheckRequired() []CheckResult {
	var results []CheckResult
	for _, dep := range GetRequiredDeps() {
		results = append(results, Check(dep))
	}
	return results
}

// MissingRequired returns a list of missing required dependencies
func MissingRequired() []CheckResult {
	var missing []CheckResult
	for _, dep := range GetRequiredDeps() {
		result := Check(dep)
		if !result.Available {
			missing = append(missing, result)
		}
	}
	return missing
}

// HasAllRequired returns true if all required dependencies are available
func HasAllRequired() bool {
	return len(MissingRequired()) == 0
}

// FormatMissing returns a formatted string of missing dependencies
func FormatMissing(results []CheckResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Missing dependencies:\n\n")

	for _, r := range results {
		status := "MISSING"
		if r.Dependency.Required {
			status = "REQUIRED"
		}
		sb.WriteString(fmt.Sprintf("  • %s (%s)\n", r.Dependency.Name, status))
		sb.WriteString(fmt.Sprintf("    %s\n\n", r.Dependency.Description))
	}

	return sb.String()
}

// FormatAll returns a formatted string of all dependency check results
func FormatAll(required, optional []CheckResult) string {
	var sb strings.Builder

	sb.WriteString("Required dependencies:\n")
	for _, r := range required {
		status := "✓"
		if !r.Available {
			status = "✗"
		}
		sb.WriteString(fmt.Sprintf("  %s %s - %s\n", status, r.Dependency.Name, r.Dependency.Description))
		if r.Available {
			sb.WriteString(fmt.Sprintf("      Path: %s\n", r.Path))
		}
	}

	sb.WriteString("\nOptional dependencies:\n")
	for _, r := range optional {
		status := "✓"
		if !r.Available {
			status = "○"
		}
		sb.WriteString(fmt.Sprintf("  %s %s - %s\n", status, r.Dependency.Name, r.Dependency.Description))
		if r.Available {
			sb.WriteString(fmt.Sprintf("      Path: %s\n", r.Path))
		}
	}

	return sb.String()
}
