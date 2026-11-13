package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// dialogWidth is the width of every dialog (Figma: 720px).
const dialogWidth = 60

// dialog is a modal drawn over the dimmed view (Figma 32:13454). The top
// dialog on the App's stack receives every key.
type dialog interface {
	// update handles a key. done closes the dialog.
	update(a *App, msg tea.KeyPressMsg) (done bool, cmd tea.Cmd)
	view(a *App, t theme.Theme) string
}

func help(t theme.Theme, s string) string { return t.Style(t.Fade(t.C.Text), t.C.Bg).Render(s) }

// --- confirmation (Figma 40:14869, decision G4.9) ---

type confirmDialog struct {
	message string
	action  string // destructive button label: "archive" or "delete"
	focus   int    // 0 = cancel (starts selected), 1 = action
	run     tea.Cmd
}

func (d *confirmDialog) update(_ *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "shift+tab":
		d.focus = 0
	case "right", "l", "tab":
		d.focus = 1
	case "enter":
		if d.focus == 1 {
			return true, d.run
		}
		return true, nil
	case "esc":
		return true, nil
	}
	return false, nil
}

func (d *confirmDialog) view(_ *App, t theme.Theme) string {
	bw := w.FrameBodyWidth(dialogWidth, false)
	base := t.Base()
	buttons := w.Button(t, "cancel", w.ButtonDefault, d.focus == 0, false) + base.Render("  ") +
		w.Button(t, d.action, w.ButtonDestructive, d.focus == 1, false)
	body := []string{
		base.Render(w.Truncate(t, d.message, bw)),
		"",
		buttons,
		"",
		w.HintRow(t, bw, []w.Hint{{Key: t.G.Left + " / " + t.G.Right, Label: "select"},
			{Key: "enter", Label: d.action}, {Key: "esc", Label: "cancel"}}),
	}
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Focused: true}, body)
}

// --- Goto Project (Figma 40:14667, decision G4.7) ---

// gotoVisible is how many project lines the dialog shows (Figma: 8 rows).
const gotoVisible = 8

type gotoDialog struct {
	sel int // index into active followed by archived
	top int // first visible line
}

func (a *App) splitProjects() (active, archived []task.Project) {
	for _, p := range a.projectList {
		if p.Archived() {
			archived = append(archived, p)
		} else {
			active = append(active, p)
		}
	}
	return active, archived
}

// selectable returns active projects then archived ones.
func (d *gotoDialog) selectable(a *App) (all []task.Project, nActive int) {
	active, archived := a.splitProjects()
	return append(active, archived...), len(active)
}

// lineOf maps a selectable index to its line, skipping the "Archived"
// heading.
func lineOf(i, nActive int) int {
	if i < nActive {
		return i
	}
	return i + 1
}

func (d *gotoDialog) clamp(a *App) {
	all, nActive := d.selectable(a)
	d.sel = max(min(d.sel, len(all)-1), 0)
	line := lineOf(d.sel, nActive)
	if line < d.top {
		d.top = line
	}
	if line >= d.top+gotoVisible {
		d.top = line - gotoVisible + 1
	}
	if d.sel == nActive && d.top == line { // keep the heading visible above the first archived project
		d.top = max(line-1, 0)
	}
}

// selectProject moves the selection to id.
func (d *gotoDialog) selectProject(a *App, id task.ID) {
	all, _ := d.selectable(a)
	for i, p := range all {
		if p.ID == id {
			d.sel = i
		}
	}
	d.clamp(a)
}

