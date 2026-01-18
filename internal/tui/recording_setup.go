package tui

import (
	"fmt"
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

// RecordingSetupField represents which field is currently focused
type RecordingSetupField int

const (
	FieldTitle RecordingSetupField = iota
	FieldNumber
	FieldTopic
	FieldRecordAudio
	FieldRecordWebcam
	FieldRecordScreen
	FieldScreenSelect
	FieldVerticalVideo
	FieldAddLogos
	FieldDescription
	FieldGoLive
)

// RecordingSetupModel handles the recording setup form
type RecordingSetupModel struct {
	width  int
	height int

	focusedField     RecordingSetupField
	titleInput       textinput.Model
	numberInput      textinput.Model
	descriptionInput textarea.Model

	topics        []models.Topic
	selectedTopic int
	config        *config.Config
	validationMsg string

	// Options
	recordAudio    bool
	recordWebcam   bool
	recordScreen   bool
	verticalVideo  bool
	addLogos       bool

	// Screen selection
	monitors        []models.Monitor
	selectedMonitor int
}

// NewRecordingSetupModel creates a new recording setup model
func NewRecordingSetupModel() *RecordingSetupModel {
	cfg, _ := config.Load()
	recordingNumber := config.GetCurrentRecordingNumber()

	titleInput := textinput.New()
	titleInput.Placeholder = "A nice recording"
	titleInput.CharLimit = 100
	titleInput.Width = 50
	titleInput.Focus()

	numberInput := textinput.New()
	numberInput.Placeholder = "001"
	numberInput.CharLimit = 10
	numberInput.Width = 10
	numberInput.SetValue(fmt.Sprintf("%03d", recordingNumber))

	descInput := textarea.New()
	descInput.Placeholder = "Description of this recording..."
	descInput.CharLimit = 2000
	descInput.SetWidth(50)
	descInput.SetHeight(6)
	descInput.ShowLineNumbers = false

	topics := cfg.Topics
	if len(topics) == 0 {
		topics = models.DefaultTopics()
	}

	// Get available monitors
	monitors, _ := monitor.ListMonitors()

	return &RecordingSetupModel{
		focusedField:     FieldTitle,
		titleInput:       titleInput,
		numberInput:      numberInput,
		descriptionInput: descInput,
		topics:           topics,
		selectedTopic:    0,
		config:           cfg,
		// Default options
		recordAudio:     true,
		recordWebcam:    true,
		recordScreen:    true,
		verticalVideo:   true,
		addLogos:        true,
		monitors:        monitors,
		selectedMonitor: 0,
	}
}

func (m *RecordingSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RecordingSetupModel) Update(msg tea.Msg) (*RecordingSetupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize description to fill remaining space
		// Account for more rows now with options
		descHeight := m.height - 28
		if descHeight < 3 {
			descHeight = 3
		}
		m.descriptionInput.SetHeight(descHeight)
		m.descriptionInput.SetWidth(m.width - 20)

	case tea.KeyMsg:
		m.validationMsg = ""

		switch msg.String() {
		case "tab", "down", "j":
			if m.focusedField == FieldDescription {
				// Let textarea handle down if it has multiline content
				if strings.Contains(m.descriptionInput.Value(), "\n") {
					var cmd tea.Cmd
					m.descriptionInput, cmd = m.descriptionInput.Update(msg)
					return m, cmd
				}
			}
			m.nextField()
			return m, nil

		case "shift+tab", "up", "k":
			m.prevField()
			return m, nil

		case "left", "h":
			return m.handleLeft()

		case "right", "l":
			return m.handleRight()

		case " ":
			// Space toggles checkboxes
			return m.handleSpace()

		case "enter":
			if m.focusedField == FieldGoLive {
				if m.Validate() {
					return m, func() tea.Msg { return recordingSetupCompleteMsg{} }
				}
				return m, nil
			}
			// Space/enter on checkboxes toggle them
			return m.handleSpace()
		}

		// Update focused input
		var cmd tea.Cmd
		switch m.focusedField {
		case FieldTitle:
			m.titleInput, cmd = m.titleInput.Update(msg)
		case FieldNumber:
			m.numberInput, cmd = m.numberInput.Update(msg)
		case FieldDescription:
			m.descriptionInput, cmd = m.descriptionInput.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *RecordingSetupModel) handleLeft() (*RecordingSetupModel, tea.Cmd) {
	switch m.focusedField {
	case FieldTopic:
		m.selectedTopic--
		if m.selectedTopic < 0 {
			m.selectedTopic = len(m.topics) - 1
		}
	case FieldScreenSelect:
		m.selectedMonitor--
		if m.selectedMonitor < 0 {
			m.selectedMonitor = len(m.monitors) - 1
		}
	}
	return m, nil
}

func (m *RecordingSetupModel) handleRight() (*RecordingSetupModel, tea.Cmd) {
	switch m.focusedField {
	case FieldTopic:
		m.selectedTopic++
		if m.selectedTopic >= len(m.topics) {
			m.selectedTopic = 0
		}
	case FieldScreenSelect:
		m.selectedMonitor++
		if m.selectedMonitor >= len(m.monitors) {
			m.selectedMonitor = 0
		}
	}
	return m, nil
}

func (m *RecordingSetupModel) handleSpace() (*RecordingSetupModel, tea.Cmd) {
	switch m.focusedField {
	case FieldRecordAudio:
		m.recordAudio = !m.recordAudio
	case FieldRecordWebcam:
		m.recordWebcam = !m.recordWebcam
	case FieldRecordScreen:
		m.recordScreen = !m.recordScreen
	case FieldVerticalVideo:
		m.verticalVideo = !m.verticalVideo
	case FieldAddLogos:
		m.addLogos = !m.addLogos
	}
	return m, nil
}

func (m *RecordingSetupModel) nextField() {
	m.blurAll()
	m.focusedField++

	// Skip screen select if record screen is disabled
	if m.focusedField == FieldScreenSelect && !m.recordScreen {
		m.focusedField++
	}

	if m.focusedField > FieldGoLive {
		m.focusedField = FieldTitle
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) prevField() {
	m.blurAll()
	m.focusedField--

	// Skip screen select if record screen is disabled
	if m.focusedField == FieldScreenSelect && !m.recordScreen {
		m.focusedField--
	}

	if m.focusedField < FieldTitle {
		m.focusedField = FieldGoLive
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) blurAll() {
	m.titleInput.Blur()
	m.numberInput.Blur()
	m.descriptionInput.Blur()
}

func (m *RecordingSetupModel) focusCurrent() {
	switch m.focusedField {
	case FieldTitle:
		m.titleInput.Focus()
	case FieldNumber:
		m.numberInput.Focus()
	case FieldDescription:
		m.descriptionInput.Focus()
	}
}

func (m *RecordingSetupModel) Validate() bool {
	if strings.TrimSpace(m.titleInput.Value()) == "" {
		m.validationMsg = "Please enter a title"
		m.focusedField = FieldTitle
		m.focusCurrent()
		return false
	}
	return true
}

func (m *RecordingSetupModel) GetMetadata() models.RecordingMetadata {
	topic := ""
	if m.selectedTopic >= 0 && m.selectedTopic < len(m.topics) {
		topic = m.topics[m.selectedTopic].Name
	}

	// Parse recording number from input
	recordingNumber := 1
	if num, err := strconv.Atoi(strings.TrimSpace(m.numberInput.Value())); err == nil && num > 0 {
		recordingNumber = num
	}

	metadata := models.RecordingMetadata{
		Number:      recordingNumber,
		Title:       strings.TrimSpace(m.titleInput.Value()),
		Description: strings.TrimSpace(m.descriptionInput.Value()),
		Topic:       topic,
		Presenter:   m.config.DefaultPresenter,
	}
	metadata.GenerateFolderName()

	return metadata
}

// GetRecordingOptions returns the recording options based on form selections
func (m *RecordingSetupModel) GetRecordingOptions() models.RecordingOptions {
	monitorName := ""
	if m.recordScreen && m.selectedMonitor >= 0 && m.selectedMonitor < len(m.monitors) {
		monitorName = m.monitors[m.selectedMonitor].Name
	}

	return models.RecordingOptions{
		Monitor:        monitorName,
		NoAudio:        !m.recordAudio,
		NoWebcam:       !m.recordWebcam,
		CreateVertical: m.verticalVideo && m.recordWebcam, // Only create vertical if webcam is enabled
		// AddLogos is a future feature
	}
}

func (m *RecordingSetupModel) View() string {
	labelWidth := 14
	label := lipgloss.NewStyle().Foreground(ColorGray).Width(labelWidth).Align(lipgloss.Right)
	activeLabel := lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Width(labelWidth).Align(lipgloss.Right)

	var rows []string

	// Title row
	titleLabel := label.Render("Title")
	if m.focusedField == FieldTitle {
		titleLabel = activeLabel.Render("Title")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, titleLabel, "  ", m.titleInput.View()))
	rows = append(rows, "")

	// Number row
	numberLabel := label.Render("Number")
	if m.focusedField == FieldNumber {
		numberLabel = activeLabel.Render("Number")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, numberLabel, "  ", m.numberInput.View()))
	rows = append(rows, "")

	// Topic row
	topicLabel := label.Render("Topic")
	if m.focusedField == FieldTopic {
		topicLabel = activeLabel.Render("Topic")
	}
	topicValue := m.renderTopicSelector()
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, topicLabel, "  ", topicValue))
	rows = append(rows, "")

	// Options section
	optionsLabel := label.Render("Options")
	if m.focusedField >= FieldRecordAudio && m.focusedField <= FieldAddLogos {
		optionsLabel = activeLabel.Render("Options")
	}

	optionsContent := m.renderOptions()
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, optionsLabel, "  ", optionsContent))
	rows = append(rows, "")

	// Description row
	descLabel := label.Render("Description")
	if m.focusedField == FieldDescription {
		descLabel = activeLabel.Render("Description")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, descLabel, "  ", m.descriptionInput.View()))
	rows = append(rows, "")

	// Go Live button row
	var goLiveBtn string
	if m.focusedField == FieldGoLive {
		goLiveBtn = lipgloss.NewStyle().
			Background(ColorOrange).
			Foreground(lipgloss.Color("#000")).
			Bold(true).
			Padding(0, 2).
			Render("Go Live!")
	} else {
		goLiveBtn = lipgloss.NewStyle().
			Background(ColorDarkGray).
			Foreground(ColorWhite).
			Padding(0, 2).
			Render("Go Live!")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(labelWidth).Render(""),
		"  ",
		goLiveBtn))

	// Validation message
	if m.validationMsg != "" {
		rows = append(rows, "")
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render(""),
			"  ",
			lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render(m.validationMsg)))
	}

	// Join all rows left-aligned
	form := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Center the form horizontally on screen
	return lipgloss.Place(m.width, m.height-6, lipgloss.Center, lipgloss.Top, form)
}

