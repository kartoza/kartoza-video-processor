package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/monitor"
)

// Field indices for navigation
const (
	fieldTitle = iota
	fieldNumber
	fieldTopic
	fieldRecordAudio
	fieldRecordWebcam
	fieldRecordScreen
	fieldMonitor
	fieldVerticalVideo
	fieldAddLogos
	fieldLeftLogo
	fieldRightLogo
	fieldBottomLogo
	fieldTitleColor
	fieldGifLoopMode
	fieldDescription
	fieldConfirm
	fieldCount
)

// RecordingSetupModel handles the recording setup form
type RecordingSetupModel struct {
	width  int
	height int

	focusedField int
	inputMode    bool // When true, text input captures all keys; tab exits
	config       *config.Config

	// Text inputs
	titleInput  textinput.Model
	numberInput textinput.Model
	descInput   textarea.Model

	// Options (bool toggles)
	recordAudio   bool
	recordWebcam  bool
	recordScreen  bool
	verticalVideo bool
	addLogos      bool

	// Logo selection
	logoDirectory     string   // Directory containing logos
	availableLogos    []string // List of logo files in the directory
	leftLogo          string   // Selected left logo path
	rightLogo         string   // Selected right logo path
	bottomLogo          string            // Selected bottom logo path
	selectedLeftIdx     int               // Index in availableLogos for left
	selectedRightIdx    int               // Index in availableLogos for right
	selectedBottomIdx   int               // Index in availableLogos for bottom
	titleColor          string            // Selected title text color
	selectedColorIdx    int               // Index in TitleColors
	gifLoopMode         config.GifLoopMode // How to loop animated GIFs
	selectedGifLoopIdx  int               // Index in GifLoopModes

	// Screen selection
	monitors        []models.Monitor
	selectedMonitor int

	// Available topics
	topics        []models.Topic
	selectedTopic int

	// Confirm selection (true = Go Live, false = Cancel)
	confirmSelected bool
}

// NewRecordingSetupModel creates a new recording setup model
func NewRecordingSetupModel() *RecordingSetupModel {
	cfg, _ := config.Load()
	recordingNumber := config.GetCurrentRecordingNumber()

	topics := cfg.Topics
	if len(topics) == 0 {
		topics = models.DefaultTopics()
	}

	// Get available monitors
	monitors, _ := monitor.ListMonitors()

	// Create text inputs
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter recording title..."
	titleInput.CharLimit = 100
	titleInput.Width = 30
	titleInput.Focus()

	numberInput := textinput.New()
	numberInput.Placeholder = "001"
	numberInput.CharLimit = 10
	numberInput.Width = 30
	numberInput.SetValue(fmt.Sprintf("%03d", recordingNumber))

	descInput := textarea.New()
	descInput.Placeholder = "Enter description..."
	descInput.CharLimit = 2000
	descInput.SetWidth(35)
	descInput.SetHeight(4)
	descInput.ShowLineNumbers = false

	// Determine title color - use last used or default
	titleColor := cfg.LastUsedLogos.TitleColor
	if titleColor == "" {
		titleColor = config.DefaultTitleColor
	}
	// Find index of title color
	colorIdx := 0
	for i, c := range config.TitleColors {
		if c == titleColor {
			colorIdx = i
			break
		}
	}

	// Determine GIF loop mode - use last used or default to continuous
	gifLoopMode := cfg.LastUsedLogos.GifLoopMode
	if gifLoopMode == "" {
		gifLoopMode = config.GifLoopContinuous
	}
	// Find index of GIF loop mode
	gifLoopIdx := 0
	for i, mode := range config.GifLoopModes {
		if mode == gifLoopMode {
			gifLoopIdx = i
			break
		}
	}

	// Load recording presets (use defaults if not set)
	presets := cfg.RecordingPresets
	// Check if presets have been saved before (if all bools are false, use defaults)
	presetsExist := presets.RecordAudio || presets.RecordWebcam || presets.RecordScreen || presets.VerticalVideo || presets.AddLogos
	if !presetsExist {
		presets = config.DefaultRecordingPresets()
	}

	// Find topic index from saved preset
	selectedTopicIdx := 0
	if presets.Topic != "" {
		for i, t := range topics {
			if t.Name == presets.Topic {
				selectedTopicIdx = i
				break
			}
		}
	}

	m := &RecordingSetupModel{
		config:             cfg,
		focusedField:       fieldTitle,
		titleInput:         titleInput,
		numberInput:        numberInput,
		descInput:          descInput,
		recordAudio:        presets.RecordAudio,
		recordWebcam:       presets.RecordWebcam,
		recordScreen:       presets.RecordScreen,
		verticalVideo:      presets.VerticalVideo,
		addLogos:           presets.AddLogos,
		monitors:           monitors,
		selectedMonitor:    0,
		topics:             topics,
		selectedTopic:      selectedTopicIdx,
		confirmSelected:    true, // Default to "Go Live"
		logoDirectory:      cfg.LogoDirectory,
		leftLogo:           cfg.LastUsedLogos.LeftLogo,
		rightLogo:          cfg.LastUsedLogos.RightLogo,
		bottomLogo:         cfg.LastUsedLogos.BottomLogo,
		titleColor:         titleColor,
		selectedColorIdx:   colorIdx,
		gifLoopMode:        gifLoopMode,
		selectedGifLoopIdx: gifLoopIdx,
	}

	// Load available logos from directory
	m.loadAvailableLogos()

	return m
}

