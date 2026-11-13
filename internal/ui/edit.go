package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/nlp"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// The edit dialogs (Phase 6): one per task field. Each works on the task
// selected in a list or open in the Single Task view (G6.6) and saves with
// UpdateTask.

// editKeys maps the "Selected Tasks" keys to the dialog they open (G6.6).
var editKeys = map[string]func(a *App, t task.Task) dialog{
	"n": newNameDialog, "d": newDescriptionDialog, "p": newPriorityDialog, "v": newEditProjectDialog,
	"r": func(a *App, t task.Task) dialog { return newDateDialog(a, t, false) },
	"t": func(a *App, t task.Task) dialog { return newDateDialog(a, t, true) },
	"y": newDurationDialog,
}

// saveTask updates t. If the result has a start date after its due date,
// the status bar warns (G6.5).
func (a *App) saveTask(t task.Task) tea.Cmd {
	s := a.store
	return func() tea.Msg {
		got, err := s.UpdateTask(context.Background(), t)
		if err != nil {
			return errMsg{fmt.Errorf("saving %q: %w", t.Name, err)}
		}
		if !got.Start.IsZero() && !got.Due.IsZero() && got.Start.After(got.Due) {
			return doneMsg{notice: "Start date is after the due date."}
		}
		return doneMsg{}
	}
}

// --- text fields: name, description, dates, duration ---

// textEditDialog edits one text field. parse turns the input into the
// updated task or an error shown in red in place of the help (G6.5).
type textEditDialog struct {
	title string
	help  []string
	input *w.Input
	field bool // shaded multi-row field (name, description) vs a plain line
	t     task.Task
	parse func(a *App, value string) (task.Task, string)
	err   string
}

func (d *textEditDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return true, nil
	case "shift+enter", "alt+enter", "ctrl+j": // G6.2: newline (description only)
		if d.input.Multiline {
			d.input.InsertNewline()
			d.err = ""
		}
		return false, nil
	case "enter":
		t, errText := d.parse(a, d.input.Value())
		if errText != "" {
			d.err = errText
			return false, nil
		}
		return true, a.saveTask(t)
	}
	if d.input.Update(msg) {
		d.err = ""
	}
	return false, nil
}

func (d *textEditDialog) paste(msg tea.PasteMsg) { d.input.Update(msg) }

func (d *textEditDialog) view(a *App, t theme.Theme) string {
	bw := w.FrameBodyWidth(dialogWidth, d.field)
	body := []string{""}
	for i, h := range d.help {
		if i == 0 && d.err != "" {
			body = append(body, t.Style(t.C.TextError, t.C.Bg).Render(w.Truncate(t, d.err, bw)))
			continue
		}
		body = append(body, help(t, h))
	}
	body = append(body, "")
	body = append(body, w.Lines(d.input.View(t, d.field))...)
	hints := []w.Hint{{Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"}}
	if d.input.Multiline {
		nl := w.Hint{Key: "ctrl+j", Label: "newline"}
		if a.keyDisambiguation { // G6.2
			nl = w.Hint{Key: "shift + enter", Label: "newline"}
		}
		hints = append([]w.Hint{nl}, hints...)
	}
	body = append(body, "", w.HintRow(t, bw, hints))
	var scroll *w.Scroll
	if d.field { // Name and Description frames have a scrollbar (15:5973, 40:16847)
		scroll = &w.Scroll{}
	}
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: d.title, Focused: true, Scroll: scroll}, body)
}

func newTextEdit(title string, help []string, value string, rows int, field, multiline bool, t task.Task,
	parse func(a *App, value string) (task.Task, string)) *textEditDialog {
	in := &w.Input{Multiline: multiline}
	in.SetSize(w.FrameBodyWidth(dialogWidth, field), rows)
	in.SetValue(value)
	in.Focus()
	return &textEditDialog{title: title, help: help, input: in, field: field, t: t, parse: parse}
}

