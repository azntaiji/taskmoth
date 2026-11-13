package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

func typeText(a *App, s string) {
	for _, r := range s {
		if r == ' ' {
			press(a, "space")
			continue
		}
		press(a, string(r))
	}
}

func addDialog(t *testing.T, a *App) *addTaskDialog {
	t.Helper()
	d, ok := a.topDialog().(*addTaskDialog)
	if !ok {
		t.Fatal("Add Task is not open")
	}
	return d
}

func findTask(t *testing.T, s store.Store, name string) task.Task {
	t.Helper()
	all, _ := s.ListTasks(context.Background())
	for _, x := range all {
		if x.Name == name {
			return x
		}
	}
	t.Fatalf("no task named %q", name)
	return task.Task{}
}

// TestAddTaskDesignSample is the Phase 5 exit criterion: the design's sample
// input (Figma 40:16708) produces the expected task.
func TestAddTaskDesignSample(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	press(a, "q")
	typeText(a, "Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow")
	assertScreen(t, a, 80, 24)
	text := strings.Join(screen(a), "\n")
	if !strings.Contains(text, "→ #oneofftasks · due 2026-11-12 · s:2026-11-11 · ~30m ·") ||
		!strings.Contains(text, "│ P2 ") { // the preview wraps onto a second line
		t.Errorf("preview line missing:\n%s", text)
	}
	golden(t, "add_task", a.View().Content)

	press(a, "enter")
	if a.topDialog() != nil {
		t.Fatal("the dialog should close after adding")
	}
	got := findTask(t, s, "Change the laundry")
	one, _ := s.ProjectByName(context.Background(), "oneofftasks")
	if got.ProjectID != one.ID || got.Due.String() != "2026-11-12" || got.Start.String() != "2026-11-11" ||
		got.DurationMin != 30 || got.Priority != 2 {
		t.Errorf("task = %+v", got)
	}
	if !strings.Contains(screen(a)[21], `Added "Change the laundry".`) {
		t.Errorf("notice = %q", screen(a)[21])
	}
}

func TestAddTaskSelectsNewTaskAndDefaults(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	a := newApp(t, s, clk, 80, 24)

	// Plain text goes to the Inbox as P4; due today puts it in Today, selected.
	press(a, "q")
	typeText(a, "Buy milk today")
	press(a, "enter")
	if sel, _ := a.current.Selected(); sel.Name != "Buy milk" {
		t.Errorf("selected %q, want the new task", sel.Name)
	}
	if got := findTask(t, s, "Buy milk"); got.ProjectID != "" || got.Priority != task.P4 {
		t.Errorf("defaults: %+v", got)
	}

	// From a project list, the project is the default (G5.4).
	openProject(t, a, "work")
	press(a, "q")
	if !strings.Contains(strings.Join(screen(a), "\n"), "→ #work · P4") {
		t.Error("preview should show the default project")
	}
	typeText(a, "Write agenda")
	press(a, "enter")
	w, _ := s.ProjectByName(context.Background(), "work")
	if got := findTask(t, s, "Write agenda"); got.ProjectID != w.ID {
		t.Errorf("project = %q, want #work", got.ProjectID)
	}
	// An explicit # overrides it.
	press(a, "q")
	typeText(a, "Buy eggs #groceries")
	press(a, "enter")
	g, _ := s.ProjectByName(context.Background(), "groceries")
	if got := findTask(t, s, "Buy eggs"); got.ProjectID != g.ID {
		t.Error("explicit # should override the default project")
	}
}

func TestAddTaskAutocomplete(t *testing.T) {
	clk := &clock{}
	a := newApp(t, projectStore(t, clk), clk, 80, 24)
	press(a, "q")
	typeText(a, "Buy #gro")
	d := addDialog(t, a)
	if sugg := d.suggestions(a); len(sugg) != 1 || sugg[0].Name != "groceries" {
		t.Fatalf("suggestions = %v", sugg)
	}
	if !strings.Contains(strings.Join(screen(a), "\n"), "▶ #groceries") {
		t.Error("suggestion list not drawn")
	}
	golden(t, "add_task_autocomplete", a.View().Content)
	press(a, "tab")
	if d.input.Value() != "Buy #groceries " {
		t.Errorf("after tab: %q", d.input.Value())
	}

	// Fuzzy: #musc-style typos still list the project.
	press(a, "backspace")
	for range "#groceries" {
		press(a, "backspace")
	}
	typeText(a, "#onof")
	if sugg := d.suggestions(a); len(sugg) == 0 || sugg[0].Name != "oneofftasks" {
		t.Errorf("fuzzy suggestions = %v", sugg)
	}
	// esc hides the list first, then closes the dialog.
	press(a, "esc")
	if len(d.suggestions(a)) != 0 || a.topDialog() == nil {
		t.Error("first esc should only hide the list")
	}
	press(a, "esc")
	if a.topDialog() != nil {
		t.Error("second esc should close the dialog")
	}
}

