package nlp

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/azntaiji/taskmoth/internal/task"
)

// today is Tue 2026-11-10, as in PLAN.md Appendix A.
var today = task.Date{Year: 2026, Month: 11, Day: 10}

func d(s string) task.Date {
	if s == "" {
		return task.Date{}
	}
	dt, err := task.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return dt
}

// TestAppendixA is the approved example table (decision G5.1), row for row.
func TestAppendixA(t *testing.T) {
	tests := []struct {
		row               int
		in, name, project string
		due, start        string
		minutes, priority int
	}{
		{1, "Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow", "Change the laundry", "oneofftasks", "2026-11-12", "2026-11-11", 30, 2},
		{2, "Buy milk", "Buy milk", "", "", "", 0, 0},
		{3, "Buy milk tomorrow", "Buy milk", "", "2026-11-11", "", 0, 0},
		{4, "Call mom tomorrow at 4pm", "Call mom at 4pm", "", "2026-11-11", "", 0, 0},
		{5, "Submit report friday", "Submit report", "", "2026-11-13", "", 0, 0},
		{6, "Submit report next friday", "Submit report", "", "2026-11-20", "", 0, 0},
		{7, "Plan sprint next monday", "Plan sprint", "", "2026-11-16", "", 0, 0},
		{8, "Water plants tuesday", "Water plants", "", "2026-11-17", "", 0, 0},
		{9, "Clean garage next week", "Clean garage", "", "2026-11-16", "", 0, 0},
		{10, "Pay rent next month", "Pay rent", "", "2026-12-01", "", 0, 0},
		{11, "Hike this weekend", "Hike", "", "2026-11-14", "", 0, 0},
		{12, "Visit gran next weekend", "Visit gran", "", "2026-11-21", "", 0, 0},
		{13, "Renew passport in 3 weeks", "Renew passport", "", "2026-12-01", "", 0, 0},
		{14, "Dentist in a week", "Dentist", "", "2026-11-17", "", 0, 0},
		{15, "File taxes 2027-04-15", "File taxes", "", "2027-04-15", "", 0, 0},
		{16, "Gift for sam 11/20", "Gift for sam", "", "2026-11-20", "", 0, 0},
		{17, "Concert 27/1", "Concert", "", "2027-01-27", "", 0, 0},
		{18, "Party nov 20", "Party", "", "2026-11-20", "", 0, 0},
		{19, "Trip 20 november", "Trip", "", "2026-11-20", "", 0, 0},
		{20, "Renew lease jan 5", "Renew lease", "", "2027-01-05", "", 0, 0},
		{21, "Pay card 27th", "Pay card", "", "2026-11-27", "", 0, 0},
		{22, "Budget end of month", "Budget", "", "2026-11-30", "", 0, 0},
		{23, "Review mid december", "Review", "", "2026-12-15", "", 0, 0},
		{24, "Order cake 2 days before nov 20", "Order cake", "", "2026-11-18", "", 0, 0},
		{25, "Fix login bug p1", "Fix login bug", "", "", "", 0, 1},
		{26, "Fix login bug !p3", "Fix login bug", "", "", "", 0, 3},
		{27, "Read ~1.5h", "Read", "", "", "", 90, 0},
		{28, "Read ~1h30m", "Read", "", "", "", 90, 0},
		{29, "Read ~2 hours", "Read", "", "", "", 120, 0},
		{30, "Walk ~10 minutes", "Walk", "", "", "", 10, 0},
		{31, "Prep slides s:monday @friday", "Prep slides", "", "2026-11-13", "2026-11-16", 0, 0},
		{32, "Watch Tomorrowland", "Watch Tomorrowland", "", "", "", 0, 0},
		{33, "Buy sun cream", "Buy sun cream", "", "", "", 0, 0},
		{34, "Run @sun", "Run", "", "2026-11-15", "", 0, 0},
		{35, "Call dad @tom", "Call dad", "", "2026-11-11", "", 0, 0},
		{36, "May 5k race", "May 5k race", "", "", "", 0, 0},
		{37, "Pay may 5 bill", "Pay bill", "", "2027-05-05", "", 0, 0},
		{38, "Learn C# basics", "Learn C# basics", "", "", "", 0, 0},
		{39, "Email @ 5", "Email @ 5", "", "", "", 0, 0},
		{40, `Fix \friday bug`, "Fix friday bug", "", "", "", 0, 0},
		{41, "Plan #work #home", "Plan #work", "home", "", "", 0, 0},
		{42, "Groceries tomorrow friday", "Groceries tomorrow", "", "2026-11-13", "", 0, 0},
		{43, "Report @2026-02-30", "Report @2026-02-30", "", "", "", 0, 0},
		{44, "#Work standup", "standup", "work", "", "", 0, 0},
		{45, "p5 test", "p5 test", "", "", "", 0, 0},
		{46, "#work", "", "work", "", "", 0, 0},
	}
	for _, tt := range tests {
		r := Parse(tt.in, today)
		if r.Name != tt.name || r.Project != tt.project || r.Due != d(tt.due) || r.Start != d(tt.start) ||
			r.DurationMin != tt.minutes || r.Priority != tt.priority {
			t.Errorf("row %d %q:\n got  name=%q project=%q due=%s start=%s min=%d pri=%d\n want name=%q project=%q due=%s start=%s min=%d pri=%d",
				tt.row, tt.in, r.Name, r.Project, r.Due, r.Start, r.DurationMin, r.Priority,
				tt.name, tt.project, tt.due, tt.start, tt.minutes, tt.priority)
		}
	}
}