// newNameDialog is Edit Name (40:16847): a 3-row field, single-line value
// (G2.1d), so no newline hint (G6.2).
func newNameDialog(_ *App, t task.Task) dialog {
	return newTextEdit("Edit Name", []string{"Edit task name:"}, t.Name, 3, true, false, t,
		func(_ *App, v string) (task.Task, string) {
			if strings.TrimSpace(v) == "" {
				return t, "Enter a task name."
			}
			t.Name = strings.TrimSpace(v)
			return t, ""
		})
}

// newDescriptionDialog is Edit Description (15:5973): a 5-row multi-line
// field. Saving it empty clears the description (G6.3).
func newDescriptionDialog(_ *App, t task.Task) dialog {
	return newTextEdit("Edit Description", []string{"Edit task description:"}, t.Description, 5, true, true, t,
		func(_ *App, v string) (task.Task, string) {
			t.Description = strings.TrimRight(v, " \n")
			return t, ""
		})
}

// dateHelp is the date dialogs' help (40:15310, 40:15226).
var dateHelp = []string{
	"Enter date in YYYY-MM-DD format or use natural language.",
	"Examples: 2026-11-02, today, in 3 days, next monday",
}

// newDateDialog is Edit Start date (40:15310) or Edit Due date (40:15226).
// Empty input or "no date" clears it (G6.3).
func newDateDialog(_ *App, t task.Task, due bool) dialog {
	title, cur := "Edit Start date", t.Start
	if due {
		title, cur = "Edit Due date", t.Due
	}
	return newTextEdit(title, dateHelp, cur.String(), 1, false, false, t,
		func(a *App, v string) (task.Task, string) {
			var d task.Date
			if strings.TrimSpace(v) != "" {
				var ok bool
				if d, ok = nlp.ParseDate(v, task.DateOf(a.clock)); !ok {
					return t, "Can't read that date. Try 2026-11-20 or next monday."
				}
			}
			if due {
				t.Due = d
			} else {
				t.Start = d
			}
			return t, ""
		})
}

// durationText formats minutes for editing: 30m, 2h, 1h30m.
func durationText(m int) string {
	switch {
	case m <= 0:
		return ""
	case m < 60:
		return fmt.Sprintf("%dm", m)
	case m%60 == 0:
		return fmt.Sprintf("%dh", m/60)
	}
	return fmt.Sprintf("%dh%dm", m/60, m%60)
}

// newDurationDialog is Edit Duration (40:16639), opening with the current
// duration rather than the design's "tomorrow" sample (G6.4).
func newDurationDialog(_ *App, t task.Task) dialog {
	text := "Enter duration in minutes, decimal hours or use natural language. Examples: 30m, 1.5h, 2 hours, 10 minutes"
	helpLines := strings.Split(ansi.Wordwrap(text, w.FrameBodyWidth(dialogWidth, false), ""), "\n")
	return newTextEdit("Edit Duration", helpLines, durationText(t.DurationMin), 1, false, false, t,
		func(_ *App, v string) (task.Task, string) {
			if strings.TrimSpace(v) == "" {
				t.DurationMin = 0
				return t, ""
			}
			m, ok := nlp.ParseDuration(v)
			if !ok {
				return t, "Can't read that duration. Try 30m or 1.5h."
			}
			t.DurationMin = m
			return t, ""
		})
}

// --- priority (15:6168) ---

type priorityDialog struct {
	t   task.Task
	sel int // 1–4
}

func newPriorityDialog(_ *App, t task.Task) dialog {
	return &priorityDialog{t: t, sel: t.Priority}
}

func (d *priorityDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k := msg.String(); k {
	case "esc":
		return true, nil
	case "left", "h":
		d.sel = max(d.sel-1, 1)
	case "right", "l":
		d.sel = min(d.sel+1, 4)
	case "1", "2", "3", "4":
		d.sel = int(k[0] - '0')
		fallthrough
	case "enter":
		t := d.t
		t.Priority = d.sel
		return true, a.saveTask(t)
	}
	return false, nil
}

