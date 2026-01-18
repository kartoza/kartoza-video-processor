package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMenuModel(t *testing.T) {
	m := NewMenuModel()

	if m == nil {
		t.Fatal("NewMenuModel returned nil")
	}

	if m.selectedItem != 0 {
		t.Errorf("expected selectedItem to be 0, got %d", m.selectedItem)
	}

	if len(m.menuItems) != 4 {
		t.Errorf("expected 4 menu items, got %d", len(m.menuItems))
	}

	// Check menu item labels
	expectedLabels := []string{"New Recording", "Recording History", "Options", "Quit"}
	for i, item := range m.menuItems {
		if item.label != expectedLabels[i] {
			t.Errorf("expected menu item %d to be %q, got %q", i, expectedLabels[i], item.label)
		}
		if !item.enabled {
			t.Errorf("expected menu item %d to be enabled", i)
		}
	}
}

func TestMenuModel_NavigationDown(t *testing.T) {
	m := NewMenuModel()

	// Navigate down through all items
	for i := 0; i < 4; i++ {
		if m.selectedItem != i {
			t.Errorf("expected selectedItem to be %d, got %d", i, m.selectedItem)
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyDown}
		newM, _ := m.Update(keyMsg)
		m = newM
	}

	// Should wrap to 0
	if m.selectedItem != 0 {
		t.Errorf("expected selectedItem to wrap to 0, got %d", m.selectedItem)
	}
}

func TestMenuModel_NavigationUp(t *testing.T) {
	m := NewMenuModel()

	// Navigate up from 0 should wrap to last item
	keyMsg := tea.KeyMsg{Type: tea.KeyUp}
	newM, _ := m.Update(keyMsg)
	m = newM

	if m.selectedItem != 3 {
		t.Errorf("expected selectedItem to wrap to 3, got %d", m.selectedItem)
	}
}

func TestMenuModel_VimKeysJ(t *testing.T) {
	m := NewMenuModel()

	// Test vim 'j' key for down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ := m.Update(keyMsg)
	m = newM

	if m.selectedItem != 1 {
		t.Errorf("expected selectedItem to be 1 after 'j', got %d", m.selectedItem)
	}
}

func TestMenuModel_VimKeysK(t *testing.T) {
	m := NewMenuModel()
	m.selectedItem = 2 // Start at item 2

	// Test vim 'k' key for up
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ := m.Update(keyMsg)
	m = newM

	if m.selectedItem != 1 {
		t.Errorf("expected selectedItem to be 1 after 'k', got %d", m.selectedItem)
	}
}

func TestMenuModel_SelectNewRecording(t *testing.T) {
	m := NewMenuModel()
	m.selectedItem = 0 // New Recording

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(keyMsg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	// Execute the command to get the message
	msg := cmd()
	actionMsg, ok := msg.(menuActionMsg)
	if !ok {
		t.Fatal("expected menuActionMsg")
	}

	if actionMsg.action != MenuNewRecording {
		t.Errorf("expected MenuNewRecording action, got %d", actionMsg.action)
	}
}

func TestMenuModel_SelectQuit(t *testing.T) {
	m := NewMenuModel()
	m.selectedItem = 3 // Quit

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(keyMsg)

	if cmd == nil {
		t.Fatal("expected command to be returned for Quit")
	}
}

func TestMenuModel_SpaceKeySelect(t *testing.T) {
	m := NewMenuModel()
	m.selectedItem = 1 // Recording History

	keyMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := m.Update(keyMsg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	msg := cmd()
	actionMsg, ok := msg.(menuActionMsg)
	if !ok {
		t.Fatal("expected menuActionMsg")
	}

	if actionMsg.action != MenuRecordingHistory {
		t.Errorf("expected MenuRecordingHistory action, got %d", actionMsg.action)
	}
}

func TestMenuModel_WindowResize(t *testing.T) {
	m := NewMenuModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newM, _ := m.Update(msg)
	m = newM

	if m.width != 120 {
		t.Errorf("expected width to be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height to be 40, got %d", m.height)
	}
}

func TestMenuModel_SelectedAction(t *testing.T) {
	m := NewMenuModel()

	tests := []struct {
		selectedItem int
		expected     MenuItem
	}{
		{0, MenuNewRecording},
		{1, MenuRecordingHistory},
		{2, MenuOptions},
		{3, MenuQuit},
	}

	for _, tt := range tests {
		m.selectedItem = tt.selectedItem
		got := m.SelectedAction()
		if got != tt.expected {
			t.Errorf("SelectedAction() with selectedItem=%d: expected %d, got %d",
				tt.selectedItem, tt.expected, got)
		}
	}
}

func TestMenuModel_View(t *testing.T) {
	m := NewMenuModel()
	m.width = 80
	m.height = 24

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check that menu items are rendered
	if !containsString(view, "New Recording") {
		t.Error("expected view to contain 'New Recording'")
	}
	if !containsString(view, "Recording History") {
		t.Error("expected view to contain 'Recording History'")
	}
	if !containsString(view, "Options") {
		t.Error("expected view to contain 'Options'")
	}
	if !containsString(view, "Quit") {
		t.Error("expected view to contain 'Quit'")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
