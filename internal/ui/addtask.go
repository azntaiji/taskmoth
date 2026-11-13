package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/nlp"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// addTaskHelp is the Add Task dialog's help text (Figma 40:16708). From a
// project's list, the last sentence names that project instead (G7.7).
const addTaskHelp = "Add a new task using natural language. Options: # to assign to a project, " +
	"s: to set start date, @ to set due date, ~ to set duration, ! to set priority. "

func (d *addTaskDialog) helpText(a *App) string {
	if name, ok := a.projects[d.defaultProject]; ok && d.defaultProject != "" {
		return addTaskHelp + "Tasks without a # project are added to #" + name + "."
	}
	return addTaskHelp + "Tasks without options are sent to the Inbox."
}

const (
	addTaskInputRows = 4 // Figma: a 96px input
	maxSuggestions   = 4
)

// addTaskDialog is quick add (decisions G5.1–G5.4).
type addTaskDialog struct {
	input          *w.Input
	defaultProject task.ID // the project list it was opened from, if any (G5.4)

	suggestSel int    // highlighted autocomplete suggestion
	dismissed  string // the #word whose suggestions esc hid

	prompt *newProjectPrompt // asking before creating a project (G5.2)
	err    string
}

// newProjectPrompt is "Create #musc? Did you mean #music?".
type newProjectPrompt struct {
	name       string // the unknown project typed
	suggestion string // closest existing project, or ""
}

func newAddTaskDialog(defaultProject task.ID) *addTaskDialog {
	in := &w.Input{}
	in.SetSize(w.FrameBodyWidth(dialogWidth, false), addTaskInputRows)
	in.Focus()
	return &addTaskDialog{input: in, defaultProject: defaultProject}
}

func (d *addTaskDialog) parse(a *App) nlp.Result {
	return nlp.Parse(d.input.Value(), task.DateOf(a.clock))
}

// hashWord returns the "#…" word the cursor is in or just after, with its
// rune offsets, or ok=false.
func (d *addTaskDialog) hashWord() (word string, start, end int, ok bool) {
	rs := []rune(d.input.Value())
	cur := d.input.Cursor()
	start = cur
	for start > 0 && rs[start-1] != ' ' {
		start--
	}
	end = cur
	for end < len(rs) && rs[end] != ' ' {
		end++
	}
	word = string(rs[start:end])
	if !strings.HasPrefix(word, "#") {
		return "", 0, 0, false
	}
	return word, start, end, true
}

// suggestions are the active projects matching the #word being typed.
func (d *addTaskDialog) suggestions(a *App) []task.Project {
	word, _, _, ok := d.hashWord()
	if !ok || word == d.dismissed || d.prompt != nil {
		return nil
	}
	active, _ := a.splitProjects()
	out := rankProjects(task.NormalizeProjectName(word), active)
	if len(out) > maxSuggestions {
		out = out[:maxSuggestions]
	}
	return out
}

func (d *addTaskDialog) update(a *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	sugg := d.suggestions(a)
	key := msg.String()

	if d.prompt != nil {
		switch key {
		case "enter":
			return false, d.submit(a, true)
		case "tab":
			if d.prompt.suggestion != "" {
				d.replaceProject(d.prompt.name, d.prompt.suggestion)
				d.prompt = nil
				return false, d.submit(a, false)
			}
		case "esc":
			d.prompt = nil
		}
		return false, nil
	}

	switch key {
	case "esc":
		if len(sugg) > 0 {
			d.dismissed, _, _, _ = d.hashWord()
			return false, nil
		}
		return true, nil
	case "enter":
		return false, d.submit(a, false)
	case "tab":
		if len(sugg) > 0 {
			_, start, end, _ := d.hashWord()
			rs := []rune(d.input.Value())
			repl := []rune("#" + sugg[min(d.suggestSel, len(sugg)-1)].Name + " ")
			if end < len(rs) && rs[end] == ' ' {
				end++
			}
			next := string(rs[:start]) + string(repl) + string(rs[end:])
			d.input.SetValueCursor(next, start+len(repl))
			d.suggestSel = 0
		}
		return false, nil
	case "up":
		if len(sugg) > 0 {
			d.suggestSel = max(d.suggestSel-1, 0)
			return false, nil
		}
	case "down":
		if len(sugg) > 0 {
			d.suggestSel = min(d.suggestSel+1, len(sugg)-1)
			return false, nil
		}
	case "ctrl+t":
		if s, cur, ok := nlp.ToggleAt(d.input.Value(), d.input.Cursor(), task.DateOf(a.clock)); ok {
			d.input.SetValueCursor(s, cur)
		}
		return false, nil
	}
	if d.input.Update(msg) {
		d.err, d.suggestSel = "", 0
	}
	return false, nil
}