// loadAvailableLogos scans the logo directory for image files
func (m *RecordingSetupModel) loadAvailableLogos() {
	m.availableLogos = []string{"(none)"} // First option is always "none"

	if m.logoDirectory == "" {
		return
	}

	entries, err := os.ReadDir(m.logoDirectory)
	if err != nil {
		return
	}

	var logos []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			logos = append(logos, entry.Name())
		}
	}

	sort.Strings(logos)
	m.availableLogos = append(m.availableLogos, logos...)

	// Find indices for current selections
	m.selectedLeftIdx = m.findLogoIndex(m.leftLogo)
	m.selectedRightIdx = m.findLogoIndex(m.rightLogo)
	m.selectedBottomIdx = m.findLogoIndex(m.bottomLogo)
}

// findLogoIndex finds the index of a logo path in availableLogos
func (m *RecordingSetupModel) findLogoIndex(logoPath string) int {
	if logoPath == "" {
		return 0 // (none)
	}
	name := filepath.Base(logoPath)
	for i, logo := range m.availableLogos {
		if logo == name {
			return i
		}
	}
	return 0
}

// getLogoPath returns the full path for a selected logo index
func (m *RecordingSetupModel) getLogoPath(idx int) string {
	if idx == 0 || idx >= len(m.availableLogos) || m.logoDirectory == "" {
		return ""
	}
	return filepath.Join(m.logoDirectory, m.availableLogos[idx])
}

func (m *RecordingSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RecordingSetupModel) isTextField() bool {
	return m.focusedField == fieldTitle || m.focusedField == fieldNumber || m.focusedField == fieldDescription
}

