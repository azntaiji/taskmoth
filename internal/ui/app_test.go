package ui

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

var update = flag.Bool("update", false, "rewrite golden files")

// designNow is the time shown in the designs' header: tues · 2026-11-10 · 15:26.
var designNow = time.Date(2026, 11, 10, 15, 26, 0, 0, time.UTC)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time  { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Set(t time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.t = t }
func date(s string) task.Date    { d, _ := task.ParseDate(s); return d }
func keyText(s string) tea.KeyPressMsg {
	switch s {
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "ctrl+q":
		return tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

// designStore holds exactly the tasks in the Today (32:9525) and Inbox
// (40:18095) frames, with the two completed tasks completed on the design day.
func designStore(t *testing.T, clk *clock) store.Store {
	t.Helper()
	ctx := context.Background()
	s := store.NewMemory(store.Options{Now: clk.Now})
	proj := map[string]task.ID{}
	for _, n := range []string{"oneofftasks", "reminders", "parenting", "groceries", "homelab-tech", "movies-tv"} {
		p, err := s.CreateProject(ctx, n)
		if err != nil {
			t.Fatal(err)
		}
		proj[n] = p.ID
	}
	add := func(tk task.Task, done bool) {
		t.Helper()
		created, err := s.CreateTask(ctx, tk)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			if _, err := s.SetDone(ctx, created.ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	clk.Set(designNow.Add(-6 * time.Hour)) // created and completed earlier that day
	add(task.Task{Name: "File insurance reimbursements", ProjectID: proj["oneofftasks"], Priority: 1, Start: date("2026-11-10"), Due: date("2026-11-10"), DurationMin: 120}, false)
	add(task.Task{Name: "Practice meditating", ProjectID: proj["reminders"], Priority: 2, Due: date("2026-11-10"), DurationMin: 30}, false)
	add(task.Task{Name: "Pack Azalea's books in backpack", ProjectID: proj["parenting"], Priority: 2, Due: date("2026-11-10"), DurationMin: 60}, false)
	add(task.Task{Name: "Get eggs", ProjectID: proj["groceries"], Priority: 3, Start: date("2026-11-09"), Due: date("2026-11-09"), DurationMin: 15}, false)
	add(task.Task{Name: "Finish creating task management TUI app. Then put it into production", ProjectID: proj["homelab-tech"], Priority: 3, Start: date("2026-11-07"), Due: date("2026-11-09"), DurationMin: 210}, true)
	add(task.Task{Name: "Watch Tales of the Loop", ProjectID: proj["movies-tv"], Priority: 4, Start: date("2026-11-10"), Due: date("2026-11-10"), DurationMin: 45}, true)
	for _, n := range []string{"Try Godot exercise", "Citizen sleeper", "Sable", "Season a letter to the future", "Pacific drive", "More graphic novels"} {
		add(task.Task{Name: n}, false)
	}
	clk.Set(designNow)
	return s
}

// run executes cmd and feeds resulting messages back into the app until no
// command remains. It never runs the clock tick, which waits for real time.
func run(a *App, cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		switch msg := msg.(type) {
		case nil:
			return
		case tea.BatchMsg:
			for _, c := range msg {
				run(a, c)
			}
			return
		case tickMsg:
			return
		default:
			_, cmd = a.Update(msg)
		}
	}
}

func newApp(t *testing.T, s store.Store, clk *clock, w, h int) *App {
	t.Helper()
	a := NewApp(theme.New(false), s, clk.Now)
	a.Update(tea.WindowSizeMsg{Width: w, Height: h})
	run(a, a.load(true))
	return a
}

func press(a *App, keys ...string) {
	for _, k := range keys {
		_, cmd := a.Update(keyText(k))
		run(a, cmd)
	}
}

func screen(a *App) []string { return strings.Split(ansi.Strip(a.View().Content), "\n") }

func assertScreen(t *testing.T, a *App, w, h int) {
	t.Helper()
	lines := strings.Split(a.View().Content, "\n")
	if len(lines) != h {
		t.Fatalf("%d lines, want %d", len(lines), h)
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Errorf("line %d is %d cells, want %d", i, got, w)
		}
	}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", name+".txt"), []byte(ansi.Strip(got)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run with -update): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("%s differs from golden.\n--- want\n%s\n--- got\n%s", name, ansi.Strip(string(want)), ansi.Strip(got))
	}
}

func TestTodayMatchesDesign(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	assertScreen(t, a, 80, 24)
	lines := screen(a)

	if !strings.HasPrefix(lines[0], "taskmoth ▸ Today") || !strings.HasSuffix(lines[0], "tues · 2026-11-10 · 15:26") {
		t.Errorf("header = %q", lines[0])
	}
	// G3.3c: open first, completed last; P2 tie A–Z (swapped vs the frame).
	wantRows := map[int]string{
		2:  "▶ [ ]  P1  File insurance reimbursements",
		3:  "           #oneofftasks  ~2h  s:2026-11-10",
		5:  "  [ ]  P2  Pack Azalea's books in backpack",
		8:  "  [ ]  P2  Practice meditating",
		11: "  [ ]  P3  Get eggs",
		12: "           #groceries  ~15m  s:2026-11-09",
		14: "  [x]  P3  Finish creating task management TUI app. Then put it int…",
		17: "  [x]  P4  Watch Tales of the Loop",
	}
	for row, prefix := range wantRows {
		if !strings.HasPrefix(lines[row], prefix) {
			t.Errorf("row %d = %q\n      want prefix %q", row, lines[row], prefix)
		}
	}
	if !strings.HasSuffix(strings.TrimRight(lines[2], " │█"), "today") ||
		!strings.HasSuffix(strings.TrimRight(lines[11], " │█"), "overdue") {
		t.Errorf("due badges: %q / %q", lines[2], lines[11])
	}
	if want := "  Today   4/6 tasks · ~8h total · ~3.75h remaining"; !strings.HasPrefix(lines[21], want) ||
		!strings.HasSuffix(lines[21], "s:priority ") {
		t.Errorf("status = %q", lines[21])
	}
	golden(t, "today", a.View().Content)
}

func TestInboxMatchesDesign(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "g", "i")
	assertScreen(t, a, 80, 24)
	lines := screen(a)
	if !strings.HasPrefix(lines[0], "taskmoth ▸ Inbox") {
		t.Errorf("header = %q", lines[0])
	}
	// One-line rows (Figma 40:18095), name at column 7, A–Z under the default
	// priority sort since all are P4.
	want := []string{"Citizen sleeper", "More graphic novels", "Pacific drive", "Sable", "Season a letter to the future", "Try Godot exercise"}
	for i, name := range want {
		row := lines[2+2*i]
		if row[7:7+len(name)] != name && !strings.Contains(row, name) {
			t.Errorf("row %d = %q, want %q", i, row, name)
		}
		if idx := strings.Index(row, name); idx != 7+2 && idx != 7 { // "▶ " is 2 bytes wider than 2 cells
			t.Errorf("%q starts at byte %d", name, idx)
		}
	}
	if !strings.HasPrefix(lines[21], "  Inbox   6 tasks to organize") || strings.Contains(lines[21], "s:") {
		t.Errorf("status = %q", lines[21])
	}
	golden(t, "inbox", a.View().Content)
}

func selectedName(a *App) string {
	t, _ := a.current.Selected()
	return t.Name
}

func TestMovement(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	steps := []struct {
		keys []string
		want string
	}{
		{nil, "File insurance reimbursements"},
		{[]string{"j"}, "Pack Azalea's books in backpack"},
		{[]string{"down", "down"}, "Get eggs"},
		{[]string{"k"}, "Practice meditating"},
		{[]string{"G"}, "Watch Tales of the Loop"},
		{[]string{"j"}, "Watch Tales of the Loop"}, // clamped at the end
		{[]string{"g", "g"}, "File insurance reimbursements"},
		{[]string{"end"}, "Watch Tales of the Loop"},
		{[]string{"home"}, "File insurance reimbursements"},
		{[]string{"pgdown"}, "Watch Tales of the Loop"}, // a page holds all six rows
		{[]string{"pgup"}, "File insurance reimbursements"},
	}
	for _, s := range steps {
		press(a, s.keys...)
		if got := selectedName(a); got != s.want {
			t.Errorf("after %v: selected %q, want %q", s.keys, got, s.want)
		}
	}
}

func TestChords(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "g")
	if !strings.HasSuffix(screen(a)[21], "g… ") {
		t.Errorf("pending chord not shown: %q", screen(a)[21])
	}
	press(a, "esc")
	if a.pendingG || a.current.Spec.Kind != view.Today {
		t.Error("esc should cancel the chord")
	}
	press(a, "g", "i")
	if a.current.Spec.Kind != view.Inbox {
		t.Error("g i should open Inbox")
	}
	press(a, "g", "t")
	if a.current.Spec.Kind != view.Today {
		t.Error("g t should open Today")
	}
	// Every g chord lands on its view (Phase 4 exit criterion).
	for key, kind := range map[string]view.Kind{
		"u": view.Upcoming, "o": view.Overdue, "a": view.All, "c": view.Completed, "x": view.Archive, "i": view.Inbox, "t": view.Today,
	} {
		press(a, "g", key)
		if a.current.Spec.Kind != kind {
			t.Errorf("g %s opened %v", key, a.current.Spec.Title)
		}
		if !strings.Contains(screen(a)[0], a.current.Spec.Title) {
			t.Errorf("g %s: header %q", key, screen(a)[0])
		}
	}
	press(a, "g", "p")
	if _, ok := a.topDialog().(*gotoDialog); !ok {
		t.Error("g p should open Goto Project")
	}
}

