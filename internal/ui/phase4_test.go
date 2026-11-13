package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

// projectStore holds the #oneofftasks tasks from the project frame
// (40:17686) plus a few others, all open, on the design day.
func projectStore(t *testing.T, clk *clock) store.Store {
	t.Helper()
	ctx := context.Background()
	clk.Set(designNow.Add(-time.Hour))
	s := store.NewMemory(store.Options{Now: clk.Now})
	ids := map[string]task.ID{}
	for _, n := range []string{"oneofftasks", "groceries", "work"} {
		p, err := s.CreateProject(ctx, n)
		if err != nil {
			t.Fatal(err)
		}
		ids[n] = p.ID
	}
	add := func(tk task.Task) {
		t.Helper()
		if _, err := s.CreateTask(ctx, tk); err != nil {
			t.Fatal(err)
		}
	}
	one := ids["oneofftasks"]
	add(task.Task{Name: "Pay credit card balances", ProjectID: one, Priority: 1, Start: date("2026-11-10"), Due: date("2026-11-10"), DurationMin: 30})
	add(task.Task{Name: "Move some PFML money to the taxes vault?", ProjectID: one, Priority: 2, Due: date("2026-11-20"), DurationMin: 15})
	add(task.Task{Name: "Water plants", ProjectID: one, Priority: 2, Due: date("2026-11-18"), DurationMin: 5})
	add(task.Task{Name: "Shave, clean CPAP, cut nails", ProjectID: one, Priority: 2, Start: date("2026-11-07"), Due: date("2026-11-07"), DurationMin: 60})
	add(task.Task{Name: "Put out the garbage", ProjectID: one, Priority: 3, Due: date("2026-11-15"), DurationMin: 15})
	add(task.Task{Name: "Get oliver a gift card", ProjectID: one, Priority: 4, Start: date("2026-11-10"), Due: date("2026-11-14"), DurationMin: 45})
	add(task.Task{Name: "Get eggs", ProjectID: ids["groceries"], Priority: 3, Due: date("2026-11-12"), DurationMin: 15})
	add(task.Task{Name: "Quarterly review", ProjectID: ids["work"], Priority: 2, Start: date("2026-11-11"), Due: date("2026-11-30")})
	clk.Set(designNow)
	return s
}

func openProject(t *testing.T, a *App, name string) {
	t.Helper()
	press(a, "g", "p")
	g, ok := a.topDialog().(*gotoDialog)
	if !ok {
		t.Fatal("Goto Project did not open")
	}
	for _, p := range a.projectList {
		if p.Name == name {
			g.selectProject(a, p.ID)
		}
	}
	press(a, "enter")
	if a.current.Spec.Title != "#"+name {
		t.Fatalf("opened %q, want #%s", a.current.Spec.Title, name)
	}
}

func TestProjectListMatchesDesign(t *testing.T) {
	clk := &clock{}
	a := newApp(t, projectStore(t, clk), clk, 80, 24)
	openProject(t, a, "oneofftasks")
	assertScreen(t, a, 80, 24)
	lines := screen(a)
	if !strings.HasPrefix(lines[0], "taskmoth ▸ #oneofftasks") {
		t.Errorf("header = %q", lines[0])
	}
	// G4.3: project lists show dates, coloured by state.
	for row, want := range map[int]string{
		2: "▶ [ ]  P1  Pay credit card balances", 5: "  [ ]  P2  Move some PFML money",
		11: "  [ ]  P2  Water plants", 14: "  [ ]  P3  Put out the garbage", 17: "  [ ]  P4  Get oliver a gift card",
	} {
		if !strings.HasPrefix(lines[row], want) {
			t.Errorf("row %d = %q, want %q", row, lines[row], want)
		}
	}
	if !strings.Contains(lines[2], "2026-11-10") || strings.Contains(lines[2], "today") {
		t.Errorf("today's badge should be the date: %q", lines[2])
	}
	if !strings.HasPrefix(lines[21], "  #oneofftasks   6 tasks · ~2.83h total · ~2.83h remaining") {
		t.Errorf("status = %q", lines[21])
	}
	golden(t, "project", a.View().Content)
}

