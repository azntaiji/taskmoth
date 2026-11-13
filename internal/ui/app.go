// Package ui holds taskmoth's Bubble Tea models.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/widget"
)

// App is the root model. So far it draws a placeholder and handles quitting
// and terminal size; views, dialogs and key chords arrive in later phases.
type App struct {
	theme         theme.Theme
	width, height int
}

// NewApp returns the root model drawn with t.
func NewApp(t theme.Theme) App {
	return App{theme: t}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		if isQuit(msg) {
			return a, tea.Quit
		}
	}
	return a, nil
}

// isQuit reports whether msg is a quit key. `q` is "add task" in the designs,
// so quitting is ctrl+c or ctrl+q (decision G0.4 in PLAN.md).
func isQuit(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		return true
	}
	return false
}

// View implements tea.Model.
func (a App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.WindowTitle = "taskmoth"
	return v
}

func (a App) render() string {
	t := a.theme
	switch {
	case a.width == 0 || a.height == 0:
		return "" // size not known yet
	case a.width < widget.MinWidth || a.height < widget.MinHeight:
		return widget.TooSmall(t, a.width, a.height)
	}
	lines := []string{widget.Header(t, a.width, "Today", "")}
	for len(lines) < a.height-2 {
		lines = append(lines, "")
	}
	lines = append(lines, widget.StatusBar(t, a.width, "Today", theme.ToneInfo, "Task views arrive in Phase 3", ""))
	lines = append(lines, widget.KeyHints(t, a.width, []widget.Hint{{Key: "ctrl+q", Label: "quit"}})...)
	return widget.Screen(t, a.width, a.height, lines)
}
