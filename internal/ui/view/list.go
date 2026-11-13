package view

import (
	"fmt"
	"slices"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// Kind identifies a list view.
type Kind int

// List views.
const (
	Today Kind = iota
	Inbox
	Upcoming
	Overdue
	All
	Completed
	Archive
	Project
)

// Spec configures a list view. Every list view is the same List with a
// different Spec (PLAN.md §4).
type Spec struct {
	Kind    Kind
	Project task.ID // Project views only
	Title   string  // header section and status badge
	// Key names the view in stored settings: "sort." + Key.
	Key string
	// OneLine draws one-line rows with no priority, badges or metadata
	// (Inbox, Figma 40:18095). Otherwise rows are two lines (TaskItemv2).
	OneLine bool
	// ShowSort shows the sort at the right of the status bar.
	ShowSort bool
	// Select reports whether a task is listed.
	Select func(t task.Task, now time.Time) bool
	// Badge draws a row's right-hand badge.
	Badge func(th theme.Theme, t task.Task, now time.Time) string
	// Group, if set, groups rows under day headers (Upcoming, G4.1).
	Group func(t task.Task, today task.Date) (task.Date, bool)
	// Cycle is the sort cycle; its first key is the default.
	Cycle []task.SortKey
	// Summary is the status bar message.
	Summary func(th theme.Theme, rows []task.Task) string
	// Empty is shown when nothing is listed (G3.8, G4.6); EmptyHint adds
	// "q to add a task".
	Empty     string
	EmptyHint bool
	// Hints replaces the first key-hint row (Archive, G4.5a).
	Hints []w.Hint
}

// openView selects open tasks matching in, plus today's completions (G3.1).
func openView(in func(t task.Task, today task.Date) bool) func(task.Task, time.Time) bool {
	return func(t task.Task, now time.Time) bool {
		today := task.DateOf(now)
		return task.Listed(t, now, func(x task.Task) bool { return in(x, today) })
	}
}

// SpecFor returns the Spec of a fixed (non-project) view.
func SpecFor(k Kind) Spec {
	switch k {
	case Inbox:
		return Spec{
			Kind: Inbox, Title: "Inbox", Key: "inbox", OneLine: true,
			Select:  openView(func(t task.Task, _ task.Date) bool { return task.InInbox(t) }),
			Cycle:   task.DefaultCycle,
			Summary: inboxSummary,
			Empty:   "Inbox is empty.", EmptyHint: true,
		}
	case Upcoming:
		return Spec{
			Kind: Upcoming, Title: "Upcoming", Key: "upcoming", ShowSort: true,
			Select: openView(task.InUpcoming), Badge: dueDateBadge, Group: task.UpcomingDay,
			Cycle: task.DefaultCycle, Summary: openSummary,
			Empty: "Nothing starting or due in the next 7 days.", EmptyHint: true,
		}
	case Overdue:
		return Spec{
			Kind: Overdue, Title: "Overdue", Key: "overdue", ShowSort: true,
			Select: openView(task.InOverdue), Badge: dueWordBadge,
			Cycle: task.DefaultCycle, Summary: openSummary,
			Empty: "Nothing overdue.", EmptyHint: true,
		}
	case All:
		return Spec{
			Kind: All, Title: "All", Key: "all", ShowSort: true,
			Select: openView(func(t task.Task, _ task.Date) bool { return task.InAll(t) }), Badge: dueDateBadge,
			Cycle: task.DefaultCycle, Summary: countSummary,
			Empty: "No open tasks.", EmptyHint: true,
		}
	case Completed:
		return Spec{
			Kind: Completed, Title: "Completed", Key: "completed", ShowSort: true,
			Select: func(t task.Task, _ time.Time) bool { return task.InCompleted(t) },
			Badge:  completedBadge, Cycle: task.CompletedCycle,
			Summary: func(_ theme.Theme, rows []task.Task) string {
				return fmt.Sprintf("%d completed", len(rows))
			},
			Empty: "No completed tasks yet.",
		}
	case Archive:
		return Spec{
			Kind: Archive, Title: "Archive", Key: "archive", ShowSort: true,
			Select: func(t task.Task, _ time.Time) bool { return task.InArchive(t) },
			Badge:  archivedBadge, Cycle: task.ArchiveCycle,
			Summary: func(_ theme.Theme, rows []task.Task) string {
				return fmt.Sprintf("%d archived", len(rows))
			},
			Empty: "The archive is empty.",
			Hints: []w.Hint{{Key: "u", Label: "unarchive"}, {Key: "x", Label: "delete forever"},
				{Key: "s", Label: "sort"}, {Key: "?", Label: "shortcuts"}, {Key: "/", Label: "search"}},
		}
	default:
		return Spec{
			Kind: Today, Title: "Today", Key: "today", ShowSort: true,
			Select: openView(task.InToday), Badge: dueWordBadge,
			Cycle: task.DefaultCycle, Summary: openSummary,
			Empty: "Nothing due or started today.", EmptyHint: true,
		}
	}
}

// ProjectSpec returns the Spec of a project's list (Figma 40:17686).
func ProjectSpec(p task.Project) Spec {
	return Spec{
		Kind: Project, Project: p.ID, Title: "#" + p.Name, Key: "project." + string(p.ID), ShowSort: true,
		Select: openView(func(t task.Task, _ task.Date) bool { return task.InProject(t, p.ID) }),
		Badge:  dueDateBadge, Cycle: task.DefaultCycle, Summary: countSummary,
		Empty: "#" + p.Name + " has no open tasks.", EmptyHint: true,
	}
}

// --- summaries ---

func durations(rows []task.Task) (open, total, remaining int) {
	for _, r := range rows {
		total += r.DurationMin
		if !r.Done() {
			open++
			remaining += r.DurationMin
		}
	}
	return
}

func withTimes(t theme.Theme, s string, total, remaining int) string {
	if total == 0 {
		return s
	}
	dot := " " + t.G.Dot + " "
	return s + dot + Duration(total) + " total" + dot + Duration(remaining) + " remaining"
}

// openSummary is "4/6 tasks · ~8h total · ~3.75h remaining" (G3.2).
func openSummary(t theme.Theme, rows []task.Task) string {
	open, total, remaining := durations(rows)
	return withTimes(t, fmt.Sprintf("%d/%d %s", open, len(rows), plural(len(rows), "task", "tasks")), total, remaining)
}

// countSummary is "32 tasks · ~28h total · ~10.25h remaining" (40:17686).
func countSummary(t theme.Theme, rows []task.Task) string {
	_, total, remaining := durations(rows)
	return withTimes(t, fmt.Sprintf("%d %s", len(rows), plural(len(rows), "task", "tasks")), total, remaining)
}

// inboxSummary is "6 tasks to organize" (Figma 40:18095).
func inboxSummary(_ theme.Theme, rows []task.Task) string {
	open, _, _ := durations(rows)
	return fmt.Sprintf("%d %s to organize", open, plural(open, "task", "tasks"))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// --- badges (G4.3) ---

// dueWordBadge: "today" / "overdue" words, future dates as dates (Today and
// Overdue, 32:9525).
func dueWordBadge(t theme.Theme, tk task.Task, now time.Time) string {
	switch task.DueStateOf(tk, task.DateOf(now)) {
	case task.DueToday:
		return w.DueLabel(t, w.DueToday, "today", tk.Done())
	case task.DueOverdue:
		return w.DueLabel(t, w.DueOverdue, "overdue", tk.Done())
	case task.DueUpcoming:
		return w.DueLabel(t, w.DueUpcoming, tk.Due.String(), tk.Done())
	}
	return ""
}

// dueDateBadge: always the date, coloured by state (Upcoming, All, projects).
func dueDateBadge(t theme.Theme, tk task.Task, now time.Time) string {
	states := map[task.DueState]w.DueState{task.DueToday: w.DueToday, task.DueOverdue: w.DueOverdue, task.DueUpcoming: w.DueUpcoming}
	s, ok := states[task.DueStateOf(tk, task.DateOf(now))]
	if !ok {
		return ""
	}
	return w.DueLabel(t, s, tk.Due.String(), tk.Done())
}

// completedBadge shows the local completion date (G4.2).
func completedBadge(t theme.Theme, tk task.Task, now time.Time) string {
	return w.DueLabel(t, w.DueUpcoming, task.DateOf(tk.CompletedAt.In(now.Location())).String(), tk.Done())
}

// archivedBadge shows the local archive date (G4.7).
func archivedBadge(t theme.Theme, tk task.Task, now time.Time) string {
	return w.DueLabel(t, w.DueUpcoming, task.DateOf(tk.ArchivedAt.In(now.Location())).String(), tk.Done())
}

// --- the list ---

// block is one laid-out piece of the list: a task row or a day header.
type block struct {
	row    int       // index into rows, or -1 for a header
	day    task.Date // header date
	start  int       // first line
	height int
}

// List is one list view's state: its tasks, sort, selection and scroll.
type List struct {
	Spec Spec
	Sort task.SortKey

	rows    []task.Task
	blocks  []block
	lines   int // total lines
	sel     int
	selID   task.ID
	topLine int
	height  int // body rows available
	today   task.Date
}

// NewList returns an empty list for spec, sorted by the spec's default.
func NewList(spec Spec) *List {
	return &List{Spec: spec, Sort: spec.Cycle[0]}
}

// SetTasks refilters and resorts from all tasks, keeping the selection on
// the same task when it is still listed.
func (l *List) SetTasks(all []task.Task, now time.Time) {
	l.today = task.DateOf(now)
	l.rows = task.Filter(all, func(t task.Task) bool { return l.Spec.Select(t, now) })
	l.order()
}

// SetSort changes the sort and resorts.
func (l *List) SetSort(k task.SortKey) {
	l.Sort = k
	l.order()
}

// NextSort steps the sort along the view's cycle and returns the new key.
func (l *List) NextSort() task.SortKey {
	l.SetSort(task.Next(l.Spec.Cycle, l.Sort))
	return l.Sort
}

func (l *List) order() {
	task.Sort(l.rows, l.Sort)
	if l.Spec.Group != nil {
		slices.SortStableFunc(l.rows, func(a, b task.Task) int {
			da, _ := l.Spec.Group(a, l.today)
			db, _ := l.Spec.Group(b, l.today)
			return da.Compare(db)
		})
	}
	l.layout()
	l.reselect()
}

func (l *List) itemHeight() int {
	if l.Spec.OneLine {
		return 2
	}
	return 3
}

// layout places rows (and day headers) on lines; each block starts with a
// blank line.
func (l *List) layout() {
	l.blocks = l.blocks[:0]
	line := 0
	var day task.Date
	for i, r := range l.rows {
		if l.Spec.Group != nil {
			if d, _ := l.Spec.Group(r, l.today); i == 0 || d != day {
				day = d
				l.blocks = append(l.blocks, block{row: -1, day: d, start: line, height: 2})
				line += 2
			}
		}
		l.blocks = append(l.blocks, block{row: i, start: line, height: l.itemHeight()})
		line += l.itemHeight()
	}
	l.lines = line
}

// reselect moves the selection back onto selID if it is still listed;
// otherwise it keeps the same position, clamped to the list.
func (l *List) reselect() {
	for i, r := range l.rows {
		if r.ID == l.selID {
			l.sel = i
			l.scrollToSelection()
			return
		}
	}
	l.setSel(l.sel)
}

func (l *List) setSel(i int) {
	l.sel = max(min(i, len(l.rows)-1), 0)
	l.selID = ""
	if len(l.rows) > 0 {
		l.selID = l.rows[l.sel].ID
	}
	l.scrollToSelection()
}

// Rows returns the listed tasks in display order.
func (l *List) Rows() []task.Task { return l.rows }

// Selected returns the selected task.
func (l *List) Selected() (task.Task, bool) {
	if len(l.rows) == 0 {
		return task.Task{}, false
	}
	return l.rows[l.sel], true
}

// Move moves the selection by d rows.
func (l *List) Move(d int) { l.setSel(l.sel + d) }

// Page moves the selection by about a screenful in direction d (±1).
func (l *List) Page(d int) { l.setSel(l.sel + d*max(l.height/l.itemHeight(), 1)) }

// Top selects the first row; Bottom the last.
func (l *List) Top()    { l.setSel(0) }
func (l *List) Bottom() { l.setSel(len(l.rows) - 1) }

// SetHeight sets the body height in terminal rows.
func (l *List) SetHeight(h int) {
	l.height = h
	l.scrollToSelection()
}

// rowBlock returns the block holding row i, and the header just above it if
// the row is the first of its day.
func (l *List) rowBlock(i int) (b block, header *block) {
	for bi, x := range l.blocks {
		if x.row == i {
			if bi > 0 && l.blocks[bi-1].row == -1 {
				header = &l.blocks[bi-1]
			}
			return x, header
		}
	}
	return block{}, nil
}

func (l *List) scrollToSelection() {
	if len(l.rows) == 0 || l.height <= 0 {
		l.topLine = 0
		return
	}
	b, header := l.rowBlock(l.sel)
	first, last := b.start, b.start+b.height
	if header != nil {
		first = header.start // keep a day's header on screen with its first task
	}
	if first < l.topLine {
		l.topLine = first
	}
	if last > l.topLine+l.height {
		l.topLine = last - l.height
	}
	l.topLine = max(min(l.topLine, l.lines-l.height), 0)
}

// BodyWidth is the row width for a screen of the given width: one column is
// kept blank and one holds the scrollbar (Figma: 936px body, scrollbar at
// 948px).
func BodyWidth(width int) int { return width - 2 }

// Body renders the list area: height rows of width cells, rows on the left
// and the scrollbar in the last column.
func (l *List) Body(t theme.Theme, width int, projects map[task.ID]string, now time.Time) []string {
	base := t.Base()
	bw := BodyWidth(width)

	lines := make([]string, l.height)
	if len(l.rows) == 0 && l.height > 1 {
		muted := t.Style(t.Fade(t.C.Text), t.C.Bg)
		msg := w.Blank(base, w.MarkerWidth) + muted.Render(w.Truncate(t, l.Spec.Empty, bw-20))
		if l.Spec.EmptyHint {
			msg += base.Render("  ") + t.Style(t.C.Key, t.C.Bg).Bold(true).Render("q") +
				base.Render(" ") + t.Style(t.C.KeyLabel, t.C.Bg).Render("to add a task")
		}
		lines[1] = msg
	}
	for _, b := range l.blocks {
		if b.start+b.height <= l.topLine || b.start >= l.topLine+l.height {
			continue
		}
		var content []string
		if b.row < 0 {
			content = []string{"", w.Blank(base, w.MarkerWidth) +
				t.Style(t.C.TextAccent, t.C.Bg).Bold(true).Render(Day(t, b.day))}
		} else {
			content = append([]string{""}, l.renderRow(t, bw, l.rows[b.row], b.row == l.sel, projects, now)...)
		}
		for j, c := range content {
			if y := b.start + j - l.topLine; y >= 0 && y < l.height {
				lines[y] = c
			}
		}
	}

	bar := w.Scrollbar(t, l.height, w.Scroll{Total: l.lines, Visible: l.height, Offset: l.topLine})
	out := make([]string, l.height)
	for i := range out {
		out[i] = w.Fit(lines[i], bw, base) + base.Render(" ") + bar[i]
	}
	return out
}

// renderRow draws one task (Figma TaskItemv2, 14:1199): marker, checkbox,
// priority, name and badge on line 1; project, duration and start date on
// line 2. Completed rows use the "done" variant: muted name, dimmed priority
// and badge.
func (l *List) renderRow(t theme.Theme, width int, tk task.Task, selected bool, projects map[task.ID]string, now time.Time) []string {
	bg, nameFg := t.C.Bg, t.C.Text
	if selected {
		bg, nameFg = t.C.SelectionBg, t.C.SelectionFg
	}
	if tk.Done() {
		nameFg = t.C.TextMuted
	}
	fill := t.Style(t.C.Text, bg)
	lead := w.Marker(t, selected, bg) + w.Checkbox(t, tk.Done(), bg)

	if l.Spec.OneLine {
		name := w.Truncate(t, tk.Name, width-w.Width(lead)-2)
		return []string{w.Fit(lead+fill.Render(" ")+t.Style(nameFg, bg).Render(name), width, fill)}
	}

	lead += w.PriorityMark(t, tk.Priority, tk.Done(), bg)
	right := ""
	if l.Spec.Badge != nil {
		if b := l.Spec.Badge(t, tk, now); b != "" {
			right = fill.Render(" ") + b // one cell between name and badge
		}
	}
	nameRoom := width - w.Width(lead) - 1 - w.Width(right)
	name := t.Style(nameFg, bg).Render(w.Truncate(t, tk.Name, nameRoom))
	line1 := w.Spread(lead+fill.Render(" ")+name, right, width, fill)

	meta := t.Style(t.FadeOn(t.C.Text, bg), bg) // metadata at 50% opacity on the row
	line2 := w.Blank(fill, w.Width(lead))
	for _, s := range metadata(tk, projects) {
		line2 += meta.Render(" " + s + " ")
	}
	return []string{line1, w.Fit(line2, width, fill)}
}

// metadata is the second row: #project, ~duration, s:start.
func metadata(tk task.Task, projects map[task.ID]string) []string {
	var out []string
	if name, ok := projects[tk.ProjectID]; ok {
		out = append(out, "#"+name)
	}
	if tk.DurationMin > 0 {
		out = append(out, Duration(tk.DurationMin))
	}
	if !tk.Start.IsZero() {
		out = append(out, "s:"+tk.Start.String())
	}
	return out
}

// StatusBar renders the view's status bar (decision G3.2). pending replaces
// the right-hand text, e.g. "g…" while a chord is waiting; message, if set,
// replaces the summary (errors and notices).
func (l *List) StatusBar(t theme.Theme, width int, pending, message string) string {
	msg := l.Spec.Summary(t, l.rows)
	if message != "" {
		msg = message
	}
	right := ""
	if l.Spec.ShowSort {
		right = "s:" + l.Sort.String()
	}
	if pending != "" {
		right = pending
	}
	return w.StatusBar(t, width, l.Spec.Title, theme.ToneInfo, msg, right)
}
