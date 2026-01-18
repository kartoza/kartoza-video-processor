package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/monitor"
	"github.com/kartoza/kartoza-video-processor/internal/recorder"
)

// Screen represents the current screen being displayed
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenRecordingSetup
	ScreenRecording
	ScreenHistory
	ScreenOptions
)

// AppModel is the main application model that coordinates screens
type AppModel struct {
	screen          Screen
	menu            *MenuModel
	recordingSetup  *RecordingSetupModel
	options         *OptionsModel
	history         *HistoryModel
	recorder        *recorder.Recorder
	status          models.RecordingStatus
	monitors        []models.Monitor
	spinner         spinner.Model
	width           int
	height          int
	showHelp        bool
	blinkOn         bool
	err             error
	state           appState
	countdownNum    int
	processing      *ProcessingState
	processingFrame int
	metadata        models.RecordingMetadata
	recordingInfo   *models.RecordingInfo
	outputDir       string

	// External recording detection
	externalRecordingActive bool
	externalRecordingPIDs   []string
}

// countRecordings counts the number of valid recordings in the screencasts folder
func countRecordings() int {
	videosDir := config.GetDefaultVideosDir()

	if _, err := os.Stat(videosDir); os.IsNotExist(err) {
		return 0
	}

	entries, err := os.ReadDir(videosDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if folder has recording.json
		infoPath := filepath.Join(videosDir, entry.Name(), "recording.json")
		if _, err := os.Stat(infoPath); err == nil {
			count++
		}
	}

	return count
}

// updateGlobalAppState updates the global app state for header display
func updateGlobalAppState(isRecording bool, blinkOn bool, status string) {
	GlobalAppState.IsRecording = isRecording
	GlobalAppState.BlinkOn = blinkOn
	GlobalAppState.Status = status
	GlobalAppState.TotalRecordings = countRecordings()
}

// checkExternalRecording checks if wl-screenrec processes are running externally
func checkExternalRecording() (bool, []string) {
	// Use pgrep to find wl-screenrec processes
	cmd := exec.Command("pgrep", "-a", "wl-screenrec")
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 if no processes found
		return false, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var pids []string
	for _, line := range lines {
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				pids = append(pids, parts[0])
			}
		}
	}

	return len(pids) > 0, pids
}

// NewAppModel creates a new application model
func NewAppModel() AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorRed)

	rec := recorder.New()

	// Check if a recording is already in progress
	status := rec.GetStatus()
	initialScreen := ScreenMenu
	initialState := stateReady

	if status.IsRecording {
		initialScreen = ScreenRecording
		initialState = stateRecording
	}

	// Check for external wl-screenrec processes
	externalActive, externalPIDs := checkExternalRecording()

	// Create menu and set external recording state
	menu := NewMenuModel()
	menu.SetExternalRecording(externalActive, externalPIDs)

	// Initialize global app state
	GlobalAppState.TotalRecordings = countRecordings()
	if status.IsRecording {
		GlobalAppState.IsRecording = true
		GlobalAppState.Status = "Recording"
	} else {
		GlobalAppState.IsRecording = false
		GlobalAppState.Status = "Ready"
	}

	return AppModel{
		screen:                  initialScreen,
		menu:                    menu,
		recordingSetup:          NewRecordingSetupModel(),
		options:                 NewOptionsModel(),
		history:                 NewHistoryModel(),
		recorder:                rec,
		spinner:                 s,
		blinkOn:                 true,
		state:                   initialState,
		status:                  status,
		countdownNum:            5,
		processing:              NewProcessingState(),
		processingFrame:         0,
		externalRecordingActive: externalActive,
		externalRecordingPIDs:   externalPIDs,
	}
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		blinkCmd(),
		updateStatus(m.recorder),
		updateMonitors(),
	)
}

