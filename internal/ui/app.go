// Package ui holds taskmoth's root Bubble Tea model and its dialogs.
package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
	"github.com/azntaiji/taskmoth/internal/ui/widget"
)

// fixedViews are the views reachable by a `g` chord (project lists open from
// Goto Project).
var fixedViews = map[string]view.Kind{
	"t": view.Today, "i": view.Inbox, "u": view.Upcoming, "o": view.Overdue,
	"a": view.All, "c": view.Completed, "x": view.Archive, "s": view.Someday,
}

// App is the root model: it owns the store, the list views, the clock, the
// key chords and the dialog stack.
type App struct {
	theme theme.Theme
	store store.Store
	now   func() time.Time

	clock         time.Time
	width, height int

	tasks       []task.Task
	projectList []task.Project     // every project, A–Z
	projects    map[task.ID]string // names of every project
	loaded      bool

	lists       map[view.Kind]*view.List
	projectView map[task.ID]*view.List
	current     *view.List

	dialogs             []dialog
	selectAfterLoad     task.ID   // project to select in Goto Project once reloaded
	selectTaskAfterLoad task.ID   // task to select in the current view once reloaded
	taskView            *taskView // the Single Task view, when open
	keyDisambiguation   bool      // the terminal reports shift+enter apart from enter (G6.2)
	pendingG            bool      // `g` pressed, waiting for the second key (G3.4: no timeout)
	message             string    // notice or error in the status bar until the next key
	search              *search   // the Search view, once opened (G7.2b keeps its query)
}

// NewApp returns the root model. now supplies the current time (injected so
// tests can fix the clock).
func NewApp(t theme.Theme, s store.Store, now func() time.Time) *App {
	a := &App{theme: t, store: s, now: now, clock: now(),
		lists: map[view.Kind]*view.List{}, projectView: map[task.ID]*view.List{}}
	for _, k := range fixedViews {
		a.lists[k] = view.NewList(view.SpecFor(k))
	}
	a.current = a.lists[view.Today] // the default view (brief)
	return a
}

// Messages.
type (
	loadedMsg struct {
		tasks    []task.Task
		projects []task.Project
		sorts    map[string]task.SortKey // view Key → saved sort; first load only
	}
	reloadMsg      struct{}
	errMsg         struct{ err error }
	tickMsg        time.Time
	projectSortMsg struct {
		id  task.ID
		key task.SortKey
	}
)

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.load(true), tick(a.clock))
}

func settingKey(l *view.List) string { return "sort." + l.Spec.Key }

// load reads everything the views need. withSorts also reads each fixed
// view's saved sort (decision G3.3b).
func (a *App) load(withSorts bool) tea.Cmd {
	s := a.store
	var keys []string
	if withSorts {
		for _, l := range a.lists {
			keys = append(keys, l.Spec.Key)
		}
	}
	return func() tea.Msg {
		ctx := context.Background()
		tasks, err := s.ListTasks(ctx)
		if err != nil {
			return errMsg{err}
		}
		ps, err := s.ListProjects(ctx)
		if err != nil {
			return errMsg{err}
		}
		msg := loadedMsg{tasks: tasks, projects: ps, sorts: map[string]task.SortKey{}}
		for _, k := range keys {
			if key, ok, err := savedSort(ctx, s, k); err != nil {
				return errMsg{err}
			} else if ok {
				msg.sorts[k] = key
			}
		}
		return msg
	}
}

