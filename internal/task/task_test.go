package task

import (
	"errors"
	"testing"
	"time"
)

func d(s string) Date {
	dt, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return dt
}

var today = d("2026-11-10")

func TestDate(t *testing.T) {
	if !(Date{}).IsZero() || d("2026-01-01").IsZero() {
		t.Error("IsZero")
	}
	if got := d("2026-02-28").AddDays(1).String(); got != "2026-03-01" {
		t.Errorf("AddDays over month end = %s", got)
	}
	if got := d("2026-12-31").AddDays(1).String(); got != "2027-01-01" {
		t.Errorf("AddDays over year end = %s", got)
	}
	if got := d("2028-02-28").AddDays(1).String(); got != "2028-02-29" {
		t.Errorf("leap day = %s", got)
	}
	if !d("2026-11-09").Before(today) || !d("2026-11-11").After(today) || today.Compare(d("2026-11-10")) != 0 {
		t.Error("comparisons")
	}
	if (Date{}).String() != "" {
		t.Error("zero date should format as empty")
	}
	for _, bad := range []string{"2026-13-01", "26-11-10", "2026/11/10", "tomorrow"} {
		if _, err := ParseDate(bad); err == nil {
			t.Errorf("ParseDate(%q) should fail", bad)
		}
	}
	if dt, err := ParseDate(""); err != nil || !dt.IsZero() {
		t.Error("empty string should parse to zero date")
	}
	// DateOf uses the time's own location, not UTC.
	tz := time.FixedZone("UTC-8", -8*3600)
	late := time.Date(2026, 11, 10, 23, 30, 0, 0, tz) // 07:30 UTC on the 11th
	if got := DateOf(late).String(); got != "2026-11-10" {
		t.Errorf("DateOf local = %s", got)
	}
}

func TestValidate(t *testing.T) {
	ok := Task{Name: "Get eggs", Priority: P4}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid task: %v", err)
	}
	bad := map[string]Task{
		"empty name":       {Name: "  ", Priority: P4},
		"multi-line name":  {Name: "a\nb", Priority: P4},
		"carriage return":  {Name: "a\rb", Priority: P4},
		"priority 0":       {Name: "x", Priority: 0},
		"priority 5":       {Name: "x", Priority: 5},
		"negative minutes": {Name: "x", Priority: P1, DurationMin: -5},
	}
	for name, task := range bad {
		if err := task.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func TestProjectNames(t *testing.T) {
	if got := NormalizeProjectName("  #HomeLab-Tech "); got != "homelab-tech" {
		t.Errorf("normalize = %q", got)
	}
	for _, ok := range []string{"oneofftasks", "homelab-tech", "movies_tv", "2026"} {
		if err := ValidateProjectName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "has space", "Caps", "émoji", "a/b", "#x"} {
		if err := ValidateProjectName(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestViews(t *testing.T) {
	done := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name                                 string
		task                                 Task
		today, upcoming, overdue, inbox, cmp bool
	}{
		{"no dates", Task{}, false, false, false, true, false},
		{"due today", Task{Due: today}, true, false, false, true, false},
		{"due yesterday", Task{Due: today.AddDays(-1)}, true, false, true, true, false},
		{"due tomorrow", Task{Due: today.AddDays(1)}, false, true, false, true, false},
		{"due in 7 days", Task{Due: today.AddDays(7)}, false, true, false, true, false},
		{"due in 8 days", Task{Due: today.AddDays(8)}, false, false, false, true, false},
		{"started today", Task{Start: today}, true, false, false, true, false},
		{"started last month, due next month", Task{Start: today.AddDays(-30), Due: today.AddDays(30)}, true, false, false, true, false},
		{"starts tomorrow", Task{Start: today.AddDays(1)}, false, true, false, true, false},
		{"starts in 8 days", Task{Start: today.AddDays(8)}, false, false, false, true, false},
		{"started today, due in 3 days (both lists)", Task{Start: today, Due: today.AddDays(3)}, true, true, false, true, false},
		{"in a project", Task{ProjectID: "p", Due: today}, true, false, false, false, false},
		{"completed", Task{Due: today, CompletedAt: done}, false, false, false, false, true},
		{"trashed", Task{Due: today, DeletedAt: done}, false, false, false, false, false},
		{"trashed and completed", Task{CompletedAt: done, DeletedAt: done}, false, false, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := [5]bool{InToday(tt.task, today), InUpcoming(tt.task, today), InOverdue(tt.task, today), InInbox(tt.task), InCompleted(tt.task)}
			want := [5]bool{tt.today, tt.upcoming, tt.overdue, tt.inbox, tt.cmp}
			if got != want {
				t.Errorf("today/upcoming/overdue/inbox/completed = %v, want %v", got, want)
			}
			if InTrash(tt.task) != tt.task.Trashed() {
				t.Error("InTrash")
			}
		})
	}
	if !InProject(Task{ProjectID: "p"}, "p") || InProject(Task{ProjectID: "p"}, "q") || InProject(Task{}, "p") {
		t.Error("InProject")
	}
}

func TestDueState(t *testing.T) {
	cases := map[Date]DueState{
		{}:                DueNone,
		today.AddDays(-1): DueOverdue,
		today:             DueToday,
		today.AddDays(1):  DueUpcoming,
	}
	for due, want := range cases {
		if got := DueStateOf(Task{Due: due}, today); got != want {
			t.Errorf("due %s: %v, want %v", due, got, want)
		}
	}
}

func names(ts []Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Name
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSort(t *testing.T) {
	base := []Task{
		{ID: "05", Name: "eggs", Priority: P3, Due: d("2026-11-09"), DurationMin: 15},
		{ID: "01", Name: "Insurance", Priority: P1, Due: d("2026-11-10"), Start: d("2026-11-10"), DurationMin: 120},
		{ID: "04", Name: "apples", Priority: P4},
		{ID: "02", Name: "Meditate", Priority: P2, Start: d("2026-11-01"), DurationMin: 30},
		{ID: "03", Name: "apples", Priority: P2, Due: d("2026-11-09")},
	}
	tests := []struct {
		key  SortKey
		want []string
	}{
		{SortName, []string{"apples", "apples", "eggs", "Insurance", "Meditate"}},
		{SortStart, []string{"Meditate", "Insurance", "apples", "apples", "eggs"}},
		{SortDue, []string{"apples", "eggs", "Insurance", "apples", "Meditate"}},
		{SortPriority, []string{"Insurance", "apples", "Meditate", "eggs", "apples"}},
		{SortDuration, []string{"eggs", "Meditate", "Insurance", "apples", "apples"}},
	}
	for _, tt := range tests {
		t.Run(tt.key.String(), func(t *testing.T) {
			ts := append([]Task(nil), base...)
			Sort(ts, tt.key)
			if got := names(ts); !eq(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	// Equal names fall back to ID (creation order), whatever the input order.
	ts := []Task{{ID: "04", Name: "apples"}, {ID: "03", Name: "apples"}}
	Sort(ts, SortName)
	if ts[0].ID != "03" {
		t.Error("tie should break by ID")
	}
}

func TestFilterKeepsOrder(t *testing.T) {
	ts := []Task{{Name: "a"}, {Name: "b", ProjectID: "p"}, {Name: "c"}}
	got := Filter(ts, InInbox)
	if !eq(names(got), []string{"a", "c"}) {
		t.Errorf("Filter = %v", names(got))
	}
}
