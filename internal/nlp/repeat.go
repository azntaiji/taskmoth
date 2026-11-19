package nlp

import (
	"strconv"
	"strings"

	"github.com/azntaiji/taskmoth/internal/task"
)

// Repeat phrases (PLAN.md Appendix A, decision G15.7):
//
//	every|ev [other|N] day(s)|week(s)|month(s)|year(s) [on [the] <days>]
//	every|ev [other] <days>
//	daily | weekly | monthly | yearly | annually
//	... [starting <date phrase>]
//
// where <days> is a weekday list (mon, wed / mon and wed), weekday,
// weekend, an ordinal day (15th), last day, an nth weekday (first mon,
// last fri) or a month and day (nov 20, 20 nov).

// repeatPhrase is a recognised repeat phrase: the rule as written (not yet
// anchored), the "starting" date if any, and the index of its last word.
type repeatPhrase struct {
	rule     task.Rule
	starting task.Date
	last     int
}

// bare drops a trailing comma, so "mon," reads as "mon".
func bare(w string) string { return strings.TrimSuffix(w, ",") }

var units = map[string]task.Unit{
	"day": task.Day, "days": task.Day, "week": task.Week, "weeks": task.Week,
	"month": task.Month, "months": task.Month, "year": task.Year, "years": task.Year,
}

var adverbs = map[string]task.Unit{
	"daily": task.Day, "weekly": task.Week, "monthly": task.Month, "yearly": task.Year, "annually": task.Year,
}

var nths = map[string]int{
	"first": 1, "1st": 1, "second": 2, "2nd": 2, "third": 3, "3rd": 3, "fourth": 4, "4th": 4, "last": task.Last,
}

// parseRepeat reads the repeat phrase starting at word i, if there is one.
func parseRepeat(lower []string, i int, today task.Date) (repeatPhrase, bool) {
	w := bare(lower[i])
	var p repeatPhrase
	if u, ok := adverbs[w]; ok {
		p.rule, p.last = task.Rule{Every: 1, Unit: u}, i
	} else {
		if w != "every" && w != "ev" {
			return p, false
		}
		r, last, ok := parseEvery(lower, i+1)
		if !ok {
			return p, false
		}
		p.rule, p.last = r, last
	}
	// "starting <date>": the longest date phrase that follows.
	if j := p.last + 1; j < len(lower) && bare(lower[j]) == "starting" {
		for n := min(maxPhrase, len(lower)-j-1); n >= 1; n-- {
			if d, ok := parseDate(lower[j+1:j+1+n], today, true); ok && !d.IsZero() {
				p.starting, p.last = d, j+n
				break
			}
		}
	}
	return p, true
}

// parseEvery reads what follows "every", from word j. last is the index of
// the phrase's last word.
func parseEvery(lower []string, j int) (r task.Rule, last int, ok bool) {
	at := func(k int) string {
		if k < len(lower) {
			return bare(lower[k])
		}
		return ""
	}
	r.Every = 1
	counted := false
	switch w := at(j); {
	case w == "other":
		r.Every = 2
		j++
	case isCount(w):
		n, _ := strconv.Atoi(w)
		r.Every, counted = n, true
		j++
	}
	if u, ok := units[at(j)]; ok {
		r.Unit, last = u, j
		// "on [the] <days>" says which days of the period.
		if at(j+1) == "on" {
			k := j + 2
			if at(k) == "the" {
				k++
			}
			if s, end, ok := parseDays(lower, k); ok && s.Unit == u {
				s.Every, last = r.Every, end
				return s, last, true
			}
		}
		return r, last, true
	}
	if counted {
		return r, 0, false // "every 3" needs a unit
	}
	s, end, ok := parseDays(lower, j)
	if !ok {
		return r, 0, false
	}
	s.Every = r.Every
	return s, end, true
}

// isCount is a whole number of periods, 1–999.
func isCount(w string) bool {
	n, err := strconv.Atoi(w)
	return err == nil && n >= 1 && n <= 999 && w[0] != '0'
}

// parseDays reads <days> at word j: the rule's unit and days, and the
// index of the last word.
func parseDays(lower []string, j int) (task.Rule, int, bool) {
	at := func(k int) string {
		if k < len(lower) {
			return bare(lower[k])
		}
		return ""
	}
	w := at(j)
	switch w {
	case "weekday", "weekdays", "workday", "workdays":
		return task.Rule{Unit: task.Week, Days: task.WorkWeek}, j, true
	case "weekend", "weekends":
		return task.Rule{Unit: task.Week, Days: task.Weekend}, j, true
	}
	// A month and day: "nov 20", "20 nov", "20th november".
	if m, ok := months[w]; ok {
		if d, ok := dayNum(at(j + 1)); ok && d <= task.DaysIn(2028, m) {
			return task.Rule{Unit: task.Year, Month: m, MonthDay: d}, j + 1, true
		}
	}
	if d, ok := dayNum(w); ok {
		if m, ok := months[at(j+1)]; ok && d <= task.DaysIn(2028, m) {
			return task.Rule{Unit: task.Year, Month: m, MonthDay: d}, j + 1, true
		}
	}
	// "first mon", "last fri", "last day".
	if n, ok := nths[w]; ok {
		if d, ok := anyWeekday(at(j + 1)); ok {
			return task.Rule{Unit: task.Month, Nth: n, Weekday: d}, j + 1, true
		}
		if n == task.Last && at(j+1) == "day" {
			return task.Rule{Unit: task.Month, MonthDay: task.Last}, j + 1, true
		}
	}
	// "15th".
	if m := ordinalRe.FindStringSubmatch(w); m != nil {
		if d, _ := strconv.Atoi(m[1]); d >= 1 && d <= 31 {
			return task.Rule{Unit: task.Month, MonthDay: d}, j, true
		}
	}
	// A weekday list: "mon", "mon, wed", "mon and wed", "mon , wed".
	d, ok := anyWeekday(w)
	if !ok {
		return task.Rule{}, 0, false
	}
	days, last := task.Weekdays(0).With(d), j
	for k := j + 1; k < len(lower); {
		sep := strings.HasSuffix(lower[last], ",") // "mon, wed"
		if lower[k] == "," || lower[k] == "and" {  // "mon , wed", "mon and wed"
			sep, k = true, k+1
		}
		if !sep || k >= len(lower) {
			break
		}
		d, ok := anyWeekday(bare(lower[k]))
		if !ok {
			break
		}
		days, last, k = days.With(d), k, k+1
	}
	return task.Rule{Unit: task.Week, Days: days}, last, true
}

// ParseRepeat parses a whole repeat phrase, as typed into Edit Due date
// (G15.2): the rule, anchored, and its first due date on or after today
// (or the "starting" date). ok is false if s isn't one whole repeat phrase.
func ParseRepeat(s string, today task.Date) (task.Rule, task.Date, bool) {
	ws := strings.Fields(strings.ToLower(s))
	if len(ws) == 0 {
		return task.Rule{}, task.Date{}, false
	}
	p, ok := parseRepeat(ws, 0, today)
	if !ok || p.last != len(ws)-1 {
		return task.Rule{}, task.Date{}, false
	}
	r, first := anchor(p, task.Date{}, today)
	return r, first, true
}

// anchor fixes a repeat phrase to its first due date: the first occurrence
// on or after due (if one was given), the "starting" date, or today.
func anchor(p repeatPhrase, due, today task.Date) (task.Rule, task.Date) {
	from := today
	switch {
	case !due.IsZero():
		from = due
	case !p.starting.IsZero():
		from = p.starting
	}
	first := p.rule.First(from)
	return p.rule.Anchor(first), first
}
