package task

import (
	"testing"
	"time"
)

func TestLoadByDay(t *testing.T) {
	d := func(s string) Date { x, _ := ParseDate(s); return x }
	done := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "a", Due: d("2026-11-24"), DurationMin: 60},
		{ID: "b", Due: d("2026-11-24"), DurationMin: 30, CompletedAt: done},
		{ID: "c", Due: d("2026-11-24"), DurationMin: 15},
		{ID: "d", Due: d("2026-11-24"), ArchivedAt: done},  // archived: not counted
		{ID: "e", Start: d("2026-11-24")},                  // start dates don't count
		{ID: "f"},                                          // no due date
		{ID: "g", Due: d("2026-11-10")},                    // overdue
		{ID: "h", Due: d("2026-11-12"), CompletedAt: done}, // done in the past: not overdue
		{ID: "i", Due: d("2026-11-25"), DurationMin: 45},
	}
	got := LoadByDay(tasks)
	if want := (DayLoad{Due: 3, Done: 1, RemainingMin: 75}); got[d("2026-11-24")] != want {
		t.Errorf("2026-11-24 = %+v, want %+v", got[d("2026-11-24")], want)
	}
	if len(got) != 4 {
		t.Errorf("days = %d, want 4: %+v", len(got), got)
	}
	if l := LoadOn(tasks, d("2026-11-24"), "a"); l != (DayLoad{Due: 2, Done: 1, RemainingMin: 15}) {
		t.Errorf("LoadOn skipping a = %+v", l)
	}
	if l := LoadOn(tasks, d("2026-11-24"), ""); l != got[d("2026-11-24")] {
		t.Errorf("LoadOn = %+v, want LoadByDay's %+v", l, got[d("2026-11-24")])
	}
	if l := OverdueLoad(tasks, d("2026-11-17")); l != (DayLoad{Due: 1}) {
		t.Errorf("OverdueLoad = %+v, want 1 due", l)
	}
	if o := (DayLoad{Due: 5, Done: 2}).Open(); o != 3 {
		t.Errorf("Open = %d", o)
	}
}

func TestWeekStart(t *testing.T) {
	for in, want := range map[string]string{
		"2026-11-16": "2026-11-16", // Monday
		"2026-11-17": "2026-11-16",
		"2026-11-22": "2026-11-16", // Sunday ends the week
		"2026-11-23": "2026-11-23",
		"2027-01-01": "2026-12-28", // across a year
	} {
		d, _ := ParseDate(in)
		if got := WeekStart(d).String(); got != want {
			t.Errorf("WeekStart(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestDaysBetween(t *testing.T) {
	a, _ := ParseDate("2026-11-17")
	for _, n := range []int{0, 1, 6, 7, 45, 400} {
		if got := DaysBetween(a, a.AddDays(n)); got != n {
			t.Errorf("DaysBetween(+%d) = %d", n, got)
		}
		if got := DaysBetween(a.AddDays(n), a); got != -n {
			t.Errorf("DaysBetween(-%d) = %d", n, got)
		}
	}
}