// Update handles messages
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle recording setup completion messages first (from any screen)
	switch msg.(type) {
	case recordingSetupCompleteMsg:
		// Recording setup is complete, save metadata and start countdown
		m.metadata = m.recordingSetup.GetMetadata()
		m.screen = ScreenRecording
		m.state = stateCountdown
		m.countdownNum = 5
		go playBeep(5)
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return countdownTickMsg{}
		})
	case backToMenuMsg:
		m.screen = ScreenMenu
		m.recordingSetup = NewRecordingSetupModel()
		return m, nil
	}

	// If on recording setup screen, pass messages to the form
	if m.screen == ScreenRecordingSetup {
		// Handle escape to go back (before passing to form)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
				m.screen = ScreenMenu
				return m, nil
			}
			if key.Matches(keyMsg, key.NewBinding(key.WithKeys("ctrl+c"))) {
				return m, tea.Quit
			}
		}

		// Pass message to the form
		newSetup, cmd := m.recordingSetup.Update(msg)
		m.recordingSetup = newSetup

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu.width = msg.Width
		m.menu.height = msg.Height
		// Also update recording setup dimensions
		if m.recordingSetup != nil {
			m.recordingSetup.width = msg.Width
			m.recordingSetup.height = msg.Height
		}
		return m, nil

	case tea.KeyMsg:
		// Handle based on current screen and state
		return m.handleKeyMsg(msg)

	case tickMsg:
		if m.state != stateCountdown {
			// Re-check for external recordings
			externalActive, externalPIDs := checkExternalRecording()
			if m.externalRecordingActive != externalActive {
				m.externalRecordingActive = externalActive
				m.externalRecordingPIDs = externalPIDs
				m.menu.SetExternalRecording(externalActive, externalPIDs)
			}

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
		return m.handleCountdownTick()

	case statusUpdateMsg:
		m.status = models.RecordingStatus(msg)
		if m.status.IsRecording {
			m.state = stateRecording
			m.screen = ScreenRecording
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

	case processingTickMsg:
		if m.state == stateProcessing {
			m.processingFrame++
			return m, processingTickCmd()
		}
		return m, nil

	case processingStepMsg:
		if m.state == stateProcessing && m.processing != nil {
			if !msg.Completed {
				// Step is starting
				m.processing.SetStepByIndex(msg.Step, StepRunning)
			} else if msg.Skipped {
				// Step was skipped
				m.processing.SetStepByIndex(msg.Step, StepSkipped)
			} else if msg.Error != nil {
				// Step failed
				m.processing.SetStepByIndex(msg.Step, StepFailed)
				m.processing.Error = msg.Error
			} else {
				// Step completed successfully
				m.processing.SetStepByIndex(msg.Step, StepComplete)

				// If step 0 (stopping recorders) completed, start the processing pipeline
				if msg.Step == 0 {
					return m, startProcessingPipeline(m.recorder, m.processing)
				}
			}
		}
		return m, nil

	case processingCompleteMsg:
		if m.state == stateProcessing && m.processing != nil {
			m.processing.Complete()
			return m, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
				return processingDoneMsg{}
			})
		}
		return m, nil

	case processingDoneMsg:
		m.state = stateReady
		m.screen = ScreenMenu
		m.processing.Reset()
		// Update global state - recording complete, refresh count
		updateGlobalAppState(false, true, "Ready")
		return m, updateStatus(m.recorder)

	case processingErrorMsg:
		if m.state == stateProcessing && m.processing != nil {
			m.processing.FailStep(msg.Error)
			m.err = msg.Error
		}
		return m, nil

	case menuActionMsg:
		return m.handleMenuAction(msg.action)

	case recordingSetupCompleteMsg:
		// Recording setup is complete, save metadata and start countdown
		m.metadata = m.recordingSetup.GetMetadata()
		m.screen = ScreenRecording
		m.state = stateCountdown
		m.countdownNum = 5
		go playBeep(5)
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return countdownTickMsg{}
		})

	case backToMenuMsg:
		// Return to main menu from history
		m.screen = ScreenMenu
		return m, nil

	case recordingsLoadedMsg:
		// Forward to history model
		if m.screen == ScreenHistory && m.history != nil {
			newHistory, cmd := m.history.Update(msg)
			m.history = newHistory
			return m, cmd
		}
		return m, nil

	case recordingSavedMsg:
		// Forward to history model
		if m.screen == ScreenHistory && m.history != nil {
			newHistory, cmd := m.history.Update(msg)
			m.history = newHistory
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input based on current state
func (m AppModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle processing state - only allow quit
	if m.state == stateProcessing {
		if key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))) {
			return m, tea.Quit
		}
		return m, nil
	}

	// Handle countdown state
	if m.state == stateCountdown {
		if key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))) {
			m.state = stateReady
			m.countdownNum = 5
			m.screen = ScreenMenu
			return m, nil
		}
		return m, nil
	}

	// Handle based on screen
	switch m.screen {
	case ScreenMenu:
		return m.handleMenuKeys(msg)
	case ScreenRecordingSetup:
		return m.handleRecordingSetupKeys(msg)
	case ScreenRecording:
		return m.handleRecordingKeys(msg)
	case ScreenHistory:
		return m.handleHistoryKeys(msg)
	case ScreenOptions:
		return m.handleOptionsKeys(msg)
	}

	return m, nil
}

