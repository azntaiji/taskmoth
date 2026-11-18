package task

import "time"

// DayLoad is how many tasks fall on one Calendar row (decisions G14.2,
// G14.6): Due counts every task in the row, Done the completed ones among
// them, and RemainingMin the estimates of the open ones.
type DayLoad struct {
	Due, Done    int
	RemainingMin int
}

// Open is the number of tasks not yet done.
func (l DayLoad) Open() int { return l.Due - l.Done }

// Add adds l's counts to m.
func (l DayLoad) Add(m DayLoad) DayLoad {
	return DayLoad{Due: l.Due + m.Due, Done: l.Done + m.Done, RemainingMin: l.RemainingMin + m.RemainingMin}
}

func (l *DayLoad) count(t Task) {
	l.Due++
	if t.Done() {
		l.Done++
	} else {
		l.RemainingMin += t.DurationMin
	}
}

// DueOn reports whether t counts towards day d: due that day and not
// archived, done or not (G14.2). Start dates don't count.
func DueOn(t Task, d Date) bool { return !t.Archived() && !t.Due.IsZero() && t.Due == d }

// LoadByDay counts every task that isn't archived under its due date
// (G14.2). Tasks without a due date aren't counted.
func LoadByDay(tasks []Task) map[Date]DayLoad {
	out := map[Date]DayLoad{}
	for _, t := range tasks {
		if t.Archived() || t.Due.IsZero() {
			continue
		}
		l := out[t.Due]
		l.count(t)
		out[t.Due] = l
	}
	return out
}

// LoadOn counts the tasks due on d, leaving out the task skip (the one
// being edited, G14.12; "" skips nothing).
func LoadOn(tasks []Task, d Date, skip ID) DayLoad {
	var l DayLoad
	for _, t := range tasks {
		if t.ID != skip && DueOn(t, d) {
			l.count(t)
		}
	}
	return l
}

// OverdueLoad counts the open tasks due before today (G14.6), the same
// tasks as the Overdue view.
func OverdueLoad(tasks []Task, today Date) DayLoad {
	var l DayLoad
	for _, t := range tasks {
		if InOverdue(t, today) {
			l.count(t)
		}
	}
	return l
}

// Weekday is d's day of the week.
func (d Date) Weekday() time.Weekday { return d.time().Weekday() }

// WeekStart is the Monday on or before d (weeks start on Monday, G14.5).
func WeekStart(d Date) Date {
	return d.AddDays(-((int(d.Weekday()) + 6) % 7))
}

// DaysBetween is the number of days from d to e (negative if e is before d).
func DaysBetween(d, e Date) int {
	return int(e.time().Sub(d.time()).Round(time.Hour).Hours() / 24)
}
