// Package nlp parses taskmoth's natural-language input: quick-add text
// (Parse), date phrases (ParseDate) and durations (ParseDuration). The
// grammar is PLAN.md Appendix A (decision G5.1). Nothing here reads the
// clock; "today" is always passed in.
package nlp

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
)

// ParseDate parses a whole date phrase as typed after @ or s:, or into an
// edit dialog (short forms allowed). ok is false if the phrase isn't a date.
// "no date" parses as the zero Date with ok true.
func ParseDate(s string, today task.Date) (task.Date, bool) {
	ws := strings.Fields(strings.ToLower(s))
	if len(ws) == 0 {
		return task.Date{}, false
	}
	return parseDate(ws, today, true)
}

var months = map[string]time.Month{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4,
	"may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8,
	"sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11,
	"dec": 12, "december": 12,
}

// fullWeekdays may be used bare; shortWeekdays need a sigil when alone
// (decision G5.1f).
var (
	fullWeekdays = map[string]time.Weekday{
		"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4, "friday": 5, "saturday": 6,
	}
	shortWeekdays = map[string]time.Weekday{
		"sun": 0, "mon": 1, "tue": 2, "tues": 2, "wed": 3, "weds": 3, "thu": 4, "thur": 4, "thurs": 4, "fri": 5, "sat": 6,
	}
)

func weekday(w string, sigil bool) (time.Weekday, bool) {
	if d, ok := fullWeekdays[w]; ok {
		return d, true
	}
	if d, ok := shortWeekdays[w]; ok && sigil {
		return d, true
	}
	return 0, false
}

// anyWeekday accepts short forms too; used after "next"/"this".
func anyWeekday(w string) (time.Weekday, bool) { return weekday(w, true) }

func wd(d task.Date) time.Weekday {
	return time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC).Weekday()
}

// monday is the Monday of d's week (weeks start on Monday).
func monday(d task.Date) task.Date { return d.AddDays(-((int(wd(d)) + 6) % 7)) }

