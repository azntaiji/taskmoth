package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// binding is one entry in the key map. The list views dispatch keys through
// the map, and the hint rows and the Keyboard Shortcuts dialog are generated
// from it, so the three can't drift apart (Phase 7).
type binding struct {
	keys  []string                                       // key strings as Bubble Tea reports them
	show  func(g theme.Glyphs) string                    // how the key is drawn; nil = keys[0]
	short string                                         // label in hint rows
	long  string                                         // label in the shortcuts dialog
	act   func(a *App, l *view.List, key string) tea.Cmd // nil: handled elsewhere (esc, quit)
}

func (b binding) key(g theme.Glyphs) string {
	if b.show != nil {
		return b.show(g)
	}
	return b.keys[0]
}

func (b binding) hint(g theme.Glyphs) w.Hint { return w.Hint{Key: b.key(g), Label: b.short} }

// keyGroup is a column of the shortcuts dialog (Figma 40:15079, G7.3b).
type keyGroup struct {
	title    string
	bindings []binding
}

// gotoView is a `g` chord binding that switches to a fixed view.
func gotoView(k string, v view.Kind, label string) binding {
	return binding{keys: []string{k}, show: func(theme.Glyphs) string { return "g " + k }, short: label, long: label,
		act: func(a *App, _ *view.List, _ string) tea.Cmd { a.current = a.lists[v]; return nil }}
}

// onSelected runs f on the selected task, if any.
func onSelected(f func(a *App, l *view.List, t task.Task) tea.Cmd) func(a *App, l *view.List, key string) tea.Cmd {
	return func(a *App, l *view.List, _ string) tea.Cmd {
		if t, ok := l.Selected(); ok {
			return f(a, l, t)
		}
		return nil
	}
}

// editBinding opens one of the field edit dialogs (G6.6); not in the Archive.
func editBinding(f editField) binding {
	open := f.open
	return binding{keys: []string{f.key}, short: f.short, long: "edit " + f.short,
		act: onSelected(func(a *App, l *view.List, t task.Task) tea.Cmd {
			if l.Spec.Kind != view.Archive {
				a.push(open(a, t))
			}
			return nil
		})}
}

var (
	gotoGroup = keyGroup{"Goto Lists", []binding{
		gotoView("i", view.Inbox, "inbox"), gotoView("t", view.Today, "today"), gotoView("u", view.Upcoming, "upcoming"),
		{keys: []string{"p"}, show: func(theme.Glyphs) string { return "g p" }, short: "projects", long: "projects",
			act: func(a *App, l *view.List, _ string) tea.Cmd {
				g := &gotoDialog{}
				if l.Spec.Kind == view.Project {
					g.selectProject(a, l.Spec.Project) // start on the project being viewed
				}
				a.push(g)
				return nil
			}},
		gotoView("o", view.Overdue, "overdue"), gotoView("a", view.All, "all"),
		gotoView("c", view.Completed, "completed"), gotoView("x", view.Archive, "archive"),
	}}

	listGroup = keyGroup{"Within Lists", []binding{
		{keys: []string{"up", "down"}, show: func(g theme.Glyphs) string { return g.Up + " / " + g.Down },
			short: "select", long: "select task", act: move},
		{keys: []string{"j", "k"}, show: func(theme.Glyphs) string { return "j / k" }, short: "select", long: "select task", act: move},
		{keys: []string{"pgup", "pgdown"}, show: func(theme.Glyphs) string { return "pgup / pgdn" }, short: "scroll", long: "scroll", act: move},
		{keys: []string{"home", "end", "G", "shift+g"}, show: func(theme.Glyphs) string { return "home / end" },
			short: "top/bottom", long: "first/last", act: move},
		{keys: []string{"q"}, short: "add", long: "add task", act: func(a *App, l *view.List, _ string) tea.Cmd {
			a.push(newAddTaskDialog(l.Spec.Project)) // G5.4: a project list defaults to its project
			return nil
		}},
		{keys: []string{"s"}, short: "sort", long: "cycle sorting", act: func(a *App, l *view.List, _ string) tea.Cmd {
			if l.Spec.Order != nil {
				return nil // Search keeps best match first (G7.1d)
			}
			return a.saveSort(l, l.NextSort())
		}},
		{keys: []string{"/"}, short: "search", long: "search for task", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			a.openSearch()
			return nil
		}},
		{keys: []string{"u"}, short: "unarchive", long: "unarchive",
			act: onSelected(func(a *App, l *view.List, t task.Task) tea.Cmd {
				if l.Spec.Kind == view.Archive {
					return a.unarchiveTask(t)
				}
				return nil
			})},
	}}

	taskGroup = keyGroup{"Selected Tasks", []binding{
		{keys: []string{"space"}, short: "done", long: "mark done", act: func(a *App, l *view.List, _ string) tea.Cmd {
			if l.Spec.Kind == view.Archive {
				return nil
			}
			return a.toggleDone()
		}},
		{keys: []string{"e"}, short: "edit/view", long: "edit/view task", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			a.openTaskView()
			return nil
		}},
		{keys: []string{"x"}, short: "archive", long: "archive", act: onSelected(func(a *App, l *view.List, t task.Task) tea.Cmd {
			if l.Spec.Kind == view.Archive {
				a.push(&confirmDialog{message: "Delete " + quote(a.theme, t.Name) + " forever?", action: "delete", run: a.deleteTask(t)})
			} else {
				a.push(&confirmDialog{message: "Archive " + quote(a.theme, t.Name) + "?", action: "archive", run: a.archiveTask(t)})
			}
			return nil
		})},
	}}

	generalGroup = keyGroup{"General", []binding{
		{keys: []string{"?"}, short: "shortcuts", long: "shortcuts", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			a.push(&shortcutsDialog{})
			return nil
		}},
		{keys: []string{"esc"}, short: "back", long: "back/cancel"},
		{keys: []string{"ctrl+q", "ctrl+c"}, short: "quit", long: "quit"},
	}}

	keyGroups = []keyGroup{gotoGroup, listGroup, taskGroup, generalGroup}
)

