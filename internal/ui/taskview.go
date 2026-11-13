package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// taskView is the Single Task view (Figma 29:7123), opened with `e`. It
// replaces the list until `esc`; the list keeps its selection.
type taskView struct {
	id       task.ID
	readOnly bool // opened from the Archive (G6.7)
	top      int  // first visible line
	visible  int  // body lines on screen, from the last render
}

func (a *App) openTaskView() {
	t, ok := a.current.Selected()
	if !ok {
		return
	}
	a.taskView = &taskView{id: t.ID, readOnly: a.current.Spec.Kind == view.Archive}
}

// viewedTask is the task the Single Task view shows.
func (a *App) viewedTask() (task.Task, bool) {
	for _, t := range a.tasks {
		if t.ID == a.taskView.id {
			return t, true
		}
	}
	return task.Task{}, false
}

// syncTaskView closes the view once its task is gone or has moved in or out
// of the archive (archived, unarchived or deleted forever).
func (a *App) syncTaskView() {
	if a.taskView == nil {
		return
	}
	t, ok := a.viewedTask()
	if !ok || t.Archived() != a.taskView.readOnly {
		a.taskView = nil
	}
}

func (a *App) handleTaskViewKey(key string) tea.Cmd {
	tv := a.taskView
	t, ok := a.viewedTask()
	if !ok {
		a.taskView = nil
		return nil
	}
	switch key {
	case "esc":
		a.taskView = nil
	case "up", "k":
		tv.top--
	case "down", "j":
		tv.top++
	case "pgup":
		tv.top -= max(tv.visible-1, 1)
	case "pgdown":
		tv.top += max(tv.visible-1, 1)
	case "home":
		tv.top = 0
	case "end", "G":
		tv.top = 1 << 20
	case "space":
		if !tv.readOnly {
			if !t.Done() {
				a.taskView = nil // G6.13: completing returns to the list
			}
			return a.setDone(t)
		}
	case "x":
		if tv.readOnly {
			a.push(&confirmDialog{message: "Delete " + quote(a.theme, t.Name) + " forever?", action: "delete", run: a.deleteTask(t)})
		} else {
			a.push(&confirmDialog{message: "Archive " + quote(a.theme, t.Name) + "?", action: "archive", run: a.archiveTask(t)})
		}
	case "u":
		if tv.readOnly {
			return a.unarchiveTask(t)
		}
	default:
		if open, ok := editKeys[key]; ok && !tv.readOnly {
			a.push(open(a, t))
		}
	}
	return nil
}

// taskViewHints are the view's hint rows (29:7123), with `x` meaning
// archive (G6.4); read-only for archived tasks (G6.7).
func (a *App) taskViewHints(t theme.Theme) [][]w.Hint {
	nav := w.Hint{Key: t.G.Up + " / " + t.G.Down, Label: "scroll"}
	if a.taskView.readOnly {
		return [][]w.Hint{{nav, {Key: "u", Label: "unarchive"}, {Key: "x", Label: "delete forever"}, {Key: "esc", Label: "return to list"}}, {}}
	}
	return [][]w.Hint{
		{nav, {Key: "space", Label: "done"}, {Key: "esc", Label: "return to list"}, {Key: "x", Label: "archive"}, {Key: "n", Label: "name"}},
		{{Key: "p", Label: "priority"}, {Key: "v", Label: "project"}, {Key: "r", Label: "start date"},
			{Key: "t", Label: "due date"}, {Key: "y", Label: "duration"}, {Key: "d", Label: "description"}},
	}
}

func (a *App) renderTaskView(t theme.Theme) string {
	tk, _ := a.viewedTask()
	tv := a.taskView
	lines := []string{w.Header(t, a.width, "Task", view.Clock(t, a.clock))}

	panelH := a.height - 3 // header and two hint rows
	if a.message != "" {
		panelH-- // G6.10: a status bar only while there's a message
	}
	body := a.taskBody(t, tk, w.FrameBodyWidth(a.width, true))
	visible := max(panelH-2, 1)
	tv.visible = visible
	tv.top = max(min(tv.top, len(body)-visible), 0)
	shown := make([]string, visible)
	for i := range shown {
		if tv.top+i < len(body) {
			shown[i] = body[tv.top+i]
		}
	}
	lines = append(lines, w.Lines(w.Frame(t, w.FrameOptions{Width: a.width,
		Scroll: &w.Scroll{Total: len(body), Visible: visible, Offset: tv.top}}, shown))...)
	if a.message != "" {
		lines = append(lines, w.StatusBar(t, a.width, "Task", theme.ToneInfo, a.message, ""))
	}
	lines = append(lines, w.KeyHints(t, a.width, a.taskViewHints(t)...)...)
	return w.Screen(t, a.width, a.height, lines)
}

// wrap word-wraps text to width, hard-breaking words longer than a line.
// Control characters are cleaned first (stored text may predate CleanText).
func wrap(text string, width int) []string {
	text = w.CleanText(text, true)
	return strings.Split(ansi.Hardwrap(ansi.Wordwrap(text, width, ""), width, true), "\n")
}

// taskBody lays out the task (29:7123): name, project, badges, then the
// description.
func (a *App) taskBody(t theme.Theme, tk task.Task, bw int) []string {
	base := t.Base()
	var out []string
	for _, l := range wrap(tk.Name, bw) {
		out = append(out, base.Bold(true).Render(l))
	}
	out = append(out, "")
	if name, ok := a.projects[tk.ProjectID]; ok {
		out = append(out, base.Render("#"+name))
	} else {
		out = append(out, help(t, "Inbox")) // G6.10
	}
	out = append(out, "")

	// Badge row: priority, due, s:start, ~duration, two cells apart (G6.1).
	parts := []string{w.PriorityMark(t, tk.Priority, false, t.C.Bg)}
	if b := view.DueWordBadge(t, tk, a.clock); b != "" {
		parts = append(parts, b)
	}
	if !tk.Start.IsZero() {
		parts = append(parts, base.Render("s:"+tk.Start.String()))
	}
	if tk.DurationMin > 0 {
		parts = append(parts, base.Render(view.Duration(tk.DurationMin)))
	}
	out = append(out, w.Fit(strings.Join(parts, base.Render("  ")), bw, base))
	if tk.Done() {
		out = append(out, help(t, "Completed "+task.DateOf(tk.CompletedAt.In(a.clock.Location())).String()))
	}
	if tk.Archived() {
		out = append(out, help(t, "Archived "+task.DateOf(tk.ArchivedAt.In(a.clock.Location())).String()))
	}

	out = append(out, "", help(t, "Description"))
	if strings.TrimSpace(tk.Description) == "" {
		msg := "No description. Press d to add one."
		if a.taskView.readOnly {
			msg = "No description."
		}
		out = append(out, help(t, msg))
	} else {
		for _, l := range wrap(tk.Description, bw) {
			out = append(out, base.Render(l))
		}
	}
	return append(out, "")
}