func (d *addTaskDialog) paste(msg tea.PasteMsg) { d.input.Update(msg) }

// replaceProject swaps the #from token for #to in the input.
func (d *addTaskDialog) replaceProject(from, to string) {
	fields := strings.Split(d.input.Value(), " ")
	for i := len(fields) - 1; i >= 0; i-- { // the last one counts (Appendix A)
		if task.NormalizeProjectName(fields[i]) == from && strings.HasPrefix(fields[i], "#") {
			fields[i] = "#" + to
			break
		}
	}
	d.input.SetValue(strings.Join(fields, " "))
}

// submit validates and adds the task. confirmedNew allows creating an
// unknown project (the user answered the G5.2 prompt).
func (d *addTaskDialog) submit(a *App, confirmedNew bool) tea.Cmd {
	r := d.parse(a)
	if strings.TrimSpace(r.Name) == "" {
		d.err = "Enter a task name."
		return nil
	}
	t := task.Task{Name: r.Name, Priority: r.Priority, Start: r.Start, Due: r.Due, DurationMin: r.DurationMin,
		ProjectID: d.defaultProject}
	newProject := ""
	if r.Project != "" {
		t.ProjectID = ""
		p, found := a.projectNamed(r.Project)
		switch {
		case found && p.Archived():
			d.err = fmt.Sprintf("#%s is archived; unarchive it instead.", p.Name)
			return nil
		case found:
			t.ProjectID = p.ID
		case !confirmedNew:
			active, _ := a.splitProjects()
			d.prompt = &newProjectPrompt{name: r.Project, suggestion: closestProject(r.Project, active)}
			return nil
		default:
			newProject = r.Project
		}
	}
	d.prompt = nil
	s := a.store
	return func() tea.Msg {
		ctx := context.Background()
		if newProject != "" {
			p, err := s.CreateProject(ctx, newProject)
			if err != nil {
				return addFailedMsg{createError(newProject, err)}
			}
			t.ProjectID = p.ID
		}
		created, err := s.CreateTask(ctx, t)
		if err != nil {
			return addFailedMsg{"Error: " + err.Error()}
		}
		return taskAddedMsg{t: created, newProject: newProject}
	}
}

type (
	taskAddedMsg struct {
		t          task.Task
		newProject string
	}
	addFailedMsg struct{ text string }
)

func (d *addTaskDialog) view(a *App, t theme.Theme) string {
	bw := w.FrameBodyWidth(dialogWidth, false)
	r := d.parse(a)

	// Highlight what the parser recognised (G5.1a).
	d.input.Highlights = d.input.Highlights[:0]
	for _, s := range r.Spans {
		d.input.Highlights = append(d.input.Highlights, w.Highlight{Start: s.Start, End: s.End, Fg: t.C.TextLink})
	}

	body := []string{""}
	for _, l := range strings.Split(ansi.Wordwrap(d.helpText(a), bw, ""), "\n") {
		body = append(body, help(t, l))
	}
	body = append(body, "")
	body = append(body, w.Lines(d.input.View(t, true))...)

	sugg := d.suggestions(a)
	for i, p := range sugg {
		if i == min(d.suggestSel, len(sugg)-1) {
			s := t.Style(t.C.SelectionFg, t.C.SelectionBg)
			body = append(body, w.Fit(w.Marker(t, true, t.C.SelectionBg)+s.Render("#"+p.Name), bw, s))
		} else {
			body = append(body, w.Marker(t, false, t.C.Bg)+t.Base().Render("#"+p.Name))
		}
	}

	var status []string
	var hints []w.Hint
	previewLines := func() []string {
		var out []string
		for _, l := range strings.Split(ansi.Wordwrap(d.preview(a, t, r), bw, ""), "\n") {
			out = append(out, help(t, w.Truncate(t, l, bw)))
		}
		return out
	}
	switch {
	case d.err != "":
		status = []string{t.Style(t.C.TextError, t.C.Bg).Render(w.Truncate(t, d.err, bw))}
		hints = []w.Hint{{Key: "enter", Label: "add task"}, {Key: "esc", Label: "cancel"}}
	case d.prompt != nil:
		q := fmt.Sprintf("Create #%s?", d.prompt.name)
		hints = []w.Hint{{Key: "enter", Label: "create"}, {Key: "esc", Label: "edit"}}
		if d.prompt.suggestion != "" {
			q += fmt.Sprintf(" Did you mean #%s?", d.prompt.suggestion)
			hints = append([]w.Hint{{Key: "tab", Label: "#" + d.prompt.suggestion}}, hints...)
		}
		status = []string{t.Style(t.C.TextWarning, t.C.Bg).Render(w.Truncate(t, q, bw))}
	case len(sugg) > 0:
		status = previewLines()
		hints = []w.Hint{{Key: "tab", Label: "complete"}, {Key: t.G.Up + " / " + t.G.Down, Label: "select"},
			{Key: "enter", Label: "add task"}, {Key: "esc", Label: "hide list"}}
	default:
		status = previewLines()
		hints = []w.Hint{{Key: "enter", Label: "add task"}, {Key: "esc", Label: "cancel"}, {Key: "ctrl+t", Label: "keep as text"}}
	}
	body = append(body, status...)
	body = append(body, "", w.HintRow(t, bw, hints))
	return w.Frame(t, w.FrameOptions{Width: dialogWidth, Title: "Add Task", Focused: true}, body)
}

