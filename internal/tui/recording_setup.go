package tui

import (
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
	FieldTitle RecordingSetupField = iota
	FieldPresenter
	FieldTopic
	FieldDescription
	FieldStart
)

// RecordingSetupModel handles the recording setup form
type RecordingSetupModel struct {
	width  int
	height int

	focusedField     RecordingSetupField
	titleInput       textinput.Model
	presenterInput   textinput.Model
	descriptionInput textarea.Model

	recordingNumber int
	topics          []models.Topic
	selectedTopic   int
	config          *config.Config
	validationMsg   string
}

// NewRecordingSetupModel creates a new recording setup model
func NewRecordingSetupModel() *RecordingSetupModel {
	cfg, _ := config.Load()
	recordingNumber := config.GetCurrentRecordingNumber()

	titleInput := textinput.New()
	titleInput.Placeholder = "My Tutorial Video"
	titleInput.CharLimit = 100
	titleInput.Width = 50
	titleInput.Focus()

	presenterInput := textinput.New()
	presenterInput.Placeholder = "Your Name"
	presenterInput.CharLimit = 100
	presenterInput.Width = 50
	if cfg.DefaultPresenter != "" {
		presenterInput.SetValue(cfg.DefaultPresenter)
	}

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

	return &RecordingSetupModel{
		focusedField:     FieldTitle,
		titleInput:       titleInput,
		presenterInput:   presenterInput,
		descriptionInput: descInput,
		recordingNumber:  recordingNumber,
		topics:           topics,
		selectedTopic:    0,
		config:           cfg,
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
		// Header ~3, title+folder ~3, presenter ~2, topic ~2, button ~2, footer ~2 = ~14 lines used
		descHeight := m.height - 18
		if descHeight < 3 {
			descHeight = 3
		}
		m.descriptionInput.SetHeight(descHeight)
		m.descriptionInput.SetWidth(m.width - 16) // Leave room for label

	case tea.KeyMsg:
		m.validationMsg = ""

		switch msg.String() {
		case "tab", "down":
			if m.focusedField == FieldDescription {
				// Let textarea handle down if it has content
				if strings.Contains(m.descriptionInput.Value(), "\n") {
					var cmd tea.Cmd
					m.descriptionInput, cmd = m.descriptionInput.Update(msg)
					return m, cmd
				}
			}
			m.nextField()
			return m, nil

		case "shift+tab", "up":
			m.prevField()
			return m, nil

		case "left":
			if m.focusedField == FieldTopic {
				m.selectedTopic--
				if m.selectedTopic < 0 {
					m.selectedTopic = len(m.topics) - 1
				}
				return m, nil
			}

		case "right":
			if m.focusedField == FieldTopic {
				m.selectedTopic++
				if m.selectedTopic >= len(m.topics) {
					m.selectedTopic = 0
				}
				return m, nil
			}

		case "enter":
			if m.focusedField == FieldStart {
				if m.Validate() {
					return m, func() tea.Msg { return recordingSetupCompleteMsg{} }
				}
				return m, nil
			}
			m.nextField()
			return m, nil
		}

		// Update focused input
		var cmd tea.Cmd
		switch m.focusedField {
		case FieldTitle:
			m.titleInput, cmd = m.titleInput.Update(msg)
		case FieldPresenter:
			m.presenterInput, cmd = m.presenterInput.Update(msg)
		case FieldDescription:
			m.descriptionInput, cmd = m.descriptionInput.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *RecordingSetupModel) nextField() {
	m.blurAll()
	m.focusedField++
	if m.focusedField > FieldStart {
		m.focusedField = FieldTitle
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) prevField() {
	m.blurAll()
	m.focusedField--
	if m.focusedField < FieldTitle {
		m.focusedField = FieldStart
	}
	m.focusCurrent()
}

func (m *RecordingSetupModel) blurAll() {
	m.titleInput.Blur()
	m.presenterInput.Blur()
	m.descriptionInput.Blur()
}

func (m *RecordingSetupModel) focusCurrent() {
	switch m.focusedField {
	case FieldTitle:
		m.titleInput.Focus()
	case FieldPresenter:
		m.presenterInput.Focus()
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

	presenter := strings.TrimSpace(m.presenterInput.Value())
	if presenter != "" && presenter != m.config.DefaultPresenter {
		m.config.DefaultPresenter = presenter
		config.Save(m.config)
	}

	metadata := models.RecordingMetadata{
		Number:      m.recordingNumber,
		Title:       strings.TrimSpace(m.titleInput.Value()),
		Description: strings.TrimSpace(m.descriptionInput.Value()),
		Topic:       topic,
		Presenter:   presenter,
	}
	metadata.GenerateFolderName()

	return metadata
}

func (m *RecordingSetupModel) View() string {
	labelWidth := 14
	label := lipgloss.NewStyle().Foreground(ColorGray).Width(labelWidth).Align(lipgloss.Right)
	activeLabel := lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Width(labelWidth).Align(lipgloss.Right)
	dim := lipgloss.NewStyle().Foreground(ColorGray).Italic(true)

	var rows []string

	// Title row
	titleLabel := label.Render("Title")
	if m.focusedField == FieldTitle {
		titleLabel = activeLabel.Render("Title")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, titleLabel, "  ", m.titleInput.View()))

	// Folder preview
	meta := m.GetMetadata()
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(labelWidth).Render(""),
		"  ",
		dim.Render("→ "+meta.FolderName+"/")))
	rows = append(rows, "")

	// Presenter row
	presenterLabel := label.Render("Presenter")
	if m.focusedField == FieldPresenter {
		presenterLabel = activeLabel.Render("Presenter")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, presenterLabel, "  ", m.presenterInput.View()))
	rows = append(rows, "")

	// Topic row
	topicLabel := label.Render("Topic")
	if m.focusedField == FieldTopic {
		topicLabel = activeLabel.Render("Topic")
	}
	var topicChips []string
	for i, t := range m.topics {
		if i == m.selectedTopic {
			if m.focusedField == FieldTopic {
				topicChips = append(topicChips, lipgloss.NewStyle().
					Background(ColorOrange).
					Foreground(lipgloss.Color("#000")).
					Padding(0, 1).
					Render(t.Name))
			} else {
				topicChips = append(topicChips, lipgloss.NewStyle().
					Background(ColorGray).
					Foreground(ColorWhite).
					Padding(0, 1).
					Render(t.Name))
			}
		} else {
			topicChips = append(topicChips, lipgloss.NewStyle().
				Foreground(ColorGray).
				Padding(0, 1).
				Render(t.Name))
		}
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, topicLabel, "  ", lipgloss.JoinHorizontal(lipgloss.Top, topicChips...)))
	rows = append(rows, "")

	// Description row
	descLabel := label.Render("Description")
	if m.focusedField == FieldDescription {
		descLabel = activeLabel.Render("Description")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, descLabel, "  ", m.descriptionInput.View()))
	rows = append(rows, "")

	// Start button row
	var startBtn string
	if m.focusedField == FieldStart {
		startBtn = lipgloss.NewStyle().
			Background(ColorOrange).
			Foreground(lipgloss.Color("#000")).
			Bold(true).
			Padding(0, 2).
			Render("▶ Start Recording")
	} else {
		startBtn = lipgloss.NewStyle().
			Background(ColorDarkGray).
			Foreground(ColorWhite).
			Padding(0, 2).
			Render("▶ Start Recording")
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(labelWidth).Render(""),
		"  ",
		startBtn))

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

type recordingSetupCompleteMsg struct{}