func (m *RecordingSetupModel) Update(msg tea.Msg) (*RecordingSetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// In input mode, only tab/shift+tab/esc exit; all other keys go to text input
		if m.inputMode {
			switch msg.String() {
			case "tab":
				m.inputMode = false
				m.nextField()
				return m, nil
			case "shift+tab":
				m.inputMode = false
				m.prevField()
				return m, nil
			case "esc":
				// Escape exits input mode but stays on current field
				m.inputMode = false
				return m, nil
			case "enter":
				// For single-line inputs, enter exits input mode
				// For textarea (description), enter adds a newline
				if m.focusedField == fieldDescription {
					m.descInput, cmd = m.descInput.Update(msg)
					return m, cmd
				}
				m.inputMode = false
				m.nextField()
				return m, nil
			default:
				// Pass all other keys to the text input
				switch m.focusedField {
				case fieldTitle:
					m.titleInput, cmd = m.titleInput.Update(msg)
				case fieldNumber:
					m.numberInput, cmd = m.numberInput.Update(msg)
				case fieldDescription:
					m.descInput, cmd = m.descInput.Update(msg)
				}
				return m, cmd
			}
		}

		// Navigation mode
		switch msg.String() {
		case "tab", "down", "j":
			m.nextField()
			return m, nil
		case "shift+tab", "up", "k":
			m.prevField()
			return m, nil
		case "left", "h":
			m.handleLeft()
			return m, nil
		case "right", "l":
			m.handleRight()
			return m, nil
		case " ":
			// Space toggles boolean fields or activates confirm button
			if m.focusedField == fieldConfirm {
				if m.confirmSelected {
					// Go Live selected
					if m.Validate() {
						return m, func() tea.Msg { return recordingSetupCompleteMsg{} }
					}
				} else {
					// Cancel selected
					return m, func() tea.Msg { return backToMenuMsg{} }
				}
				return m, nil
			}
			if m.handleToggle() {
				return m, nil
			}
		case "enter":
			// On text fields, enter activates input mode
			if m.isTextField() {
				m.inputMode = true
				return m, nil
			}
			if m.focusedField == fieldConfirm {
				if m.confirmSelected {
					// Go Live selected
					if m.Validate() {
						return m, func() tea.Msg { return recordingSetupCompleteMsg{} }
					}
				} else {
					// Cancel selected
					return m, func() tea.Msg { return backToMenuMsg{} }
				}
				return m, nil
			}
			// Enter moves to next field for other fields
			m.nextField()
			return m, nil
		default:
			// If on a text field and user types a printable character, enter input mode
			if m.isTextField() && len(msg.String()) == 1 {
				m.inputMode = true
				// Pass the key to the text input
				switch m.focusedField {
				case fieldTitle:
					m.titleInput, cmd = m.titleInput.Update(msg)
				case fieldNumber:
					m.numberInput, cmd = m.numberInput.Update(msg)
				case fieldDescription:
					m.descInput, cmd = m.descInput.Update(msg)
				}
				return m, cmd
			}
		}
	}

	return m, cmd
}

// isBottomLogoGif returns true if the bottom logo is a GIF file
func (m *RecordingSetupModel) isBottomLogoGif() bool {
	if m.bottomLogo == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(m.bottomLogo))
	return ext == ".gif"
}

