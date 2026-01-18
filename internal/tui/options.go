package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// OptionsField represents which field is focused in options
type OptionsField int

const (
	OptionsFieldTopicList OptionsField = iota
	OptionsFieldAddTopic
	OptionsFieldRemoveTopic
	OptionsFieldDefaultPresenter
	OptionsFieldProductLogo1
	OptionsFieldProductLogo2
	OptionsFieldCompanyLogo
	OptionsFieldSave
)

// OptionsModel handles the options screen
type OptionsModel struct {
	width  int
	height int

	// Configuration
	config *config.Config

	// Topics management
	topics        []models.Topic
	selectedTopic int

	// Focus state
	focusedField OptionsField

	// Inputs
	newTopicInput      textinput.Model
	presenterInput     textinput.Model
	productLogo1Input  textinput.Model
	productLogo2Input  textinput.Model
	companyLogoInput   textinput.Model

	// State
	message string
	err     error
}

// NewOptionsModel creates a new options model
func NewOptionsModel() *OptionsModel {
	cfg, _ := config.Load()

	// Get topics from config or use defaults
	topics := cfg.Topics
	if len(topics) == 0 {
		topics = models.DefaultTopics()
	}

	// New topic input
	newTopicInput := textinput.New()
	newTopicInput.Placeholder = "New topic name"
	newTopicInput.CharLimit = 50
	newTopicInput.Width = 30

	// Presenter input
	presenterInput := textinput.New()
	presenterInput.Placeholder = "Default presenter name"
	presenterInput.CharLimit = 100
	presenterInput.Width = 40
	if cfg.DefaultPresenter != "" {
		presenterInput.SetValue(cfg.DefaultPresenter)
	}

	// Product Logo 1 input (top-left)
	productLogo1Input := textinput.New()
	productLogo1Input.Placeholder = "/path/to/logo1.png"
	productLogo1Input.CharLimit = 255
	productLogo1Input.Width = 40
	if cfg.ProductLogo1Path != "" {
		productLogo1Input.SetValue(cfg.ProductLogo1Path)
	}

	// Product Logo 2 input (top-right)
	productLogo2Input := textinput.New()
	productLogo2Input.Placeholder = "/path/to/logo2.png"
	productLogo2Input.CharLimit = 255
	productLogo2Input.Width = 40
	if cfg.ProductLogo2Path != "" {
		productLogo2Input.SetValue(cfg.ProductLogo2Path)
	}

	// Company Logo input (lower third)
	companyLogoInput := textinput.New()
	companyLogoInput.Placeholder = "/path/to/company_logo.png"
	companyLogoInput.CharLimit = 255
	companyLogoInput.Width = 40
	if cfg.CompanyLogoPath != "" {
		companyLogoInput.SetValue(cfg.CompanyLogoPath)
	}

	return &OptionsModel{
		config:            cfg,
		topics:            topics,
		selectedTopic:     0,
		focusedField:      OptionsFieldTopicList,
		newTopicInput:     newTopicInput,
		presenterInput:    presenterInput,
		productLogo1Input: productLogo1Input,
		productLogo2Input: productLogo2Input,
		companyLogoInput:  companyLogoInput,
	}
}

