package task

// The list-view filters. Each reports whether an open task belongs in a view
// on the given local date. Archived tasks belong in no view except Archive.
// Use Listed to also keep tasks completed today (decision G3.1).

import "time"

// UpcomingDays is the length of the Upcoming window after today.
const UpcomingDays = 7

// CompletedOn reports whether t was completed on the local date of now
// (in now's location).
func CompletedOn(t Task, now time.Time) bool {
	return t.Done() && DateOf(t.CompletedAt.In(now.Location())) == DateOf(now)
}

// Listed reports whether t appears in a view whose open-task filter is in.
// Tasks completed today stay listed until midnight if they would belong
// there when open (decisions G3.1, G3.5). Archived tasks never appear.
func Listed(t Task, now time.Time, in func(Task) bool) bool {
	if t.Archived() {
		return false
	}
	if !t.Done() {
		return in(t)
	}
	if !CompletedOn(t, now) {
		return false
	}
	reopened := t
	reopened.CompletedAt = time.Time{}
	return in(reopened)
}

func open(t Task) bool { return !t.Archived() && !t.Done() }

// InToday: due today or overdue, or started on or before today (G2.1c).
func InToday(t Task, today Date) bool {
	if !open(t) {
		return false
	}
	return (!t.Due.IsZero() && !t.Due.After(today)) ||
		(!t.Start.IsZero() && !t.Start.After(today))
}

// InUpcoming: starts or is due from tomorrow through today+7 (G2.1c).
func InUpcoming(t Task, today Date) bool {
	if !open(t) {
		return false
	}
	end := today.AddDays(UpcomingDays)
	inWindow := func(d Date) bool { return !d.IsZero() && d.After(today) && !d.After(end) }
	return inWindow(t.Start) || inWindow(t.Due)
}

// InOverdue: due before today.
func InOverdue(t Task, today Date) bool {
	return open(t) && !t.Due.IsZero() && t.Due.Before(today)
}

// InInbox: not assigned to a project.
func InInbox(t Task) bool { return open(t) && t.ProjectID == "" }

// InProject: assigned to project p.
func InProject(t Task, p ID) bool { return open(t) && t.ProjectID == p }

// InAll: every open task (decision G4.2).
func InAll(t Task) bool { return open(t) }

// InSomeday: open with no due date, whatever its start date (G7.6).
func InSomeday(t Task) bool { return open(t) && t.Due.IsZero() }

// InCompleted: completed and not archived (decision G4.2).
func InCompleted(t Task) bool { return !t.Archived() && t.Done() }

// InArchive: archived, completed or not (decision G4.7).
func InArchive(t Task) bool { return t.Archived() }

// UpcomingDay is the day an Upcoming task is grouped under (decision G4.1):
// its due date if that is in the window, otherwise its start date. ok is
// false if neither falls in the window.
func UpcomingDay(t Task, today Date) (day Date, ok bool) {
	end := today.AddDays(UpcomingDays)
	inWindow := func(d Date) bool { return !d.IsZero() && d.After(today) && !d.After(end) }
	switch {
	case inWindow(t.Due):
		return t.Due, true
	case inWindow(t.Start):
		return t.Start, true
	}
	return Date{}, false
}

// Filter returns the tasks for which keep is true, in their original order.
func Filter(tasks []Task, keep func(Task) bool) []Task {
	var out []Task
	for _, t := range tasks {
		if keep(t) {
			out = append(out, t)
		}
	}
	return out
}

// DueState is how a task's due date relates to today, for its badge.
type DueState int

// Due states.
const (
	DueNone DueState = iota
	DueUpcoming
	DueToday
	DueOverdue
)

// DueStateOf classifies t's due date relative to today.
func DueStateOf(t Task, today Date) DueState {
	switch {
	case t.Due.IsZero():
		return DueNone
	case t.Due.Before(today):
		return DueOverdue
	case t.Due == today:
		return DueToday
	default:
		return DueUpcoming
	}
}