// handleMenuKeys handles keys on the menu screen
func (m AppModel) handleMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu
	return m, cmd
}

// handleRecordingSetupKeys handles keys on the recording setup screen
func (m AppModel) handleRecordingSetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape to go back
	if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
		m.screen = ScreenMenu
		return m, nil
	}

	// Handle quit
	if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
		return m, tea.Quit
	}

	// Update the setup form
	newSetup, cmd := m.recordingSetup.Update(msg)
	m.recordingSetup = newSetup
	return m, cmd
}

// handleRecordingKeys handles keys on the recording screen
func (m AppModel) handleRecordingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
		return m, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "enter"))):
		if m.status.IsRecording {
			// Stop recording - transition to processing state
			m.state = stateProcessing
			m.processing.Reset()
			m.processing.Start()
			m.processingFrame = 0

			return m, tea.Batch(
				processingTickCmd(),
				m.stopAndProcess(),
			)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Go back to menu (only if not recording)
		if !m.status.IsRecording {
			m.screen = ScreenMenu
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("?"))):
		m.showHelp = !m.showHelp
		return m, nil
	}

	return m, nil
}

// handleHistoryKeys handles keys on the history screen
func (m AppModel) handleHistoryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let history model handle keys
	newHistory, cmd := m.history.Update(msg)
	m.history = newHistory
	return m, cmd
}

// handleOptionsKeys handles keys on the options screen
func (m AppModel) handleOptionsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape to go back
	if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
		m.screen = ScreenMenu
		return m, nil
	}

	// Handle quit
	if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
		return m, tea.Quit
	}

	// Update the options form
	newOptions, cmd := m.options.Update(msg)
	m.options = newOptions
	return m, cmd
}

// handleMenuAction handles menu item selection
func (m AppModel) handleMenuAction(action MenuItem) (tea.Model, tea.Cmd) {
	switch action {
	case MenuNewRecording:
		// Go to recording setup screen first
		m.screen = ScreenRecordingSetup
		m.recordingSetup = NewRecordingSetupModel()
		m.recordingSetup.width = m.width
		m.recordingSetup.height = m.height
		return m, m.recordingSetup.Init()

	case MenuRecordingHistory:
		m.screen = ScreenHistory
		m.history = NewHistoryModel()
		m.history.width = m.width
		m.history.height = m.height
		return m, m.history.Init()

	case MenuOptions:
		m.screen = ScreenOptions
		m.options = NewOptionsModel()
		m.options.width = m.width
		m.options.height = m.height
		return m, m.options.Init()

	case MenuQuit:
		return m, tea.Quit
	}

	return m, nil
}

