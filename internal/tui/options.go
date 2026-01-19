package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// fileEntry represents a file or directory in the browser
type fileEntry struct {
	name  string
	path  string
	isDir bool
}

// OptionsField represents which field is focused in options
type OptionsField int

const (
	OptionsFieldTopicList OptionsField = iota
	OptionsFieldAddTopic
	OptionsFieldRemoveTopic
	OptionsFieldDefaultPresenter
	OptionsFieldLogoDirectory
	OptionsFieldYouTubeSetup
	OptionsFieldSave
)

// FileBrowserField represents which part of the file browser is focused
type FileBrowserField int

const (
	FileBrowserFieldList FileBrowserField = iota
	FileBrowserFieldPathInput
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

	// Logo directory path
	logoDirectory string

	// Custom file browser (for selecting logo directory)
	showFileBrowser      bool
	selectingDirectory   bool // true when selecting directory, not file
	browserCurrentDir    string
	browserEntries       []fileEntry
	browserSelected      int
	browserScrollTop     int
	browserPathInput     textinput.Model
	browserField         FileBrowserField

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

	// Path input for file browser
	pathInput := textinput.New()
	pathInput.Placeholder = "Enter or paste path..."
	pathInput.CharLimit = 500
	pathInput.Width = 50

	// Start in home directory or logo directory if set
	home, _ := os.UserHomeDir()
	browserDir := home
	if cfg.LogoDirectory != "" {
		browserDir = cfg.LogoDirectory
	}

	return &OptionsModel{
		config:             cfg,
		topics:             topics,
		selectedTopic:      0,
		focusedField:       OptionsFieldTopicList,
		newTopicInput:      newTopicInput,
		presenterInput:     presenterInput,
		logoDirectory:      cfg.LogoDirectory,
		showFileBrowser:    false,
		selectingDirectory: false,
		browserCurrentDir:  browserDir,
		browserPathInput:   pathInput,
		browserField:       FileBrowserFieldList,
	}
}

// loadBrowserEntries loads the directory contents for the file browser
func (m *OptionsModel) loadBrowserEntries() {
	m.browserEntries = nil
	m.browserSelected = 0
	m.browserScrollTop = 0

	entries, err := os.ReadDir(m.browserCurrentDir)
	if err != nil {
		return
	}

	// Add parent directory entry if not at root
	if m.browserCurrentDir != "/" {
		m.browserEntries = append(m.browserEntries, fileEntry{
			name:  "..",
			path:  filepath.Dir(m.browserCurrentDir),
			isDir: true,
		})
	}

	// Collect directories (and files if not selecting directory)
	var dirs []fileEntry
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(m.browserCurrentDir, entry.Name())
		fe := fileEntry{
			name:  entry.Name(),
			path:  fullPath,
			isDir: entry.IsDir(),
		}

		if entry.IsDir() {
			dirs = append(dirs, fe)
		}
		// When selecting directory, we only show directories
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name)
	})

	m.browserEntries = append(m.browserEntries, dirs...)
}

// openDirectoryBrowser opens the file browser for selecting a directory
func (m *OptionsModel) openDirectoryBrowser() {
	m.showFileBrowser = true
	m.selectingDirectory = true
	m.browserField = FileBrowserFieldList
	m.browserPathInput.SetValue(m.browserCurrentDir)
	m.browserPathInput.Blur()
	m.loadBrowserEntries()
}

// closeFileBrowser closes the file browser
func (m *OptionsModel) closeFileBrowser() {
	m.showFileBrowser = false
	m.selectingDirectory = false
}

// IsFileBrowserActive returns true if the file browser is currently shown
func (m *OptionsModel) IsFileBrowserActive() bool {
	return m.showFileBrowser
}

// RenderFileBrowser renders the file browser with full screen layout
func (m *OptionsModel) RenderFileBrowser(width, height int) string {
	m.width = width
	m.height = height
	return m.renderFileBrowser()
}

