package view

import (
	"fmt"
	"slices"
	"strings"
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
	Archive
	Someday
	Project
	Search
	CalendarView // the Calendar (G14.1); drawn by Calendar, not a List
	DayList      // a Calendar day's tasks (G14.8)
)

// Spec configures a list view. Every list view is the same List with a
// different Spec (PLAN.md §4).
type Spec struct {
	Kind    Kind
	Project task.ID   // Project views only
	Date    task.Date // Day lists only (G14.8)
	Title   string    // header section and status badge
	// Key names the view in stored settings: "sort." + Key.
	Key string
	// ShowSort shows the sort at the right of the status bar.
	ShowSort bool
	// Open reports whether an open task belongs in the list. The list adds
	// completed tasks as its Completed mode says (G12.1), so views built on
	// Open answer to `c`. Dated marks the date-based views (Today, Upcoming,
	// Overdue), which list only today's completions even in CompletedAll.
	Open  func(t task.Task, today task.Date) bool
	Dated bool
	// Select, for lists without Open (Archive, Search), reports whether a
	// task is listed; `c` doesn't affect them.
	Select func(t task.Task, now time.Time) bool
	// Badge draws a row's right-hand badge.
	Badge func(th theme.Theme, t task.Task, now time.Time) string
	// Group, if set, groups rows under day headers (Upcoming, G4.1).
	Group func(t task.Task, today task.Date) (task.Date, bool)
	// Groups, if set, is the grouping cycle `o` steps through (G13.1); its
	// first entry is the default. Lists without it don't group.
	Groups []task.GroupBy
	// Cycle is the sort cycle; its first key is the default.
	Cycle []task.SortKey
	// Summary is the status bar message.
	Summary func(th theme.Theme, rows []task.Task) string
	// Empty is shown when nothing is listed (G3.8, G4.6); EmptyHint adds
	// "q to add a task".
	Empty     string
	EmptyHint bool
	// EmptyNote, if set, adds a muted line below the empty message, picked
	// for the day (Inbox G6.12; Someday and projects G7.9, each its own set).
	EmptyNote func(today task.Date) string
	// EmptyArt, if set, is drawn below EmptyNote when it fits (Inbox, G6.15;
	// Someday and projects, G7.9).
	EmptyArt []string
	// DoneNote, if set, replaces the status bar summary once every listed
	// task is done, picked for the day (Today, G6.14).
	DoneNote func(today task.Date) string
	// Order, if set, replaces the sort (Search, G7.1d); s does nothing.
	Order func(rows []task.Task)
}

// SpecFor returns the Spec of a fixed (non-project) view.
func SpecFor(k Kind) Spec {
	switch k {
	case Inbox:
		return Spec{
			Kind: Inbox, Title: "Inbox", Key: "inbox",
			Open:    func(t task.Task, _ task.Date) bool { return task.InInbox(t) },
			Badge:   dueDateBadge, // G6.11: standard rows, not the one-line design
			Cycle:   task.DefaultCycle,
			Summary: inboxSummary,
			Empty:   "Inbox is empty.", EmptyHint: true, EmptyNote: InboxNote, EmptyArt: MothArt,
		}
	case Upcoming:
		return Spec{
			Kind: Upcoming, Title: "Upcoming", Key: "upcoming", ShowSort: true,
			Open: task.InUpcoming, Dated: true, Badge: dueDateBadge, Group: task.UpcomingDay,
			Cycle: task.DefaultCycle, Summary: openSummary,
			Empty: "Nothing starting or due in the next 7 days.", EmptyHint: true,
		}
	case Overdue:
		return Spec{
			Kind: Overdue, Title: "Overdue", Key: "overdue", ShowSort: true,
			Open: task.InOverdue, Dated: true, Badge: dueWordBadge,
			Cycle: task.DefaultCycle, Groups: task.ListGroups, Summary: openSummary,
			Empty: "Nothing overdue.", EmptyHint: true,
		}
	case All:
		return Spec{
			Kind: All, Title: "All", Key: "all", ShowSort: true,
			Open: func(t task.Task, _ task.Date) bool { return task.InAll(t) }, Badge: dueDateBadge,
			Cycle: task.DefaultCycle, Groups: task.ListGroups, Summary: countSummary,
			Empty: "No open tasks.", EmptyHint: true,
		}
	case Someday:
		return Spec{
			Kind: Someday, Title: "Someday", Key: "someday", ShowSort: true,
			Open:  func(t task.Task, _ task.Date) bool { return task.InSomeday(t) },
			Cycle: task.DefaultCycle, Groups: task.ListGroups, Summary: openSummary,
			Empty: "Every open task has a due date.", EmptyHint: true,
			EmptyNote: SomedayNote, EmptyArt: MothArt,
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
		}
	default:
		return Spec{
			Kind: Today, Title: "Today", Key: "today", ShowSort: true,
			Open: task.InToday, Dated: true, Badge: dueWordBadge,
			Cycle: task.DefaultCycle, Groups: task.TodayGroups, Summary: openSummary,
			Empty: "Nothing due or started today.", EmptyHint: true, DoneNote: TodayDoneNote,
		}
	}
}