func (m *RecordingSetupModel) nextField() {
	m.blurAll()
	m.focusedField++

	// Skip fields based on conditions
	for {
		skip := false
		if m.focusedField == fieldMonitor && !m.recordScreen {
			skip = true
		}
		if (m.focusedField == fieldLeftLogo || m.focusedField == fieldRightLogo || m.focusedField == fieldBottomLogo || m.focusedField == fieldTitleColor) && !m.addLogos {
			skip = true
		}
		// Skip GIF loop mode if logos not enabled or bottom logo is not a GIF
		if m.focusedField == fieldGifLoopMode && (!m.addLogos || !m.isBottomLogoGif()) {
			skip = true
		}
		if !skip {
			break
		}
		m.focusedField++
		if m.focusedField >= fieldCount {
			m.focusedField = fieldTitle
			break
		}
	}

	if m.focusedField >= fieldCount {
		m.focusedField = fieldTitle
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) prevField() {
	m.blurAll()
	m.focusedField--

	// Skip fields based on conditions
	for {
		skip := false
		if m.focusedField == fieldMonitor && !m.recordScreen {
			skip = true
		}
		if (m.focusedField == fieldLeftLogo || m.focusedField == fieldRightLogo || m.focusedField == fieldBottomLogo || m.focusedField == fieldTitleColor) && !m.addLogos {
			skip = true
		}
		// Skip GIF loop mode if logos not enabled or bottom logo is not a GIF
		if m.focusedField == fieldGifLoopMode && (!m.addLogos || !m.isBottomLogoGif()) {
			skip = true
		}
		if !skip {
			break
		}
		m.focusedField--
		if m.focusedField < fieldTitle {
			m.focusedField = fieldConfirm
			break
		}
	}

	if m.focusedField < fieldTitle {
		m.focusedField = fieldConfirm
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) blurAll() {
	m.titleInput.Blur()
	m.numberInput.Blur()
	m.descInput.Blur()
	m.inputMode = false
}

func (m *RecordingSetupModel) focusCurrent() {
	switch m.focusedField {
	case fieldTitle:
		m.titleInput.Focus()
	case fieldNumber:
		m.numberInput.Focus()
	case fieldDescription:
		m.descInput.Focus()
	}
}

func (m *RecordingSetupModel) handleLeft() {
	switch m.focusedField {
	case fieldTopic:
		m.selectedTopic--
		if m.selectedTopic < 0 {
			m.selectedTopic = len(m.topics) - 1
		}
	case fieldMonitor:
		m.selectedMonitor--
		if m.selectedMonitor < 0 {
			m.selectedMonitor = len(m.monitors) - 1
		}
	case fieldLeftLogo:
		m.selectedLeftIdx--
		if m.selectedLeftIdx < 0 {
			m.selectedLeftIdx = len(m.availableLogos) - 1
		}
		m.leftLogo = m.getLogoPath(m.selectedLeftIdx)
	case fieldRightLogo:
		m.selectedRightIdx--
		if m.selectedRightIdx < 0 {
			m.selectedRightIdx = len(m.availableLogos) - 1
		}
		m.rightLogo = m.getLogoPath(m.selectedRightIdx)
	case fieldBottomLogo:
		m.selectedBottomIdx--
		if m.selectedBottomIdx < 0 {
			m.selectedBottomIdx = len(m.availableLogos) - 1
		}
		m.bottomLogo = m.getLogoPath(m.selectedBottomIdx)
	case fieldTitleColor:
		m.selectedColorIdx--
		if m.selectedColorIdx < 0 {
			m.selectedColorIdx = len(config.TitleColors) - 1
		}
		m.titleColor = config.TitleColors[m.selectedColorIdx]
	case fieldGifLoopMode:
		m.selectedGifLoopIdx--
		if m.selectedGifLoopIdx < 0 {
			m.selectedGifLoopIdx = len(config.GifLoopModes) - 1
		}
		m.gifLoopMode = config.GifLoopModes[m.selectedGifLoopIdx]
	case fieldRecordAudio, fieldRecordWebcam, fieldRecordScreen, fieldVerticalVideo, fieldAddLogos:
		m.handleToggle()
	case fieldConfirm:
		m.confirmSelected = !m.confirmSelected
	}
}

func (m *RecordingSetupModel) handleRight() {
	switch m.focusedField {
	case fieldTopic:
		m.selectedTopic++
		if m.selectedTopic >= len(m.topics) {
			m.selectedTopic = 0
		}
	case fieldMonitor:
		m.selectedMonitor++
		if m.selectedMonitor >= len(m.monitors) {
			m.selectedMonitor = 0
		}
	case fieldLeftLogo:
		m.selectedLeftIdx++
		if m.selectedLeftIdx >= len(m.availableLogos) {
			m.selectedLeftIdx = 0
		}
		m.leftLogo = m.getLogoPath(m.selectedLeftIdx)
	case fieldRightLogo:
		m.selectedRightIdx++
		if m.selectedRightIdx >= len(m.availableLogos) {
			m.selectedRightIdx = 0
		}
		m.rightLogo = m.getLogoPath(m.selectedRightIdx)
	case fieldBottomLogo:
		m.selectedBottomIdx++
		if m.selectedBottomIdx >= len(m.availableLogos) {
			m.selectedBottomIdx = 0
		}
		m.bottomLogo = m.getLogoPath(m.selectedBottomIdx)
	case fieldTitleColor:
		m.selectedColorIdx++
		if m.selectedColorIdx >= len(config.TitleColors) {
			m.selectedColorIdx = 0
		}
		m.titleColor = config.TitleColors[m.selectedColorIdx]
	case fieldGifLoopMode:
		m.selectedGifLoopIdx++
		if m.selectedGifLoopIdx >= len(config.GifLoopModes) {
			m.selectedGifLoopIdx = 0
		}
		m.gifLoopMode = config.GifLoopModes[m.selectedGifLoopIdx]
	case fieldRecordAudio, fieldRecordWebcam, fieldRecordScreen, fieldVerticalVideo, fieldAddLogos:
		m.handleToggle()
	case fieldConfirm:
		m.confirmSelected = !m.confirmSelected
	}
}

func (m *RecordingSetupModel) handleToggle() bool {
	switch m.focusedField {
	case fieldRecordAudio:
		m.recordAudio = !m.recordAudio
		return true
	case fieldRecordWebcam:
		m.recordWebcam = !m.recordWebcam
		// Disable vertical video if neither webcam nor screen is enabled
		if !m.recordWebcam && !m.recordScreen {
			m.verticalVideo = false
		}
		return true
	case fieldRecordScreen:
		m.recordScreen = !m.recordScreen
		// Disable vertical video if neither webcam nor screen is enabled
		if !m.recordWebcam && !m.recordScreen {
			m.verticalVideo = false
		}
		return true
	case fieldVerticalVideo:
		// Only allow toggle if webcam or screen is enabled
		if m.recordWebcam || m.recordScreen {
			m.verticalVideo = !m.verticalVideo
		}
		return true
	case fieldAddLogos:
		m.addLogos = !m.addLogos
		return true
	}
	return false
}

func (m *RecordingSetupModel) canEnableVerticalVideo() bool {
	return m.recordWebcam || m.recordScreen
}

func (m *RecordingSetupModel) Validate() bool {
	// Title is required
	if strings.TrimSpace(m.titleInput.Value()) == "" {
		return false
	}
	// At least one recording source must be enabled
	if !m.recordAudio && !m.recordWebcam && !m.recordScreen {
		return false
	}
	return true
}

func (m *RecordingSetupModel) hasRecordingSource() bool {
	return m.recordAudio || m.recordWebcam || m.recordScreen
}

func (m *RecordingSetupModel) GetMetadata() models.RecordingMetadata {
	topic := ""
	if m.selectedTopic >= 0 && m.selectedTopic < len(m.topics) {
		topic = m.topics[m.selectedTopic].Name
	}

	recordingNumber := 1
	if num, err := strconv.Atoi(strings.TrimSpace(m.numberInput.Value())); err == nil && num > 0 {
		recordingNumber = num
	}

	metadata := models.RecordingMetadata{
		Number:      recordingNumber,
		Title:       strings.TrimSpace(m.titleInput.Value()),
		Description: strings.TrimSpace(m.descInput.Value()),
		Topic:       topic,
		Presenter:   m.config.DefaultPresenter,
	}
	metadata.GenerateFolderName()

	return metadata
}

func (m *RecordingSetupModel) GetRecordingOptions() models.RecordingOptions {
	monitorName := ""
	if m.recordScreen && m.selectedMonitor >= 0 && m.selectedMonitor < len(m.monitors) {
		monitorName = m.monitors[m.selectedMonitor].Name
	}

	return models.RecordingOptions{
		Monitor:        monitorName,
		NoAudio:        !m.recordAudio,
		NoWebcam:       !m.recordWebcam,
		NoScreen:       !m.recordScreen,
		CreateVertical: m.verticalVideo && m.recordWebcam && m.recordScreen,
	}
}

// GetLogoSelection returns the selected logos
func (m *RecordingSetupModel) GetLogoSelection() config.LogoSelection {
	if !m.addLogos {
		return config.LogoSelection{}
	}
	return config.LogoSelection{
		LeftLogo:    m.leftLogo,
		RightLogo:   m.rightLogo,
		BottomLogo:  m.bottomLogo,
		TitleColor:  m.titleColor,
		GifLoopMode: m.gifLoopMode,
	}
}

// SaveLogoSelection saves the current logo selection to config for next time
func (m *RecordingSetupModel) SaveLogoSelection() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.LastUsedLogos = m.GetLogoSelection()
	return config.Save(cfg)
}

// GetRecordingPresets returns the current recording presets
func (m *RecordingSetupModel) GetRecordingPresets() config.RecordingPresets {
	topic := ""
	if m.selectedTopic >= 0 && m.selectedTopic < len(m.topics) {
		topic = m.topics[m.selectedTopic].Name
	}

	return config.RecordingPresets{
		RecordAudio:   m.recordAudio,
		RecordWebcam:  m.recordWebcam,
		RecordScreen:  m.recordScreen,
		VerticalVideo: m.verticalVideo,
		AddLogos:      m.addLogos,
		Topic:         topic,
	}
}

// SaveAllPresets saves all recording presets (toggles, topic, logos) to config
// This should be called when starting a recording to remember settings for next time
func (m *RecordingSetupModel) SaveAllPresets() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Save recording presets (toggles and topic)
	cfg.RecordingPresets = m.GetRecordingPresets()

	// Save logo selection
	cfg.LastUsedLogos = m.GetLogoSelection()

	return config.Save(cfg)
}