// Init initializes the model
func (m *OptionsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *OptionsModel) Update(msg tea.Msg) (*OptionsModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle file browser if active
	if m.showFileBrowser {
		return m.updateFileBrowser(msg)
	}

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

		case "enter", " ":
			switch m.focusedField {
			case OptionsFieldAddTopic:
				m.addTopic()
				return m, nil
			case OptionsFieldRemoveTopic:
				m.removeTopic()
				return m, nil
			case OptionsFieldLogoDirectory:
				m.openDirectoryBrowser()
				return m, nil
			case OptionsFieldYouTubeSetup:
				return m, func() tea.Msg { return goToYouTubeSetupMsg{} }
			case OptionsFieldSave:
				m.save()
				return m, nil
			default:
				m.nextField()
				return m, nil
			}

		case "c":
			// Clear logo directory if on that field
			if m.focusedField == OptionsFieldLogoDirectory {
				m.logoDirectory = ""
				m.message = "Logo directory cleared"
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
	m.config.LogoDirectory = m.logoDirectory

	if err := config.Save(m.config); err != nil {
		m.err = err
		return
	}

	m.message = "Settings saved successfully"
}

// View renders the options screen
func (m *OptionsModel) View() string {
	// If file browser is shown, render it instead
	if m.showFileBrowser {
		return m.renderFileBrowser()
	}

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBlue).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(18).
		Align(lipgloss.Right)

	labelActiveStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Width(18).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	valueActiveStyle := lipgloss.NewStyle().
		Foreground(ColorOrange)

	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)

	activeButtonStyle := buttonStyle.
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	inactiveButtonStyle := buttonStyle.
		Background(ColorGray).
		Foreground(ColorWhite)

	// Topic Management Section
	topicSection := sectionStyle.Render("Topics")

	// Topic list - simple inline display
	var topicItems []string
	for i, topic := range m.topics {
		style := lipgloss.NewStyle().Foreground(ColorGray)
		if i == m.selectedTopic {
			if m.focusedField == OptionsFieldTopicList {
				style = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000000"))
			} else {
				style = lipgloss.NewStyle().Foreground(ColorWhite)
			}
		}
		topicItems = append(topicItems, style.Render(" "+topic.Name+" "))
	}
	topicListStr := lipgloss.JoinHorizontal(lipgloss.Center, topicItems...)

	topicLabel := labelStyle.Render("Topics: ")
	if m.focusedField == OptionsFieldTopicList {
		topicLabel = labelActiveStyle.Render("Topics: ")
	}
	topicRow := lipgloss.JoinHorizontal(lipgloss.Center, topicLabel, topicListStr)

	// Add topic input
	addLabel := labelStyle.Render("Add: ")
	if m.focusedField == OptionsFieldAddTopic {
		addLabel = labelActiveStyle.Render("Add: ")
	}
	addTopicRow := lipgloss.JoinHorizontal(lipgloss.Center, addLabel, m.newTopicInput.View())

	// Remove button
	removeLabel := labelStyle.Render("")
	removeBtn := inactiveButtonStyle.Render("Remove")
	if m.focusedField == OptionsFieldRemoveTopic {
		removeBtn = activeButtonStyle.Render("Remove")
	}
	removeRow := lipgloss.JoinHorizontal(lipgloss.Center, removeLabel, "  ", removeBtn)

	// Default Presenter Section
	presenterSection := sectionStyle.Render("Presenter")
	presenterLabel := labelStyle.Render("Default: ")
	if m.focusedField == OptionsFieldDefaultPresenter {
		presenterLabel = labelActiveStyle.Render("Default: ")
	}
	presenterRow := lipgloss.JoinHorizontal(lipgloss.Center, presenterLabel, m.presenterInput.View())

	// Logo Settings Section
	logoSection := sectionStyle.Render("Logos")

	// Logo directory
	logoDirLabel := labelStyle.Render("Directory: ")
	if m.focusedField == OptionsFieldLogoDirectory {
		logoDirLabel = labelActiveStyle.Render("Directory: ")
	}
	var logoDirValue string
	if m.logoDirectory == "" {
		if m.focusedField == OptionsFieldLogoDirectory {
			logoDirValue = valueActiveStyle.Render("(browse...)")
		} else {
			logoDirValue = hintStyle.Render("(not set)")
		}
	} else {
		if m.focusedField == OptionsFieldLogoDirectory {
			logoDirValue = valueActiveStyle.Render(m.logoDirectory)
		} else {
			logoDirValue = valueStyle.Render(m.logoDirectory)
		}
	}
	logoDirRow := lipgloss.JoinHorizontal(lipgloss.Center, logoDirLabel, logoDirValue)
	logoDirHint := hintStyle.Render("                    logos selected per-recording")

	// YouTube Section
	youtubeSection := sectionStyle.Render("YouTube")
	youtubeLabel := labelStyle.Render("Status: ")
	if m.focusedField == OptionsFieldYouTubeSetup {
		youtubeLabel = labelActiveStyle.Render("Status: ")
	}

	// Get YouTube status
	cfg, _ := config.Load()
	youtubeStatus := cfg.GetYouTubeAuthStatus()
	var youtubeStatusText string
	var youtubeStatusColor lipgloss.Color
	switch youtubeStatus {
	case 3: // AuthStatusAuthenticated
		youtubeStatusText = "Connected"
		youtubeStatusColor = ColorGreen
		if cfg.YouTube.ChannelName != "" {
			youtubeStatusText = "Connected: " + cfg.YouTube.ChannelName
		}
	case 2: // AuthStatusConfigured
		youtubeStatusText = "Not Connected (press enter to connect)"
		youtubeStatusColor = ColorOrange
	default:
		youtubeStatusText = "Not Set Up (press enter to configure)"
		youtubeStatusColor = ColorGray
	}
	if m.focusedField == OptionsFieldYouTubeSetup {
		youtubeStatusText = "▶ " + youtubeStatusText
	}
	youtubeStatusStyled := lipgloss.NewStyle().Foreground(youtubeStatusColor).Render(youtubeStatusText)
	youtubeRow := lipgloss.JoinHorizontal(lipgloss.Center, youtubeLabel, youtubeStatusStyled)

	// Save button
	saveLabel := labelStyle.Render("")
	saveBtn := inactiveButtonStyle.Render("Save")
	if m.focusedField == OptionsFieldSave {
		saveBtn = activeButtonStyle.Render("Save")
	}
	saveRow := lipgloss.JoinHorizontal(lipgloss.Center, saveLabel, "  ", saveBtn)

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

	// Build the view
	return lipgloss.JoinVertical(lipgloss.Left,
		topicSection,
		topicRow,
		addTopicRow,
		removeRow,
		presenterSection,
		presenterRow,
		logoSection,
		logoDirRow,
		logoDirHint,
		youtubeSection,
		youtubeRow,
		"",
		saveRow,
		"",
		statusLine,
	)
}