func (m *RecordingSetupModel) renderTopicSelector() string {
	var chips []string
	for i, t := range m.topics {
		if i == m.selectedTopic {
			if m.focusedField == FieldTopic {
				chips = append(chips, lipgloss.NewStyle().
					Background(ColorOrange).
					Foreground(lipgloss.Color("#000")).
					Padding(0, 1).
					Render(t.Name))
			} else {
				chips = append(chips, lipgloss.NewStyle().
					Background(ColorGray).
					Foreground(ColorWhite).
					Padding(0, 1).
					Render(t.Name))
			}
		} else {
			chips = append(chips, lipgloss.NewStyle().
				Foreground(ColorGray).
				Padding(0, 1).
				Render(t.Name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, chips...)
}

func (m *RecordingSetupModel) renderOptions() string {
	var lines []string

	// Record Audio
	lines = append(lines, m.renderCheckbox("Record Audio", m.recordAudio, m.focusedField == FieldRecordAudio))

	// Record Webcam
	lines = append(lines, m.renderCheckbox("Record Webcam", m.recordWebcam, m.focusedField == FieldRecordWebcam))

	// Record Screen
	lines = append(lines, m.renderCheckbox("Record Screen", m.recordScreen, m.focusedField == FieldRecordScreen))

	// Screen selection (nested, only if record screen is enabled)
	if m.recordScreen && len(m.monitors) > 0 {
		lines = append(lines, m.renderScreenSelector())
	}

	// Generate Vertical Video
	lines = append(lines, m.renderCheckbox("Generate Vertical Video", m.verticalVideo, m.focusedField == FieldVerticalVideo))

	// Add logos
	lines = append(lines, m.renderCheckbox("Add logos", m.addLogos, m.focusedField == FieldAddLogos))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *RecordingSetupModel) renderCheckbox(label string, checked bool, focused bool) string {
	checkmark := " "
	if checked {
		checkmark = "x"
	}

	style := lipgloss.NewStyle().Foreground(ColorGray)
	if focused {
		style = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
	}

	return style.Render(fmt.Sprintf("(%s) %s", checkmark, label))
}

func (m *RecordingSetupModel) renderScreenSelector() string {
	indent := "    "
	var lines []string

	for i, mon := range m.monitors {
		selected := i == m.selectedMonitor
		focused := m.focusedField == FieldScreenSelect

		marker := " "
		if selected {
			marker = "x"
		}

		style := lipgloss.NewStyle().Foreground(ColorGray)
		if focused && selected {
			style = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		} else if focused {
			style = lipgloss.NewStyle().Foreground(ColorWhite)
		}

		label := fmt.Sprintf("%s (%dx%d)", mon.Name, mon.Width, mon.Height)
		lines = append(lines, indent+style.Render(fmt.Sprintf("(%s) %s", marker, label)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

type recordingSetupCompleteMsg struct{}