// View renders the two-column form layout
func (m *RecordingSetupModel) View() string {
	// Column widths
	labelWidth := 20
	widgetWidth := 35

	// Styles
	labelStyle := lipgloss.NewStyle().
		Width(labelWidth).
		Align(lipgloss.Right).
		Foreground(ColorGray).
		PaddingRight(2)

	labelFocusedStyle := lipgloss.NewStyle().
		Width(labelWidth).
		Align(lipgloss.Right).
		Foreground(ColorOrange).
		Bold(true).
		PaddingRight(2)

	widgetStyle := lipgloss.NewStyle().
		Width(widgetWidth).
		Align(lipgloss.Left)

	// Build rows
	var rows []string

	// Title
	rows = append(rows, m.renderRow(fieldTitle, "Title", m.titleInput.View(), labelStyle, labelFocusedStyle, widgetStyle))

	// Number
	rows = append(rows, m.renderRow(fieldNumber, "Number", m.numberInput.View(), labelStyle, labelFocusedStyle, widgetStyle))

	// Topic
	topicValue := m.renderSelector(m.topics, m.selectedTopic, func(t models.Topic) string { return t.Name }, m.focusedField == fieldTopic)
	rows = append(rows, m.renderRow(fieldTopic, "Topic", topicValue, labelStyle, labelFocusedStyle, widgetStyle))

	// Spacer
	rows = append(rows, "")

	// Record Audio
	rows = append(rows, m.renderRow(fieldRecordAudio, "Record Audio", m.renderToggle(m.recordAudio, m.focusedField == fieldRecordAudio), labelStyle, labelFocusedStyle, widgetStyle))

	// Record Webcam
	rows = append(rows, m.renderRow(fieldRecordWebcam, "Record Webcam", m.renderToggle(m.recordWebcam, m.focusedField == fieldRecordWebcam), labelStyle, labelFocusedStyle, widgetStyle))

	// Record Screen
	rows = append(rows, m.renderRow(fieldRecordScreen, "Record Screen", m.renderToggle(m.recordScreen, m.focusedField == fieldRecordScreen), labelStyle, labelFocusedStyle, widgetStyle))

	// Monitor (only if screen recording enabled)
	if m.recordScreen && len(m.monitors) > 0 {
		monitorValue := m.renderMonitorSelector()
		rows = append(rows, m.renderRow(fieldMonitor, "Monitor", monitorValue, labelStyle, labelFocusedStyle, widgetStyle))
	}

	// Spacer
	rows = append(rows, "")

	// Vertical Video (disabled if neither webcam nor screen is enabled)
	verticalDisabled := !m.canEnableVerticalVideo()
	rows = append(rows, m.renderRow(fieldVerticalVideo, "Vertical Video", m.renderToggleWithDisabled(m.verticalVideo, m.focusedField == fieldVerticalVideo, verticalDisabled), labelStyle, labelFocusedStyle, widgetStyle))

	// Add Logos
	rows = append(rows, m.renderRow(fieldAddLogos, "Add Logos", m.renderToggle(m.addLogos, m.focusedField == fieldAddLogos), labelStyle, labelFocusedStyle, widgetStyle))

	// Logo selection fields (only show if addLogos is enabled)
	if m.addLogos {
		// Logo size hints
		hintStyle := lipgloss.NewStyle().Foreground(ColorGray).Italic(true)
		rows = append(rows, hintStyle.Render("  Logos: 216x216px @ 72dpi • Banner: 1080x200px @ 72dpi"))

		leftLogoValue := m.renderLogoSelector(m.selectedLeftIdx, m.focusedField == fieldLeftLogo)
		rows = append(rows, m.renderRow(fieldLeftLogo, "Left Logo", leftLogoValue, labelStyle, labelFocusedStyle, widgetStyle))

		rightLogoValue := m.renderLogoSelector(m.selectedRightIdx, m.focusedField == fieldRightLogo)
		rows = append(rows, m.renderRow(fieldRightLogo, "Right Logo", rightLogoValue, labelStyle, labelFocusedStyle, widgetStyle))

		bottomLogoValue := m.renderLogoSelector(m.selectedBottomIdx, m.focusedField == fieldBottomLogo)
		rows = append(rows, m.renderRow(fieldBottomLogo, "Bottom Banner", bottomLogoValue, labelStyle, labelFocusedStyle, widgetStyle))

		titleColorValue := m.renderColorSelector(m.focusedField == fieldTitleColor)
		rows = append(rows, m.renderRow(fieldTitleColor, "Title Color", titleColorValue, labelStyle, labelFocusedStyle, widgetStyle))

		// Only show GIF loop mode if bottom logo is a GIF
		if m.isBottomLogoGif() {
			gifLoopValue := m.renderGifLoopSelector(m.focusedField == fieldGifLoopMode)
			rows = append(rows, m.renderRow(fieldGifLoopMode, "GIF Animation", gifLoopValue, labelStyle, labelFocusedStyle, widgetStyle))
		}
	}

	// Spacer
	rows = append(rows, "")

	// Description
	rows = append(rows, m.renderRow(fieldDescription, "Description", m.descInput.View(), labelStyle, labelFocusedStyle, widgetStyle))

	// Spacer
	rows = append(rows, "")

	// Confirm buttons
	rows = append(rows, m.renderConfirmRow(labelWidth, widgetWidth))

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *RecordingSetupModel) renderRow(field int, label, widget string, labelStyle, labelFocusedStyle, widgetStyle lipgloss.Style) string {
	ls := labelStyle
	if m.focusedField == field {
		ls = labelFocusedStyle
		// Show input mode indicator for text fields
		if m.inputMode && m.isTextField() {
			label = "» " + label
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, ls.Render(label), widgetStyle.Render(widget))
}

func (m *RecordingSetupModel) renderToggle(value bool, focused bool) string {
	return m.renderToggleWithDisabled(value, focused, false)
}

func (m *RecordingSetupModel) renderToggleWithDisabled(value bool, focused bool, disabled bool) string {
	var yes, no string

	if disabled {
		// Disabled state - show dimmed
		yes = lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Padding(0, 1).Render("Yes")
		no = lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Padding(0, 1).Render("No")
		return fmt.Sprintf("%s  %s  (disabled)", yes, no)
	}

	if focused {
		if value {
			yes = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000")).Bold(true).Padding(0, 1).Render("Yes")
			no = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 1).Render("No")
		} else {
			yes = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 1).Render("Yes")
			no = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000")).Bold(true).Padding(0, 1).Render("No")
		}
	} else {
		if value {
			yes = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Padding(0, 1).Render("Yes")
			no = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 1).Render("No")
		} else {
			yes = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 1).Render("Yes")
			no = lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Padding(0, 1).Render("No")
		}
	}

	return fmt.Sprintf("%s  %s", yes, no)
}

