package task

import (
	"slices"
	"testing"
)

func TestGroupOf(t *testing.T) {
	today := Date{2026, 11, 10}
	names := func(id ID) string { return map[ID]string{"w": "work", "c": "chores"}[id] }
	tests := []struct {
		name string
		task Task
		by   GroupBy
		want Group
	}{
		{"inbox", Task{}, GroupProject, Group{0, "Inbox"}},
		{"project", Task{ProjectID: "w"}, GroupProject, Group{1, "#work"}},
		{"priority", Task{Priority: 2}, GroupPriority, Group{2, "P2"}},
		{"overdue", Task{Due: today.AddDays(-1)}, GroupDue, Group{0, "overdue"}},
		{"due today", Task{Due: today}, GroupDue, Group{1, "today"}},
		{"tomorrow", Task{Due: today.AddDays(1)}, GroupDue, Group{2, "tomorrow"}},
		{"in 2 days", Task{Due: today.AddDays(2)}, GroupDue, Group{3, "this week"}},
		{"in 7 days", Task{Due: today.AddDays(7)}, GroupDue, Group{3, "this week"}},
		{"in 8 days", Task{Due: today.AddDays(8)}, GroupDue, Group{4, "later"}},
		{"no due date", Task{}, GroupDue, Group{5, "no due date"}},
		{"15m", Task{DurationMin: 15}, GroupDuration, Group{0, "up to 15m"}},
		{"16m", Task{DurationMin: 16}, GroupDuration, Group{1, "up to 30m"}},
		{"1h", Task{DurationMin: 60}, GroupDuration, Group{2, "up to 1h"}},
		{"2h", Task{DurationMin: 120}, GroupDuration, Group{3, "up to 2h"}},
		{"3h", Task{DurationMin: 180}, GroupDuration, Group{4, "over 2h"}},
		{"no estimate", Task{}, GroupDuration, Group{5, "no estimate"}},
		{"none", Task{Priority: 1}, GroupNone, Group{}},
	}
	for _, tt := range tests {
		if got := GroupOf(tt.task, tt.by, today, names); got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
	}

	// Projects: the Inbox first, then A–Z.
	gs := []Group{GroupOf(Task{ProjectID: "w"}, GroupProject, today, names), GroupOf(Task{}, GroupProject, today, names),
		GroupOf(Task{ProjectID: "c"}, GroupProject, today, names)}
	slices.SortFunc(gs, Group.Compare)
	if gs[0].Label != "Inbox" || gs[1].Label != "#chores" || gs[2].Label != "#work" {
		t.Errorf("project group order = %v", gs)
	}
}

func TestGroupCyclesAndNames(t *testing.T) {
	walk := func(cycle []GroupBy) []string {
		var out []string
		g := cycle[0]
		for range len(cycle) + 1 {
			out = append(out, g.String())
			g = NextGroup(cycle, g)
		}
		return out
	}
	if got := walk(TodayGroups); !slices.Equal(got, []string{"none", "project", "priority", "duration", "none"}) {
		t.Errorf("Today cycle = %v", got)
	}
	if got := walk(ListGroups); !slices.Equal(got, []string{"none", "project", "priority", "due", "duration", "none"}) {
		t.Errorf("list cycle = %v", got)
	}
	if got := walk(ProjectGroups); !slices.Equal(got, []string{"none", "priority", "due", "duration", "none"}) {
		t.Errorf("project cycle = %v", got)
	}
	if NextGroup(ProjectGroups, GroupProject) != GroupNone {
		t.Error("a grouping not in the cycle should restart it")
	}
	for _, g := range ListGroups {
		if got, ok := ParseGroupBy(g.String()); !ok || got != g {
			t.Errorf("ParseGroupBy(%q) = %v, %v", g.String(), got, ok)
		}
	}
	if _, ok := ParseGroupBy("bogus"); ok {
		t.Error("ParseGroupBy(bogus) should fail")
	}
}
