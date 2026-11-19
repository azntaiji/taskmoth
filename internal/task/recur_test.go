package task

import (
	"testing"
	"time"
)

func dt(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

var (
	monWed  = Weekdays(0).With(time.Monday).With(time.Wednesday)
	tuesday = Weekdays(0).With(time.Tuesday)
)

func TestRuleNext(t *testing.T) {
	for _, c := range []struct {
		name string
		r    Rule
		from string
		want []string // successive Next dates
	}{
		{"every day", Rule{Every: 1, Unit: Day}, "2026-11-17", []string{"2026-11-18", "2026-11-19"}},
		{"every 3 days", Rule{Every: 3, Unit: Day}, "2026-11-30", []string{"2026-12-03", "2026-12-06"}},
		{"every tue", Rule{Every: 1, Unit: Week, Days: tuesday}, "2026-11-17", []string{"2026-11-24", "2026-12-01"}},
		{"every mon, wed", Rule{Every: 1, Unit: Week, Days: monWed}, "2026-11-16", []string{"2026-11-18", "2026-11-23", "2026-11-25"}},
		{"every other mon, wed", Rule{Every: 2, Unit: Week, Days: monWed}, "2026-11-16", []string{"2026-11-18", "2026-11-30", "2026-12-02", "2026-12-14"}},
		{"every weekday", Rule{Every: 1, Unit: Week, Days: WorkWeek}, "2026-11-20", []string{"2026-11-23", "2026-11-24"}},
		{"every weekend", Rule{Every: 1, Unit: Week, Days: Weekend}, "2026-11-18", []string{"2026-11-21", "2026-11-22", "2026-11-28"}},
		{"every 15th", Rule{Every: 1, Unit: Month, MonthDay: 15}, "2026-11-15", []string{"2026-12-15", "2027-01-15"}},
		{"every 31st clamps", Rule{Every: 1, Unit: Month, MonthDay: 31}, "2027-01-31", []string{"2027-02-28", "2027-03-31", "2027-04-30"}},
		{"every last day", Rule{Every: 1, Unit: Month, MonthDay: Last}, "2027-01-31", []string{"2027-02-28", "2027-03-31"}},
		{"every last day, leap year", Rule{Every: 1, Unit: Month, MonthDay: Last}, "2028-01-31", []string{"2028-02-29"}},
		{"every first mon", Rule{Every: 1, Unit: Month, Nth: 1, Weekday: time.Monday}, "2026-11-02", []string{"2026-12-07", "2027-01-04"}},
		{"every last fri", Rule{Every: 1, Unit: Month, Nth: Last, Weekday: time.Friday}, "2026-11-27", []string{"2026-12-25", "2027-01-29"}},
		{"every 3 months on the 15th", Rule{Every: 3, Unit: Month, MonthDay: 15}, "2026-11-15", []string{"2027-02-15", "2027-05-15"}},
		{"every nov 20", Rule{Every: 1, Unit: Year, Month: 11, MonthDay: 20}, "2026-11-20", []string{"2027-11-20"}},
		{"every feb 29", Rule{Every: 1, Unit: Year, Month: 2, MonthDay: 29}, "2028-02-29", []string{"2029-02-28", "2030-02-28"}},
		{"rescheduled early keeps the month", Rule{Every: 1, Unit: Month, MonthDay: 15}, "2026-11-10", []string{"2026-11-15", "2026-12-15"}},
	} {
		d := dt(c.from)
		for i, w := range c.want {
			d = c.r.Next(d)
			if d.String() != w {
				t.Errorf("%s: Next #%d = %s, want %s", c.name, i+1, d, w)
				break
			}
		}
	}
}

func TestRuleFirstAndAnchor(t *testing.T) {
	today := dt("2026-11-18") // Wednesday
	for _, c := range []struct {
		name string
		r    Rule
		want string
		str  string // String() after Anchor
	}{
		{"every day", Rule{Every: 1, Unit: Day}, "2026-11-18", "every day"},
		{"every week", Rule{Every: 1, Unit: Week}, "2026-11-18", "every wed"},
		{"every other week", Rule{Every: 2, Unit: Week}, "2026-11-18", "every other wed"},
		{"every mon", Rule{Every: 1, Unit: Week, Days: Weekdays(0).With(time.Monday)}, "2026-11-23", "every mon"},
		{"every wed", Rule{Every: 1, Unit: Week, Days: Weekdays(0).With(time.Wednesday)}, "2026-11-18", "every wed"},
		{"every weekday", Rule{Every: 1, Unit: Week, Days: WorkWeek}, "2026-11-18", "every weekday"},
		{"every month", Rule{Every: 1, Unit: Month}, "2026-11-18", "every 18th"},
		{"every 15th", Rule{Every: 1, Unit: Month, MonthDay: 15}, "2026-12-15", "every 15th"},
		{"every last day", Rule{Every: 1, Unit: Month, MonthDay: Last}, "2026-11-30", "every last day"},
		{"every first mon", Rule{Every: 1, Unit: Month, Nth: 1, Weekday: time.Monday}, "2026-12-07", "every first mon"},
		{"every 2 months on the 1st", Rule{Every: 2, Unit: Month, MonthDay: 1}, "2026-12-01", "every other month on the 1st"},
		{"every 3 months", Rule{Every: 3, Unit: Month}, "2026-11-18", "every 3 months on the 18th"},
		{"every year", Rule{Every: 1, Unit: Year}, "2026-11-18", "every nov 18"},
		{"every nov 20", Rule{Every: 1, Unit: Year, Month: 11, MonthDay: 20}, "2026-11-20", "every nov 20"},
		{"every jan 5", Rule{Every: 1, Unit: Year, Month: 1, MonthDay: 5}, "2027-01-05", "every jan 5"},
		{"every 3 days", Rule{Every: 3, Unit: Day}, "2026-11-18", "every 3 days"},
		{"every 3 weeks on mon, wed", Rule{Every: 3, Unit: Week, Days: monWed}, "2026-11-18", "every 3 weeks on mon, wed"},
	} {
		first := c.r.First(today)
		if first.String() != c.want {
			t.Errorf("%s: First = %s, want %s", c.name, first, c.want)
		}
		if got := c.r.Anchor(first).String(); got != c.str {
			t.Errorf("%s: String = %q, want %q", c.name, got, c.str)
		}
	}
}

func TestRuleEncodeRoundTrip(t *testing.T) {
	for _, r := range []Rule{
		{},
		{Every: 1, Unit: Day},
		{Every: 3, Unit: Day},
		{Every: 2, Unit: Week, Days: monWed},
		{Every: 1, Unit: Week, Days: WorkWeek},
		{Every: 1, Unit: Month, MonthDay: 15},
		{Every: 1, Unit: Month, MonthDay: Last},
		{Every: 1, Unit: Month, Nth: Last, Weekday: time.Friday},
		{Every: 2, Unit: Month, Nth: 1, Weekday: time.Monday},
		{Every: 1, Unit: Year, Month: 11, MonthDay: 20},
	} {
		got, err := DecodeRule(r.Encode())
		if err != nil || got != r {
			t.Errorf("round trip of %+v (%q) = %+v, %v", r, r.Encode(), got, err)
		}
	}
	for _, bad := range []string{"x/1", "d", "d/0", "w/1/funday", "m/1/n5:mon", "m/1/d40", "y/1/13-1", "d/1/2/3"} {
		if _, err := DecodeRule(bad); err == nil {
			t.Errorf("DecodeRule(%q) should fail", bad)
		}
	}
}

func TestNextOccurrence(t *testing.T) {
	done := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	weekly := Rule{Every: 1, Unit: Week, Days: tuesday}
	base := Task{ID: "a", Name: "Water plants", Description: "the ferns", ProjectID: "p", Priority: 2,
		Start: dt("2026-11-15"), Due: dt("2026-11-17"), DurationMin: 10, Repeat: weekly, SeriesID: "a",
		CreatedAt: done, UpdatedAt: done, CompletedAt: done}

	// On time: the next Tuesday, start moved by the same 7 days (G15.4).
	n := NextOccurrence(base, dt("2026-11-17"))
	if n.Due != dt("2026-11-24") || n.Start != dt("2026-11-22") {
		t.Errorf("next = due %s start %s", n.Due, n.Start)
	}
	if n.ID != "" || !n.CompletedAt.IsZero() || !n.CreatedAt.IsZero() || n.SeriesID != "a" || n.Repeat != weekly ||
		n.Name != base.Name || n.Description != base.Description || n.ProjectID != "p" || n.Priority != 2 || n.DurationMin != 10 {
		t.Errorf("next copies the wrong fields: %+v", n)
	}

	// Overdue by weeks: skips to the first Tuesday after today (G15.3).
	n = NextOccurrence(base, dt("2026-12-10"))
	if n.Due != dt("2026-12-15") {
		t.Errorf("overdue next = %s, want 2026-12-15", n.Due)
	}

	// Completed early: still the occurrence after the due date.
	base.Due, base.Start = dt("2026-11-24"), Date{}
	if n = NextOccurrence(base, dt("2026-11-18")); n.Due != dt("2026-12-01") || !n.Start.IsZero() {
		t.Errorf("early next = %s (start %s)", n.Due, n.Start)
	}
}

func TestRepeatNeedsDue(t *testing.T) {
	tk := Task{Name: "x", Priority: 4, Repeat: Rule{Every: 1, Unit: Day}}
	if tk.Validate() == nil {
		t.Error("a repeating task without a due date should be invalid")
	}
	tk.Due = dt("2026-11-18")
	if err := tk.Validate(); err != nil {
		t.Error(err)
	}
}

func TestOrdinal(t *testing.T) {
	for n, want := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st", 22: "22nd", 23: "23rd", 31: "31st"} {
		if got := Ordinal(n); got != want {
			t.Errorf("Ordinal(%d) = %s", n, got)
		}
	}
}