// TestAppendixARepeating is Appendix A's repeating examples (G15.7), row
// for row. Repeat is the anchored rule as shown (Rule.String).
func TestAppendixARepeating(t *testing.T) {
	tests := []struct {
		row                       int
		in, name, project, due, r string
		priority                  int
	}{
		{1, "Water plants every day", "Water plants", "", "2026-11-10", "every day", 0},
		{2, "Standup every weekday", "Standup", "", "2026-11-10", "every weekday", 0},
		{3, "Team sync every monday", "Team sync", "", "2026-11-16", "every mon", 0},
		{4, "Gym every mon, wed and fri", "Gym", "", "2026-11-11", "every mon, wed, fri", 0},
		{5, "Bins every other thursday", "Bins", "", "2026-11-12", "every other thu", 0},
		{6, "Backups every 3 days", "Backups", "", "2026-11-10", "every 3 days", 0},
		{7, "Pay rent every month", "Pay rent", "", "2026-11-10", "every 10th", 0},
		{8, "Pay card every 15th", "Pay card", "", "2026-11-15", "every 15th", 0},
		{9, "Budget every last day", "Budget", "", "2026-11-30", "every last day", 0},
		{10, "Book club every first mon", "Book club", "", "2026-12-07", "every first mon", 0},
		{11, "Anniversary every nov 20", "Anniversary", "", "2026-11-20", "every nov 20", 0},
		{12, "Review weekly", "Review", "", "2026-11-10", "every tue", 0},
		{13, "Daily standup", "standup", "", "2026-11-10", "every day", 0},
		{14, "Water ferns ev 2 weeks", "Water ferns", "", "2026-11-10", "every other tue", 0},
		{15, "Taxes every 3 months starting jan 1", "Taxes", "", "2027-01-01", "every 3 months on the 1st", 0},
		{16, "Stretch every day starting tomorrow", "Stretch", "", "2026-11-11", "every day", 0},
		{17, "Inspect every other month on the 1st", "Inspect", "", "2026-12-01", "every other month on the 1st", 0},
		{18, "Clean every 2 weeks on sat, sun", "Clean", "", "2026-11-14", "every other weekend", 0},
		{19, "Call mom every sunday #family p2", "Call mom", "family", "2026-11-15", "every sun", 2},
		{20, "Every time I travel", "Every time I travel", "", "", "", 0},
		{21, "Pay every! 3 days", "Pay every! 3 days", "", "", "", 0},
		{22, `Walk \every \monday`, "Walk every monday", "", "", "", 0},
	}
	for _, tt := range tests {
		r := Parse(tt.in, today)
		if r.Name != tt.name || r.Project != tt.project || r.Due != d(tt.due) || r.Repeat.String() != tt.r || r.Priority != tt.priority {
			t.Errorf("row %d %q:\n got  name=%q project=%q due=%s repeat=%q pri=%d\n want name=%q project=%q due=%s repeat=%q pri=%d",
				tt.row, tt.in, r.Name, r.Project, r.Due, r.Repeat, r.Priority, tt.name, tt.project, tt.due, tt.r, tt.priority)
		}
	}
}

