package ui

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

// calendarStore is the design store plus tasks due over the next weeks: on
// Thursday 2026-11-12 three open and one completed, one archived (not
// counted), one only starting that day (not counted), and a busy Monday
// 2026-11-23 with more tasks than the bar has room for at 80 columns.
func calendarStore(t *testing.T, clk *clock) store.Store {
	t.Helper()
	ctx := context.Background()
	s := designStore(t, clk)
	add := func(tk task.Task) task.Task {
		t.Helper()
		c, err := s.CreateTask(ctx, tk)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	for i, n := range []string{"Call the bank", "Book flights", "Renew passport"} {
		add(task.Task{Name: n, Priority: 2, Due: date("2026-11-12"), DurationMin: 30 * (i + 1)})
	}
	done := add(task.Task{Name: "Water plants", Priority: 4, Due: date("2026-11-12"), DurationMin: 5})
	if _, err := s.SetDone(ctx, done.ID, true); err != nil {
		t.Fatal(err)
	}
	gone := add(task.Task{Name: "Old idea", Priority: 4, Due: date("2026-11-12")})
	if _, err := s.Archive(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}
	add(task.Task{Name: "Start the essay", Priority: 3, Start: date("2026-11-12")})
	add(task.Task{Name: "Pay rent", Priority: 1, Due: date("2026-11-16"), DurationMin: 15})
	for i := range 30 {
		add(task.Task{Name: "Busy " + string(rune('a'+i%26)), Priority: 3, Due: date("2026-11-23")})
	}
	return s
}

func calendarApp(t *testing.T) (*App, *clock) {
	t.Helper()
	clk := &clock{}
	a := newApp(t, calendarStore(t, clk), clk, 80, 24)
	press(a, "g", "c")
	if !a.inCalendar() {
		t.Fatal("g c should open the Calendar")
	}
	return a, clk
}

func selectedDay(a *App) string {
	d, overdue := a.calendar.Selected()
	if overdue {
		return "overdue"
	}
	return d.String()
}

// G14.1–G14.6: g c opens the Calendar on today; days, bars, counts, week
// headers and the overdue row.
func TestCalendar(t *testing.T) {
	a, _ := calendarApp(t)
	assertScreen(t, a, 80, 24)
	if got := selectedDay(a); got != "2026-11-10" {
		t.Errorf("opened on %s, want today", got)
	}
	text := strings.Join(screen(a), "\n")
	for _, want := range []string{
		"taskmoth ▸ Calendar",
		"overdue", "1 due", // Get eggs, due 2026-11-09
		"week of 2026-11-09",
		"tues · 2026-11-10", "1 done / 4 due", "~3.5h", // today: three open (~3.5h), one done
		"thurs · 2026-11-12", "1 done / 4 due", "~3h", // archived and start-only tasks don't count
		"week of 2026-11-16", "mon · 2026-11-16", "0 done / 1 due",
		"week of 2026-11-09 · 3 done / 10 due · ~6.75h remaining", // the status bar sums the whole week
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Calendar lacks %q", want)
		}
	}
	golden(t, "calendar", a.View().Content)

	// The bar: one cell per task, completed ones after the open ones.
	for _, l := range screen(a) {
		if strings.Contains(l, "thurs · 2026-11-12") && !strings.Contains(l, "▆▆▆▆  ") {
			t.Errorf("2026-11-12's bar should have four cells: %q", l)
		}
	}
}

// G14.7: movement, weeks, today and the overdue row.
func TestCalendarMovement(t *testing.T) {
	a, _ := calendarApp(t)
	steps := []struct {
		keys []string
		want string
	}{
		{[]string{"j"}, "2026-11-11"},
		{[]string{"down", "down"}, "2026-11-13"},
		{[]string{"]"}, "2026-11-20"},
		{[]string{"["}, "2026-11-13"},
		{[]string{"[", "["}, "2026-11-10"}, // never before today
		{[]string{"k"}, "overdue"},
		{[]string{"k"}, "overdue"},
		{[]string{"j"}, "2026-11-10"},
		{[]string{"pgdown"}, "2026-11-26"},
		{[]string{"t"}, "2026-11-10"},
		{[]string{"home"}, "overdue"},
		{[]string{"]"}, "2026-11-17"},
		{[]string{"g", "g"}, "overdue"},
	}
	for _, s := range steps {
		press(a, s.keys...)
		if got := selectedDay(a); got != s.want {
			t.Errorf("after %v: selected %s, want %s", s.keys, got, s.want)
		}
	}
}

