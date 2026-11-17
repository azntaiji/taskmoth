package ui

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

// headers returns the current list's group headers that are in want.
func headers(a *App, want []string) []string {
	var got []string
	for _, h := range a.current.Headers() {
		if slices.Contains(want, h) {
			got = append(got, h)
		}
	}
	return got
}

// TestGrouping covers `o` (G13.1–G13.4).
func TestGrouping(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 40)

	// Today: none → project → priority → duration → none.
	press(a, "g", "t", "o")
	if a.current.GroupBy != task.GroupProject || !strings.HasSuffix(screen(a)[37], "o:project  s:priority ") {
		t.Fatalf("Today after o: %v, status %q", a.current.GroupBy, screen(a)[37])
	}
	want := []string{"#groceries", "#homelab-tech", "#movies-tv", "#oneofftasks", "#parenting", "#reminders"}
	if got := headers(a, want); !slices.Equal(got, want) {
		t.Errorf("Today by project: headers %v, want A–Z %v", got, want)
	}
	golden(t, "today_by_project", a.View().Content)
	press(a, "o")
	if got := headers(a, []string{"P1", "P2", "P3", "P4"}); !slices.Equal(got, []string{"P1", "P2", "P3", "P4"}) {
		t.Errorf("Today by priority: %v", got)
	}
	// Within a group the sort applies, completed tasks last (G3.3c).
	var p3 []task.Task
	for _, r := range a.current.Rows() {
		if r.Priority == 3 {
			p3 = append(p3, r)
		}
	}
	if len(p3) != 2 || p3[0].Done() || !p3[1].Done() {
		t.Errorf("P3 group = %v, want the open task first", p3)
	}
	press(a, "o")
	if a.current.GroupBy != task.GroupDuration {
		t.Errorf("third o: %v, want duration", a.current.GroupBy)
	}
	press(a, "o")
	if a.current.GroupBy != task.GroupNone || strings.Contains(screen(a)[37], "o:") {
		t.Errorf("fourth o: %v, status %q", a.current.GroupBy, screen(a)[37])
	}
	if got := a.current.Rows(); len(got) != 6 {
		t.Errorf("ungrouped Today lists %d tasks", len(got))
	}

	// All: none → project → priority → due; the Inbox group first.
	press(a, "g", "a", "o")
	if got := headers(a, []string{"Inbox", "#groceries"}); len(got) == 0 || got[0] != "Inbox" {
		t.Errorf("All by project should start with the Inbox: %v", got)
	}
	press(a, "o", "o")
	if a.current.GroupBy != task.GroupDue {
		t.Fatalf("All: %v, want due", a.current.GroupBy)
	}
	buckets := []string{"overdue", "today", "tomorrow", "this week", "later", "no due date"}
	if got := headers(a, buckets); !slices.Equal(got, []string{"overdue", "today", "no due date"}) {
		t.Errorf("All by due: %v", got)
	}

	// Lists without grouping ignore o.
	for _, k := range []string{"i", "u", "x"} {
		press(a, "g", k, "o")
		if a.current.GroupBy != task.GroupNone {
			t.Errorf("%s grouped by %v", a.current.Spec.Key, a.current.GroupBy)
		}
	}

	// A project list: none → priority → due → duration (no project groups).
	openProject(t, a, "oneofftasks")
	press(a, "o")
	if a.current.GroupBy != task.GroupPriority {
		t.Errorf("project list after o: %v, want priority", a.current.GroupBy)
	}

	// Saved per list: a restart keeps All on due, Today ungrouped and the
	// project on priority.
	if v, _ := s.GetSetting(context.Background(), "group.all"); v != "due" {
		t.Errorf("saved group.all = %q", v)
	}
	b := newApp(t, s, clk, 80, 40)
	if b.lists[view.All].GroupBy != task.GroupDue || b.lists[view.Today].GroupBy != task.GroupNone {
		t.Errorf("restart: All %v, Today %v", b.lists[view.All].GroupBy, b.lists[view.Today].GroupBy)
	}
	openProject(t, b, "oneofftasks")
	if b.current.GroupBy != task.GroupPriority {
		t.Errorf("restart: project grouping %v, want priority", b.current.GroupBy)
	}
}

// TestGroupingKeepsSelection checks that regrouping keeps the selected task
// and that moving down steps over headers.
func TestGroupingKeepsSelection(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "j", "j")
	sel, _ := a.current.Selected()
	press(a, "o")
	if now, _ := a.current.Selected(); now.ID != sel.ID {
		t.Errorf("selection moved from %q to %q", sel.Name, now.Name)
	}
	assertScreen(t, a, 80, 24)
	for range 10 {
		press(a, "j")
		assertScreen(t, a, 80, 24)
	}
	if last, _ := a.current.Selected(); last.ID != a.current.Rows()[len(a.current.Rows())-1].ID {
		t.Error("j should reach the last task")
	}
}