func init() {
	for _, f := range editFields {
		taskGroup.bindings = append(taskGroup.bindings, editBinding(f))
	}
	keyGroups[2] = taskGroup
}

// move handles the movement keys (G3.4).
func move(_ *App, l *view.List, key string) tea.Cmd {
	switch key {
	case "up", "k":
		l.Move(-1)
	case "down", "j":
		l.Move(1)
	case "pgup":
		l.Page(-1)
	case "pgdown":
		l.Page(1)
	case "home":
		l.Top()
	default: // end, G
		l.Bottom()
	}
	return nil
}

// lookup finds the binding for key in groups.
func lookup(key string, groups ...keyGroup) (binding, bool) {
	for _, g := range groups {
		for _, b := range g.bindings {
			if slices.Contains(b.keys, key) {
				return b, true
			}
		}
	}
	return binding{}, false
}

// hintFor is the hint for key, optionally relabelled.
func hintFor(g theme.Glyphs, key string, label ...string) w.Hint {
	b, ok := lookup(key, listGroup, taskGroup, generalGroup)
	if !ok {
		panic("no binding for hint " + key) // caught by TestHintsMatchKeyMap
	}
	h := b.hint(g)
	if len(label) > 0 {
		h.Label = label[0]
	}
	return h
}

// gotoHint is the hint for a `g` chord.
func gotoHint(g theme.Glyphs, key string) w.Hint {
	for _, b := range gotoGroup.bindings {
		if b.keys[0] == key {
			return b.hint(g)
		}
	}
	panic("no goto binding for " + key)
}

// listHints are a list view's two hint rows (32:9525); the Archive has its
// own first row (G4.5a) and Search drops `s` (G7.1d).
func listHints(g theme.Glyphs, k view.Kind) [][]w.Hint {
	row1 := []w.Hint{hintFor(g, "q"), hintFor(g, "space"), hintFor(g, "e"), hintFor(g, "x"),
		hintFor(g, "s"), hintFor(g, "?"), hintFor(g, "/")}
	switch k {
	case view.Archive:
		row1 = []w.Hint{hintFor(g, "u"), hintFor(g, "x", "delete forever"), hintFor(g, "s"), hintFor(g, "?"), hintFor(g, "/")}
	case view.Search:
		row1 = slices.DeleteFunc(row1, func(h w.Hint) bool { return h.Key == "s" })
	}
	row2 := []w.Hint{gotoHint(g, "i"), gotoHint(g, "t"), gotoHint(g, "u"), gotoHint(g, "p"), gotoHint(g, "o"), gotoHint(g, "a")}
	return [][]w.Hint{row1, row2}
}