// G14.4: the Calendar scrolls forward without end, keeping the selection
// on screen.
func TestCalendarScrollsForward(t *testing.T) {
	a, _ := calendarApp(t)
	for range 60 {
		press(a, "]")
	}
	assertScreen(t, a, 80, 24)
	want := date("2026-11-10").AddDays(7 * 60)
	if got := selectedDay(a); got != want.String() {
		t.Fatalf("selected %s, want %s", got, want)
	}
	if text := strings.Join(screen(a), "\n"); !strings.Contains(text, "▶ "+view.Day(a.theme, want)) {
		t.Errorf("the selected day is off screen:\n%s", text)
	}
	press(a, "t")
	if text := strings.Join(screen(a), "\n"); !strings.Contains(text, "overdue") {
		t.Error("t should scroll back to the top")
	}
}

// G14.3: a day with more tasks than the bar holds ends in "+".
func TestCalendarBarOverflow(t *testing.T) {
	a, _ := calendarApp(t)
	press(a, "]", "]")
	for _, l := range screen(a) {
		if strings.Contains(l, "mon · 2026-11-23") {
			if !strings.Contains(l, "▆+") || !strings.Contains(l, "0 done / 30 due") {
				t.Errorf("busy day: %q", l)
			}
			return
		}
	}
	t.Error("2026-11-23 not on screen")
}

// G14.8: enter opens a day's list, every task due that day; esc returns to
// the Calendar on the same day. From the overdue row it opens Overdue.
func TestCalendarOpensDay(t *testing.T) {
	a, _ := calendarApp(t)
	press(a, "j", "j", "enter")
	if a.current.Spec.Kind != view.DayList {
		t.Fatalf("enter opened %q", a.current.Spec.Title)
	}
	assertScreen(t, a, 80, 24)
	var names []string
	for _, r := range a.current.Rows() {
		names = append(names, r.Name)
	}
	slices.Sort(names)
	if want := []string{"Book flights", "Call the bank", "Renew passport", "Water plants"}; !slices.Equal(names, want) {
		t.Errorf("day list = %v, want %v", names, want)
	}
	text := strings.Join(screen(a), "\n")
	for _, want := range []string{"taskmoth ▸ thurs · 2026-11-12", "3/4 tasks", "esc back", "t due date"} {
		if !strings.Contains(text, want) {
			t.Errorf("day list lacks %q", want)
		}
	}
	golden(t, "calendar_day", a.View().Content)

	press(a, "esc")
	if !a.inCalendar() || selectedDay(a) != "2026-11-12" {
		t.Errorf("esc: calendar %v on %s, want the Calendar on 2026-11-12", a.inCalendar(), selectedDay(a))
	}

	press(a, "home", "enter")
	if a.current.Spec.Kind != view.Overdue {
		t.Fatalf("enter on the overdue row opened %q", a.current.Spec.Title)
	}
	press(a, "esc")
	if !a.inCalendar() {
		t.Error("esc from Overdue should return to the Calendar")
	}

	// Once you've gone elsewhere, esc in Overdue no longer returns.
	press(a, "enter", "g", "o", "esc")
	if a.current.Spec.Kind != view.Overdue {
		t.Error("esc in Overdue opened with g o shouldn't leave it")
	}
}

// G14.7: q adds a task due on the selected day, from the Calendar and from
// a day's list.
func TestCalendarAddsTaskOnDay(t *testing.T) {
	a, _ := calendarApp(t)
	press(a, "j", "q")
	d, ok := a.topDialog().(*addTaskDialog)
	if !ok {
		t.Fatal("q should open Add Task")
	}
	if got := d.input.Value(); got != " @2026-11-11" {
		t.Errorf("Add Task starts with %q", got)
	}
	press(a, "B", "u", "y", " ", "m", "i", "l", "k", "enter")
	if a.topDialog() != nil {
		t.Fatal("Add Task should have closed")
	}
	if !slices.ContainsFunc(a.tasks, func(tk task.Task) bool { return tk.Name == "Buy milk" && tk.Due == date("2026-11-11") }) {
		t.Error("Buy milk should be due 2026-11-11")
	}
	if text := strings.Join(screen(a), "\n"); !strings.Contains(text, "0 done / 1 due") {
		t.Error("the Calendar should count the new task")
	}

	press(a, "enter", "q")
	if d, ok := a.topDialog().(*addTaskDialog); !ok || d.input.Value() != " @2026-11-11" {
		t.Error("q in a day's list should fill in the day")
	}
	press(a, "esc", "esc", "home", "q")
	if d, ok := a.topDialog().(*addTaskDialog); !ok || d.input.Value() != "" {
		t.Error("q on the overdue row should start empty")
	}
}

