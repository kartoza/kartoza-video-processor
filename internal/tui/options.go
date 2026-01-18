package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// filePickerResultMsg is sent when a file is selected from the filepicker
type filePickerResultMsg struct {
	path string
}

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

// LogoType identifies which logo is being selected
type LogoType int

const (
	LogoNone LogoType = iota
	LogoProduct1
	LogoProduct2
	LogoCompany
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
	newTopicInput  textinput.Model
	presenterInput textinput.Model

	// Logo paths (displayed as text, selected via filepicker)
	productLogo1Path string
	productLogo2Path string
	companyLogoPath  string

	// Filepicker for logo selection
	filepicker       filepicker.Model
	showFilepicker   bool
	selectingLogoFor LogoType

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

	// Initialize filepicker for image files
	fp := filepicker.New()
	fp.AllowedTypes = []string{".png", ".jpg", ".jpeg", ".PNG", ".JPG", ".JPEG"}
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = true
	// Start in home directory
	home, _ := os.UserHomeDir()
	fp.CurrentDirectory = home

	return &OptionsModel{
		config:           cfg,
		topics:           topics,
		selectedTopic:    0,
		focusedField:     OptionsFieldTopicList,
		newTopicInput:    newTopicInput,
		presenterInput:   presenterInput,
		productLogo1Path: cfg.ProductLogo1Path,
		productLogo2Path: cfg.ProductLogo2Path,
		companyLogoPath:  cfg.CompanyLogoPath,
		filepicker:       fp,
		showFilepicker:   false,
		selectingLogoFor: LogoNone,
	}
}

// Init initializes the model
func (m *OptionsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *OptionsModel) Update(msg tea.Msg) (*OptionsModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle filepicker if active
	if m.showFilepicker {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.showFilepicker = false
				m.selectingLogoFor = LogoNone
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)

		// Check if a file was selected
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			// Store the selected path based on which logo we're selecting
			switch m.selectingLogoFor {
			case LogoProduct1:
				m.productLogo1Path = path
			case LogoProduct2:
				m.productLogo2Path = path
			case LogoCompany:
				m.companyLogoPath = path
			}
			m.showFilepicker = false
			m.selectingLogoFor = LogoNone
			m.message = "Logo selected: " + filepath.Base(path)
			return m, nil
		}

		// Check if a file was disabled (invalid type)
		if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
			m.message = "Invalid file type: " + filepath.Base(path)
			return m, cmd
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update filepicker size
		m.filepicker.Height = m.height - 10

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

		case "enter", " ":
			switch m.focusedField {
			case OptionsFieldAddTopic:
				m.addTopic()
				return m, nil
			case OptionsFieldRemoveTopic:
				m.removeTopic()
				return m, nil
			case OptionsFieldProductLogo1:
				m.showFilepicker = true
				m.selectingLogoFor = LogoProduct1
				return m, m.filepicker.Init()
			case OptionsFieldProductLogo2:
				m.showFilepicker = true
				m.selectingLogoFor = LogoProduct2
				return m, m.filepicker.Init()
			case OptionsFieldCompanyLogo:
				m.showFilepicker = true
				m.selectingLogoFor = LogoCompany
				return m, m.filepicker.Init()
			case OptionsFieldSave:
				m.save()
				return m, nil
			default:
				m.nextField()
				return m, nil
			}

		case "c":
			// Clear logo path if on a logo field
			switch m.focusedField {
			case OptionsFieldProductLogo1:
				m.productLogo1Path = ""
				m.message = "Product Logo 1 cleared"
				return m, nil
			case OptionsFieldProductLogo2:
				m.productLogo2Path = ""
				m.message = "Product Logo 2 cleared"
				return m, nil
			case OptionsFieldCompanyLogo:
				m.companyLogoPath = ""
				m.message = "Company Logo cleared"
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
}

// focusCurrent focuses the current field
func (m *OptionsModel) focusCurrent() {
	switch m.focusedField {
	case OptionsFieldAddTopic:
		m.newTopicInput.Focus()
	case OptionsFieldDefaultPresenter:
		m.presenterInput.Focus()
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
	m.config.ProductLogo1Path = m.productLogo1Path
	m.config.ProductLogo2Path = m.productLogo2Path
	m.config.CompanyLogoPath = m.companyLogoPath

	if err := config.Save(m.config); err != nil {
		m.err = err
		return
	}

	m.message = "Settings saved successfully"
}

// View renders the options screen
func (m *OptionsModel) View() string {
	// If filepicker is shown, render it instead
	if m.showFilepicker {
		return m.renderFilepicker()
	}

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

	// Helper to format logo path for display
	formatLogoPath := func(path string) string {
		if path == "" {
			return "(not set - press Enter to browse)"
		}
		// Show just the filename, or truncate if too long
		name := filepath.Base(path)
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		return name
	}

	// Product Logo 1 (top-left)
	logo1Style := inactiveStyle.Width(44)
	if m.focusedField == OptionsFieldProductLogo1 {
		logo1Style = activeStyle.Width(44)
	}
	logo1Row := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Product Logo 1: "),
		logo1Style.Render(formatLogoPath(m.productLogo1Path)),
	)
	logo1Hint := lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("  (top-left corner)")

	// Product Logo 2 (top-right)
	logo2Style := inactiveStyle.Width(44)
	if m.focusedField == OptionsFieldProductLogo2 {
		logo2Style = activeStyle.Width(44)
	}
	logo2Row := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Product Logo 2: "),
		logo2Style.Render(formatLogoPath(m.productLogo2Path)),
	)
	logo2Hint := lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("  (top-right corner)")

	// Company Logo (lower third)
	companyLogoStyle := inactiveStyle.Width(44)
	if m.focusedField == OptionsFieldCompanyLogo {
		companyLogoStyle = activeStyle.Width(44)
	}
	companyLogoRow := lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Width(20).Render("Company Logo: "),
		companyLogoStyle.Render(formatLogoPath(m.companyLogoPath)),
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
	logoHint := hintStyle.Render("enter/space to browse • c to clear")

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
		logoHint,
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

// renderFilepicker renders the file picker overlay
func (m *OptionsModel) renderFilepicker() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var title string
	switch m.selectingLogoFor {
	case LogoProduct1:
		title = "Select Product Logo 1 (top-left)"
	case LogoProduct2:
		title = "Select Product Logo 2 (top-right)"
	case LogoCompany:
		title = "Select Company Logo (lower third)"
	default:
		title = "Select Logo"
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		titleStyle.Render(title),
		"",
		m.filepicker.View(),
		"",
		hintStyle.Render("↑/↓ navigate • enter select • esc cancel"),
		hintStyle.Render("Allowed types: .png, .jpg, .jpeg"),
	)

	// Center on screen
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}