// updateFileBrowser handles messages when the file browser is active
func (m *OptionsModel) updateFileBrowser(msg tea.Msg) (*OptionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle path input mode
		if m.browserField == FileBrowserFieldPathInput {
			switch msg.String() {
			case "esc":
				m.browserField = FileBrowserFieldList
				m.browserPathInput.Blur()
				return m, nil
			case "enter":
				// Try to navigate to the entered path
				path := m.browserPathInput.Value()
				if info, err := os.Stat(path); err == nil {
					if info.IsDir() {
						m.browserCurrentDir = path
						m.loadBrowserEntries()
					}
				}
				m.browserField = FileBrowserFieldList
				m.browserPathInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.browserPathInput, cmd = m.browserPathInput.Update(msg)
				return m, cmd
			}
		}

		// Handle file list mode
		switch msg.String() {
		case "esc", "q":
			m.closeFileBrowser()
			return m, nil

		case "up", "k":
			if m.browserSelected > 0 {
				m.browserSelected--
				// Scroll up if needed
				if m.browserSelected < m.browserScrollTop {
					m.browserScrollTop = m.browserSelected
				}
			}
			return m, nil

		case "down", "j":
			if m.browserSelected < len(m.browserEntries)-1 {
				m.browserSelected++
				// Scroll down if needed
				visibleHeight := m.height - 12 // Account for header, footer, path input
				if m.browserSelected >= m.browserScrollTop+visibleHeight {
					m.browserScrollTop = m.browserSelected - visibleHeight + 1
				}
			}
			return m, nil

		case "enter", " ":
			if len(m.browserEntries) > 0 && m.browserSelected < len(m.browserEntries) {
				entry := m.browserEntries[m.browserSelected]
				if entry.isDir {
					// Navigate into directory
					m.browserCurrentDir = entry.path
					m.browserPathInput.SetValue(entry.path)
					m.loadBrowserEntries()
				}
			}
			return m, nil

		case "s":
			// Select current directory (when selecting directory)
			if m.selectingDirectory {
				m.logoDirectory = m.browserCurrentDir
				m.message = "Logo directory set: " + m.browserCurrentDir
				m.closeFileBrowser()
			}
			return m, nil

		case "backspace":
			// Go to parent directory
			if m.browserCurrentDir != "/" {
				m.browserCurrentDir = filepath.Dir(m.browserCurrentDir)
				m.browserPathInput.SetValue(m.browserCurrentDir)
				m.loadBrowserEntries()
			}
			return m, nil

		case "tab", "/":
			// Switch to path input
			m.browserField = FileBrowserFieldPathInput
			m.browserPathInput.Focus()
			return m, textinput.Blink

		case "~":
			// Go to home directory
			if home, err := os.UserHomeDir(); err == nil {
				m.browserCurrentDir = home
				m.browserPathInput.SetValue(home)
				m.loadBrowserEntries()
			}
			return m, nil
		}
	}

	return m, nil
}