func (m *RecordingSetupModel) renderSelector(topics []models.Topic, selected int, getName func(models.Topic) string, focused bool) string {
	if len(topics) == 0 {
		return lipgloss.NewStyle().Foreground(ColorGray).Render("(none)")
	}

	name := getName(topics[selected])

	if focused {
		return fmt.Sprintf("◀ %s ▶", lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render(name))
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Render(name)
}

func (m *RecordingSetupModel) renderMonitorSelector() string {
	if len(m.monitors) == 0 {
		return lipgloss.NewStyle().Foreground(ColorGray).Render("(none)")
	}

	mon := m.monitors[m.selectedMonitor]
	name := fmt.Sprintf("%s (%dx%d)", mon.Name, mon.Width, mon.Height)

	if m.focusedField == fieldMonitor {
		return fmt.Sprintf("◀ %s ▶", lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render(name))
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Render(name)
}

func (m *RecordingSetupModel) renderLogoSelector(selectedIdx int, focused bool) string {
	if len(m.availableLogos) == 0 {
		return lipgloss.NewStyle().Foreground(ColorGray).Render("(no logos)")
	}

	name := m.availableLogos[selectedIdx]

	if focused {
		return fmt.Sprintf("◀ %s ▶", lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render(name))
	}
	if name == "(none)" {
		return lipgloss.NewStyle().Foreground(ColorGray).Render(name)
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Render(name)
}

func (m *RecordingSetupModel) renderColorSelector(focused bool) string {
	name := m.titleColor

	// Create a color preview block - use the actual color for the block
	// Handle both hex colors (#rrggbb) and named colors
	colorBlock := "██"
	var previewStyle lipgloss.Style
	if strings.HasPrefix(m.titleColor, "#") {
		previewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.titleColor))
	} else {
		// Named colors - map to lipgloss colors
		previewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.titleColor))
	}
	preview := previewStyle.Render(colorBlock)

	if focused {
		return fmt.Sprintf("◀ %s %s ▶", preview, lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render(name))
	}
	return fmt.Sprintf("%s %s", preview, lipgloss.NewStyle().Foreground(ColorWhite).Render(name))
}

