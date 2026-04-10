package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewAppModel(t *testing.T) {
	m := NewAppModel()
	if m.width != 0 {
		t.Errorf("expected width 0, got %d", m.width)
	}
	if m.height != 0 {
		t.Errorf("expected height 0, got %d", m.height)
	}
}

func TestQuitOnQ(t *testing.T) {
	m := NewAppModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing q")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestQuitOnCtrlC(t *testing.T) {
	m := NewAppModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected non-nil cmd when pressing ctrl+c")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewAppModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)

	model, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel, got %T", updated)
	}

	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}

func TestViewRendersPlaceholder(t *testing.T) {
	m := NewAppModel()
	// Set dimensions so View can render properly
	m.width = 100
	m.height = 30
	view := m.View()

	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Check that the placeholder text is present
	expected := "crypto-tracker"
	if !contains(view, expected) {
		t.Errorf("expected view to contain %q, got %q", expected, view)
	}

	expectedQuit := "press q to quit"
	if !contains(view, expectedQuit) {
		t.Errorf("expected view to contain %q, got %q", expectedQuit, view)
	}
}

func TestIgnoresOtherKeys(t *testing.T) {
	m := NewAppModel()
	otherKeys := []rune{'a', 'b', 'c', 'x', 'z', '1', ' '}
	for _, key := range otherKeys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
		_, cmd := m.Update(msg)
		if cmd != nil {
			t.Errorf("expected nil cmd for key %q, got non-nil cmd", key)
		}
	}
}

// contains checks if s contains substr (simple substring check).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