// renderFileBrowser renders the custom file browser
func (m *OptionsModel) renderFileBrowser() string {
	// Page title
	pageTitle := "Select Logo Directory"

	header := RenderHeader(pageTitle)

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(10).
		Align(lipgloss.Right)

	labelActiveStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Width(10).
		Align(lipgloss.Right)

	dirStyle := lipgloss.NewStyle().
		Foreground(ColorBlue)

	selectedStyle := lipgloss.NewStyle().
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	// Path input row
	pathLabel := labelStyle.Render("Path: ")
	if m.browserField == FileBrowserFieldPathInput {
		pathLabel = labelActiveStyle.Render("Path: ")
	}
	pathRow := lipgloss.JoinHorizontal(lipgloss.Center, pathLabel, m.browserPathInput.View())

	// Current directory display
	dirLabel := labelStyle.Render("In: ")
	dirRow := lipgloss.JoinHorizontal(lipgloss.Center, dirLabel, lipgloss.NewStyle().Foreground(ColorGray).Render(m.browserCurrentDir))

	// File list
	visibleHeight := m.height - 14 // Account for header, footer, path input, etc
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	var fileLines []string
	for i := m.browserScrollTop; i < len(m.browserEntries) && i < m.browserScrollTop+visibleHeight; i++ {
		entry := m.browserEntries[i]
		var line string
		if entry.isDir {
			line = dirStyle.Render("📁 " + entry.name)
		}

		if i == m.browserSelected && m.browserField == FileBrowserFieldList {
			line = selectedStyle.Render("▶ 📁 " + entry.name)
		}
		if line != "" {
			fileLines = append(fileLines, line)
		}
	}

	fileList := lipgloss.JoinVertical(lipgloss.Left, fileLines...)
	if len(m.browserEntries) == 0 {
		fileList = lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("(no subdirectories)")
	}

	// Content
	content := lipgloss.JoinVertical(lipgloss.Left,
		pathRow,
		dirRow,
		"",
		fileList,
	)

	// Help footer
	helpText := "↑/k ↓/j: navigate • enter: open dir • s: select this dir • backspace: parent • ~: home • esc: cancel"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// goToYouTubeSetupMsg signals navigation to YouTube setup screen
type goToYouTubeSetupMsg struct{}
