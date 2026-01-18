package tui

import (
	"fmt"
	"strconv"
	"strings"

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
	fieldDescription
	fieldConfirm
	fieldCount
)

// RecordingSetupModel handles the recording setup form
type RecordingSetupModel struct {
	width  int
	height int

	focusedField int
	config       *config.Config

	// Text inputs
	titleInput  textinput.Model
	numberInput textinput.Model
	descInput   textinput.Model

	// Form values
	topic string

	// Options (bool toggles)
	recordAudio   bool
	recordWebcam  bool
	recordScreen  bool
	verticalVideo bool
	addLogos      bool

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

	descInput := textinput.New()
	descInput.Placeholder = "Enter description..."
	descInput.CharLimit = 500
	descInput.Width = 30

	m := &RecordingSetupModel{
		config:          cfg,
		focusedField:    fieldTitle,
		titleInput:      titleInput,
		numberInput:     numberInput,
		descInput:       descInput,
		recordAudio:     true,
		recordWebcam:    true,
		recordScreen:    true,
		verticalVideo:   true,
		addLogos:        true,
		monitors:        monitors,
		selectedMonitor: 0,
		topics:          topics,
		selectedTopic:   0,
		confirmSelected: true, // Default to "Go Live"
	}

	return m
}

func (m *RecordingSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RecordingSetupModel) Update(msg tea.Msg) (*RecordingSetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
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
			// Space toggles boolean fields
			if m.handleToggle() {
				return m, nil
			}
		case "enter":
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
		}

		// Update text input if focused on a text field
		switch m.focusedField {
		case fieldTitle:
			m.titleInput, cmd = m.titleInput.Update(msg)
		case fieldNumber:
			m.numberInput, cmd = m.numberInput.Update(msg)
		case fieldDescription:
			m.descInput, cmd = m.descInput.Update(msg)
		}
	}

	return m, cmd
}

func (m *RecordingSetupModel) nextField() {
	m.blurAll()
	m.focusedField++

	// Skip monitor field if screen recording is disabled
	if m.focusedField == fieldMonitor && !m.recordScreen {
		m.focusedField++
	}

	if m.focusedField >= fieldCount {
		m.focusedField = fieldTitle
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) prevField() {
	m.blurAll()
	m.focusedField--

	// Skip monitor field if screen recording is disabled
	if m.focusedField == fieldMonitor && !m.recordScreen {
		m.focusedField--
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
		return true
	case fieldRecordScreen:
		m.recordScreen = !m.recordScreen
		return true
	case fieldVerticalVideo:
		m.verticalVideo = !m.verticalVideo
		return true
	case fieldAddLogos:
		m.addLogos = !m.addLogos
		return true
	}
	return false
}

func (m *RecordingSetupModel) Validate() bool {
	return strings.TrimSpace(m.titleInput.Value()) != ""
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
		CreateVertical: m.verticalVideo && m.recordWebcam,
	}
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

	// Vertical Video
	rows = append(rows, m.renderRow(fieldVerticalVideo, "Vertical Video", m.renderToggle(m.verticalVideo, m.focusedField == fieldVerticalVideo), labelStyle, labelFocusedStyle, widgetStyle))

	// Add Logos
	rows = append(rows, m.renderRow(fieldAddLogos, "Add Logos", m.renderToggle(m.addLogos, m.focusedField == fieldAddLogos), labelStyle, labelFocusedStyle, widgetStyle))

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
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, ls.Render(label), widgetStyle.Render(widget))
}

func (m *RecordingSetupModel) renderToggle(value bool, focused bool) string {
	yes := "Yes"
	no := "No"

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

func (m *RecordingSetupModel) renderConfirmRow(labelWidth, widgetWidth int) string {
	// Center the buttons
	spacer := lipgloss.NewStyle().Width(labelWidth).Render("")

	var goLive, cancel string

	if m.focusedField == fieldConfirm {
		if m.confirmSelected {
			goLive = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000")).Bold(true).Padding(0, 3).Render("Go Live!")
			cancel = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Cancel")
		} else {
			goLive = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Go Live!")
			cancel = lipgloss.NewStyle().Background(ColorGray).Foreground(ColorWhite).Bold(true).Padding(0, 3).Render("Cancel")
		}
	} else {
		goLive = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Go Live!")
		cancel = lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 3).Render("Cancel")
	}

	buttons := fmt.Sprintf("%s    %s", goLive, cancel)
	return lipgloss.JoinHorizontal(lipgloss.Center, spacer, buttons)
}

type recordingSetupCompleteMsg struct{}