func TestToggleDoneKeepsSelectionAndLingers(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	press(a, "j", "j", "j") // Get eggs (P3, open)
	if selectedName(a) != "Get eggs" {
		t.Fatalf("selected %q", selectedName(a))
	}
	press(a, "space")
	// Completed tasks sort last (G3.3c) and stay listed today (G3.1); the
	// selection follows the task.
	sel, _ := a.current.Selected()
	if sel.Name != "Get eggs" || !sel.Done() {
		t.Errorf("after space: selected %q done=%v", sel.Name, sel.Done())
	}
	// Completed group, by priority then A–Z: Finish… (P3), Get eggs (P3), Watch… (P4).
	rows := a.current.Rows()
	if rows[len(rows)-2].Name != "Get eggs" || rows[len(rows)-3].Name != "Finish creating task management TUI app. Then put it into production" {
		t.Errorf("completed task should move into the completed group, A–Z within P3")
	}
	if !strings.Contains(screen(a)[21], "3/6 tasks") {
		t.Errorf("summary = %q", screen(a)[21])
	}
	press(a, "space") // reopen
	if sel, _ := a.current.Selected(); sel.Done() || a.current.Rows()[3].Name != "Get eggs" {
		t.Errorf("reopen failed: %+v", sel)
	}
	got, _ := s.GetTask(context.Background(), sel.ID)
	if got.Done() {
		t.Error("store not updated")
	}
}

