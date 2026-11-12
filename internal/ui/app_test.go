package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      tea.KeyPressMsg
		wantQuit bool
	}{
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, true},
		{"ctrl+q", tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}, true},
		{"q is add, not quit", tea.KeyPressMsg{Code: 'q', Text: "q"}, false},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := NewApp().Update(tt.key)
			gotQuit := cmd != nil && isQuitMsg(cmd())
			if gotQuit != tt.wantQuit {
				t.Errorf("key %q: quit = %v, want %v", tt.key.String(), gotQuit, tt.wantQuit)
			}
		})
	}
}

func isQuitMsg(msg tea.Msg) bool {
	_, ok := msg.(tea.QuitMsg)
	return ok
}

func TestViewUsesAltScreen(t *testing.T) {
	if !NewApp().View().AltScreen {
		t.Error("View should request the alternate screen")
	}
}

func TestWindowSizeIsStored(t *testing.T) {
	m, _ := NewApp().Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a := m.(App)
	if a.width != 80 || a.height != 24 {
		t.Errorf("size = %dx%d, want 80x24", a.width, a.height)
	}
}