func savedSort(ctx context.Context, s store.Store, viewKey string) (task.SortKey, bool, error) {
	v, err := s.GetSetting(ctx, "sort."+viewKey)
	if errors.Is(err, store.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	key, ok := task.ParseSortKey(v)
	return key, ok, nil
}

// tick fires at the start of the next minute, for the header clock and for
// day rollover (Today's contents and G3.1's end-of-day rule).
func tick(now time.Time) tea.Cmd {
	next := now.Truncate(time.Minute).Add(time.Minute)
	return tea.Tick(next.Sub(now), func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a *App) allLists() []*view.List {
	out := make([]*view.List, 0, len(a.lists)+len(a.projectView))
	for _, l := range a.lists {
		out = append(out, l)
	}
	for _, l := range a.projectView {
		out = append(out, l)
	}
	if a.search != nil {
		out = append(out, a.search.list)
	}
	return out
}

func (a *App) refresh() {
	for _, l := range a.allLists() {
		l.SetTasks(a.tasks, a.clock)
	}
}

// bodyHeight is the list area: everything but the header, status bar and
// two rows of key hints.
func (a *App) bodyHeight() int { return max(a.height-4, 0) }

func (a *App) push(d dialog) { a.dialogs = append(a.dialogs, d) }

func (a *App) pop() {
	if len(a.dialogs) > 0 {
		a.dialogs = a.dialogs[:len(a.dialogs)-1]
	}
}

func (a *App) topDialog() dialog {
	if len(a.dialogs) == 0 {
		return nil
	}
	return a.dialogs[len(a.dialogs)-1]
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		for _, l := range a.allLists() {
			l.SetHeight(a.bodyHeight())
		}
	case loadedMsg:
		a.applyLoad(msg)
	case reloadMsg:
		return a, a.load(false)
	case doneMsg:
		a.message = msg.notice
		return a, a.load(false)
	case errMsg:
		a.message = "Error: " + msg.err.Error()
	case tickMsg:
		a.clock = a.now()
		a.refresh()
		return a, tick(a.clock)
	case openProjectMsg:
		return a, a.openProject(msg.id)
	case projectSortMsg:
		if l, ok := a.projectView[msg.id]; ok {
			l.SetSort(msg.key)
		}
	case projectCreatedMsg:
		a.pop() // the Create New Project dialog
		a.message = fmt.Sprintf("Created #%s.", msg.p.Name)
		a.selectAfterLoad = msg.p.ID
		return a, a.load(false)
	case createFailedMsg:
		if c, ok := a.topDialog().(*createProjectDialog); ok {
			c.err = msg.text
		}
	case tea.KeyPressMsg:
		return a, a.handleKey(msg)
	case tea.KeyboardEnhancementsMsg:
		a.keyDisambiguation = msg.SupportsKeyDisambiguation()
	case taskAddedMsg:
		a.pop() // the Add Task dialog
		a.message = fmt.Sprintf("Added %s.", quote(a.theme, msg.t.Name))
		if msg.newProject != "" {
			a.message += fmt.Sprintf(" Created #%s.", msg.newProject)
		}
		a.selectTaskAfterLoad = msg.t.ID
		return a, a.load(false)
	case addFailedMsg:
		if d, ok := a.topDialog().(*addTaskDialog); ok {
			d.err = msg.text
		}
	case tea.PasteMsg:
		if p, ok := a.topDialog().(interface{ paste(tea.PasteMsg) }); ok {
			p.paste(msg)
		} else if a.searching() && a.search.input.Focused() && a.taskView == nil {
			a.search.input.Update(msg)
			a.updateSearch()
		}
	}
	return a, nil
}

func (a *App) applyLoad(msg loadedMsg) {
	a.tasks, a.projectList, a.loaded = msg.tasks, msg.projects, true
	a.projects = make(map[task.ID]string, len(msg.projects))
	active := map[task.ID]task.Project{}
	for _, p := range msg.projects {
		a.projects[p.ID] = p.Name
		if !p.Archived() {
			active[p.ID] = p
		}
	}
	for _, l := range a.lists {
		if k, ok := msg.sorts[l.Spec.Key]; ok {
			l.Sort = k
		}
	}
	// Drop project views whose project is gone or archived; leave them for
	// Today if they were showing.
	for id, l := range a.projectView {
		p, ok := active[id]
		if !ok {
			delete(a.projectView, id)
			if a.current == l {
				a.current = a.lists[view.Today]
			}
			continue
		}
		l.Spec = view.ProjectSpec(p) // picks up a renamed project
	}
	a.refresh()
	if a.selectAfterLoad != "" {
		switch d := a.topDialog().(type) {
		case *gotoDialog:
			d.selectProject(a, a.selectAfterLoad)
		case *editProjectDialog: // G6.8
			d.selectProject(a, a.selectAfterLoad)
		}
		a.selectAfterLoad = ""
	}
	a.syncTaskView()
	if a.selectTaskAfterLoad != "" {
		a.current.Select(a.selectTaskAfterLoad) // G5.4: if it's in this view
		a.selectTaskAfterLoad = ""
	}
}

// openProject switches to a project's list, creating it on first use and
// restoring its saved sort.
func (a *App) openProject(id task.ID) tea.Cmd {
	for _, p := range a.projectList {
		if p.ID != id || p.Archived() {
			continue
		}
		if l, ok := a.projectView[id]; ok {
			a.current = l
			return nil
		}
		l := view.NewList(view.ProjectSpec(p))
		l.SetHeight(a.bodyHeight())
		l.SetTasks(a.tasks, a.clock)
		a.projectView[id] = l
		a.current = l
		s, key := a.store, l.Spec.Key
		return func() tea.Msg {
			k, ok, err := savedSort(context.Background(), s, key)
			if err != nil {
				return errMsg{err}
			}
			if !ok {
				return nil
			}
			return projectSortMsg{id: id, key: k}
		}
	}
	return nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if isQuit(msg) {
		return tea.Quit
	}
	a.message = ""

	if d := a.topDialog(); d != nil {
		done, cmd := d.update(a, msg)
		if done {
			// Remove this dialog, wherever it is now (it may have pushed one).
			for i := len(a.dialogs) - 1; i >= 0; i-- {
				if a.dialogs[i] == d {
					a.dialogs = append(a.dialogs[:i], a.dialogs[i+1:]...)
					break
				}
			}
		}
		return cmd
	}

	key := msg.String()
	if a.taskView != nil {
		return a.handleTaskViewKey(key)
	}
	if a.searching() && a.search.input.Focused() {
		return a.handleSearchInput(msg)
	}
	if a.pendingG {
		a.pendingG = false
		if key == "g" {
			a.current.Top()
		} else if b, ok := lookup(key, gotoGroup); ok {
			return b.act(a, a.current, key)
		}
		return nil
	}
	if a.searching() && (key == "esc" || key == "/") {
		a.search.input.Focus() // G7.2a: back to the input
		return nil
	}
	if key == "g" {
		a.pendingG = true
		return nil
	}
	if b, ok := lookup(key, listGroup, taskGroup, generalGroup); ok && b.act != nil {
		return b.act(a, a.current, key)
	}
	return nil
}

// searching reports whether the Search view is showing.
func (a *App) searching() bool { return a.search != nil && a.current == a.search.list }

// isQuit reports whether msg is a quit key. `q` is "add task" in the designs,
// so quitting is ctrl+c or ctrl+q (decision G0.4 in PLAN.md).
func isQuit(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		return true
	}
	return false
}

