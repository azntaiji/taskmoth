package view

import (
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

func TestDuration(t *testing.T) {
	// Every example in the designs, plus the rounding cases from G3.7.
	cases := map[int]string{
		5: "~5m", 15: "~15m", 45: "~45m", 59: "~59m",
		60: "~1h", 90: "~1.5h", 100: "~1.67h", 120: "~2h", 210: "~3.5h", 225: "~3.75h", 480: "~8h", 1690: "~28.17h",
	}
	for min, want := range cases {
		if got := Duration(min); got != want {
			t.Errorf("Duration(%d) = %q, want %q", min, got, want)
		}
	}
}

func TestClock(t *testing.T) {
	uni, ascii := theme.New(false), theme.New(true)
	tue := time.Date(2026, 11, 10, 15, 26, 0, 0, time.UTC)
	if got := Clock(uni, tue); got != "tues · 2026-11-10 · 15:26" {
		t.Errorf("Clock = %q", got)
	}
	if got := Clock(ascii, tue); got != "tues - 2026-11-10 - 15:26" {
		t.Errorf("ASCII Clock = %q", got)
	}
	want := []string{"sun", "mon", "tues", "wed", "thurs", "fri", "sat"}
	for i, w := range want {
		day := time.Date(2026, 11, 8+i, 9, 5, 0, 0, time.UTC) // 2026-11-08 is a Sunday
		if got := Clock(uni, day)[:len(w)]; got != w {
			t.Errorf("%s: weekday %q, want %q", day.Weekday(), got, w)
		}
	}
	if got := Clock(uni, time.Date(2026, 1, 2, 0, 7, 0, 0, time.UTC)); got[len(got)-5:] != "00:07" {
		t.Errorf("24h time: %q", got)
	}
}

func TestInboxNote(t *testing.T) {
	d := func(s string) task.Date { x, _ := task.ParseDate(s); return x }
	day := d("2026-11-13")
	if first := InboxNote(day); InboxNote(d(day.String())) != first {
		t.Error("the note must stay the same all day")
	}
	seen := map[string]bool{}
	start := d("2026-01-01")
	for i := range 60 {
		seen[InboxNote(start.AddDays(i))] = true
	}
	if len(seen) < 10 {
		t.Errorf("only %d different notes over 60 days", len(seen))
	}
	if len(todayDoneNotes) < 10 || TodayDoneNote(day) == "" {
		t.Error("Today needs its own notes")
	}
	for _, n := range todayDoneNotes {
		if len(n) > 50 { // badge and s:priority share the 80-column status bar
			t.Errorf("Today note too long for the status bar: %q", n)
		}
	}
	// G7.9: Inbox, Someday and projects each have their own lines.
	owner := map[string]string{}
	for name, pool := range map[string][]string{"inbox": inboxNotes, "someday": somedayNotes, "project": projectNotes} {
		if len(pool) < 10 {
			t.Errorf("%s has only %d notes", name, len(pool))
		}
		for _, n := range pool {
			if o, dup := owner[n]; dup {
				t.Errorf("%q is in both %s and %s", n, o, name)
			}
			owner[n] = name
		}
	}
	if n := ProjectNote("work", day); strings.Contains(n, "%") {
		t.Errorf("ProjectNote left a verb: %q", n)
	}
	for _, n := range append(append(inboxNotes, somedayNotes...), projectNotes...) {
		if len(strings.ReplaceAll(n, "%s", "homelab-tech")) > 70 {
			t.Errorf("note too long for 80 columns: %q", n)
		}
	}
}