// Every rule's String reads back as the same rule (the edit dialog starts
// with it, G15.2).
func TestRepeatRoundTrip(t *testing.T) {
	for _, in := range []string{"every day", "every weekday", "every mon, wed, fri", "every other thu", "every 3 days",
		"every 15th", "every last day", "every first mon", "every last fri", "every nov 20", "every other tue",
		"every 3 months on the 1st", "every other month on the last day", "every other year on feb 29",
		"every 3 weeks on mon, wed", "every other weekend", "every 4 months on the second wed"} {
		r, first, ok := ParseRepeat(in, today)
		if !ok {
			t.Errorf("ParseRepeat(%q) failed", in)
			continue
		}
		if r.String() != in {
			t.Errorf("ParseRepeat(%q).String() = %q", in, r.String())
		}
		again, first2, ok := ParseRepeat(r.String(), first)
		if !ok || again != r || first2 != first {
			t.Errorf("%q doesn't read back: %+v %s vs %+v %s", in, again, first2, r, first)
		}
	}
	for _, bad := range []string{"", "every", "every 3", "every blue moon", "every day please", "tomorrow", "every 0 days"} {
		if _, _, ok := ParseRepeat(bad, today); ok {
			t.Errorf("ParseRepeat(%q) should fail", bad)
		}
	}
}

func TestRepeatSpanAndToggle(t *testing.T) {
	in := "Gym every mon and wed #health"
	r := Parse(in, today)
	rs := []rune(in)
	for _, s := range r.Spans {
		if s.Kind == KindRepeat && string(rs[s.Start:s.End]) != "every mon and wed" {
			t.Errorf("repeat span = %q", string(rs[s.Start:s.End]))
		}
	}
	out, _, ok := ToggleAt(in, 6, today)
	if !ok || out != `Gym \every \mon \and \wed #health` {
		t.Errorf("ToggleAt = %q", out)
	}
	if r := Parse(out, today); !r.Repeat.IsZero() || r.Name != "Gym every mon and wed" {
		t.Errorf("escaped repeat parsed as %+v", r)
	}
}

func TestSpans(t *testing.T) {
	in := "Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow"
	r := Parse(in, today)
	rs := []rune(in)
	got := map[Kind]string{}
	for _, s := range r.Spans {
		got[s.Kind] = string(rs[s.Start:s.End])
	}
	want := map[Kind]string{KindProject: "#oneofftasks", KindDue: "@in 2 days", KindDuration: "~30m", KindPriority: "!p2", KindStart: "s:tomorrow"}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("span %v = %q, want %q", k, got[k], w)
		}
	}
	if !r.Has(KindDue) || Parse("Buy milk", today).Has(KindDue) {
		t.Error("Has")
	}
}