// Init initializes the model
func (m *OptionsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *OptionsModel) Update(msg tea.Msg) (*OptionsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Clear messages on any key
		m.message = ""
		m.err = nil

		switch msg.String() {
		case "tab", "down":
			m.nextField()
			return m, nil

		case "shift+tab", "up":
			m.prevField()
			return m, nil

		case "j":
			if m.focusedField == OptionsFieldTopicList {
				m.selectedTopic++
				if m.selectedTopic >= len(m.topics) {
					m.selectedTopic = 0
				}
				return m, nil
			}

		case "k":
			if m.focusedField == OptionsFieldTopicList {
				m.selectedTopic--
				if m.selectedTopic < 0 {
					m.selectedTopic = len(m.topics) - 1
				}
				return m, nil
			}

		case "enter":
			switch m.focusedField {
			case OptionsFieldAddTopic:
				m.addTopic()
				return m, nil
			case OptionsFieldRemoveTopic:
				m.removeTopic()
				return m, nil
			case OptionsFieldSave:
				m.save()
				return m, nil
			default:
				m.nextField()
				return m, nil
			}

		case "d", "delete", "backspace":
			if m.focusedField == OptionsFieldTopicList && len(m.topics) > 1 {
				m.removeTopic()
				return m, nil
			}
		}
	}

	// Update focused input
	switch m.focusedField {
	case OptionsFieldAddTopic:
		var cmd tea.Cmd
		m.newTopicInput, cmd = m.newTopicInput.Update(msg)
		cmds = append(cmds, cmd)

	case OptionsFieldDefaultPresenter:
		var cmd tea.Cmd
		m.presenterInput, cmd = m.presenterInput.Update(msg)
		cmds = append(cmds, cmd)

	case OptionsFieldProductLogo1:
		var cmd tea.Cmd
		m.productLogo1Input, cmd = m.productLogo1Input.Update(msg)
		cmds = append(cmds, cmd)

	case OptionsFieldProductLogo2:
		var cmd tea.Cmd
		m.productLogo2Input, cmd = m.productLogo2Input.Update(msg)
		cmds = append(cmds, cmd)

	case OptionsFieldCompanyLogo:
		var cmd tea.Cmd
		m.companyLogoInput, cmd = m.companyLogoInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// nextField moves to the next field
func (m *OptionsModel) nextField() {
	m.unfocusAll()
	m.focusedField++
	if m.focusedField > OptionsFieldSave {
		m.focusedField = OptionsFieldTopicList
	}
	m.focusCurrent()
}

// prevField moves to the previous field
func (m *OptionsModel) prevField() {
	m.unfocusAll()
	m.focusedField--
	if m.focusedField < OptionsFieldTopicList {
		m.focusedField = OptionsFieldSave
	}
	m.focusCurrent()
}

// unfocusAll removes focus from all inputs
func (m *OptionsModel) unfocusAll() {
	m.newTopicInput.Blur()
	m.presenterInput.Blur()
	m.productLogo1Input.Blur()
	m.productLogo2Input.Blur()
	m.companyLogoInput.Blur()
}

// focusCurrent focuses the current field
func (m *OptionsModel) focusCurrent() {
	switch m.focusedField {
	case OptionsFieldAddTopic:
		m.newTopicInput.Focus()
	case OptionsFieldDefaultPresenter:
		m.presenterInput.Focus()
	case OptionsFieldProductLogo1:
		m.productLogo1Input.Focus()
	case OptionsFieldProductLogo2:
		m.productLogo2Input.Focus()
	case OptionsFieldCompanyLogo:
		m.companyLogoInput.Focus()
	}
}

// addTopic adds a new topic
func (m *OptionsModel) addTopic() {
	name := strings.TrimSpace(m.newTopicInput.Value())
	if name == "" {
		m.err = nil
		return
	}

	// Check for duplicates
	for _, t := range m.topics {
		if strings.EqualFold(t.Name, name) {
			m.message = "Topic already exists"
			return
		}
	}

	// Generate ID from name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	m.topics = append(m.topics, models.Topic{
		ID:   id,
		Name: name,
	})

	m.newTopicInput.SetValue("")
	m.message = "Topic added: " + name
}

// removeTopic removes the selected topic
func (m *OptionsModel) removeTopic() {
	if len(m.topics) <= 1 {
		m.message = "Cannot remove last topic"
		return
	}

	if m.selectedTopic >= 0 && m.selectedTopic < len(m.topics) {
		name := m.topics[m.selectedTopic].Name
		m.topics = append(m.topics[:m.selectedTopic], m.topics[m.selectedTopic+1:]...)
		if m.selectedTopic >= len(m.topics) {
			m.selectedTopic = len(m.topics) - 1
		}
		m.message = "Topic removed: " + name
	}
}

// save saves the configuration
func (m *OptionsModel) save() {
	m.config.Topics = m.topics
	m.config.DefaultPresenter = strings.TrimSpace(m.presenterInput.Value())
	m.config.ProductLogo1Path = strings.TrimSpace(m.productLogo1Input.Value())
	m.config.ProductLogo2Path = strings.TrimSpace(m.productLogo2Input.Value())
	m.config.CompanyLogoPath = strings.TrimSpace(m.companyLogoInput.Value())

	if err := config.Save(m.config); err != nil {
		m.err = err
		return
	}

	m.message = "Settings saved successfully"
}

