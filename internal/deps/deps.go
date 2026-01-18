package deps

import (
	"fmt"
	"os/exec"
	"strings"
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

// RequiredDeps lists all required dependencies for the application
var RequiredDeps = []Dependency{
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
		Name:        "wl-screenrec",
		Description: "Wayland screen recording",
		Required:    true,
	},
	{
		Name:        "pw-record",
		Description: "PipeWire audio recording",
		Required:    true,
	},
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
	for _, dep := range RequiredDeps {
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
	for _, dep := range RequiredDeps {
		results = append(results, Check(dep))
	}
	return results
}

// MissingRequired returns a list of missing required dependencies
func MissingRequired() []CheckResult {
	var missing []CheckResult
	for _, dep := range RequiredDeps {
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