func TestUpcomingGroupsByDay(t *testing.T) {
	clk := &clock{}
	a := newApp(t, projectStore(t, clk), clk, 80, 30)
	press(a, "g", "u")
	assertScreen(t, a, 80, 30)
	text := strings.Join(screen(a), "\n")
	// G4.1: due day if in the window, else start day; headers in date order.
	order := []string{
		"wed · 2026-11-11", "Quarterly review", // start only (due is outside the window)
		"thurs · 2026-11-12", "Get eggs",
		"sat · 2026-11-14", "Get oliver a gift card",
		"sun · 2026-11-15", "Put out the garbage",
	}
	pos := -1
	for _, s := range order {
		i := strings.Index(text, s)
		if i < 0 || i < pos {
			t.Fatalf("%q missing or out of order in:\n%s", s, text)
		}
		pos = i
	}
	for _, outside := range []string{"PFML", "Water plants"} { // due +10 and +8 days
		if strings.Contains(text, outside) {
			t.Errorf("%s is outside the 7-day window", outside)
		}
	}
	golden(t, "upcoming", a.View().Content)
}

func TestOverdueAllCompleted(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 24)

	// Overdue: the open Get eggs, then "Finish creating…", completed today
	// and still listed until midnight (G3.1 applied to every open-task view).
	press(a, "g", "o")
	rows := a.current.Rows()
	if len(rows) != 2 || rows[0].Name != "Get eggs" || !rows[1].Done() || !strings.Contains(screen(a)[2], "overdue") {
		t.Errorf("Overdue = %d rows / %q", len(rows), screen(a)[2])
	}

	press(a, "g", "a")
	if n := len(a.current.Rows()); n != 12 { // 4 open dated + 6 inbox + 2 completed today (G3.1)
		t.Errorf("All lists %d tasks", n)
	}

	press(a, "g", "c")
	rows = a.current.Rows()
	if len(rows) != 2 || a.current.Sort != task.SortCompleted || !strings.HasSuffix(screen(a)[21], "s:completed ") {
		t.Fatalf("Completed: %d rows, sort %v", len(rows), a.current.Sort)
	}
	if !strings.Contains(screen(a)[2], "2026-11-10") {
		t.Errorf("completed badge should show the completion date: %q", screen(a)[2])
	}
	press(a, "space") // G4.2: reopens
	if len(a.current.Rows()) != 1 {
		t.Error("space in Completed should reopen the task")
	}
	press(a, "s")
	if a.current.Sort != task.SortPriority {
		t.Errorf("Completed cycle after completed = %v", a.current.Sort)
	}
}

func TestArchiveTaskFlow(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 24)

	press(a, "x") // File insurance reimbursements
	c, ok := a.topDialog().(*confirmDialog)
	if !ok || c.focus != 0 {
		t.Fatal("x should open the archive dialog with cancel selected (G4.9)")
	}
	golden(t, "confirm_archive", a.View().Content)
	press(a, "enter") // cancel
	if a.topDialog() != nil || len(a.lists[view.Archive].Rows()) != 0 {
		t.Fatal("enter on cancel should do nothing")
	}

	press(a, "x", "right", "enter")
	if a.topDialog() != nil {
		t.Fatal("dialog should close")
	}
	if got := a.lists[view.Archive].Rows(); len(got) != 1 || got[0].Name != "File insurance reimbursements" {
		t.Fatalf("archive = %v", got)
	}
	for _, r := range a.current.Rows() {
		if r.Name == "File insurance reimbursements" {
			t.Error("archived task still in Today")
		}
	}

	press(a, "g", "x")
	lines := screen(a)
	if !strings.HasPrefix(lines[22], " u unarchive  x delete forever  s sort") {
		t.Errorf("archive hint row (G4.5a) = %q", lines[22])
	}
	press(a, "space") // no-op in the archive
	if task, _ := a.current.Selected(); task.Done() {
		t.Error("space should do nothing in the archive")
	}
	press(a, "u")
	if len(a.current.Rows()) != 0 {
		t.Error("u should unarchive")
	}

	// Archive the selected task again, then delete it forever (G4.8).
	press(a, "g", "t")
	victim, _ := a.current.Selected()
	press(a, "x", "right", "enter", "g", "x", "x")
	if c, ok := a.topDialog().(*confirmDialog); !ok || c.action != "delete" || !strings.Contains(c.message, "forever") {
		t.Fatal("x in the archive should ask to delete forever")
	}
	press(a, "esc")
	if a.topDialog() != nil || len(a.current.Rows()) != 1 {
		t.Fatal("esc should cancel")
	}
	press(a, "x", "right", "enter")
	if len(a.current.Rows()) != 0 {
		t.Error("task should be deleted")
	}
	all, _ := s.ListTasks(context.Background())
	for _, x := range all {
		if x.ID == victim.ID {
			t.Errorf("deleted task %q still in the store", x.Name)
		}
	}
}