// nextWeekday is the first such weekday after today (today's own weekday
// means next week's; Appendix A).
func nextWeekday(today task.Date, w time.Weekday) task.Date {
	delta := (int(w) - int(wd(today)) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	return today.AddDays(delta)
}

// nextWeekWeekday is Todoist's "next <weekday>": that weekday in the
// following Monday–Sunday week.
func nextWeekWeekday(today task.Date, w time.Weekday) task.Date {
	return monday(today).AddDays(7 + (int(w)+6)%7)
}

// saturday is the coming Saturday (today if it is Saturday).
func saturday(today task.Date) task.Date {
	return today.AddDays((6 - int(wd(today)) + 7) % 7)
}

func daysIn(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func valid(y int, m time.Month, d int) bool {
	return m >= 1 && m <= 12 && d >= 1 && d <= daysIn(y, m)
}

// addMonths adds n months, clamping the day to the target month's length.
func addMonths(d task.Date, n int) task.Date {
	t := time.Date(d.Year, d.Month+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	y, m := t.Year(), t.Month()
	return task.Date{Year: y, Month: m, Day: min(d.Day, daysIn(y, m))}
}

// nextOccurrence returns month/day in today's year, or next year if that has
// passed (or is invalid this year, e.g. Feb 29).
func nextOccurrence(today task.Date, m time.Month, day int) (task.Date, bool) {
	for y := today.Year; y <= today.Year+4; y++ {
		if !valid(y, m, day) {
			continue
		}
		d := task.Date{Year: y, Month: m, Day: day}
		if !d.Before(today) {
			return d, true
		}
	}
	return task.Date{}, false
}

var (
	isoRe     = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	numericRe = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2}|\d{4}))?$`)
	ordinalRe = regexp.MustCompile(`^(\d{1,2})(st|nd|rd|th)$`)
	dayRe     = regexp.MustCompile(`^(\d{1,2})(st|nd|rd|th)?$`)
	yearRe    = regexp.MustCompile(`^\d{4}$`)
)

// dayNum parses "20" or "20th".
func dayNum(w string) (int, bool) {
	m := dayRe.FindStringSubmatch(w)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return n, n >= 1 && n <= 31
}

func count(w string) (int, bool) {
	if w == "a" || w == "an" {
		return 1, true
	}
	n, err := strconv.Atoi(w)
	return n, err == nil && n >= 0 && n <= 1000
}

// unit maps day/week/month/year words to a shift function.
func unit(w string) (func(task.Date, int) task.Date, bool) {
	switch strings.TrimSuffix(w, "s") {
	case "day":
		return func(d task.Date, n int) task.Date { return d.AddDays(n) }, true
	case "week":
		return func(d task.Date, n int) task.Date { return d.AddDays(7 * n) }, true
	case "month":
		return addMonths, true
	case "year":
		return func(d task.Date, n int) task.Date { return addMonths(d, 12*n) }, true
	}
	return nil, false
}

// parseDate parses lowercase words as one whole date phrase. sigil allows
// the short forms that bare text may not use (G5.1f).
func parseDate(ws []string, today task.Date, sigil bool) (task.Date, bool) {
	switch len(ws) {
	case 1:
		return parseOneWord(ws[0], today, sigil)
	case 2:
		if d, ok := parseTwoWords(ws, today); ok {
			return d, true
		}
	case 3:
		if d, ok := parseThreeWords(ws, today); ok {
			return d, true
		}
	}
	// "N units before|after <date>"
	if len(ws) >= 4 {
		n, ok1 := count(ws[0])
		shift, ok2 := unit(ws[1])
		if ok1 && ok2 && (ws[2] == "before" || ws[2] == "after") {
			if base, ok := parseDate(ws[3:], today, sigil); ok && !base.IsZero() {
				if ws[2] == "before" {
					n = -n
				}
				return shift(base, n), true
			}
		}
	}
	if len(ws) == 3 && ws[0] == "no" && ws[1] == "due" && ws[2] == "date" {
		return task.Date{}, true
	}
	return task.Date{}, false
}

func parseOneWord(w string, today task.Date, sigil bool) (task.Date, bool) {
	if m := isoRe.FindStringSubmatch(w); m != nil {
		d, err := task.ParseDate(w)
		return d, err == nil
	}
	if m := numericRe.FindStringSubmatch(w); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		mon, day := a, b // US month/day (G5.1c) ...
		if a > 12 && b <= 12 {
			mon, day = b, a // ... unless the first number can't be a month
		}
		if m[3] != "" {
			y, _ := strconv.Atoi(m[3])
			if y < 100 {
				y += 2000
			}
			if !valid(y, time.Month(mon), day) {
				return task.Date{}, false
			}
			return task.Date{Year: y, Month: time.Month(mon), Day: day}, true
		}
		return nextOccurrence(today, time.Month(mon), day)
	}
	switch w {
	case "today":
		return today, true
	case "tomorrow":
		return today.AddDays(1), true
	case "yesterday":
		return today.AddDays(-1), true
	case "weekend":
		return saturday(today), true
	case "tod":
		return today, sigil
	case "tom":
		return today.AddDays(1), sigil
	case "eom":
		return task.Date{Year: today.Year, Month: today.Month, Day: daysIn(today.Year, today.Month)}, sigil
	}
	if d, ok := weekday(w, sigil); ok {
		return nextWeekday(today, d), true
	}
	if m := ordinalRe.FindStringSubmatch(w); m != nil {
		n, _ := strconv.Atoi(m[1])
		for i := 0; i < 12; i++ { // the next month that has this day
			c := addMonths(task.Date{Year: today.Year, Month: today.Month, Day: 1}, i)
			if valid(c.Year, c.Month, n) {
				if d := (task.Date{Year: c.Year, Month: c.Month, Day: n}); !d.Before(today) {
					return d, true
				}
			}
		}
		return task.Date{}, false
	}
	if m, ok := months[w]; ok && sigil { // a lone month is its 1st
		return nextOccurrence(today, m, 1)
	}
	return task.Date{}, false
}

func parseTwoWords(ws []string, today task.Date) (task.Date, bool) {
	a, b := ws[0], ws[1]
	switch a {
	case "next":
		switch b {
		case "week":
			return monday(today).AddDays(7), true
		case "month":
			return addMonths(task.Date{Year: today.Year, Month: today.Month, Day: 1}, 1), true
		case "year":
			return task.Date{Year: today.Year + 1, Month: 1, Day: 1}, true
		case "weekend":
			return monday(today).AddDays(5 + 7), true
		}
		if d, ok := anyWeekday(b); ok {
			return nextWeekWeekday(today, d), true
		}
	case "this":
		if b == "weekend" {
			return saturday(today), true
		}
		if d, ok := anyWeekday(b); ok {
			return nextWeekday(today, d), true
		}
	case "mid":
		if m, ok := months[b]; ok {
			return nextOccurrence(today, m, 15)
		}
	case "no":
		if b == "date" {
			return task.Date{}, true
		}
	}
	if m, ok := months[a]; ok { // nov 20
		if d, ok := dayNum(b); ok {
			return nextOccurrence(today, m, d)
		}
	}
	if m, ok := months[b]; ok { // 20 nov, 20th november
		if d, ok := dayNum(a); ok {
			return nextOccurrence(today, m, d)
		}
	}
	return task.Date{}, false
}

func parseThreeWords(ws []string, today task.Date) (task.Date, bool) {
	if ws[0] == "in" {
		if n, ok := count(ws[1]); ok {
			if shift, ok := unit(ws[2]); ok {
				return shift(today, n), true
			}
		}
	}
	if ws[0] == "end" && ws[1] == "of" {
		if ws[2] == "month" {
			return task.Date{Year: today.Year, Month: today.Month, Day: daysIn(today.Year, today.Month)}, true
		}
		if m, ok := months[ws[2]]; ok {
			for y := today.Year; y <= today.Year+1; y++ {
				if d := (task.Date{Year: y, Month: m, Day: daysIn(y, m)}); !d.Before(today) {
					return d, true
				}
			}
		}
	}
	if yearRe.MatchString(ws[2]) { // nov 20 2027, 20 nov 2027
		y, _ := strconv.Atoi(ws[2])
		var m time.Month
		var day int
		var ok1, ok2 bool
		if m, ok1 = months[ws[0]]; ok1 {
			day, ok2 = dayNum(ws[1])
		} else if m, ok1 = months[ws[1]]; ok1 {
			day, ok2 = dayNum(ws[0])
		}
		if ok1 && ok2 && valid(y, m, day) {
			return task.Date{Year: y, Month: m, Day: day}, true
		}
	}
	return task.Date{}, false
}

var (
	minutesRe = regexp.MustCompile(`^(\d+(?:\.\d+)?) ?(?:m|min|mins|minute|minutes)?$`)
	hoursRe   = regexp.MustCompile(`^(\d+(?:\.\d+)?) ?(?:h|hr|hrs|hour|hours)$`)
	hmRe      = regexp.MustCompile(`^(\d+) ?(?:h|hr|hrs|hour|hours) ?(\d+) ?(?:m|min|mins|minute|minutes)?$`)
)

// ParseDuration parses a duration as typed after ~ or into the Edit
// Duration dialog: 30m, 30 minutes, 1.5h, 2 hours, 1h30m, 1h 30m, or a bare
// number of minutes. It returns whole minutes, rounded; zero is invalid.
func ParseDuration(s string) (int, bool) {
	s = strings.Join(strings.Fields(strings.ToLower(s)), " ")
	var minutes float64
	switch {
	case hmRe.MatchString(s):
		m := hmRe.FindStringSubmatch(s)
		h, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		minutes = float64(h*60 + mm)
	case hoursRe.MatchString(s):
		h, _ := strconv.ParseFloat(hoursRe.FindStringSubmatch(s)[1], 64)
		minutes = h * 60
	case minutesRe.MatchString(s):
		minutes, _ = strconv.ParseFloat(minutesRe.FindStringSubmatch(s)[1], 64)
	default:
		return 0, false
	}
	n := int(minutes + 0.5)
	return n, n > 0 && n <= 100000
}