func (m *RecordingSetupModel) renderGifLoopSelector(focused bool) string {
	label := config.GifLoopModeLabels[m.gifLoopMode]
	if label == "" {
		label = string(m.gifLoopMode)
	}

	if focused {
		return fmt.Sprintf("◀ %s ▶", lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render(label))
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Render(label)
}

func (m *RecordingSetupModel) renderConfirmRow(labelWidth, widgetWidth int) string {
	// Center the buttons
	spacer := lipgloss.NewStyle().Width(labelWidth).Render("")

	var goLive, cancel string
	hasSource := m.hasRecordingSource()
	hasTitle := strings.TrimSpace(m.titleInput.Value()) != ""
	canGoLive := hasSource && hasTitle

	if m.focusedField == fieldConfirm {
		if m.confirmSelected {
			if canGoLive {
				goLive = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000")).Bold(true).Padding(0, 3).Render("Go Live!")
			} else {
				// Disabled state - show as dim
				goLive = lipgloss.NewStyle().Background(ColorGray).Foreground(lipgloss.Color("#666")).Padding(0, 3).Render("Go Live!")
			}
			cancel = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Cancel")
		} else {
			if canGoLive {
				goLive = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Go Live!")
			} else {
				goLive = lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Padding(0, 3).Render("Go Live!")
			}
			cancel = lipgloss.NewStyle().Background(ColorGray).Foreground(ColorWhite).Bold(true).Padding(0, 3).Render("Cancel")
		}
	} else {
		if canGoLive {
			goLive = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Go Live!")
		} else {
			goLive = lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Padding(0, 3).Render("Go Live!")
		}
		cancel = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Cancel")
	}

	buttons := fmt.Sprintf("%s    %s", goLive, cancel)

	// Show validation warnings
	var warnings []string
	if !hasTitle {
		warnings = append(warnings, "Title is required")
	}
	if !hasSource {
		warnings = append(warnings, "Enable at least one recording source")
	}

	if len(warnings) > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(ColorRed).Italic(true)
		warningText := warningStyle.Render(strings.Join(warnings, " • "))
		return lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.JoinHorizontal(lipgloss.Center, spacer, buttons),
			lipgloss.JoinHorizontal(lipgloss.Center, spacer, warningText),
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, spacer, buttons)
}

type recordingSetupCompleteMsg struct{}