func TestProjectArchiveFlow(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	openProject(t, a, "oneofftasks")

	// Archive #oneofftasks while viewing it.
	press(a, "g", "p")
	golden(t, "goto_project", a.View().Content)
	g := a.topDialog().(*gotoDialog)
	if all, _ := g.selectable(a); all[g.sel].Name != "oneofftasks" {
		t.Errorf("Goto Project should start on the viewed project, got %q", all[g.sel].Name)
	}
	press(a, "x")
	c, ok := a.topDialog().(*confirmDialog)
	if !ok || c.message != "Archive #oneofftasks and its 6 tasks?" {
		t.Fatalf("confirm = %+v", a.topDialog())
	}
	press(a, "right", "enter")
	if _, ok := a.topDialog().(*gotoDialog); !ok {
		t.Fatal("should return to Goto Project")
	}
	if a.current.Spec.Kind != view.Today {
		t.Error("the archived project's view should give way to Today")
	}
	lines := strings.Join(screen(a), "\n")
	if !strings.Contains(lines, "Archived") || !strings.Contains(lines, "#oneofftasks") {
		t.Errorf("archived project not listed under Archived:\n%s", lines)
	}
	golden(t, "goto_project_archived", a.View().Content)

	// Unarchive it (G4.4): its tasks come back.
	press(a, "esc", "g", "x")
	if n := len(a.current.Rows()); n != 6 {
		t.Fatalf("archive holds %d tasks", n)
	}
	press(a, "g", "p")
	all, nActive := a.topDialog().(*gotoDialog).selectable(a)
	a.topDialog().(*gotoDialog).sel = nActive // the archived project
	if all[nActive].Name != "oneofftasks" {
		t.Fatalf("selected %q", all[nActive].Name)
	}
	press(a, "u")
	if !strings.Contains(screen(a)[21], "Unarchived #oneofftasks.") {
		t.Errorf("notice = %q", screen(a)[21])
	}
	press(a, "esc", "g", "x")
	if n := len(a.current.Rows()); n != 0 {
		t.Errorf("archive still holds %d tasks", n)
	}

	// Archive and delete forever (G4.8).
	press(a, "g", "p")
	g = a.topDialog().(*gotoDialog)
	for i, p := range a.projectList {
		if p.Name == "oneofftasks" {
			g.sel = i
		}
	}
	press(a, "x", "right", "enter") // archive
	all, nActive = a.topDialog().(*gotoDialog).selectable(a)
	a.topDialog().(*gotoDialog).sel = nActive
	press(a, "x")
	if c := a.topDialog().(*confirmDialog); c.message != "Delete #oneofftasks and its 6 archived tasks forever?" {
		t.Errorf("delete message = %q", c.message)
	}
	press(a, "right", "enter")
	if _, err := s.ProjectByName(context.Background(), "oneofftasks"); err == nil {
		t.Error("project still exists")
	}
	_ = all
}