// DaySpec returns the Spec of a Calendar day's list (G14.8): every task due
// on d that isn't archived, open or completed, the tasks the Calendar
// counts. `c` doesn't affect it. Every list of a day shares one Key, so
// they share a sort and grouping.
func DaySpec(t theme.Theme, d task.Date) Spec {
	return Spec{
		Kind: DayList, Date: d, Title: Day(t, d), Key: "day", ShowSort: true,
		Select: func(tk task.Task, _ time.Time) bool { return task.DueOn(tk, d) },
		Cycle:  task.DefaultCycle, Groups: task.TodayGroups, Summary: openSummary,
		Empty: "Nothing due on this day.", EmptyHint: true,
	}
}

// SearchSpec returns the Spec of the Search view (Figma 40:16937) for query
// q: every task that isn't archived and matches (G7.1). project names a
// task's project ("" for the Inbox).
func SearchSpec(q task.Query, project func(task.ID) string) Spec {
	hits := map[task.ID]bool{} // name hits, filled by Select for Order
	return Spec{
		Kind: Search, Title: "Search", Key: "search",
		Select: func(t task.Task, _ time.Time) bool {
			if t.Archived() {
				return false
			}
			ok, inName := q.Match(t, project(t.ProjectID))
			hits[t.ID] = inName
			return ok
		},
		Order:   func(rows []task.Task) { task.SortSearch(rows, func(t task.Task) bool { return hits[t.ID] }) },
		Badge:   dueWordBadge,
		Cycle:   task.DefaultCycle,
		Summary: countSummary,
		Empty:   "No matching tasks.",
	}
}