func (d *priorityDialog) view(_ *App, t theme.Theme) string {
	bw := w.FrameBodyWidth(dialogWidth, false)
	base := t.Base()
	var row string
	for p := 1; p <= 4; p++ {
		if p > 1 {
			row += base.Render(" ")
		}
		row += w.Marker(t, p == d.sel, t.C.Bg) + w.PriorityMark(t, p, false, t.C.Bg) // G6.9: a marker shows the selection
	}
	body := []string{""}
	for _, l := range strings.Split(ansi.Wordwrap("Select a priority from the below options or select and save with the 1-4 key:", bw, ""), "\n") {
		body = append(body, help(t, l))
	}
	body = append(body, "", row, "", w.HintRow(t, bw, []w.Hint{
		{Key: t.G.Left + " / " + t.G.Right, Label: "select"}, {Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"},
	}))
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: "Edit Priority", Focused: true}, body)
}

// --- project (28:6404) ---

// editProjectDialog lists "Inbox" (no project, G6.3) and the active
// projects A–Z.
type editProjectDialog struct {
	t   task.Task
	sel int // 0 = Inbox, i = active[i-1]
	top int
}

func newEditProjectDialog(a *App, t task.Task) dialog {
	d := &editProjectDialog{t: t}
	d.selectProject(a, t.ProjectID)
	return d
}

func (d *editProjectDialog) options(a *App) []task.Project {
	active, _ := a.splitProjects()
	return append([]task.Project{{Name: "Inbox"}}, active...)
}

func (d *editProjectDialog) selectProject(a *App, id task.ID) {
	for i, p := range d.options(a) {
		if p.ID == id {
			d.sel = i
		}
	}
	d.clamp(a)
}

func (d *editProjectDialog) clamp(a *App) {
	n := len(d.options(a))
	d.sel = max(min(d.sel, n-1), 0)
	if d.sel < d.top {
		d.top = d.sel
	}
	if d.sel >= d.top+gotoVisible {
		d.top = d.sel - gotoVisible + 1
	}
}

func (d *editProjectDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return true, nil
	case "up", "k":
		d.sel--
	case "down", "j":
		d.sel++
	case "n": // G6.8
		a.push(newCreateProjectDialog())
	case "enter":
		t := d.t
		t.ProjectID = d.options(a)[d.sel].ID
		return true, a.saveTask(t)
	}
	d.clamp(a)
	return false, nil
}

func (d *editProjectDialog) view(a *App, t theme.Theme) string {
	d.clamp(a)
	bw := w.FrameBodyWidth(dialogWidth, true)
	base := t.Base()
	opts := d.options(a)
	body := []string{"", help(t, "Select a project:"), ""}
	for i := 0; i < gotoVisible; i++ {
		j := d.top + i
		if j >= len(opts) {
			body = append(body, "")
			continue
		}
		name := "#" + opts[j].Name
		if j == 0 {
			name = "Inbox"
		}
		switch j {
		case d.sel:
			s := t.Style(t.C.SelectionFg, t.C.SelectionBg)
			body = append(body, w.Fit(w.Marker(t, true, t.C.SelectionBg)+s.Render(w.Truncate(t, name, bw-2)), bw, s))
		case 0:
			body = append(body, w.Marker(t, false, t.C.Bg)+help(t, name))
		default:
			body = append(body, w.Marker(t, false, t.C.Bg)+base.Render(w.Truncate(t, name, bw-2)))
		}
	}
	body = append(body, "", w.HintRow(t, bw, []w.Hint{
		{Key: t.G.Up + " / " + t.G.Down, Label: "select"}, {Key: "enter", Label: "save"},
		{Key: "esc", Label: "cancel"}, {Key: "n", Label: "new"},
	}))
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: "Edit Project", Focused: true,
		Scroll: &w.Scroll{Total: len(opts), Visible: gotoVisible, Offset: d.top}}, body)
}
