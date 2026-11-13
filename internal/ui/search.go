package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// search is the Search view (Figma 40:16937): a list of results with the
// query input in place of the status bar. It lives until restart, so `/`
// brings back the last query (G7.2b).
type search struct {
	input w.Input
	list  *view.List
	prev  *view.List // the view `esc` returns to
}

// openSearch shows Search with the input focused (G7.2a).
func (a *App) openSearch() {
	if a.search == nil {
		a.search = &search{input: w.Input{Placeholder: "search for a task..."},
			list: view.NewList(view.SearchSpec(task.Query{}, a.projectName))}
		a.search.list.SetHeight(a.bodyHeight())
	}
	s := a.search
	if a.current != s.list {
		s.prev = a.current
	}
	a.current = s.list
	s.input.Focus()
	a.updateSearch()
}

func (a *App) projectName(id task.ID) string { return a.projects[id] }

// updateSearch re-runs the query (G7.1d: on every keystroke) and selects
// the best match.
func (a *App) updateSearch() {
	s := a.search
	s.list.Spec = view.SearchSpec(task.ParseQuery(s.input.Value()), a.projectName)
	s.list.SetTasks(a.tasks, a.clock)
	s.list.Top()
}

// handleSearchInput handles a key while the query input has focus: letters
// type, ↑/↓ still move the selection, enter focuses the results and esc
// leaves Search (G7.2a).
func (a *App) handleSearchInput(msg tea.KeyPressMsg) tea.Cmd {
	s := a.search
	switch key := msg.String(); key {
	case "esc":
		s.input.Blur()
		a.current = s.prev
		if a.current == nil {
			a.current = a.lists[view.Today]
		}
	case "enter":
		s.input.Blur()
	case "up", "down", "pgup", "pgdown":
		return move(a, s.list, key)
	default:
		before := s.input.Value()
		s.input.Update(msg)
		if s.input.Value() != before {
			a.updateSearch()
		}
	}
	return nil
}

// searchBar replaces the status bar in Search: the view badge, then the
// query input on statusbar/bg with a text/warning placeholder (40:17331),
// and at the right a hint for what enter and esc do from here (G7.4).
func (a *App) searchBar(t theme.Theme) string {
	bg := t.Style(t.C.Status.Fg, t.C.Status.Bg)
	left := w.Blank(bg, 1) + w.Badge(t, theme.ToneInfo, "Search", false) + w.Blank(bg, 2)
	s := a.search
	hints := []w.Hint{{Key: "esc", Label: "edit query"}}
	if s.input.Focused() {
		hints = []w.Hint{{Key: "enter", Label: "results"}, {Key: "esc", Label: "exit search"}}
	}
	right := searchHints(t, hints)
	s.input.SetSize(max(a.width-w.Width(left)-w.Width(right)-3, w.PromptWidth+1), 1)
	return w.Spread(left+s.input.ViewOn(t, t.C.Status.Bg, t.C.TextWarning), right+w.Blank(bg, 1), a.width, bg)
}

// searchHints draws key hints on statusbar/bg: keys bold in key/key, labels
// faded (key/label matches statusbar/bg, so it can't be used here).
func searchHints(t theme.Theme, hints []w.Hint) string {
	bg := t.Style(t.C.Status.Fg, t.C.Status.Bg)
	key := t.Style(t.C.Key, t.C.Status.Bg).Bold(true)
	label := t.Style(t.FadeOn(t.C.Status.Fg, t.C.Status.Bg), t.C.Status.Bg)
	out := ""
	for i, h := range hints {
		if i > 0 {
			out += bg.Render("  ")
		}
		out += key.Render(h.Key) + bg.Render(" ") + label.Render(h.Label)
	}
	return out
}