func TestLoneUnarchiveMovesToInbox(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	ctx := context.Background()
	g, _ := s.ProjectByName(ctx, "groceries")
	if _, err := s.ArchiveProject(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	run(a, a.load(false))
	press(a, "g", "x", "u")
	if !strings.Contains(screen(a)[21], "Moved to Inbox (#groceries is archived).") {
		t.Errorf("notice = %q", screen(a)[21])
	}
	press(a, "g", "i")
	if rows := a.current.Rows(); len(rows) != 1 || rows[0].Name != "Get eggs" {
		t.Errorf("Inbox = %v", rows)
	}
}

func TestCreateProjectDialog(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	ctx := context.Background()
	w, _ := s.ProjectByName(ctx, "work")
	if _, err := s.ArchiveProject(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "g", "p", "n")
	if _, ok := a.topDialog().(*createProjectDialog); !ok {
		t.Fatal("n should open Create New Project")
	}
	golden(t, "create_project", a.View().Content)

	typeKeys := func(text string) {
		for _, r := range text {
			press(a, string(r))
		}
	}
	errorLine := func() string {
		for _, l := range screen(a) {
			if strings.Contains(l, "│ ") && (strings.Contains(l, "exists") || strings.Contains(l, "archived") ||
				strings.Contains(l, "Use only") || strings.Contains(l, "Enter a name")) {
				return l
			}
		}
		return ""
	}
	press(a, "enter")
	if !strings.Contains(errorLine(), "Enter a name.") {
		t.Errorf("empty name error: %q", errorLine())
	}
	typeKeys("groceries")
	press(a, "enter")
	if !strings.Contains(errorLine(), "#groceries already exists.") {
		t.Errorf("duplicate error: %q", errorLine())
	}
	for range "groceries" {
		press(a, "backspace")
	}
	typeKeys("Work")
	press(a, "enter")
	if !strings.Contains(errorLine(), "#work is archived; unarchive it instead.") {
		t.Errorf("archived-name error: %q", errorLine())
	}
	for range "Work" {
		press(a, "backspace")
	}
	typeKeys("Home Lab")
	press(a, "enter")
	if !strings.Contains(errorLine(), "Use only a-z, 0-9, - and _.") {
		t.Errorf("invalid error: %q", errorLine())
	}
	for range "Home Lab" {
		press(a, "backspace")
	}
	typeKeys("#Hobbies-Games")
	press(a, "enter")
	g, ok := a.topDialog().(*gotoDialog)
	if !ok {
		t.Fatal("should return to Goto Project")
	}
	all, _ := g.selectable(a)
	if all[g.sel].Name != "hobbies-games" {
		t.Errorf("new project should be selected, got %q", all[g.sel].Name)
	}
	if !strings.Contains(screen(a)[21], "Created #hobbies-games.") {
		t.Errorf("notice = %q", ansi.Strip(screen(a)[21]))
	}
}

func TestEmptyViews(t *testing.T) {
	clk := &clock{t: designNow}
	a := newApp(t, store.NewMemory(store.Options{Now: clk.Now}), clk, 80, 24)
	for key, want := range map[string]string{
		"u": "Nothing starting or due in the next 7 days.  q to add a task",
		"o": "Nothing overdue.  q to add a task",
		"a": "No open tasks.  q to add a task",
		"c": "No completed tasks yet.",
		"x": "The archive is empty.",
	} {
		press(a, "g", key)
		line := strings.TrimRight(screen(a)[2], " │")
		if !strings.HasSuffix(line, want) {
			t.Errorf("g %s empty state = %q", key, line)
		}
	}
	press(a, "g", "p")
	if !strings.Contains(strings.Join(screen(a), "\n"), "No projects yet.") {
		t.Error("Goto Project should say there are no projects")
	}
	press(a, "x", "u", "enter", "j") // must not panic with nothing selected
}