// G14.4: at midnight a selection in the past moves to today.
func TestCalendarMidnight(t *testing.T) {
	a, clk := calendarApp(t)
	clk.Set(designNow.Add(12 * time.Hour)) // 03:26 the next day
	a.Update(tickMsg(clk.Now()))
	if got := selectedDay(a); got != "2026-11-11" {
		t.Errorf("after midnight selected %s, want 2026-11-11", got)
	}
}

// G14.3: --ascii draws the bar with "#".
func TestCalendarASCII(t *testing.T) {
	clk := &clock{}
	a := newAppTheme(t, theme.New(theme.Dracula(), true), calendarStore(t, clk), clk, 80, 24)
	press(a, "g", "c")
	assertScreen(t, a, 80, 24)
	text := strings.Join(screen(a), "\n")
	if !strings.Contains(text, "####") || strings.Contains(text, "▆") {
		t.Error("the ASCII Calendar should draw bars with #")
	}
}

// G14.9: in the Calendar, the shortcuts dialog shows the Calendar's keys.
func TestCalendarShortcuts(t *testing.T) {
	a, _ := calendarApp(t)
	press(a, "?")
	assertScreen(t, a, 80, 24)
	text := strings.Join(screen(a), "\n")
	for _, want := range []string{"Goto Lists", "g c calendar", "Calendar", "[ / ] previous/next week", "enter open day",
		"q add task due that day", "General"} {
		if !strings.Contains(text, want) {
			t.Errorf("shortcuts lack %q", want)
		}
	}
	if strings.Contains(text, "Selected Tasks") {
		t.Error("the Calendar's shortcuts shouldn't list task keys")
	}
}

// G14.11, G14.12: Edit Due date shows the day's count for the date typed,
// leaving out the task being edited.
func TestEditDueShowsDayLoad(t *testing.T) {
	clk := &clock{}
	a := newApp(t, calendarStore(t, clk), clk, 80, 24)
	press(a, "t") // File insurance reimbursements, due today
	text := strings.Join(screen(a), "\n")
	if !strings.Contains(text, "tues · 2026-11-10 · 1 done / 3 due") {
		t.Errorf("Edit Due date should count today without the task:\n%s", text)
	}
	golden(t, "edit_due", a.View().Content)
	press(a, "ctrl+u")
	if text := strings.Join(screen(a), "\n"); strings.Contains(text, "done /") {
		t.Error("an empty input shows no count")
	}
	press(a, "t", "h", "u")
	if text := strings.Join(screen(a), "\n"); !strings.Contains(text, "thurs · 2026-11-12 · 1 done / 4 due") {
		t.Errorf("thu should count 2026-11-12:\n%s", text)
	}
	press(a, "x", "x")
	if text := strings.Join(screen(a), "\n"); strings.Contains(text, "done /") {
		t.Error("an unparseable date shows no count")
	}
	press(a, "esc", "r")
	if text := strings.Join(screen(a), "\n"); strings.Contains(text, "done /") {
		t.Error("Edit Start date shows no count")
	}
}

// G14.11: Add Task shows the count for an @due.
func TestAddTaskShowsDayLoad(t *testing.T) {
	clk := &clock{}
	a := newApp(t, calendarStore(t, clk), clk, 80, 24)
	press(a, "q", "X", " ", "@", "t", "h", "u")
	if text := strings.Join(screen(a), "\n"); !strings.Contains(text, "thurs · 2026-11-12 · 1 done / 4 due") {
		t.Errorf("Add Task should count 2026-11-12:\n%s", text)
	}
}

// G14.9: the Calendar's hint rows come from its key map, and no key is
// bound twice in it.
func TestCalendarHintsMatchKeyMap(t *testing.T) {
	for _, th := range []theme.Theme{theme.New(theme.Dracula(), false), theme.New(theme.Dracula(), true)} {
		for _, row := range calendarHints(th.G) { // panics on an unbound key
			if len(row) == 0 {
				t.Error("the Calendar has an empty hint row")
			}
		}
	}
	seen := map[string]bool{}
	for _, g := range calendarKeyGroups[1:] {
		for _, b := range g.bindings {
			for _, k := range b.keys {
				if seen[k] {
					t.Errorf("%q is bound twice", k)
				}
				seen[k] = true
			}
		}
	}
	if _, ok := lookup("c", gotoGroup); !ok {
		t.Error("g c missing from Goto Lists")
	}
}
