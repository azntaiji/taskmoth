package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

// TestCompletedModes covers `c` (G12.1–G12.4): today → all → hidden, one
// setting for every list, remembered across restarts.
func TestCompletedModes(t *testing.T) {
	ctx := context.Background()
	clk := &clock{}
	s := designStore(t, clk)
	// Two tasks completed a week ago: one in the Inbox, one overdue in a project.
	p, err := s.ProjectByName(ctx, "groceries")
	if err != nil {
		t.Fatal(err)
	}
	clk.Set(designNow.AddDate(0, 0, -7))
	for _, tk := range []task.Task{
		{Name: "Old inbox chore", Priority: 4},
		{Name: "Old grocery run", ProjectID: p.ID, Priority: 2, Due: date("2026-11-01")},
	} {
		c, err := s.CreateTask(ctx, tk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.SetDone(ctx, c.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	clk.Set(designNow)
	a := newApp(t, s, clk, 80, 24)

	listed := func(name string) bool {
		for _, r := range a.current.Rows() {
			if r.Name == name {
				return true
			}
		}
		return false
	}
	done := func() (n int) {
		for _, r := range a.current.Rows() {
			if r.Done() {
				n++
			}
		}
		return n
	}
	status := func() string { return screen(a)[21] }

	// today (default): today's completions only, no note.
	press(a, "g", "i")
	if listed("Old inbox chore") || strings.Contains(status(), "done") {
		t.Errorf("default mode: old completion listed or note shown: %q", status())
	}
	press(a, "g", "t")
	if done() != 2 { // the two designStore tasks completed today
		t.Errorf("Today lists %d completed tasks, want today's 2", done())
	}

	// all: older completions join Inbox, All and projects; dated lists don't change.
	press(a, "c")
	if !strings.Contains(status(), "Showing all completed tasks.") {
		t.Errorf("notice = %q", status())
	}
	press(a, "g", "i")
	if !listed("Old inbox chore") || !strings.Contains(status(), "all done") {
		t.Errorf("all mode: Inbox misses the old completion or the note: %q", status())
	}
	if rows := a.current.Rows(); !rows[len(rows)-1].Done() {
		t.Error("completed tasks should sort below open ones (G3.3c)")
	}
	openProject(t, a, "groceries")
	if !listed("Old grocery run") {
		t.Error("all mode: project list misses its old completion")
	}
	press(a, "g", "o")
	if listed("Old grocery run") || strings.Contains(status(), "all done") {
		t.Errorf("Overdue is dated: it lists only today's completions, with no note: %q", status())
	}
	press(a, "g", "a")
	if !listed("Old inbox chore") || !listed("Old grocery run") {
		t.Error("all mode: All misses old completions")
	}

	// hidden: no completed tasks anywhere, with a count.
	press(a, "c")
	if !strings.Contains(status(), "Hiding completed tasks.") {
		t.Errorf("notice = %q", status())
	}
	for _, k := range []string{"t", "i", "u", "o", "a", "s"} {
		press(a, "g", k)
		if done() != 0 {
			t.Errorf("hidden mode: %s lists %d completed tasks", a.current.Spec.Key, done())
		}
	}
	press(a, "g", "a")
	if !strings.Contains(status(), "4 done hidden") { // 2 completed today + 2 last week
		t.Errorf("hidden note = %q", status())
	}

	// Completing a task in hidden mode drops it and selects the next open one (G7.8).
	press(a, "g", "t")
	first, _ := a.current.Selected()
	press(a, "space")
	if listed(first.Name) {
		t.Errorf("%q should disappear once completed while hidden", first.Name)
	}
	if sel, ok := a.current.Selected(); !ok || sel.Done() {
		t.Errorf("selection after completing = %+v", sel)
	}

	// Saved: a restart keeps hidden.
	if v, _ := s.GetSetting(ctx, "completed"); v != "hidden" {
		t.Errorf("saved setting = %q", v)
	}
	b := newApp(t, s, clk, 80, 24)
	if b.current.Completed != task.CompletedHidden || b.lists[view.Inbox].Completed != task.CompletedHidden {
		t.Error("restart lost the hidden mode")
	}

	// And back to the default.
	press(a, "c")
	if a.completed != task.CompletedToday || !strings.Contains(status(), "Showing tasks completed today.") {
		t.Errorf("third c: mode %v, notice %q", a.completed, status())
	}

	// The Archive and Search ignore the mode.
	press(a, "c", "c", "g", "x") // hidden
	if a.current.Spec.Open != nil {
		t.Error("the Archive shouldn't follow the Completed mode")
	}
}
