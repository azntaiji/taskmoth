package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Unit is the period a Rule repeats over.
type Unit int

// Units.
const (
	Day Unit = iota
	Week
	Month
	Year
)

var unitNames = [...]string{"day", "week", "month", "year"}

// Weekdays is a set of days of the week, bit i for time.Weekday(i).
type Weekdays uint8

// Weekday sets.
const (
	WorkWeek Weekdays = 1<<time.Monday | 1<<time.Tuesday | 1<<time.Wednesday | 1<<time.Thursday | 1<<time.Friday
	Weekend  Weekdays = 1<<time.Saturday | 1<<time.Sunday
)

// Has reports whether d is in the set.
func (w Weekdays) Has(d time.Weekday) bool { return w&(1<<d) != 0 }

// With returns the set plus d.
func (w Weekdays) With(d time.Weekday) Weekdays { return w | 1<<d }

// Last is the Nth value meaning "the last" (every last fri) and the
// MonthDay value meaning "the last day".
const Last = -1

// Rule is how a task repeats (decisions G15.1–G15.7). The zero Rule means
// it doesn't. A Rule is a value, so tasks stay comparable.
//
//   - Day: every Every days.
//   - Week: on Days, every Every weeks.
//   - Month: on MonthDay (1–31, or Last), or on the Nth (1–4, or Last)
//     Weekday, every Every months.
//   - Year: on Month and MonthDay, every Every years.
//
// A rule as parsed may leave the day unsaid ("every week"); Anchor fills it
// in from the first due date, so stored rules always say when.
type Rule struct {
	Every    int
	Unit     Unit
	Days     Weekdays
	MonthDay int
	Nth      int
	Weekday  time.Weekday
	Month    time.Month
}

// IsZero reports whether the rule is unset (the task doesn't repeat).
func (r Rule) IsZero() bool { return r.Every == 0 }

// Anchor fills in what the rule leaves unsaid from its first occurrence:
// "every week" from Tuesday 2026-11-17 becomes every Tuesday.
func (r Rule) Anchor(first Date) Rule {
	switch r.Unit {
	case Week:
		if r.Days == 0 {
			r.Days = r.Days.With(first.Weekday())
		}
	case Month:
		if r.MonthDay == 0 && r.Nth == 0 {
			r.MonthDay = first.Day
		}
	case Year:
		if r.Month == 0 {
			r.Month, r.MonthDay = first.Month, first.Day
		}
	}
	return r
}

// specific reports whether the rule names its days, so that only some dates
// are occurrences. Unsaid rules occur on any day until anchored.
func (r Rule) specific() bool {
	switch r.Unit {
	case Week:
		return r.Days != 0
	case Month:
		return r.MonthDay != 0 || r.Nth != 0
	case Year:
		return r.Month != 0
	}
	return false
}

// on reports whether d falls on one of the rule's days, ignoring Every.
func (r Rule) on(d Date) bool {
	switch r.Unit {
	case Week:
		return r.Days.Has(d.Weekday())
	case Month:
		return r.inMonth(d.Year, d.Month) == d
	case Year:
		return d.Month == r.Month && d.Day == clampDay(d.Year, d.Month, r.MonthDay)
	}
	return true
}

// First is the first occurrence on or after from.
func (r Rule) First(from Date) Date {
	if !r.specific() {
		return from
	}
	for d, i := from, 0; i < 366*8; d, i = d.AddDays(1), i+1 {
		if r.on(d) {
			return d
		}
	}
	return from // unreachable for valid rules
}

// Next is the occurrence after d, counting Every periods from d's own week,
// month or year, so a rescheduled occurrence keeps its rhythm from there.
func (r Rule) Next(d Date) Date {
	n := max(r.Every, 1)
	switch r.Unit {
	case Week:
		if r.Days == 0 {
			return d.AddDays(7 * n)
		}
		end := WeekStart(d).AddDays(6)
		for x := d.AddDays(1); !x.After(end); x = x.AddDays(1) {
			if r.Days.Has(x.Weekday()) {
				return x
			}
		}
		return r.First(WeekStart(d).AddDays(7 * n))
	case Month:
		if !r.specific() {
			return AddMonths(d, n)
		}
		if c := r.inMonth(d.Year, d.Month); c.After(d) {
			return c
		}
		m := AddMonths(Date{d.Year, d.Month, 1}, n)
		return r.inMonth(m.Year, m.Month)
	case Year:
		if !r.specific() {
			return AddMonths(d, 12*n)
		}
		if c := (Date{d.Year, r.Month, clampDay(d.Year, r.Month, r.MonthDay)}); c.After(d) {
			return c
		}
		y := d.Year + n
		return Date{y, r.Month, clampDay(y, r.Month, r.MonthDay)}
	}
	return d.AddDays(n)
}

