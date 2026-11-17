package task

import (
	"cmp"
	"slices"
	"strconv"
)

// GroupBy is how a list groups its tasks under headers (decisions G13.1–G13.3).
type GroupBy int

// Groupings. The zero value, GroupNone, is every list's default.
const (
	GroupNone GroupBy = iota
	GroupProject
	GroupPriority
	GroupDue
	GroupDuration
)

var groupNames = [...]string{"none", "project", "priority", "due", "duration"}

// String names the grouping as stored in settings and shown in the status
// bar ("o:priority").
func (g GroupBy) String() string {
	if g < 0 || int(g) >= len(groupNames) {
		return groupNames[0]
	}
	return groupNames[g]
}

// ParseGroupBy parses a name produced by GroupBy.String.
func ParseGroupBy(s string) (GroupBy, bool) {
	for i, n := range groupNames {
		if n == s {
			return GroupBy(i), true
		}
	}
	return GroupNone, false
}

// Grouping cycles: the order `o` steps through in each kind of list. Each
// starts with GroupNone, the default.
var (
	// TodayGroups is Today's (everything is due by today, so no due groups).
	TodayGroups = []GroupBy{GroupNone, GroupProject, GroupPriority, GroupDuration}
	// ListGroups is All's, Someday's and Overdue's.
	ListGroups = []GroupBy{GroupNone, GroupProject, GroupPriority, GroupDue, GroupDuration}
	// ProjectGroups is a project list's (one project, so no project groups).
	ProjectGroups = []GroupBy{GroupNone, GroupPriority, GroupDue, GroupDuration}
)

// NextGroup returns the grouping after g in cycle; one not in the cycle
// restarts it.
func NextGroup(cycle []GroupBy, g GroupBy) GroupBy {
	i := slices.Index(cycle, g)
	return cycle[(i+1)%len(cycle)]
}

// Group is the header a task is listed under. Groups are ordered by Rank,
// then by Label.
type Group struct {
	Rank  int
	Label string
}

// Compare orders groups: by Rank, then Label.
func (g Group) Compare(h Group) int {
	if c := cmp.Compare(g.Rank, h.Rank); c != 0 {
		return c
	}
	return cmp.Compare(g.Label, h.Label)
}

// Due-date buckets (G13.3), in order.
var dueGroups = [...]string{"overdue", "today", "tomorrow", "this week", "later", "no due date"}

// Duration buckets (G13.2): upper bounds in minutes, inclusive, shortest first.
var durationGroups = []struct {
	max   int
	label string
}{{15, "up to 15m"}, {30, "up to 30m"}, {60, "up to 1h"}, {120, "up to 2h"}}

// GroupOf returns t's group under by. project names a project ID ("" for the
// Inbox). today places due dates in their buckets.
func GroupOf(t Task, by GroupBy, today Date, project func(ID) string) Group {
	switch by {
	case GroupProject:
		// The Inbox first, then projects A–Z.
		if t.ProjectID == "" {
			return Group{0, "Inbox"}
		}
		return Group{1, "#" + project(t.ProjectID)}
	case GroupPriority:
		return Group{t.Priority, "P" + strconv.Itoa(t.Priority)}
	case GroupDue:
		i := dueBucket(t.Due, today)
		return Group{i, dueGroups[i]}
	case GroupDuration:
		if t.DurationMin <= 0 {
			return Group{len(durationGroups) + 1, "no estimate"}
		}
		for i, b := range durationGroups {
			if t.DurationMin <= b.max {
				return Group{i, b.label}
			}
		}
		return Group{len(durationGroups), "over 2h"}
	}
	return Group{}
}

// dueBucket is the index into dueGroups for a due date: before today,
// today, tomorrow, within the next 7 days, after that, or none.
func dueBucket(due, today Date) int {
	switch {
	case due.IsZero():
		return 5
	case due.Before(today):
		return 0
	case due == today:
		return 1
	case due == today.AddDays(1):
		return 2
	case !due.After(today.AddDays(UpcomingDays)):
		return 3
	}
	return 4
}
