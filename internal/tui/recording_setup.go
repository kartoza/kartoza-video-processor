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
)

// RecordingSetupField represents which field is currently focused
type RecordingSetupField int

const (
	FieldCounter RecordingSetupField = iota
	FieldTitle
	FieldDescription
	FieldTopic
	FieldPresenter
	FieldStart
)

// RecordingSetupModel handles the recording setup form
type RecordingSetupModel struct {
	width  int
	height int

	// Current focused field
	focusedField RecordingSetupField

	// Text inputs
	counterInput     textinput.Model
	titleInput       textinput.Model
	presenterInput   textinput.Model
	descriptionInput textarea.Model

	// Counter value
	recordingNumber int

	// Topic selection
	topics        []models.Topic
	selectedTopic int

	// Configuration
	config *config.Config

	// Error/validation state
	err            error
	validationMsg  string
	successMessage string
}

// NewRecordingSetupModel creates a new recording setup model
func NewRecordingSetupModel() *RecordingSetupModel {
	cfg, _ := config.Load()

	// Get the next recording number
	recordingNumber := config.GetCurrentRecordingNumber()

	// Counter input (editable, defaults to next number)
	counterInput := textinput.New()
	counterInput.Placeholder = "001"
	counterInput.CharLimit = 6
	counterInput.Width = 6
	counterInput.SetValue(fmt.Sprintf("%d", recordingNumber))

	// Title input (required)
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter recording title"
	titleInput.CharLimit = 100
	titleInput.Width = 40

	// Presenter input
	presenterInput := textinput.New()
	presenterInput.Placeholder = "Enter presenter name"
	presenterInput.CharLimit = 100
	presenterInput.Width = 40
	if cfg.DefaultPresenter != "" {
		presenterInput.SetValue(cfg.DefaultPresenter)
	}

	// Description textarea
	descInput := textarea.New()
	descInput.Placeholder = "Optional description..."
	descInput.CharLimit = 2000
	descInput.SetWidth(40)
	descInput.SetHeight(3)
	descInput.ShowLineNumbers = false

	// Get topics from config or use defaults
	topics := cfg.Topics
	if len(topics) == 0 {
		topics = models.DefaultTopics()
	}

	return &RecordingSetupModel{
		focusedField:     FieldTitle, // Start on title since it's required
		counterInput:     counterInput,
		recordingNumber:  recordingNumber,
		titleInput:       titleInput,
		presenterInput:   presenterInput,
		descriptionInput: descInput,
		topics:           topics,
		selectedTopic:    0,
		config:           cfg,
	}
}

// Init initializes the model
func (m *RecordingSetupModel) Init() tea.Cmd {
	m.focusCurrent()
	return textinput.Blink
}

// Update handles messages
func (m *RecordingSetupModel) Update(msg tea.Msg) (*RecordingSetupModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Clear validation messages on key press
		m.validationMsg = ""
		m.err = nil

		switch msg.String() {
		case "tab":
			m.nextField()
			return m, nil

		case "shift+tab":
			m.prevField()
			return m, nil

		case "down", "j":
			// In description field, let it handle internally
			if m.focusedField == FieldDescription {
				var cmd tea.Cmd
				m.descriptionInput, cmd = m.descriptionInput.Update(msg)
				return m, cmd
			}
			m.nextField()
			return m, nil

		case "up", "k":
			// In description field, let it handle internally
			if m.focusedField == FieldDescription {
				var cmd tea.Cmd
				m.descriptionInput, cmd = m.descriptionInput.Update(msg)
				return m, cmd
			}
			m.prevField()
			return m, nil

		case "left", "h":
			if m.focusedField == FieldTopic {
				m.selectedTopic--
				if m.selectedTopic < 0 {
					m.selectedTopic = len(m.topics) - 1
				}
				return m, nil
			}
			// Let text inputs handle left arrow
			return m.updateCurrentInput(msg)

		case "right", "l":
			if m.focusedField == FieldTopic {
				m.selectedTopic++
				if m.selectedTopic >= len(m.topics) {
					m.selectedTopic = 0
				}
				return m, nil
			}
			// Let text inputs handle right arrow
			return m.updateCurrentInput(msg)

		case "enter":
			if m.focusedField == FieldStart {
				// Validate and return
				if m.Validate() {
					return m, func() tea.Msg { return recordingSetupCompleteMsg{} }
				}
				return m, nil
			}
			// If in description, allow newlines
			if m.focusedField == FieldDescription {
				var cmd tea.Cmd
				m.descriptionInput, cmd = m.descriptionInput.Update(msg)
				return m, cmd
			}
			// Otherwise move to next field
			m.nextField()
			return m, nil
		}

		// Update the focused input
		return m.updateCurrentInput(msg)
	}

	return m, tea.Batch(cmds...)
}

