package gallery

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

var update = flag.Bool("update", false, "rewrite golden files")

func sized(t theme.Theme, w, h int) *Model {
	m := New(t)
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

// TestPagesFillTheScreen renders every page at the design size and a larger
// size, and checks each fills the terminal exactly. With -update it writes
// the 80×24 renders to testdata for side-by-side review against Figma.
func TestPagesFillTheScreen(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := theme.New(ascii)
		for _, size := range [][2]int{{80, 24}, {120, 40}} {
			m := sized(th, size[0], size[1])
			for i := range m.pages {
				m.page = i
				content := m.View().Content
				lines := strings.Split(content, "\n")
				if len(lines) != size[1] {
					t.Errorf("ascii=%v %v page %d: %d lines", ascii, size, i, len(lines))
				}
				for n, l := range lines {
					if w := ansi.StringWidth(l); w != size[0] {
						t.Errorf("ascii=%v %v page %d line %d: %d cells", ascii, size, i, n, w)
					}
				}
				if ascii || size[0] != 80 {
					continue
				}
				name := filepath.Join("testdata", fmt.Sprintf("page%d.golden", i))
				if *update {
					if err := os.MkdirAll("testdata", 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
					continue
				}
				want, err := os.ReadFile(name)
				if err != nil {
					t.Fatalf("missing golden %s (run with -update): %v", name, err)
				}
				if string(want) != content {
					t.Errorf("page %d (%s) differs from %s", i, m.pages[i].title, name)
				}
			}
		}
	}
}

func TestPageNavigationAndInput(t *testing.T) {
	m := sized(theme.New(false), 80, 24)
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.page != 1 {
		t.Fatalf("tab: page %d", m.page)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.page != len(m.pages)-1 {
		t.Fatalf("shift+tab wrap: page %d", m.page)
	}

	m.page = m.pageWith(m.date)
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: '9', Text: "9"})
	if m.date.Value() != "2026-10-09" {
		t.Errorf("date input = %q", m.date.Value())
	}
	m.page = 0
	before := m.desc.Value()
	m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if m.desc.Value() != before {
		t.Error("typing on a non-input page changed an input")
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+q should quit")
	}
}

func TestTooSmall(t *testing.T) {
	m := sized(theme.New(false), 60, 20)
	if !strings.Contains(ansi.Strip(m.View().Content), "Terminal too small") {
		t.Error("expected too-small message")
	}
}