func (a *App) toggleDone() tea.Cmd {
	t, ok := a.current.Selected()
	if !ok {
		return nil
	}
	if !t.Done() {
		a.selectTaskAfterLoad = nextOpen(a.current.Rows(), t.ID) // G7.8
	}
	return a.setDone(t)
}

// nextOpen is the open task to select after completing id: the first open
// one below it, else the nearest open one above, else id itself (G7.8).
func nextOpen(rows []task.Task, id task.ID) task.ID {
	i := slices.IndexFunc(rows, func(t task.Task) bool { return t.ID == id })
	for j := i + 1; j < len(rows); j++ {
		if !rows[j].Done() {
			return rows[j].ID
		}
	}
	for j := i - 1; j >= 0; j-- {
		if !rows[j].Done() {
			return rows[j].ID
		}
	}
	return id
}

// setDone completes an open task or reopens a completed one.
func (a *App) setDone(t task.Task) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if _, err := s.SetDone(context.Background(), t.ID, !t.Done()); err != nil {
			return errMsg{fmt.Errorf("updating %q: %w", t.Name, err)}
		}
		return reloadMsg{}
	}
}

func (a *App) saveSort(l *view.List, key task.SortKey) tea.Cmd {
	s, name := a.store, settingKey(l)
	return func() tea.Msg {
		if err := s.SetSetting(context.Background(), name, key.String()); err != nil {
			return errMsg{fmt.Errorf("saving sort: %w", err)}
		}
		return nil
	}
}

// View implements tea.Model.
func (a *App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.WindowTitle = "taskmoth"
	return v
}

func (a *App) render() string {
	t := a.theme
	switch {
	case a.width == 0 || a.height == 0:
		return "" // size not known yet
	case a.width < widget.MinWidth || a.height < widget.MinHeight:
		return widget.TooSmall(t, a.width, a.height)
	}
	d := a.topDialog()
	if d == nil {
		return a.renderView(t)
	}
	return widget.Overlay(a.renderView(t.Dimmed()), d.view(a, t), a.width, a.height)
}

func (a *App) renderView(t theme.Theme) string {
	if a.taskView != nil {
		return a.renderTaskView(t)
	}
	l := a.current
	lines := []string{widget.Header(t, a.width, l.Spec.Title, view.Clock(t, a.clock))}
	if a.loaded {
		lines = append(lines, l.Body(t, a.width, a.projects, a.clock)...)
	} else {
		for range a.bodyHeight() {
			lines = append(lines, "")
		}
	}
	pending := ""
	if a.pendingG {
		pending = "g" + t.G.Ellipsis
	}
	if a.searching() && a.message == "" {
		lines = append(lines, a.searchBar(t)) // G7: the input replaces the status bar
	} else {
		lines = append(lines, l.StatusBar(t, a.width, pending, a.message))
	}
	lines = append(lines, widget.KeyHints(t, a.width, listHints(t.G, l.Spec.Kind)...)...)
	return widget.Screen(t, a.width, a.height, lines)
}