// handleCountdownTick handles countdown timer ticks
func (m AppModel) handleCountdownTick() (tea.Model, tea.Cmd) {
	if m.state != stateCountdown {
		return m, nil
	}

	m.countdownNum--

	if m.countdownNum < 0 {
		// Countdown finished, start recording
		m.state = stateRecording

		// Generate folder name and create recording directory
		m.metadata.GenerateFolderName()
		baseDir := config.GetDefaultVideosDir()
		m.outputDir = filepath.Join(baseDir, m.metadata.FolderName)

		// Create the recording directory
		if err := os.MkdirAll(m.outputDir, 0755); err != nil {
			m.err = fmt.Errorf("failed to create recording directory: %w", err)
			m.state = stateReady
			m.screen = ScreenMenu
			return m, nil
		}

		// Get monitor info for recording
		monitorName, _ := monitor.GetMouseMonitor()
		if m.recordingSetup != nil && m.recordingSetup.selectedMonitor >= 0 && m.recordingSetup.selectedMonitor < len(m.recordingSetup.monitors) {
			monitorName = m.recordingSetup.monitors[m.recordingSetup.selectedMonitor].Name
		}
		monitorResolution := ""
		for _, mon := range m.monitors {
			if mon.Name == monitorName {
				monitorResolution = fmt.Sprintf("%dx%d", mon.Width, mon.Height)
				break
			}
		}

		// Create RecordingInfo and save initial metadata
		m.recordingInfo = models.NewRecordingInfo(m.metadata, monitorName, monitorResolution)
		m.recordingInfo.Files.FolderPath = m.outputDir

		// Set recording settings from setup form
		if m.recordingSetup != nil {
			m.recordingInfo.Settings.AudioEnabled = m.recordingSetup.recordAudio
			m.recordingInfo.Settings.WebcamEnabled = m.recordingSetup.recordWebcam
		}

		// Save initial recording.json
		if err := m.recordingInfo.Save(); err != nil {
			m.err = fmt.Errorf("failed to save recording metadata: %w", err)
			m.state = stateReady
			m.screen = ScreenMenu
			return m, nil
		}

		// Set up recorder options
		opts := recorder.Options{
			OutputDir:      m.outputDir,
			Monitor:        monitorName,
			Metadata:       &m.metadata,
			RecordingInfo:  m.recordingInfo,
			CreateVertical: m.recordingSetup != nil && m.recordingSetup.verticalVideo,
		}

		// Set audio/webcam/screen options from setup
		if m.recordingSetup != nil {
			opts.NoAudio = !m.recordingSetup.recordAudio
			opts.NoWebcam = !m.recordingSetup.recordWebcam
			// If recordScreen is false, we shouldn't record at all (this is a safety check)
			if !m.recordingSetup.recordScreen {
				opts.Monitor = "" // This will cause an error if no webcam/audio
			}
			// Set logo selection and save for future recordings
			opts.LogoSelection = m.recordingSetup.GetLogoSelection()
			_ = m.recordingSetup.SaveLogoSelection() // Save for next time
		}

		if err := m.recorder.StartWithOptions(opts); err != nil {
			m.err = err
			m.state = stateReady
			m.screen = ScreenMenu
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
}

// stopAndProcess stops recording and runs post-processing with progress updates
func (m AppModel) stopAndProcess() tea.Cmd {
	return func() tea.Msg {
		if err := m.recorder.Stop(); err != nil {
			return processingErrorMsg{Error: err}
		}
		// Step 0 (stopping recorders) is complete
		return processingStepMsg{Step: 0, Completed: true}
	}
}

// startProcessingPipeline begins the post-processing pipeline
func startProcessingPipeline(rec *recorder.Recorder, processing *ProcessingState) tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan recorder.ProgressUpdate, 20)

		// Start processing in a goroutine
		go rec.ProcessWithProgress(progressChan)

		// Process all updates
		for update := range progressChan {
			// Check if this is a percent update (no status change, just progress)
			if update.Percent >= 0 && !update.Completed && !update.Skipped && update.Error == nil {
				processing.SetStepProgress(update.Step, update.Percent)
				continue
			}

			// Update the processing state directly (thread-safe via the state machine)
			if !update.Completed {
				processing.SetStepByIndex(update.Step, StepRunning)
			} else if update.Skipped {
				processing.SetStepByIndex(update.Step, StepSkipped)
			} else if update.Error != nil {
				processing.SetStepByIndex(update.Step, StepFailed)
				processing.Error = update.Error
			} else {
				processing.SetStepByIndex(update.Step, StepComplete)
			}
		}

		return processingCompleteMsg{}
	}
}