// updateCurrentInput updates the currently focused input
func (m *RecordingSetupModel) updateCurrentInput(msg tea.KeyMsg) (*RecordingSetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedField {
	case FieldCounter:
		m.counterInput, cmd = m.counterInput.Update(msg)
	case FieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case FieldDescription:
		m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	case FieldPresenter:
		m.presenterInput, cmd = m.presenterInput.Update(msg)
	}

	return m, cmd
}

// nextField moves to the next field
func (m *RecordingSetupModel) nextField() {
	m.unfocusAll()
	m.focusedField++
	if m.focusedField > FieldStart {
		m.focusedField = FieldCounter
	}
	m.focusCurrent()
}

// prevField moves to the previous field
func (m *RecordingSetupModel) prevField() {
	m.unfocusAll()
	m.focusedField--
	if m.focusedField < FieldCounter {
		m.focusedField = FieldStart
	}
	m.focusCurrent()
}

// unfocusAll removes focus from all inputs
func (m *RecordingSetupModel) unfocusAll() {
	m.counterInput.Blur()
	m.titleInput.Blur()
	m.descriptionInput.Blur()
	m.presenterInput.Blur()
}

// focusCurrent focuses the current field
func (m *RecordingSetupModel) focusCurrent() {
	switch m.focusedField {
	case FieldCounter:
		m.counterInput.Focus()
	case FieldTitle:
		m.titleInput.Focus()
	case FieldDescription:
		m.descriptionInput.Focus()
	case FieldPresenter:
		m.presenterInput.Focus()
	}
}

// Validate checks if the form is valid
func (m *RecordingSetupModel) Validate() bool {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.validationMsg = "Title is required"
		m.focusedField = FieldTitle
		m.focusCurrent()
		return false
	}
	m.err = nil
	m.validationMsg = ""
	return true
}

// GetMetadata returns the recording metadata
func (m *RecordingSetupModel) GetMetadata() models.RecordingMetadata {
	topic := ""
	if m.selectedTopic >= 0 && m.selectedTopic < len(m.topics) {
		topic = m.topics[m.selectedTopic].Name
	}

	presenter := strings.TrimSpace(m.presenterInput.Value())

	// Save presenter as default for next time
	if presenter != "" && presenter != m.config.DefaultPresenter {
		m.config.DefaultPresenter = presenter
		config.Save(m.config)
	}

	// Parse counter value (fallback to recorded number if invalid)
	counterValue := m.recordingNumber
	if val, err := strconv.Atoi(strings.TrimSpace(m.counterInput.Value())); err == nil && val > 0 {
		counterValue = val
	}

	metadata := models.RecordingMetadata{
		Number:      counterValue,
		Title:       strings.TrimSpace(m.titleInput.Value()),
		Description: strings.TrimSpace(m.descriptionInput.Value()),
		Topic:       topic,
		Presenter:   presenter,
	}

	// Generate folder name
	metadata.GenerateFolderName()

	return metadata
}