// inMonth is the month rule's date in year y, month m.
func (r Rule) inMonth(y int, m time.Month) Date {
	if r.Nth == 0 {
		return Date{y, m, clampDay(y, m, r.MonthDay)}
	}
	if r.Nth == Last {
		d := Date{y, m, DaysIn(y, m)}
		for d.Weekday() != r.Weekday {
			d = d.AddDays(-1)
		}
		return d
	}
	d := Date{y, m, 1}
	for d.Weekday() != r.Weekday {
		d = d.AddDays(1)
	}
	return d.AddDays(7 * (r.Nth - 1))
}

// clampDay is day in month m of year y, or the month's last day if it's
// shorter (every 31st falls on 30 Nov) or day is Last.
func clampDay(y int, m time.Month, day int) int {
	n := DaysIn(y, m)
	if day == Last || day > n {
		return n
	}
	return day
}

// DaysIn is the number of days in month m of year y.
func DaysIn(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// AddMonths adds n months, clamping the day to the target month's length.
func AddMonths(d Date, n int) Date {
	t := time.Date(d.Year, d.Month+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	y, m := t.Year(), t.Month()
	return Date{y, m, min(d.Day, DaysIn(y, m))}
}

// NextOccurrence is the task that follows t when t is completed on today
// (G15.3, G15.4): a copy with the same rule, series, name, description,
// project, priority and duration, due on the rule's next date after t's due
// date, or after today if t was overdue. The start date moves by as many
// days as the due date. ID, timestamps and completion are cleared.
func NextOccurrence(t Task, today Date) Task {
	next := t.Repeat.Next(t.Due)
	for i := 0; !next.After(today) && i < 10000; i++ {
		next = t.Repeat.Next(next)
	}
	n := t
	n.ID, n.CreatedAt, n.UpdatedAt, n.CompletedAt, n.ArchivedAt = "", time.Time{}, time.Time{}, time.Time{}, time.Time{}
	if !t.Start.IsZero() {
		n.Start = t.Start.AddDays(DaysBetween(t.Due, next))
	}
	n.Due = next
	return n
}

// --- text ---

var dayNames = [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// mondayFirst lists the weekdays in display order.
var mondayFirst = [...]time.Weekday{1, 2, 3, 4, 5, 6, 0}

var ordinals = map[int]string{1: "first", 2: "second", 3: "third", 4: "fourth", Last: "last"}

// Ordinal is "1st", "2nd", "23rd", "11th".
func Ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

// String is the rule as shown, and as the edit dialog's starting text, in
// words the quick-add grammar reads back (G15.7, G15.8): "every day",
// "every other mon, wed", "every weekday", "every 15th", "every first mon",
// "every 3 months on the last day", "every nov 20".
func (r Rule) String() string {
	if r.IsZero() {
		return ""
	}
	spec, unit := r.spec(), unitNames[r.Unit]
	switch {
	case r.Every == 1 && spec == "":
		return "every " + unit
	case r.Every == 1:
		return "every " + spec
	case r.Every == 2 && spec == "":
		return "every other " + unit
	case r.Every == 2 && r.Unit == Week:
		return "every other " + spec
	case r.Every == 2:
		return "every other " + unit + " on " + spec
	case spec == "":
		return fmt.Sprintf("every %d %ss", r.Every, unit)
	}
	return fmt.Sprintf("every %d %ss on %s", r.Every, unit, spec)
}

// spec is the rule's days in words, or "" when unsaid: "mon, wed",
// "weekday", "15th", "last day", "first mon", "nov 20". After "on", month
// days take "the" ("every 2 months on the 15th").
func (r Rule) spec() string {
	the := ""
	if r.Every > 1 {
		the = "the "
	}
	switch r.Unit {
	case Week:
		switch r.Days {
		case 0:
			return ""
		case WorkWeek:
			return "weekday"
		case Weekend:
			return "weekend"
		}
		var days []string
		for _, d := range mondayFirst {
			if r.Days.Has(d) {
				days = append(days, dayNames[d])
			}
		}
		return strings.Join(days, ", ")
	case Month:
		switch {
		case r.Nth != 0:
			return the + ordinals[r.Nth] + " " + dayNames[r.Weekday]
		case r.MonthDay == Last:
			return the + "last day"
		case r.MonthDay != 0:
			return the + Ordinal(r.MonthDay)
		}
	case Year:
		if r.Month != 0 {
			return strings.ToLower(r.Month.String()[:3]) + " " + strconv.Itoa(r.MonthDay)
		}
	}
	return ""
}

// Encode is the rule's storage form (G15.1), e.g. "w/1/mon,wed",
// "m/2/d15", "m/1/n1:mon", "m/1/d-1", "y/1/11-20", "d/3". Decode reads it.
func (r Rule) Encode() string {
	if r.IsZero() {
		return ""
	}
	s := fmt.Sprintf("%c/%d", unitNames[r.Unit][0], r.Every)
	switch r.Unit {
	case Week:
		var days []string
		for _, d := range mondayFirst {
			if r.Days.Has(d) {
				days = append(days, dayNames[d])
			}
		}
		s += "/" + strings.Join(days, ",")
	case Month:
		if r.Nth != 0 {
			s += fmt.Sprintf("/n%d:%s", r.Nth, dayNames[r.Weekday])
		} else {
			s += fmt.Sprintf("/d%d", r.MonthDay)
		}
	case Year:
		s += fmt.Sprintf("/%d-%d", r.Month, r.MonthDay)
	}
	return s
}

// DecodeRule reads Encode's form. "" is the zero Rule.
func DecodeRule(s string) (Rule, error) {
	if s == "" {
		return Rule{}, nil
	}
	bad := func() (Rule, error) { return Rule{}, fmt.Errorf("invalid repeat rule %q", s) }
	parts := strings.Split(s, "/")
	if len(parts) < 2 {
		return bad()
	}
	var r Rule
	switch parts[0] {
	case "d":
		r.Unit = Day
	case "w":
		r.Unit = Week
	case "m":
		r.Unit = Month
	case "y":
		r.Unit = Year
	default:
		return bad()
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 1 {
		return bad()
	}
	r.Every = n
	arg := ""
	if len(parts) == 3 {
		arg = parts[2]
	} else if len(parts) > 3 {
		return bad()
	}
	day := func(name string) (time.Weekday, bool) {
		for i, d := range dayNames {
			if d == name {
				return time.Weekday(i), true
			}
		}
		return 0, false
	}
	switch r.Unit {
	case Week:
		for _, name := range strings.Split(arg, ",") {
			if name == "" {
				continue
			}
			d, ok := day(name)
			if !ok {
				return bad()
			}
			r.Days = r.Days.With(d)
		}
	case Month:
		switch {
		case strings.HasPrefix(arg, "n"):
			nth, name, ok := strings.Cut(arg[1:], ":")
			k, err := strconv.Atoi(nth)
			d, okDay := day(name)
			if !ok || err != nil || !okDay || (k != Last && (k < 1 || k > 4)) {
				return bad()
			}
			r.Nth, r.Weekday = k, d
		case strings.HasPrefix(arg, "d"):
			k, err := strconv.Atoi(arg[1:])
			if err != nil || (k != Last && (k < 1 || k > 31)) {
				return bad()
			}
			r.MonthDay = k
		case arg != "":
			return bad()
		}
	case Year:
		if arg != "" {
			m, d, ok := strings.Cut(arg, "-")
			mi, err1 := strconv.Atoi(m)
			di, err2 := strconv.Atoi(d)
			if !ok || err1 != nil || err2 != nil || mi < 1 || mi > 12 || di < 1 || di > 31 {
				return bad()
			}
			r.Month, r.MonthDay = time.Month(mi), di
		}
	default:
		if arg != "" {
			return bad()
		}
	}
	return r, nil
}
