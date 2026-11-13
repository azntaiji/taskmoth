package task

// The list-view filters. Each reports whether a task belongs in a view on the
// given local date. Trashed tasks belong in no view except Trash.
//
// Completed tasks: the filters below select open tasks. Whether recently
// completed tasks stay visible in Today, Inbox or project lists is decided in
// Phase 3 (G3.1, G3.5); what All contains is decided in Phase 4 (G4.2).

// UpcomingDays is the length of the Upcoming window after today.
const UpcomingDays = 7

func open(t Task) bool { return !t.Trashed() && !t.Done() }

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

// InCompleted: completed and not trashed.
func InCompleted(t Task) bool { return !t.Trashed() && t.Done() }

// InTrash: in the trash, completed or not.
func InTrash(t Task) bool { return t.Trashed() }

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
