package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

func newApp() App { return NewApp(theme.New(false)) }

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
			_, cmd := newApp().Update(tt.key)
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
	if !newApp().View().AltScreen {
		t.Error("View should request the alternate screen")
	}
}

func sized(w, h int) App {
	m, _ := newApp().Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m.(App)
}

func TestViewFillsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		lines := strings.Split(sized(size[0], size[1]).View().Content, "\n")
		if len(lines) != size[1] {
			t.Errorf("%v: %d lines", size, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != size[0] {
				t.Errorf("%v: line %d is %d cells", size, i, w)
			}
		}
	}
}

func TestViewTooSmall(t *testing.T) {
	for _, size := range [][2]int{{79, 24}, {80, 23}, {40, 10}} {
		content := ansi.Strip(sized(size[0], size[1]).View().Content)
		if !strings.Contains(content, "Terminal too small") {
			t.Errorf("%v: expected too-small message", size)
		}
	}
	if strings.Contains(ansi.Strip(sized(80, 24).View().Content), "too small") {
		t.Error("80×24 should be allowed")
	}
}
