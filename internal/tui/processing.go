package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProcessingStep represents a single processing step
type ProcessingStep struct {
	Name      string
	Status    StepStatus
	StartTime time.Time
	EndTime   time.Time
}

// StepStatus represents the status of a processing step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepComplete
	StepFailed
	StepSkipped
)

// ProcessingState holds the state of all processing steps
type ProcessingState struct {
	Steps        []ProcessingStep
	CurrentStep  int
	IsProcessing bool
	StartTime    time.Time
	Error        error
}

// NewProcessingState creates a new processing state with default steps
func NewProcessingState() *ProcessingState {
	return &ProcessingState{
		Steps: []ProcessingStep{
			{Name: "Stopping recorders", Status: StepPending},
			{Name: "Denoising audio", Status: StepPending},
			{Name: "Analyzing audio levels", Status: StepPending},
			{Name: "Normalizing audio", Status: StepPending},
			{Name: "Merging video & audio", Status: StepPending},
			{Name: "Creating vertical video", Status: StepPending},
		},
		CurrentStep:  -1,
		IsProcessing: false,
	}
}

// SetStepByIndex directly sets a step's status by index
func (p *ProcessingState) SetStepByIndex(index int, status StepStatus) {
	if index >= 0 && index < len(p.Steps) {
		if status == StepRunning {
			p.Steps[index].StartTime = time.Now()
			p.CurrentStep = index
		} else if status == StepComplete || status == StepSkipped || status == StepFailed {
			p.Steps[index].EndTime = time.Now()
		}
		p.Steps[index].Status = status
	}
}

// Start begins the processing
func (p *ProcessingState) Start() {
	p.IsProcessing = true
	p.StartTime = time.Now()
	p.CurrentStep = 0
	if len(p.Steps) > 0 {
		p.Steps[0].Status = StepRunning
		p.Steps[0].StartTime = time.Now()
	}
}

// NextStep advances to the next step
func (p *ProcessingState) NextStep() {
	if p.CurrentStep >= 0 && p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepComplete
		p.Steps[p.CurrentStep].EndTime = time.Now()
	}
	p.CurrentStep++
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepRunning
		p.Steps[p.CurrentStep].StartTime = time.Now()
	}
}

// SkipStep marks current step as skipped and advances
func (p *ProcessingState) SkipStep() {
	if p.CurrentStep >= 0 && p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepSkipped
		p.Steps[p.CurrentStep].EndTime = time.Now()
	}
	p.CurrentStep++
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepRunning
		p.Steps[p.CurrentStep].StartTime = time.Now()
	}
}

// FailStep marks current step as failed
func (p *ProcessingState) FailStep(err error) {
	if p.CurrentStep >= 0 && p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepFailed
		p.Steps[p.CurrentStep].EndTime = time.Now()
	}
	p.Error = err
}

// Complete marks processing as complete
func (p *ProcessingState) Complete() {
	if p.CurrentStep >= 0 && p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Status = StepComplete
		p.Steps[p.CurrentStep].EndTime = time.Now()
	}
	p.IsProcessing = false
}

// Reset resets the processing state
func (p *ProcessingState) Reset() {
	for i := range p.Steps {
		p.Steps[i].Status = StepPending
		p.Steps[i].StartTime = time.Time{}
		p.Steps[i].EndTime = time.Time{}
	}
	p.CurrentStep = -1
	p.IsProcessing = false
	p.Error = nil
}

// Messages for processing updates
type processingTickMsg struct{}
type processingStepMsg struct {
	Step      int
	Completed bool
	Skipped   bool
	Error     error
}
type processingCompleteMsg struct{}
type processingErrorMsg struct {
	Error error
}

// processingTickCmd returns a command that ticks the processing animation
func processingTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return processingTickMsg{}
	})
}

// Donut animation frames (Unicode block characters for spinning effect)
var donutFrames = []string{
	"◐", "◓", "◑", "◒",
}

// RenderProcessingView renders the processing screen with donut indicators
func RenderProcessingView(state *ProcessingState, width, height int, frame int) string {
	if state == nil {
		return ""
	}

	// Update global app state to show Processing status
	GlobalAppState.IsRecording = false
	GlobalAppState.Status = "Processing"

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		MarginBottom(1)

	title := titleStyle.Render("Processing Recording...")

	// Elapsed time
	elapsed := time.Since(state.StartTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)
	elapsedStr := timeStyle.Render(fmt.Sprintf("Elapsed: %s", elapsed))

	// Build step list
	var steps []string
	for i, step := range state.Steps {
		line := renderStepLine(step, i == state.CurrentStep, frame)
		steps = append(steps, line)
	}
	stepsContent := strings.Join(steps, "\n")

	// Status message
	var statusMsg string
	statusStyle := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(ColorGray)

	if state.Error != nil {
		statusStyle = statusStyle.Foreground(ColorRed)
		statusMsg = statusStyle.Render(fmt.Sprintf("Error: %v", state.Error))
	} else if !state.IsProcessing {
		statusStyle = statusStyle.Foreground(ColorGreen)
		statusMsg = statusStyle.Render("Processing complete!")
	} else {
		statusMsg = statusStyle.Render("Please wait...")
	}

	// Hint
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		MarginTop(2)
	hint := hintStyle.Render("Recording controls disabled during processing")

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		title,
		elapsedStr,
		"",
		stepsContent,
		"",
		statusMsg,
		hint,
	)

	// Center on screen
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderStepLine renders a single processing step with appropriate indicator
func renderStepLine(step ProcessingStep, isCurrent bool, frame int) string {
	var indicator string
	var nameStyle lipgloss.Style

	switch step.Status {
	case StepPending:
		indicator = lipgloss.NewStyle().Foreground(ColorGray).Render("○")
		nameStyle = lipgloss.NewStyle().Foreground(ColorGray)

	case StepRunning:
		// Animated donut
		donutStyle := lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		indicator = donutStyle.Render(donutFrames[frame%len(donutFrames)])
		nameStyle = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)

	case StepComplete:
		indicator = lipgloss.NewStyle().Foreground(ColorGreen).Render("●")
		nameStyle = lipgloss.NewStyle().Foreground(ColorGreen)

	case StepFailed:
		indicator = lipgloss.NewStyle().Foreground(ColorRed).Render("✗")
		nameStyle = lipgloss.NewStyle().Foreground(ColorRed)

	case StepSkipped:
		indicator = lipgloss.NewStyle().Foreground(ColorGray).Render("○")
		nameStyle = lipgloss.NewStyle().Foreground(ColorGray).Strikethrough(true)
	}

	// Duration for completed steps
	var duration string
	if step.Status == StepComplete || step.Status == StepFailed {
		d := step.EndTime.Sub(step.StartTime).Round(100 * time.Millisecond)
		durationStyle := lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
		duration = durationStyle.Render(fmt.Sprintf(" (%s)", d))
	}

	return fmt.Sprintf("  %s %s%s", indicator, nameStyle.Render(step.Name), duration)
}