func (d *gotoDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	all, _ := d.selectable(a)
	var sel *task.Project
	if len(all) > 0 {
		sel = &all[d.sel]
	}
	switch msg.String() {
	case "esc":
		return true, nil
	case "up", "k":
		d.sel--
	case "down", "j":
		d.sel++
	case "n":
		a.push(newCreateProjectDialog())
	case "enter":
		if sel != nil && !sel.Archived() {
			id := sel.ID
			return true, func() tea.Msg { return openProjectMsg{id} }
		}
	case "x":
		if sel == nil {
			break
		}
		p := *sel
		n := a.projectTaskCount(p.ID, p.Archived())
		if !p.Archived() {
			msg := fmt.Sprintf("Archive #%s?", p.Name)
			if n > 0 {
				msg = fmt.Sprintf("Archive #%s and its %d %s?", p.Name, n, plural(n, "task", "tasks"))
			}
			a.push(&confirmDialog{message: msg, action: "archive", run: a.archiveProject(p)})
		} else {
			a.push(&confirmDialog{
				message: fmt.Sprintf("Delete #%s and its %d archived %s forever?", p.Name, n, plural(n, "task", "tasks")),
				action:  "delete", run: a.deleteProject(p),
			})
		}
	case "u":
		if sel != nil && sel.Archived() {
			return false, a.unarchiveProject(*sel)
		}
	}
	d.clamp(a)
	return false, nil
}

func (d *gotoDialog) view(a *App, t theme.Theme) string {
	d.clamp(a)
	bw := w.FrameBodyWidth(dialogWidth, true)
	base := t.Base()
	all, nActive := d.selectable(a)

	var lines []string
	for i, p := range all {
		if i == nActive {
			lines = append(lines, w.Blank(base, w.MarkerWidth)+help(t, "Archived"))
		}
		name := "#" + p.Name
		switch {
		case i == d.sel:
			s := t.Style(t.C.SelectionFg, t.C.SelectionBg)
			lines = append(lines, w.Fit(w.Marker(t, true, t.C.SelectionBg)+s.Render(w.Truncate(t, name, bw-2)), bw, s))
		case p.Archived():
			lines = append(lines, w.Marker(t, false, t.C.Bg)+help(t, w.Truncate(t, name, bw-2)))
		default:
			lines = append(lines, w.Marker(t, false, t.C.Bg)+base.Render(w.Truncate(t, name, bw-2)))
		}
	}

	body := []string{"", help(t, "Select a project to open its list:"), ""}
	if len(all) == 0 {
		body[1] = help(t, "No projects yet.")
	}
	for i := 0; i < gotoVisible; i++ {
		l := ""
		if d.top+i < len(lines) {
			l = lines[d.top+i]
		}
		body = append(body, l)
	}

	nav := w.Hint{Key: t.G.Up + " / " + t.G.Down, Label: "select"}
	var hints []w.Hint
	switch {
	case len(all) == 0:
		hints = []w.Hint{{Key: "n", Label: "new"}, {Key: "esc", Label: "cancel"}}
	case all[d.sel].Archived():
		hints = []w.Hint{nav, {Key: "u", Label: "unarchive"}, {Key: "x", Label: "delete forever"}, {Key: "esc", Label: "cancel"}}
	default:
		hints = []w.Hint{nav, {Key: "enter", Label: "select"}, {Key: "esc", Label: "cancel"}, {Key: "n", Label: "new"}, {Key: "x", Label: "archive"}}
	}
	body = append(body, "", w.HintRow(t, bw, hints))

	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: "Goto Project", Focused: true,
		Scroll: &w.Scroll{Total: len(lines), Visible: gotoVisible, Offset: d.top}}, body)
}

// --- Create New Project (Figma 40:14812) ---

type createProjectDialog struct {
	input *w.Input
	err   string
}

func newCreateProjectDialog() *createProjectDialog {
	in := &w.Input{Placeholder: "yourprojectname"}
	in.SetSize(w.FrameBodyWidth(dialogWidth, false), 1)
	in.Focus()
	return &createProjectDialog{input: in}
}

func (d *createProjectDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return true, nil
	case "enter":
		return false, a.createProject(d.input.Value())
	}
	if d.input.Update(msg) {
		d.err = ""
	}
	return false, nil
}

func (d *createProjectDialog) paste(msg tea.PasteMsg) { d.input.Update(msg) }