// View renders the setup form
func (m *RecordingSetupModel) View() string {
	// Styles matching workspace associations pattern
	labelWidth := 14
	fieldWidth := 44

	labelStyle := lipgloss.NewStyle().
		Width(labelWidth).
		Align(lipgloss.Right).
		Foreground(ColorGray)

	focusedLabelStyle := lipgloss.NewStyle().
		Width(labelWidth).
		Align(lipgloss.Right).
		Foreground(ColorOrange).
		Bold(true)

	requiredMarker := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true).
		Render("*")

	// Input box styles
	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1).
		Width(fieldWidth)

	focusedInputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(0, 1).
		Width(fieldWidth)

	// Button styles
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Bold(true)

	activeButtonStyle := buttonStyle.
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	inactiveButtonStyle := buttonStyle.
		Background(ColorDarkGray).
		Foreground(ColorWhite)

	// Container style
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBlue).
		Padding(1, 2).
		Width(70)

	var rows []string

	// Row 1: Recording Number
	counterLabel := labelStyle.Render("Recording #:")
	if m.focusedField == FieldCounter {
		counterLabel = focusedLabelStyle.Render("Recording #:")
	}
	counterBoxStyle := inputBoxStyle.Width(12)
	if m.focusedField == FieldCounter {
		counterBoxStyle = focusedInputBoxStyle.Width(12)
	}
	counterBox := counterBoxStyle.Render(m.counterInput.View())
	counterRow := lipgloss.JoinHorizontal(lipgloss.Top, counterLabel, "  ", counterBox)
	rows = append(rows, counterRow)
	rows = append(rows, "")

	// Row 2: Title (required)
	titleLabel := labelStyle.Render("Title:") + requiredMarker
	if m.focusedField == FieldTitle {
		titleLabel = focusedLabelStyle.Render("Title:") + requiredMarker
	}
	titleBoxStyle := inputBoxStyle
	if m.focusedField == FieldTitle {
		titleBoxStyle = focusedInputBoxStyle
	}
	titleBox := titleBoxStyle.Render(m.titleInput.View())
	titleRow := lipgloss.JoinHorizontal(lipgloss.Top, titleLabel, " ", titleBox)
	rows = append(rows, titleRow)

	// Folder preview (shown below title)
	previewMetadata := m.GetMetadata()
	folderPreview := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		MarginLeft(labelWidth + 3).
		Render(fmt.Sprintf("→ %s/", previewMetadata.FolderName))
	rows = append(rows, folderPreview)
	rows = append(rows, "")

	// Row 3: Description
	descLabel := labelStyle.Render("Description:")
	if m.focusedField == FieldDescription {
		descLabel = focusedLabelStyle.Render("Description:")
	}
	descBoxStyle := inputBoxStyle.Height(5)
	if m.focusedField == FieldDescription {
		descBoxStyle = focusedInputBoxStyle.Height(5)
	}
	descBox := descBoxStyle.Render(m.descriptionInput.View())
	descRow := lipgloss.JoinHorizontal(lipgloss.Top, descLabel, "  ", descBox)
	rows = append(rows, descRow)
	rows = append(rows, "")

	// Row 4: Topic (horizontal selection)
	topicLabel := labelStyle.Render("Topic:")
	if m.focusedField == FieldTopic {
		topicLabel = focusedLabelStyle.Render("Topic:")
	}

	var topicOptions []string
	for i, topic := range m.topics {
		topicStyle := lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 1)

		if i == m.selectedTopic {
			if m.focusedField == FieldTopic {
				topicStyle = topicStyle.
					Background(ColorOrange).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)
			} else {
				topicStyle = topicStyle.
					Background(ColorGray).
					Foreground(ColorWhite)
			}
		} else {
			topicStyle = topicStyle.
				Foreground(ColorGray)
		}
		topicOptions = append(topicOptions, topicStyle.Render(topic.Name))
	}
	topicRow := lipgloss.JoinHorizontal(lipgloss.Top, topicLabel, "  ", lipgloss.JoinHorizontal(lipgloss.Center, topicOptions...))
	rows = append(rows, topicRow)
	rows = append(rows, "")

	// Row 5: Presenter
	presenterLabel := labelStyle.Render("Presenter:")
	if m.focusedField == FieldPresenter {
		presenterLabel = focusedLabelStyle.Render("Presenter:")
	}
	presenterBoxStyle := inputBoxStyle
	if m.focusedField == FieldPresenter {
		presenterBoxStyle = focusedInputBoxStyle
	}
	presenterBox := presenterBoxStyle.Render(m.presenterInput.View())
	presenterRow := lipgloss.JoinHorizontal(lipgloss.Top, presenterLabel, "  ", presenterBox)
	rows = append(rows, presenterRow)
	rows = append(rows, "")

	// Row 6: Start Button
	startButton := inactiveButtonStyle.Render("▶ Start Recording")
	if m.focusedField == FieldStart {
		startButton = activeButtonStyle.Render("▶ Start Recording")
	}
	// Center the button
	buttonRow := lipgloss.NewStyle().
		Width(60).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(startButton)
	rows = append(rows, buttonRow)

	// Build form content
	formContent := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Wrap in container
	form := containerStyle.Render(formContent)

	// Validation message
	if m.validationMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			MarginTop(1)
		form = lipgloss.JoinVertical(
			lipgloss.Center,
			form,
			errorStyle.Render("⚠ "+m.validationMsg),
		)
	}

	// Error message
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			MarginTop(1)
		form = lipgloss.JoinVertical(
			lipgloss.Center,
			form,
			errorStyle.Render("Error: "+m.err.Error()),
		)
	}

	return form
}

// recordingSetupCompleteMsg signals that setup is complete
type recordingSetupCompleteMsg struct{}