func TestParseDate(t *testing.T) {
	cases := map[string]string{
		"2026-11-20": "2026-11-20", "today": "2026-11-10", "tod": "2026-11-10", "tom": "2026-11-11",
		"yesterday": "2026-11-09", "wed": "2026-11-11", "Next Tues": "2026-11-17", "this friday": "2026-11-13",
		"in 2 months": "2027-01-10", "in 1 year": "2027-11-10", "next year": "2027-01-01",
		"eom": "2026-11-30", "end of february": "2027-02-28", "dec": "2026-12-01", "jan": "2027-01-01",
		"3 weeks after 2026-12-01": "2026-12-22", "1 month before dec 31": "2026-11-30",
		"11/20/27": "2027-11-20", "2/29/2028": "2028-02-29", "20th november 2027": "2027-11-20",
		"31st": "2026-12-31", "10th": "2026-11-10", "9th": "2026-12-09", "10 nov": "2026-11-10",
	}
	for in, want := range cases {
		got, ok := ParseDate(in, today)
		if !ok || got.String() != want {
			t.Errorf("ParseDate(%q) = %s, %v; want %s", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "2/30", "2/29/2027", "13/13", "someday", "every monday", "tomorrow at 4pm", "31 feb"} {
		if got, ok := ParseDate(in, today); ok {
			t.Errorf("ParseDate(%q) = %s, want no match", in, got)
		}
	}
	for _, in := range []string{"no date", "No due date"} {
		if got, ok := ParseDate(in, today); !ok || !got.IsZero() {
			t.Errorf("ParseDate(%q) should clear the date", in)
		}
	}
	// End of month clamps: Jan 31 + 1 month is Feb 28.
	if got, _ := ParseDate("in 1 month", task.Date{Year: 2027, Month: 1, Day: 31}); got.String() != "2027-02-28" {
		t.Errorf("month clamp = %s", got)
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]int{
		"30m": 30, "30min": 30, "30 mins": 30, "30 minutes": 30, "1.5h": 90, "2h": 120, "2 hours": 120,
		"1 hour": 60, "1h30m": 90, "1h 30m": 90, "1h30": 90, "90": 90, "0.25h": 15, "1.33h": 80,
	}
	for in, want := range cases {
		if got, ok := ParseDuration(in); !ok || got != want {
			t.Errorf("ParseDuration(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "0", "0m", "abc", "2 days", "-5m", "1h30x"} {
		if got, ok := ParseDuration(in); ok {
			t.Errorf("ParseDuration(%q) = %d, want no match", in, got)
		}
	}
}

func TestToggleAt(t *testing.T) {
	in := "Pay may 5 bill"
	// Cursor right after "5": the "may 5" date is escaped word by word.
	out, cur, ok := ToggleAt(in, 9, today)
	if !ok || out != `Pay \may \5 bill` || cur != 11 {
		t.Fatalf("escape: %q cursor %d ok %v", out, cur, ok)
	}
	if r := Parse(out, today); !r.Due.IsZero() || r.Name != "Pay may 5 bill" {
		t.Errorf("escaped text should parse as plain: %+v", r)
	}
	// Toggling again on the escaped run restores it.
	back, cur2, ok := ToggleAt(out, cur, today)
	if !ok || back != in || cur2 != 9 {
		t.Errorf("unescape: %q cursor %d ok %v", back, cur2, ok)
	}
	if _, _, ok := ToggleAt("Buy milk", 3, today); ok {
		t.Error("nothing to toggle")
	}
	// Multi-word phrases are escaped as a whole, so no part re-matches.
	out, _, _ = ToggleAt("Fix next friday", 15, today)
	if r := Parse(out, today); !r.Due.IsZero() {
		t.Errorf("%q still has a due date", out)
	}
}

// FuzzParse checks the parser never panics, spans stay in range, and the
// name only contains words from the input.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow", `Fix \friday bug`, "~", "@", "#", "s:",
		"2 days before 2 days before nov 20", "p1 p2 p3", "日本語 tomorrow", "\x00\xff", "  lots   of   space  ",
		"every mon, , and wed and", "ev other 3 weeks on the last", "every 999 years starting every day", "every , mon",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if !utf8.ValidString(in) {
			return
		}
		r := Parse(in, today)
		n := len([]rune(in))
		for _, s := range r.Spans {
			if s.Start < 0 || s.End > n || s.Start >= s.End {
				t.Fatalf("bad span %+v for %q", s, in)
			}
		}
		words := map[string]bool{}
		for _, w := range strings.Fields(in) {
			words[w] = true
			words[strings.TrimPrefix(w, `\`)] = true
		}
		for _, w := range strings.Fields(r.Name) {
			if !words[w] {
				t.Fatalf("name word %q not in input %q", w, in)
			}
		}
		if !r.Repeat.IsZero() {
			if r.Due.IsZero() || !r.Repeat.Next(r.Due).After(r.Due) {
				t.Fatalf("repeat %+v with due %s for %q", r.Repeat, r.Due, in)
			}
		}
		_, _, _ = ToggleAt(in, n/2, today)
	})
}
