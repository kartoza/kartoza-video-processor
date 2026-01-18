package notify

import (
	"os/exec"
)

// Urgency levels for notifications
type Urgency string

const (
	UrgencyLow      Urgency = "low"
	UrgencyNormal   Urgency = "normal"
	UrgencyCritical Urgency = "critical"
)

// Send sends a desktop notification using notify-send
func Send(title, body string, urgency Urgency, icon string) error {
	args := []string{title, body}

	if urgency != "" {
		args = append(args, "--urgency="+string(urgency))
	}

	if icon != "" {
		args = append(args, "--icon="+icon)
	}

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// Info sends an informational notification
func Info(title, body string) error {
	return Send(title, body, UrgencyNormal, "video-x-generic")
}

// Warning sends a warning notification
func Warning(title, body string) error {
	return Send(title, body, UrgencyLow, "dialog-warning")
}

// Error sends an error notification
func Error(title, body string) error {
	return Send(title, body, UrgencyCritical, "dialog-error")
}

// RecordingStarted notifies that recording has started
func RecordingStarted(monitor string) error {
	body := "Recording " + monitor + " with audio..."
	return Info("Screen Recording", body)
}

// RecordingStopped notifies that recording has stopped
func RecordingStopped() error {
	return Info("Screen Recording", "Processing recording...")
}

// RecordingComplete notifies that recording is complete
func RecordingComplete(filename string) error {
	return Info("Screen Recording Complete", filename+" saved!")
}

// VerticalComplete notifies that vertical video is complete
func VerticalComplete(filename string) error {
	return Info("Vertical Recording Complete", filename+" saved!")
}

// ProcessingStep notifies about a processing step
func ProcessingStep(step string) error {
	return Info("Screen Recording", step)
}
