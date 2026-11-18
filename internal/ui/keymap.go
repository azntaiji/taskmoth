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
		act: func(a *App, _ *view.List, _ string) tea.Cmd { a.current, a.fromCalendar = a.lists[v], nil; return nil }}
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
				g := newGotoDialog()
				if l.Spec.Kind == view.Project {
					g.selectProject(a, l.Spec.Project) // start on the project being viewed
				}
				a.push(g)
				return nil
			}},
		gotoView("o", view.Overdue, "overdue"), gotoView("a", view.All, "all"),
		{keys: []string{"c"}, show: func(theme.Glyphs) string { return "g c" }, short: "calendar", long: "calendar",
			act: func(a *App, _ *view.List, _ string) tea.Cmd { a.openCalendar(); return nil }},
		gotoView("s", view.Someday, "someday"),
		gotoView("x", view.Archive, "archive"),
	}}

	listGroup = keyGroup{"Within Lists", []binding{
		{keys: []string{"up", "down"}, show: func(g theme.Glyphs) string { return g.Up + " / " + g.Down },
			short: "select", long: "select task", act: move},
		{keys: []string{"j", "k"}, show: func(theme.Glyphs) string { return "j / k" }, short: "select", long: "select task", act: move},
		{keys: []string{"pgup", "pgdown"}, show: func(theme.Glyphs) string { return "pgup / pgdn" }, short: "scroll", long: "scroll", act: move},
		{keys: []string{"home", "end", "G", "shift+g"}, show: func(theme.Glyphs) string { return "home / end" },
			short: "top/bottom", long: "first/last", act: move},
		{keys: []string{"q"}, short: "add", long: "add task", act: func(a *App, l *view.List, _ string) tea.Cmd {
			d := newAddTaskDialog(l.Spec.Project) // G5.4: a project list defaults to its project
			if l.Spec.Kind == view.DayList {
				d.setDue(l.Spec.Date) // G14.7
			}
			a.push(d)
			return nil
		}},
		{keys: []string{"s"}, short: "sort", long: "cycle sorting", act: func(a *App, l *view.List, _ string) tea.Cmd {
			if l.Spec.Order != nil {
				return nil // Search keeps best match first (G7.1d)
			}
			return a.saveSort(l, l.NextSort())
		}},
		{keys: []string{"o"}, short: "group", long: "cycle grouping", act: func(a *App, l *view.List, _ string) tea.Cmd {
			if len(l.Spec.Groups) == 0 {
				return nil // Inbox, Upcoming, the Archive and Search don't group (G13.1)
			}
			return a.saveGroup(l, l.NextGroup())
		}},
		{keys: []string{"c"}, short: "completed", long: "cycle completed",
			act: func(a *App, _ *view.List, _ string) tea.Cmd { return a.cycleCompleted() }},
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

	// calendarGroup is the Calendar's keys (G14.7, G14.8). Its bindings
	// ignore the List they're given and act on a.calendar.
	calendarGroup = keyGroup{"Calendar", []binding{
		{keys: []string{"up", "down"}, show: func(g theme.Glyphs) string { return g.Up + " / " + g.Down },
			short: "select", long: "select day", act: calendarMove},
		{keys: []string{"j", "k"}, show: func(theme.Glyphs) string { return "j / k" }, short: "select", long: "select day", act: calendarMove},
		{keys: []string{"pgup", "pgdown"}, show: func(theme.Glyphs) string { return "pgup / pgdn" }, short: "scroll", long: "scroll", act: calendarMove},
		{keys: []string{"[", "]"}, show: func(theme.Glyphs) string { return "[ / ]" }, short: "week", long: "previous/next week", act: calendarMove},
		{keys: []string{"home"}, short: "top", long: "first day", act: calendarMove},
		{keys: []string{"t"}, short: "today", long: "today", act: calendarMove},
		{keys: []string{"enter"}, short: "open day", long: "open day", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			a.openDay()
			return nil
		}},
		{keys: []string{"q"}, short: "add", long: "add task due that day", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			d := newAddTaskDialog("")
			if day, overdue := a.calendar.Selected(); !overdue {
				d.setDue(day)
			}
			a.push(d)
			return nil
		}},
		{keys: []string{"/"}, short: "search", long: "search for task", act: func(a *App, _ *view.List, _ string) tea.Cmd {
			a.openSearch()
			return nil
		}},
	}}

	keyGroups = []keyGroup{gotoGroup, listGroup, taskGroup, generalGroup}
	// calendarKeyGroups are the shortcuts dialog's columns in the Calendar
	// (G14.9).
	calendarKeyGroups = []keyGroup{gotoGroup, calendarGroup, generalGroup}
)

// calendarMove handles the Calendar's movement keys (G14.7).
func calendarMove(a *App, _ *view.List, key string) tea.Cmd {
	c := a.calendar
	switch key {
	case "up", "k":
		c.Move(-1)
	case "down", "j":
		c.Move(1)
	case "pgup":
		c.Page(-1)
	case "pgdown":
		c.Page(1)
	case "[":
		c.Week(-1)
	case "]":
		c.Week(1)
	case "home":
		c.Top()
	case "t":
		c.Today()
	}
	return nil
}

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
	case view.DayList: // G14.8: due date in place of search, and the way back
		row1 = []w.Hint{hintFor(g, "q"), hintFor(g, "space"), hintFor(g, "e"), hintFor(g, "t", "due date"),
			hintFor(g, "x"), hintFor(g, "s"), hintFor(g, "esc")}
	}
	row2 := []w.Hint{gotoHint(g, "i"), gotoHint(g, "t"), gotoHint(g, "u"), gotoHint(g, "p"), gotoHint(g, "o"), gotoHint(g, "a")}
	return [][]w.Hint{row1, row2}
}

// calendarHint is the hint for a Calendar key.
func calendarHint(g theme.Glyphs, key string) w.Hint {
	b, ok := lookup(key, calendarGroup, generalGroup)
	if !ok {
		panic("no calendar binding for hint " + key) // caught by TestHintsMatchKeyMap
	}
	return b.hint(g)
}

// calendarHints are the Calendar's two hint rows (G14.9).
func calendarHints(g theme.Glyphs) [][]w.Hint {
	row1 := []w.Hint{calendarHint(g, "q"), calendarHint(g, "enter"), calendarHint(g, "["), calendarHint(g, "t"),
		calendarHint(g, "?"), calendarHint(g, "/")}
	row2 := []w.Hint{gotoHint(g, "i"), gotoHint(g, "t"), gotoHint(g, "u"), gotoHint(g, "p"), gotoHint(g, "o"), gotoHint(g, "a")}
	return [][]w.Hint{row1, row2}
}