// ProjectSpec returns the Spec of a project's list (Figma 40:17686).
func ProjectSpec(p task.Project) Spec {
	return Spec{
		Kind: Project, Project: p.ID, Title: "#" + p.Name, Key: "project." + string(p.ID), ShowSort: true,
		Open:  func(t task.Task, _ task.Date) bool { return task.InProject(t, p.ID) },
		Badge: dueDateBadge, Cycle: task.DefaultCycle, Groups: task.ProjectGroups, Summary: countSummary,
		Empty: "#" + p.Name + " has no open tasks.", EmptyHint: true,
		EmptyNote: func(today task.Date) string { return ProjectNote(p.Name, today) }, EmptyArt: MothArt,
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

// DueWordBadge is the Today-style due badge, also used by the Single Task
// view (29:7123).
func DueWordBadge(t theme.Theme, tk task.Task, now time.Time) string { return dueWordBadge(t, tk, now) }

// dueDateBadge: always the date, coloured by state (Upcoming, All, projects).
func dueDateBadge(t theme.Theme, tk task.Task, now time.Time) string {
	states := map[task.DueState]w.DueState{task.DueToday: w.DueToday, task.DueOverdue: w.DueOverdue, task.DueUpcoming: w.DueUpcoming}
	s, ok := states[task.DueStateOf(tk, task.DateOf(now))]
	if !ok {
		return ""
	}
	return w.DueLabel(t, s, tk.Due.String(), tk.Done())
}

// archivedBadge shows the local archive date (G4.7).
func archivedBadge(t theme.Theme, tk task.Task, now time.Time) string {
	return w.DueLabel(t, w.DueUpcoming, task.DateOf(tk.ArchivedAt.In(now.Location())).String(), tk.Done())
}

// --- the list ---

// block is one laid-out piece of the list: a task row or a group header.
type block struct {
	row    int       // index into rows, or -1 for a header
	day    task.Date // Upcoming's header date (label is "")
	label  string    // a grouping's header (G13)
	start  int       // first line
	height int
}

// List is one list view's state: its tasks, sort, selection and scroll.
type List struct {
	Spec Spec
	Sort task.SortKey
	// Completed is which completed tasks are listed (G12.1); lists without
	// Spec.Open ignore it. The App keeps every list on the same mode.
	Completed task.CompletedMode
	// GroupBy is the list's grouping (G13.1), one of Spec.Groups.
	GroupBy task.GroupBy
	// ProjectName names a project for project groups; the App sets it.
	ProjectName func(task.ID) string

	hidden  int // completed tasks left out by CompletedHidden (G12.4)
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
	l.hidden = 0
	if l.Spec.Open == nil {
		l.rows = task.Filter(all, func(t task.Task) bool { return l.Spec.Select(t, now) })
		l.order()
		return
	}
	in := func(t task.Task) bool { return l.Spec.Open(t, l.today) }
	l.rows = task.Filter(all, func(t task.Task) bool { return task.Listed(t, now, in, l.Completed, l.Spec.Dated) })
	if l.Completed == task.CompletedHidden {
		for _, t := range all {
			if t.Done() && task.Listed(t, now, in, task.CompletedAll, l.Spec.Dated) {
				l.hidden++
			}
		}
	}
	l.order()
}

// SetSort changes the sort and resorts.
func (l *List) SetSort(k task.SortKey) {
	l.Sort = k
	l.order()
}

// SetGroup changes the grouping and regroups. A grouping the list doesn't
// offer is ignored.
func (l *List) SetGroup(g task.GroupBy) {
	if !slices.Contains(l.Spec.Groups, g) {
		return
	}
	l.GroupBy = g
	l.order()
}

// NextGroup steps the grouping along the list's cycle and returns it. Lists
// without groupings return GroupNone.
func (l *List) NextGroup() task.GroupBy {
	if len(l.Spec.Groups) == 0 {
		return task.GroupNone
	}
	l.SetGroup(task.NextGroup(l.Spec.Groups, l.GroupBy))
	return l.GroupBy
}

// group returns row r's header key and block, if the list is grouped:
// Upcoming's day (G4.1) or the chosen grouping (G13).
func (l *List) group(r task.Task) (key task.Group, b block, ok bool) {
	switch {
	case l.Spec.Group != nil:
		d, _ := l.Spec.Group(r, l.today)
		return task.Group{Label: d.String()}, block{row: -1, day: d}, true
	case l.GroupBy != task.GroupNone:
		name := l.ProjectName
		if name == nil {
			name = func(task.ID) string { return "" }
		}
		g := task.GroupOf(r, l.GroupBy, l.today, name)
		return g, block{row: -1, label: g.Label}, true
	}
	return task.Group{}, block{}, false
}

// NextSort steps the sort along the view's cycle and returns the new key.
// Lists with their own Order (Search) don't sort.
func (l *List) NextSort() task.SortKey {
	if l.Spec.Order != nil {
		return l.Sort
	}
	l.SetSort(task.Next(l.Spec.Cycle, l.Sort))
	return l.Sort
}

func (l *List) order() {
	if l.Spec.Order != nil {
		l.Spec.Order(l.rows)
	} else {
		task.Sort(l.rows, l.Sort)
	}
	if l.Spec.Group != nil {
		slices.SortStableFunc(l.rows, func(a, b task.Task) int {
			da, _ := l.Spec.Group(a, l.today)
			db, _ := l.Spec.Group(b, l.today)
			return da.Compare(db)
		})
	} else if l.GroupBy != task.GroupNone {
		// Groups in order; the sort (and open-before-completed) within each.
		keys := make(map[task.ID]task.Group, len(l.rows))
		for _, r := range l.rows {
			keys[r.ID], _, _ = l.group(r)
		}
		slices.SortStableFunc(l.rows, func(a, b task.Task) int { return keys[a.ID].Compare(keys[b.ID]) })
	}
	l.layout()
	l.reselect()
}

// rowHeight is a task row's height: a blank spacer and two lines.
const rowHeight = 3

// layout places rows (and day headers) on lines; each block starts with a
// blank line.
func (l *List) layout() {
	l.blocks = l.blocks[:0]
	line := 0
	var last task.Group
	for i, r := range l.rows {
		if key, h, ok := l.group(r); ok && (i == 0 || key != last) {
			last = key
			h.start, h.height = line, 2
			l.blocks = append(l.blocks, h)
			line += 2
		}
		l.blocks = append(l.blocks, block{row: i, start: line, height: rowHeight})
		line += rowHeight
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

// Select moves the selection to task id if it is listed.
func (l *List) Select(id task.ID) {
	for i, r := range l.rows {
		if r.ID == id {
			l.setSel(i)
			return
		}
	}
}

// Move moves the selection by d rows.
func (l *List) Move(d int) { l.setSel(l.sel + d) }

// Page moves the selection by about a screenful in direction d (±1).
func (l *List) Page(d int) { l.setSel(l.sel + d*max(l.height/rowHeight, 1)) }

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
		if l.Spec.EmptyNote != nil && l.height > 3 {
			note := l.Spec.EmptyNote(task.DateOf(now))
			lines[3] = w.Blank(base, w.MarkerWidth) + muted.Render(w.Truncate(t, note, bw-w.MarkerWidth))
		}
		if art := l.Spec.EmptyArt; len(art) > 0 && l.height >= 5+len(art) {
			for i, a := range art {
				lines[5+i] = w.Blank(base, w.MarkerWidth) + muted.Render(w.Truncate(t, a, bw-w.MarkerWidth))
			}
		}
	}
	for _, b := range l.blocks {
		if b.start+b.height <= l.topLine || b.start >= l.topLine+l.height {
			continue
		}
		var content []string
		if b.row < 0 {
			label := b.label
			if label == "" {
				label = Day(t, b.day)
			}
			content = []string{"", w.Blank(base, w.MarkerWidth) +
				t.Style(t.C.TextAccent, t.C.Bg).Bold(true).Render(w.Truncate(t, label, bw-w.MarkerWidth))}
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

// allDone reports whether the list has rows and all of them are done.
func (l *List) allDone() bool {
	for _, t := range l.rows {
		if !t.Done() {
			return false
		}
	}
	return len(l.rows) > 0
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
	if tk.Done() && (!selected || !t.ANSI()) {
		// The default theme keeps selection/fg on the selection bar, where
		// faint text isn't readable (G9.6).
		nameFg = t.C.TextMuted
	}
	fill := t.Style(t.C.Text, bg)
	lead := w.Marker(t, selected, bg) + w.Checkbox(t, tk.Done(), bg)

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

	metaFg := t.C.Text
	if selected {
		metaFg = t.C.SelectionFg // the same in Dracula; readable in the default theme (G9.6)
	}
	meta := t.Style(t.FadeOn(metaFg, bg), bg) // metadata at 50% opacity on the row
	line2 := w.Blank(fill, w.Width(lead))
	if l.Spec.Kind == Project {
		projects = nil // G7.5: a project's own list doesn't repeat its name
	}
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
	if l.Spec.DoneNote != nil && l.allDone() {
		msg = l.Spec.DoneNote(l.today)
	}
	msg += l.completedNote(t)
	if message != "" {
		msg = message
	}
	right := ""
	if l.Spec.ShowSort {
		right = "s:" + l.Sort.String()
	}
	if l.GroupBy != task.GroupNone { // G13.4
		right = strings.TrimSpace("o:" + l.GroupBy.String() + "  " + right)
	}
	if pending != "" {
		right = pending
	}
	return w.StatusBar(t, width, l.Spec.Title, theme.ToneInfo, msg, right)
}

// completedNote is the status bar's note on the Completed mode (G12.4):
// "· all done" in CompletedAll, "· 3 done hidden" when CompletedHidden is
// leaving tasks out, and nothing in the default mode.
func (l *List) completedNote(t theme.Theme) string {
	if l.Spec.Open == nil {
		return ""
	}
	dot := " " + t.G.Dot + " "
	switch {
	case l.Completed == task.CompletedAll && !l.Spec.Dated:
		return dot + "all done"
	case l.Completed == task.CompletedHidden && l.hidden > 0:
		return dot + fmt.Sprintf("%d done hidden", l.hidden)
	}
	return ""
}

// Headers returns the group headers in display order (Upcoming's days as
// dates). It's empty when the list isn't grouped.
func (l *List) Headers() []string {
	var out []string
	for _, b := range l.blocks {
		if b.row >= 0 {
			continue
		}
		if b.label != "" {
			out = append(out, b.label)
		} else {
			out = append(out, b.day.String())
		}
	}
	return out
}
