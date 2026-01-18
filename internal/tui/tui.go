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

// Application states
type appState int

const (
	stateReady appState = iota
	stateCountdown
	stateRecording
)

// Key bindings
type keyMap struct {
	Toggle key.Binding
	Quit   key.Binding
	Help   key.Binding
	Cancel key.Binding
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
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// Messages
type tickMsg time.Time
type statusUpdateMsg models.RecordingStatus
type monitorsUpdateMsg []models.Monitor
type blinkMsg struct{}
type countdownTickMsg struct{}
type startRecordingMsg struct{}

// Model is the main TUI model
type Model struct {
	recorder      *recorder.Recorder
	status        models.RecordingStatus
	monitors      []models.Monitor
	spinner       spinner.Model
	width         int
	height        int
	showHelp      bool
	blinkOn       bool
	err           error
	state         appState
	countdownNum  int
}

// NewModel creates a new TUI model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorRed)

	rec := recorder.New()

	// Check if a recording is already in progress (restore state)
	initialState := stateReady
	status := rec.GetStatus()
	if status.IsRecording {
		initialState = stateRecording
	}

	return Model{
		recorder:     rec,
		spinner:      s,
		blinkOn:      true,
		state:        initialState,
		status:       status,
		countdownNum: 5,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		blinkCmd(),
		updateStatus(m.recorder),
		updateMonitors(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle cancel during countdown
		if m.state == stateCountdown {
			if key.Matches(msg, keys.Cancel) || msg.String() == "q" {
				m.state = stateReady
				m.countdownNum = 5
				return m, nil
			}
			// Ignore other keys during countdown
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Toggle):
			if m.status.IsRecording {
				// Stop recording immediately
				if err := m.recorder.Stop(); err != nil {
					m.err = err
				}
				m.state = stateReady
				return m, updateStatus(m.recorder)
			} else {
				// Start countdown
				m.state = stateCountdown
				m.countdownNum = 5
				// Play first beep
				go playBeep(5)
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					return countdownTickMsg{}
				})
			}

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.state != stateCountdown {
			return m, tea.Batch(
				tickCmd(),
				updateStatus(m.recorder),
				updateMonitors(),
			)
		}
		return m, tickCmd()

	case blinkMsg:
		m.blinkOn = !m.blinkOn
		return m, blinkCmd()

	case countdownTickMsg:
		if m.state != stateCountdown {
			return m, nil
		}

		m.countdownNum--

		if m.countdownNum < 0 {
			// Countdown finished, start recording
			m.state = stateRecording
			if err := m.recorder.Start(); err != nil {
				m.err = err
				m.state = stateReady
			}
			return m, updateStatus(m.recorder)
		}

		// Play beep for counts 5-1 (not for 0/GO)
		if m.countdownNum > 0 {
			go playBeep(m.countdownNum)
		}

		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return countdownTickMsg{}
		})

	case statusUpdateMsg:
		m.status = models.RecordingStatus(msg)
		if m.status.IsRecording {
			m.state = stateRecording
		} else if m.state == stateRecording {
			m.state = stateReady
		}
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

// View renders the UI using the standard layout
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Show countdown screen if in countdown state
	if m.state == stateCountdown {
		return m.renderCountdownView()
	}

	// Build header state
	headerState := &HeaderState{
		IsRecording: m.status.IsRecording,
		BlinkOn:     m.blinkOn,
	}

	if m.status.IsRecording {
		headerState.Duration = time.Since(m.status.StartTime).Round(time.Second).String()
		headerState.Monitor = m.status.Monitor
	}

	// Get current monitor with cursor
	cursorMonitor, _ := monitor.GetMouseMonitor()
	if cursorMonitor != "" && !m.status.IsRecording {
		headerState.Monitor = cursorMonitor
	}

	// Render header
	screenTitle := "Ready"
	if m.status.IsRecording {
		screenTitle = "Recording"
	}
	header := RenderHeader(screenTitle, headerState)

	// Render main content
	content := m.renderContent(cursorMonitor)

	// Render footer
	helpText := "space - toggle recording | q - quit | ? - help"
	footer := RenderHelpFooter(helpText, m.width)

	// Use standard layout
	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderCountdownView renders the countdown screen
