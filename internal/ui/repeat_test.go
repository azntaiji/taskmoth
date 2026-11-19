package ui

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

func screenText(a *App) string { return strings.Join(screen(a), "\n") }

func taskNamed(t *testing.T, a *App, name string, open bool) task.Task {
	t.Helper()
	i := slices.IndexFunc(a.tasks, func(x task.Task) bool { return x.Name == name && x.Done() != open })
	if i < 0 {
		t.Fatalf("no %s task %q", map[bool]string{true: "open", false: "completed"}[open], name)
	}
	return a.tasks[i]
}

// G15.2, G15.7, G15.8: quick add takes a repeat, previews it and shows it on
// the row.
func TestAddRepeatingTask(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "q")
	for _, r := range "Water plants every mon and thu #reminders" {
		press(a, string(r))
	}
	if text := screenText(a); !strings.Contains(text, "↻ every mon, thu") || !strings.Contains(text, "due 2026-11-12") {
		t.Errorf("Add Task preview lacks the repeat:\n%s", text)
	}
	golden(t, "add_task_repeat", a.View().Content)
	press(a, "enter")
	tk := taskNamed(t, a, "Water plants", true)
	if tk.Repeat.String() != "every mon, thu" || tk.Due != date("2026-11-12") || tk.SeriesID != tk.ID {
		t.Errorf("added %+v", tk)
	}
	press(a, "g", "u")
	if text := screenText(a); !strings.Contains(text, "#reminders  ↻ every mon, thu") {
		t.Errorf("Upcoming row lacks the repeat:\n%s", text)
	}
}

// G15.3, G15.8: completing a repeating task keeps it done and adds the next
// one, and says when.
func TestCompleteRepeatingTask(t *testing.T) {
	ctx := context.Background()
	clk := &clock{}
	s := designStore(t, clk)
	if _, err := s.CreateTask(ctx, task.Task{Name: "Breathe", Priority: 1, Due: date("2026-11-10"), DurationMin: 10,
		Repeat: task.Rule{Every: 1, Unit: task.Day}}); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	if sel, _ := a.current.Selected(); sel.Name != "Breathe" {
		t.Fatalf("selected %q", sel.Name)
	}
	press(a, "space")
	if text := screenText(a); !strings.Contains(text, "Next: wed · 2026-11-11.") {
		t.Errorf("no next notice:\n%s", text)
	}
	done := taskNamed(t, a, "Breathe", false)
	next := taskNamed(t, a, "Breathe", true)
	if !done.Repeat.IsZero() || next.Repeat.IsZero() || next.Due != date("2026-11-11") || next.SeriesID != done.SeriesID {
		t.Errorf("done %+v\nnext %+v", done, next)
	}

	// The Single Task view shows the rule.
	press(a, "g", "u", "e")
	if text := screenText(a); !strings.Contains(text, "↻ every day") {
		t.Errorf("task view lacks the rule:\n%s", text)
	}
}

// G15.2: Edit Due date starts with the rule, takes a new rule or a date, and
// clearing it clears the rule.
func TestEditDueRepeat(t *testing.T) {
	ctx := context.Background()
	clk := &clock{}
	s := designStore(t, clk)
	if _, err := s.CreateTask(ctx, task.Task{Name: "Breathe", Priority: 1, Due: date("2026-11-12"),
		Repeat: task.Rule{Every: 1, Unit: task.Day}}); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "g", "a") // All, P1 first
	if sel, _ := a.current.Selected(); sel.Name != "Breathe" {
		t.Fatalf("selected %q", sel.Name)
	}
	press(a, "t")
	d, ok := a.topDialog().(*textEditDialog)
	if !ok || d.input.Value() != "every day" {
		t.Fatalf("Edit Due date starts with %q", d.input.Value())
	}
	if text := screenText(a); !strings.Contains(text, "↻ every day · next 2026-11-12") {
		t.Errorf("note:\n%s", text)
	}
	press(a, "enter") // unchanged: keeps the occurrence's date
	if tk := taskNamed(t, a, "Breathe", true); tk.Due != date("2026-11-12") || tk.Repeat.String() != "every day" {
		t.Errorf("unchanged save: %+v", tk)
	}

	setInput := func(v string) {
		press(a, "t", "ctrl+u")
		for _, r := range v {
			press(a, string(r))
		}
	}
	setInput("every 15th")
	if text := screenText(a); !strings.Contains(text, "↻ every 15th · next 2026-11-15") {
		t.Errorf("note for a new rule:\n%s", text)
	}
	press(a, "enter")
	if tk := taskNamed(t, a, "Breathe", true); tk.Due != date("2026-11-15") || tk.Repeat.String() != "every 15th" {
		t.Errorf("new rule: %+v", tk)
	}

	setInput("nov 20") // reschedules this occurrence, keeps the rule
	press(a, "enter")
	if tk := taskNamed(t, a, "Breathe", true); tk.Due != date("2026-11-20") || tk.Repeat.String() != "every 15th" {
		t.Errorf("rescheduled: %+v", tk)
	}

	setInput("")
	press(a, "enter")
	if tk := taskNamed(t, a, "Breathe", true); !tk.Due.IsZero() || !tk.Repeat.IsZero() {
		t.Errorf("cleared: %+v", tk)
	}
}

// G15.8: --ascii marks a repeat with "@".
func TestRepeatASCII(t *testing.T) {
	ctx := context.Background()
	clk := &clock{}
	s := designStore(t, clk)
	if _, err := s.CreateTask(ctx, task.Task{Name: "Breathe", Priority: 1, Due: date("2026-11-10"),
		Repeat: task.Rule{Every: 1, Unit: task.Day}}); err != nil {
		t.Fatal(err)
	}
	a := newAppTheme(t, theme.New(theme.Dracula(), true), s, clk, 80, 24)
	assertScreen(t, a, 80, 24)
	if text := screenText(a); !strings.Contains(text, "@ every day") || strings.Contains(text, "↻") {
		t.Errorf("ASCII row:\n%s", text)
	}
}