// View renders the current screen
func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Show countdown screen if in countdown state
	if m.state == stateCountdown {
		return m.renderCountdownView()
	}

	// Show processing screen if in processing state
	if m.state == stateProcessing {
		return RenderProcessingView(m.processing, m.width, m.height, m.processingFrame)
	}

	// Render based on current screen
	switch m.screen {
	case ScreenMenu:
		return m.menu.View()
	case ScreenRecordingSetup:
		return m.renderRecordingSetupScreen()
	case ScreenRecording:
		return m.renderRecordingScreen()
	case ScreenHistory:
		return m.renderHistoryScreen()
	case ScreenOptions:
		return m.renderOptionsScreen()
	}

	return ""
}

// renderRecordingScreen renders the recording screen
func (m AppModel) renderRecordingScreen() string {
	// Update global app state for header
	status := "Ready"
	if m.status.IsRecording {
		status = "Recording"
	}
	updateGlobalAppState(m.status.IsRecording, m.blinkOn, status)

	// Get current monitor with cursor
	cursorMonitor, _ := monitor.GetMouseMonitor()

	// Render header
	screenTitle := "Recording"
	header := RenderHeader(screenTitle)

	// Render main content
	content := m.renderRecordingContent(cursorMonitor)

	// Render footer
	var helpText string
	if m.status.IsRecording {
		helpText = "space/enter: stop recording • q: quit"
	} else {
		helpText = "esc: back to menu • q: quit"
	}
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderRecordingContent renders the main content for the recording screen
func (m AppModel) renderRecordingContent(cursorMonitor string) string {
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
func (m AppModel) renderMonitors(cursorMonitor string) string {
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
			line := fmt.Sprintf("%s%s (%dx%d)", marker, mon.Name, mon.Width, mon.Height)
			content += style.Render(line) + "\n"
		}
	}

	return content
}

// renderRecordingInfo renders info about the current recording
func (m AppModel) renderRecordingInfo() string {
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
func (m AppModel) renderHelp() string {
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

// renderCountdownView renders the countdown screen
func (m AppModel) renderCountdownView() string {
	var bigText []string
	var color lipgloss.Color

	if m.countdownNum > 0 {
		bigText = getBigDigit(m.countdownNum)
		switch m.countdownNum {
		case 5, 4:
			color = ColorOrange
		case 3, 2:
			color = lipgloss.Color("#FF8C00")
		case 1:
			color = ColorRed
		}
	} else {
		bigText = bigGO
		color = ColorGreen
	}

	digitStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	var lines string
	for i, line := range bigText {
		lines += digitStyle.Render(line)
		if i < len(bigText)-1 {
			lines += "\n"
		}
	}

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var subtitle string
	if m.countdownNum > 0 {
		subtitle = subtitleStyle.Render("Get ready... Recording starts soon!")
	} else {
		subtitle = subtitleStyle.Render("Recording!")
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray)
	hint := hintStyle.Render("Press ESC to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		lines,
		"",
		subtitle,
		"",
		hint,
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderRecordingSetupScreen renders the recording setup screen
func (m AppModel) renderRecordingSetupScreen() string {
	header := RenderHeader("New Recording")

	// Render the setup form
	content := lipgloss.NewStyle().
		Width(HeaderWidth).
		Align(lipgloss.Center).
		Render(m.recordingSetup.View())

	footer := RenderHelpFooter("tab/↓: next • shift+tab/↑: prev • ←/→: select topic • enter: confirm • esc: back", m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderHistoryScreen renders the history screen
func (m AppModel) renderHistoryScreen() string {
	if m.history == nil {
		return "Loading..."
	}
	return m.history.View()
}

// renderOptionsScreen renders the options screen
func (m AppModel) renderOptionsScreen() string {
	// If file browser is active, it takes over the full screen
	if m.options.IsFileBrowserActive() {
		return m.options.RenderFileBrowser(m.width, m.height)
	}

	header := RenderHeader("Options")

	content := lipgloss.NewStyle().
		Width(HeaderWidth).
		Align(lipgloss.Center).
		Render(m.options.View())

	footer := RenderHelpFooter("tab/↓: next • shift+tab/↑: prev • enter: select • esc: back", m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}