func (m Model) renderCountdownView() string {
	var bigText []string
	var color lipgloss.Color

	if m.countdownNum > 0 {
		// Show digit
		bigText = getBigDigit(m.countdownNum)
		// Color changes as countdown progresses (orange -> red)
		switch m.countdownNum {
		case 5, 4:
			color = ColorOrange
		case 3, 2:
			color = lipgloss.Color("#FF8C00") // Dark orange
		case 1:
			color = ColorRed
		}
	} else {
		// Show GO!
		bigText = bigGO
		color = ColorGreen
	}

	// Style the big text
	digitStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	// Build the display
	var lines string
	for i, line := range bigText {
		lines += digitStyle.Render(line)
		if i < len(bigText)-1 {
			lines += "\n"
		}
	}

	// Add subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var subtitle string
	if m.countdownNum > 0 {
		subtitle = subtitleStyle.Render("Get ready... Recording starts soon!")
	} else {
		subtitle = subtitleStyle.Render("Recording!")
	}

	// Add cancel hint
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray)
	hint := hintStyle.Render("Press ESC to cancel")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		lines,
		"",
		subtitle,
		"",
		hint,
	)

	// Center on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderContent renders the main content area
func (m Model) renderContent(cursorMonitor string) string {
	// Monitor list
	monitorsContent := m.renderMonitors(cursorMonitor)

	// Recording info (if recording)
	var recordingInfo string
	if m.status.IsRecording {
		recordingInfo = m.renderRecordingInfo()
	}

	// Help content (if shown)
	var helpContent string
	if m.showHelp {
		helpContent = m.renderHelp()
	}

	// Combine content
	contentStyle := lipgloss.NewStyle().
		Width(HeaderWidth).
		Align(lipgloss.Center)

	var sections []string
	sections = append(sections, monitorsContent)

	if recordingInfo != "" {
		sections = append(sections, "", recordingInfo)
	}

	if helpContent != "" {
		sections = append(sections, "", helpContent)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)
	return contentStyle.Render(content)
}

// renderMonitors renders the monitor list
func (m Model) renderMonitors(cursorMonitor string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginBottom(1)

	markerStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	monitorStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	dimStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	var content string
	content += titleStyle.Render("Available Monitors") + "\n"

	if len(m.monitors) == 0 {
		content += dimStyle.Render("No monitors detected")
	} else {
		for _, mon := range m.monitors {
			marker := "  "
			style := monitorStyle
			if mon.Name == cursorMonitor {
				marker = markerStyle.Render("→ ")
				style = ActiveStyle
			}
			line := fmt.Sprintf("%s%s (%dx%d)",
				marker,
				mon.Name,
				mon.Width,
				mon.Height,
			)
			content += style.Render(line) + "\n"
		}
	}

	return content
}

// renderRecordingInfo renders info about the current recording
func (m Model) renderRecordingInfo() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorRed).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	var content string
	content += titleStyle.Render("Recording In Progress") + "\n"
	content += labelStyle.Render("Video: ") + valueStyle.Render(m.status.VideoFile) + "\n"
	content += labelStyle.Render("Audio: ") + valueStyle.Render(m.status.AudioFile) + "\n"

	if m.status.WebcamFile != "" {
		content += labelStyle.Render("Webcam: ") + valueStyle.Render(m.status.WebcamFile) + "\n"
	}

	return content
}

// renderHelp renders the help content
func (m Model) renderHelp() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	helpText := `Keyboard Shortcuts:
  space/enter  Toggle recording on/off
  q            Quit application
  ?            Toggle this help

Recording Features:
  • Video captured with wl-screenrec
  • Audio from default microphone
  • Webcam recorded if available
  • Audio denoised & normalized
  • Vertical video with webcam overlay`

	return titleStyle.Render("Help") + "\n" + helpStyle.Render(helpText)
}

// Commands

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkMsg{}
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