// View renders the options screen
func (m *OptionsModel) View() string {
	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	activeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1)

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)

	activeButtonStyle := buttonStyle.Copy().
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	inactiveButtonStyle := buttonStyle.Copy().
		Background(ColorGray).
		Foreground(ColorWhite)

	// Topic Management Section
	topicSection := sectionStyle.Render("Topic Management")

	// Topic list
	var topicList []string
	for i, topic := range m.topics {
		style := lipgloss.NewStyle().Foreground(ColorGray).Padding(0, 1)
		if i == m.selectedTopic {
			if m.focusedField == OptionsFieldTopicList {
				style = lipgloss.NewStyle().
					Background(ColorOrange).
					Foreground(lipgloss.Color("#000000")).
					Padding(0, 1)
			} else {
				style = lipgloss.NewStyle().
					Background(ColorGray).
					Foreground(ColorWhite).
					Padding(0, 1)
			}
		}
		topicList = append(topicList, style.Render(topic.Name))
	}

	topicListStr := lipgloss.JoinVertical(lipgloss.Left, topicList...)
	topicListBox := inactiveStyle.Render(topicListStr)
	if m.focusedField == OptionsFieldTopicList {
		topicListBox = activeStyle.Render(topicListStr)
	}

	// Add topic input
	addTopicStyle := inactiveStyle
	if m.focusedField == OptionsFieldAddTopic {
		addTopicStyle = activeStyle
	}
	addTopicRow := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render("Add topic: "),
		addTopicStyle.Render(m.newTopicInput.View()),
	)

	// Remove button
	removeBtn := inactiveButtonStyle.Render("Remove Selected")
	if m.focusedField == OptionsFieldRemoveTopic {
		removeBtn = activeButtonStyle.Render("Remove Selected")
	}

	// Default Presenter Section
	presenterSection := sectionStyle.Render("Default Presenter")
	presenterInputStyle := inactiveStyle
	if m.focusedField == OptionsFieldDefaultPresenter {
		presenterInputStyle = activeStyle
	}
	presenterRow := presenterInputStyle.Render(m.presenterInput.View())

	// Logo Settings Section
	logoSection := sectionStyle.Render("Logo Settings")

	// Product Logo 1 (top-left)
	logo1Style := inactiveStyle
	if m.focusedField == OptionsFieldProductLogo1 {
		logo1Style = activeStyle
	}
	logo1Row := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Product Logo 1: "),
		logo1Style.Render(m.productLogo1Input.View()),
	)
	logo1Hint := lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("  (top-left corner)")

	// Product Logo 2 (top-right)
	logo2Style := inactiveStyle
	if m.focusedField == OptionsFieldProductLogo2 {
		logo2Style = activeStyle
	}
	logo2Row := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Product Logo 2: "),
		logo2Style.Render(m.productLogo2Input.View()),
	)
	logo2Hint := lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("  (top-right corner)")

	// Company Logo (lower third)
	companyLogoStyle := inactiveStyle
	if m.focusedField == OptionsFieldCompanyLogo {
		companyLogoStyle = activeStyle
	}
	companyLogoRow := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Company Logo: "),
		companyLogoStyle.Render(m.companyLogoInput.View()),
	)
	companyLogoHint := lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("  (lower third with title)")

	// Save button
	saveBtn := inactiveButtonStyle.Render("Save Settings")
	if m.focusedField == OptionsFieldSave {
		saveBtn = activeButtonStyle.Render("Save Settings")
	}

	// Message/Error display
	var statusLine string
	if m.err != nil {
		statusLine = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Render("Error: " + m.err.Error())
	} else if m.message != "" {
		statusLine = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true).
			Render(m.message)
	}

	// Hints
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)
	topicHint := hintStyle.Render("j/k to navigate • d to delete • enter to select")

	// Build the view
	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		topicSection,
		topicListBox,
		topicHint,
		"",
		addTopicRow,
		removeBtn,
		"",
		presenterSection,
		presenterRow,
		"",
		logoSection,
		logo1Row,
		logo1Hint,
		logo2Row,
		logo2Hint,
		companyLogoRow,
		companyLogoHint,
		"",
		saveBtn,
		"",
		statusLine,
	)
}
