// Package ui holds taskmoth's Bubble Tea models.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

// App is the root model. In Phase 0 it only draws a placeholder and handles
// quitting; views, dialogs and key chords arrive in later phases.
type App struct {
	width, height int
}

// NewApp returns the root model.
func NewApp() App {
	return App{}
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
	v := tea.NewView("taskmoth\n\nctrl+c / ctrl+q to quit\n")
	v.AltScreen = true
	v.WindowTitle = "taskmoth"
	return v
}