func (d *createProjectDialog) view(_ *App, t theme.Theme) string {
	bw := w.FrameBodyWidth(dialogWidth, false)
	line := help(t, "Enter project name without #:")
	if d.err != "" {
		line = t.Style(t.C.TextError, t.C.Bg).Render(w.Truncate(t, d.err, bw))
	}
	body := []string{"", line, "", d.input.View(t, false), "",
		w.HintRow(t, bw, []w.Hint{{Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"}})}
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: "Create New Project", Focused: true}, body)
}

// createError explains why a project couldn't be created.
func createError(name string, err error) string {
	name = task.NormalizeProjectName(name)
	switch {
	case name == "":
		return "Enter a name."
	case errors.Is(err, store.ErrArchivedProjectName):
		return fmt.Sprintf("#%s is archived; unarchive it instead.", name)
	case errors.Is(err, store.ErrDuplicateProject):
		return fmt.Sprintf("#%s already exists.", name)
	case errors.Is(err, task.ErrInvalid):
		return "Use only a-z, 0-9, - and _."
	}
	return "Error: " + err.Error()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// --- store operations started from dialogs and keys ---

type (
	openProjectMsg    struct{ id task.ID }
	projectCreatedMsg struct{ p task.Project }
	createFailedMsg   struct{ text string }
	// doneMsg reports a finished change: show notice (if any) and reload.
	doneMsg struct{ notice string }
)

func (a *App) createProject(name string) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		p, err := s.CreateProject(context.Background(), name)
		if err != nil {
			return createFailedMsg{createError(name, err)}
		}
		return projectCreatedMsg{p}
	}
}

func (a *App) archiveProject(p task.Project) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if _, err := s.ArchiveProject(context.Background(), p.ID); err != nil {
			return errMsg{fmt.Errorf("archiving #%s: %w", p.Name, err)}
		}
		return doneMsg{notice: fmt.Sprintf("Archived #%s.", p.Name)}
	}
}

func (a *App) unarchiveProject(p task.Project) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if _, err := s.UnarchiveProject(context.Background(), p.ID); err != nil {
			return errMsg{fmt.Errorf("unarchiving #%s: %w", p.Name, err)}
		}
		return doneMsg{notice: fmt.Sprintf("Unarchived #%s.", p.Name)}
	}
}

func (a *App) deleteProject(p task.Project) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if err := s.DeleteProject(context.Background(), p.ID); err != nil {
			return errMsg{fmt.Errorf("deleting #%s: %w", p.Name, err)}
		}
		return doneMsg{notice: fmt.Sprintf("Deleted #%s forever.", p.Name)}
	}
}

func (a *App) archiveTask(t task.Task) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if _, err := s.Archive(context.Background(), t.ID); err != nil {
			return errMsg{fmt.Errorf("archiving %q: %w", t.Name, err)}
		}
		return doneMsg{}
	}
}

func (a *App) unarchiveTask(t task.Task) tea.Cmd {
	s := a.store
	project := a.projects[t.ProjectID]
	return func() tea.Msg {
		got, err := s.Unarchive(context.Background(), t.ID)
		if err != nil {
			return errMsg{fmt.Errorf("unarchiving %q: %w", t.Name, err)}
		}
		if t.ProjectID != "" && got.ProjectID == "" { // G4.4a
			return doneMsg{notice: fmt.Sprintf("Moved to Inbox (#%s is archived).", project)}
		}
		return doneMsg{}
	}
}

func (a *App) deleteTask(t task.Task) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		if err := s.DeleteForever(context.Background(), t.ID); err != nil {
			return errMsg{fmt.Errorf("deleting %q: %w", t.Name, err)}
		}
		return doneMsg{}
	}
}

// projectTaskCount counts a project's unarchived tasks, or for an archived
// project its archived tasks.
func (a *App) projectTaskCount(id task.ID, archived bool) int {
	n := 0
	for _, t := range a.tasks {
		if t.ProjectID == id && t.Archived() == archived {
			n++
		}
	}
	return n
}

// quote shortens a task name for a dialog message.
func quote(t theme.Theme, name string) string {
	return `"` + w.Truncate(t, strings.TrimSpace(name), 30) + `"`
}
