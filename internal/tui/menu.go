package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents a menu option
type MenuItem int

const (
	MenuNewRecording MenuItem = iota
	MenuRecordingHistory
	MenuOptions
	MenuQuit
)

// menuItem holds menu item data
type menuItem struct {
	label   string
	enabled bool
	action  MenuItem
}

// MenuModel represents the main menu screen
type MenuModel struct {
	selectedItem int
	menuItems    []menuItem
	width        int
	height       int
}

// NewMenuModel creates a new menu model
func NewMenuModel() *MenuModel {
	return &MenuModel{
		selectedItem: 0,
		menuItems: []menuItem{
			{label: "New Recording", enabled: true, action: MenuNewRecording},
			{label: "Recording History", enabled: true, action: MenuRecordingHistory},
			{label: "Options", enabled: true, action: MenuOptions},
			{label: "Quit", enabled: true, action: MenuQuit},
		},
	}
}

// Init initializes the menu
func (m *MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the menu
func (m *MenuModel) Update(msg tea.Msg) (*MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		// Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
			return m, tea.Quit

		// Navigate up
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.selectedItem--
			if m.selectedItem < 0 {
				m.selectedItem = len(m.menuItems) - 1
			}
			return m, nil

		// Navigate down
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.selectedItem++
			if m.selectedItem >= len(m.menuItems) {
				m.selectedItem = 0
			}
			return m, nil

		// Select item
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if m.selectedItem >= 0 && m.selectedItem < len(m.menuItems) {
				item := m.menuItems[m.selectedItem]
				if item.enabled {
					return m, m.handleSelection(item.action)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// handleSelection handles menu item selection
func (m *MenuModel) handleSelection(action MenuItem) tea.Cmd {
	switch action {
	case MenuNewRecording:
		return func() tea.Msg {
			return menuActionMsg{action: MenuNewRecording}
		}
	case MenuRecordingHistory:
		return func() tea.Msg {
			return menuActionMsg{action: MenuRecordingHistory}
		}
	case MenuOptions:
		return func() tea.Msg {
			return menuActionMsg{action: MenuOptions}
		}
	case MenuQuit:
		return tea.Quit
	}
	return nil
}

// View renders the menu
func (m *MenuModel) View() string {
	// Render header
	header := RenderSimpleHeader("Main Menu")

	// Render menu items
	menu := m.renderMenuItems()

	// Render help footer
	helpText := "↑/k: up • ↓/j: down • enter/space: select • q: quit"
	footer := RenderHelpFooter(helpText, m.width)

	// Use standard layout
	return LayoutWithHeaderFooter(header, menu, footer, m.width, m.height)
}

// renderMenuItems renders the menu items
func (m *MenuModel) renderMenuItems() string {
	normalStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Padding(0, 2)

	selectedStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Padding(0, 2)

	disabledStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Padding(0, 2)

	var items []string
	for i, item := range m.menuItems {
		prefix := "  "
		if i == m.selectedItem {
			prefix = "▶ "
		}

		var rendered string
		if !item.enabled {
			rendered = disabledStyle.Render(prefix + item.label + " (disabled)")
		} else if i == m.selectedItem {
			rendered = selectedStyle.Render(prefix + item.label)
		} else {
			rendered = normalStyle.Render(prefix + item.label)
		}

		items = append(items, rendered)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// SelectedAction returns the currently selected action
func (m *MenuModel) SelectedAction() MenuItem {
	if m.selectedItem >= 0 && m.selectedItem < len(m.menuItems) {
		return m.menuItems[m.selectedItem].action
	}
	return MenuNewRecording
}

// menuActionMsg is sent when a menu item is selected
type menuActionMsg struct {
	action MenuItem
}