func TestAddTaskNewProjectPrompt(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	ctx := context.Background()
	a := newApp(t, s, clk, 80, 24)

	// A near miss suggests the existing project; tab uses it (G5.2).
	press(a, "q")
	typeText(a, "Return library books #onofftasks")
	press(a, "enter")
	d := addDialog(t, a)
	if d.prompt == nil || d.prompt.suggestion != "oneofftasks" {
		t.Fatalf("prompt = %+v", d.prompt)
	}
	if !strings.Contains(strings.Join(screen(a), "\n"), "Create #onofftasks? Did you mean #oneofftasks?") {
		t.Error("did-you-mean line not shown")
	}
	golden(t, "add_task_did_you_mean", a.View().Content)
	press(a, "tab")
	one, _ := s.ProjectByName(ctx, "oneofftasks")
	if got := findTask(t, s, "Return library books"); got.ProjectID != one.ID {
		t.Error("tab should use #oneofftasks")
	}
	if _, err := s.ProjectByName(ctx, "onofftasks"); err == nil {
		t.Error("the typo project must not be created")
	}

	// No near match: enter confirms creating it.
	press(a, "q")
	typeText(a, "Mop floors #chores")
	press(a, "enter")
	if p := addDialog(t, a).prompt; p == nil || p.suggestion != "" {
		t.Fatalf("prompt = %+v", p)
	}
	press(a, "enter")
	c, err := s.ProjectByName(ctx, "chores")
	if err != nil {
		t.Fatal("#chores should be created")
	}
	if got := findTask(t, s, "Mop floors"); got.ProjectID != c.ID {
		t.Error("task should be in #chores")
	}
	if !strings.Contains(screen(a)[21], "Created #chores.") {
		t.Errorf("notice = %q", screen(a)[21])
	}

	// esc on the prompt goes back to editing.
	press(a, "q")
	typeText(a, "x #nope")
	press(a, "enter", "esc")
	if d := addDialog(t, a); d.prompt != nil {
		t.Error("esc should dismiss the prompt")
	}
}

func TestAddTaskErrorsAndToggle(t *testing.T) {
	clk := &clock{}
	s := projectStore(t, clk)
	ctx := context.Background()
	w, _ := s.ProjectByName(ctx, "work")
	if _, err := s.ArchiveProject(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "q")
	typeText(a, "#groceries")
	press(a, "enter")
	if addDialog(t, a).err != "Enter a task name." {
		t.Errorf("err = %q", addDialog(t, a).err)
	}
	for range "#groceries" {
		press(a, "backspace")
	}
	typeText(a, "Plan #work")
	press(a, "enter")
	if addDialog(t, a).err != "#work is archived; unarchive it instead." {
		t.Errorf("err = %q", addDialog(t, a).err)
	}
	for range "Plan #work" {
		press(a, "backspace")
	}

	// ctrl+t keeps a false match as text (G5.1d).
	typeText(a, "Pay may 5 bill")
	press(a, "left", "left", "left", "left", "left") // just after "5"
	press(a, "ctrl+t")
	d := addDialog(t, a)
	if d.input.Value() != `Pay \may \5 bill` {
		t.Fatalf("after ctrl+t: %q", d.input.Value())
	}
	if strings.Contains(strings.Join(screen(a), "\n"), "· due ") {
		t.Error("preview should no longer show a due date")
	}
	press(a, "enter")
	if got := findTask(t, s, "Pay may 5 bill"); !got.Due.IsZero() {
		t.Errorf("due = %s", got.Due)
	}
}

func TestAddTaskFromEveryView(t *testing.T) {
	clk := &clock{}
	a := newApp(t, store.NewMemory(store.Options{Now: clk.Now}), clk, 80, 24)
	for _, k := range []string{"t", "i", "u", "o", "a", "c", "x"} {
		press(a, "g", k, "q")
		if _, ok := a.topDialog().(*addTaskDialog); !ok {
			t.Errorf("q in %v should open Add Task", a.current.Spec.Title)
		}
		press(a, "esc")
	}
	if a.current.Spec.Kind != view.Archive {
		t.Error("esc should return to the same view")
	}
}