func TestSortCyclesAndPersists(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	press(a, "s")
	if a.current.Sort != task.SortDue || !strings.HasSuffix(screen(a)[21], "s:due ") {
		t.Errorf("after s: sort %v, status %q", a.current.Sort, screen(a)[21])
	}
	if v, _ := s.GetSetting(context.Background(), "sort.today"); v != "due" {
		t.Errorf("saved sort = %q", v)
	}
	// Inbox keeps its own sort.
	if a.lists[view.Inbox].Sort != task.DefaultSort {
		t.Error("Inbox sort changed with Today's")
	}
	// A restart restores the saved sort.
	b := newApp(t, s, clk, 80, 24)
	if b.current.Sort != task.SortDue {
		t.Errorf("restored sort = %v", b.current.Sort)
	}
}

func TestMidnightDropsYesterdaysCompletions(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	clk.Set(time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
	a.Update(tickMsg(clk.Now()))
	for _, r := range a.current.Rows() {
		if r.Done() {
			t.Errorf("%q completed yesterday should have left Today", r.Name)
		}
	}
	if !strings.Contains(screen(a)[0], "wed · 2026-11-11 · 00:00") {
		t.Errorf("clock not updated: %q", screen(a)[0])
	}
}

func TestEmptyStatesAndBigTerminal(t *testing.T) {
	clk := &clock{t: designNow}
	a := newApp(t, store.NewMemory(store.Options{Now: clk.Now}), clk, 120, 40)
	assertScreen(t, a, 120, 40)
	if !strings.Contains(screen(a)[2], "Nothing due or started today.  q to add a task") {
		t.Errorf("empty Today = %q", screen(a)[2])
	}
	if !strings.Contains(screen(a)[37], "0/0 tasks") {
		t.Errorf("empty summary = %q", screen(a)[37])
	}
	press(a, "space", "j", "s") // must not panic on an empty list
	press(a, "g", "i")
	if !strings.Contains(screen(a)[2], "Inbox is empty.") {
		t.Errorf("empty Inbox = %q", screen(a)[2])
	}
}

func TestScrollsLongLists(t *testing.T) {
	clk := &clock{t: designNow}
	s := store.NewMemory(store.Options{Now: clk.Now})
	for i := 0; i < 20; i++ {
		if _, err := s.CreateTask(context.Background(), task.Task{Name: string(rune('a'+i)) + " task", Due: date("2026-11-10")}); err != nil {
			t.Fatal(err)
		}
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "G")
	lines := screen(a)
	if !strings.Contains(strings.Join(lines, "\n"), "t task") {
		t.Error("the last task should be scrolled into view")
	}
	if strings.Contains(strings.Join(lines, "\n"), "a task") {
		t.Error("the first task should have scrolled out")
	}
	assertScreen(t, a, 80, 24)
}

func TestQuitKeys(t *testing.T) {
	clk := &clock{t: designNow}
	for _, k := range []string{"ctrl+c", "ctrl+q"} {
		a := newApp(t, store.NewMemory(store.Options{}), clk, 80, 24)
		_, cmd := a.Update(keyText(k))
		if cmd == nil {
			t.Fatalf("%s: no command", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s should quit", k)
		}
	}
	a := newApp(t, store.NewMemory(store.Options{}), clk, 80, 24)
	if _, cmd := a.Update(keyText("q")); cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("q must not quit (it is add task)")
		}
	}
}

func TestTooSmall(t *testing.T) {
	clk := &clock{t: designNow}
	a := newApp(t, store.NewMemory(store.Options{}), clk, 79, 24)
	if !strings.Contains(strings.Join(screen(a), "\n"), "Terminal too small") {
		t.Error("expected too-small message")
	}
	if !a.View().AltScreen {
		t.Error("View should request the alternate screen")
	}
}