// preview is "→ #oneofftasks · due 2026-11-12 · s:2026-11-11 · ~30m · P2"
// (decision G5.4).
func (d *addTaskDialog) preview(a *App, t theme.Theme, r nlp.Result) string {
	project := "Inbox"
	switch {
	case r.Project != "":
		project = "#" + r.Project
	case d.defaultProject != "":
		project = "#" + a.projects[d.defaultProject]
	}
	parts := []string{project}
	if !r.Due.IsZero() {
		parts = append(parts, "due "+r.Due.String())
	}
	if !r.Start.IsZero() {
		parts = append(parts, "s:"+r.Start.String())
	}
	if r.DurationMin > 0 {
		parts = append(parts, view.Duration(r.DurationMin))
	}
	p := r.Priority
	if p == 0 {
		p = task.DefaultPriority
	}
	parts = append(parts, fmt.Sprintf("P%d", p))
	return t.G.Right + " " + strings.Join(parts, " "+t.G.Dot+" ")
}

func (a *App) projectNamed(name string) (task.Project, bool) {
	for _, p := range a.projectList {
		if p.Name == name {
			return p, true
		}
	}
	return task.Project{}, false
}

// --- fuzzy project matching (G5.2) ---

// rankProjects orders projects matching typed: prefix matches first, then
// substring, then subsequence ("musc" → "music"), then near misses (edit
// distance ≤ 2). Non-matches are dropped.
func rankProjects(typed string, ps []task.Project) []task.Project {
	type scored struct {
		p     task.Project
		score int
	}
	var out []scored
	for _, p := range ps {
		s := -1
		switch {
		case strings.HasPrefix(p.Name, typed):
			s = 0
		case strings.Contains(p.Name, typed):
			s = 1
		case isSubsequence(typed, p.Name):
			s = 2
		case levenshtein(typed, p.Name) <= 2:
			s = 3
		}
		if s >= 0 {
			out = append(out, scored{p, s})
		}
	}
	slices.SortStableFunc(out, func(a, b scored) int { return a.score - b.score })
	ranked := make([]task.Project, len(out))
	for i, s := range out {
		ranked[i] = s.p
	}
	return ranked
}

// closestProject is the did-you-mean suggestion for an unknown name: the
// nearest project by edit distance among plausible matches, or "".
func closestProject(name string, ps []task.Project) string {
	best, bestD := "", 3
	for _, p := range ps {
		d := levenshtein(name, p.Name)
		if d < bestD || (d == bestD && best == "" && (isSubsequence(name, p.Name) || strings.HasPrefix(p.Name, name))) {
			best, bestD = p.Name, d
		}
	}
	return best
}

func isSubsequence(s, of string) bool {
	i := 0
	for _, r := range of {
		if i < len(s) && rune(s[i]) == r {
			i++
		}
	}
	return s != "" && i == len(s)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}
