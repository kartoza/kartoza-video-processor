package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/monitor"
	"github.com/kartoza/kartoza-video-processor/internal/recorder"
)

// Theme colors (Catppuccin-inspired)
var (
	primaryColor   = lipgloss.Color("#89b4fa") // Blue
	secondaryColor = lipgloss.Color("#a6e3a1") // Green
	warningColor   = lipgloss.Color("#f9e2af") // Yellow
	errorColor     = lipgloss.Color("#f38ba8") // Red
	textColor      = lipgloss.Color("#cdd6f4") // Text
	subtleColor    = lipgloss.Color("#6c7086") // Subtle
	surfaceColor   = lipgloss.Color("#313244") // Surface
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	inactiveStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	recordingStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Blink(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	monitorStyle = lipgloss.NewStyle().
			Foreground(textColor)

	cursorMarkerStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)
)

// Key bindings
type keyMap struct {
	Toggle key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Toggle: key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space/enter", "toggle recording"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// Messages
type tickMsg time.Time
type statusUpdateMsg models.RecordingStatus
type monitorsUpdateMsg []models.Monitor

// Model is the main TUI model
type Model struct {
	recorder   *recorder.Recorder
	status     models.RecordingStatus
	monitors   []models.Monitor
	spinner    spinner.Model
	width      int
	height     int
	showHelp   bool
	err        error
}

// NewModel creates a new TUI model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(errorColor)

	return Model{
		recorder: recorder.New(),
		spinner:  s,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		updateStatus(m.recorder),
		updateMonitors(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Toggle):
			if m.status.IsRecording {
				if err := m.recorder.Stop(); err != nil {
					m.err = err
				}
			} else {
				if err := m.recorder.Start(); err != nil {
					m.err = err
				}
			}
			return m, updateStatus(m.recorder)

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			tickCmd(),
			updateStatus(m.recorder),
			updateMonitors(),
		)

	case statusUpdateMsg:
		m.status = models.RecordingStatus(msg)
		return m, nil

	case monitorsUpdateMsg:
		m.monitors = []models.Monitor(msg)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var content string

	// Title
	title := titleStyle.Render("🎬 Kartoza Video Processor")

	// Status
	var status string
	if m.status.IsRecording {
		duration := time.Since(m.status.StartTime).Round(time.Second)
		status = fmt.Sprintf("%s %s Recording... %s",
			m.spinner.View(),
			recordingStyle.Render("●"),
			statusStyle.Render(duration.String()),
		)
	} else {
		status = inactiveStyle.Render("○ Ready to record")
	}

	// Monitors
	var monitorsView string
	cursorMonitor, _ := monitor.GetMouseMonitor()

	if len(m.monitors) > 0 {
		monitorsView = "Monitors:\n"
		for _, mon := range m.monitors {
			marker := "  "
			if mon.Name == cursorMonitor {
				marker = cursorMarkerStyle.Render("→ ")
			}
			line := fmt.Sprintf("%s%s (%dx%d)",
				marker,
				mon.Name,
				mon.Width,
				mon.Height,
			)
			monitorsView += monitorStyle.Render(line) + "\n"
		}
	}

	// Recording info
	var recordingInfo string
	if m.status.IsRecording {
		recordingInfo = fmt.Sprintf("\nRecording to:\n  Video: %s\n  Audio: %s",
			m.status.VideoFile,
			m.status.AudioFile,
		)
		if m.status.WebcamFile != "" {
			recordingInfo += fmt.Sprintf("\n  Webcam: %s", m.status.WebcamFile)
		}
	}

	// Help
	help := helpStyle.Render("[space] toggle recording  [q] quit  [?] help")

	// Combine content
	content = fmt.Sprintf("%s\n\n%s\n\n%s%s\n%s",
		title,
		status,
		monitorsView,
		recordingInfo,
		help,
	)

	if m.showHelp {
		content += "\n\n" + m.renderHelp()
	}

	return boxStyle.Render(content)
}

func (m Model) renderHelp() string {
	return `Keyboard Shortcuts:
  space/enter  Toggle recording on/off
  q            Quit application
  ?            Toggle help

Recording Info:
  • Video is recorded with wl-screenrec
  • Audio is captured from default microphone
  • Webcam is recorded if available
  • Audio is denoised and normalized automatically
  • Vertical video is created with webcam overlay`
}

// Commands

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func updateStatus(rec *recorder.Recorder) tea.Cmd {
	return func() tea.Msg {
		return statusUpdateMsg(rec.GetStatus())
	}
}

func updateMonitors() tea.Cmd {
	return func() tea.Msg {
		monitors, err := monitor.ListMonitors()
		if err != nil {
			return monitorsUpdateMsg{}
		}
		return monitorsUpdateMsg(monitors)
	}
}

// Run starts the TUI application with splash screens
func Run() error {
	// Show entry splash screen (3 seconds, skippable with any key)
	if err := ShowSplashScreen(3 * time.Second); err != nil {
		// Ignore splash errors, continue to main app
		_ = err
	}

	// Run main application
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()

	// Show exit splash screen (2 seconds, skippable with any key)
	if exitErr := ShowExitSplashScreen(2 * time.Second); exitErr != nil {
		// Ignore splash errors
		_ = exitErr
	}

	return err
}
