package task

import (
	"cmp"
	"slices"
	"strings"
)

// SortKey is a list sort order (PLAN.md "Sorting").
type SortKey int

// Sort keys, in the brief's order.
const (
	SortName SortKey = iota
	SortStart
	SortDue
	SortPriority
	SortDuration
)

// String returns the key's name as shown in the status bar ("s:priority").
func (k SortKey) String() string {
	switch k {
	case SortStart:
		return "start"
	case SortDue:
		return "due"
	case SortPriority:
		return "priority"
	case SortDuration:
		return "duration"
	default:
		return "name"
	}
}

// Sort orders tasks in place by key. Every order is total and stable: ties
// fall back to name, then ID (creation order, since IDs are UUIDv7).
//
// Directions: names A–Z (case-insensitive); dates earliest first; P1 first;
// durations shortest first. Tasks without the date or duration sort after
// those with one. Default sort, cycling and direction toggles are decided at
// G3.3 and may adjust these.
func Sort(tasks []Task, key SortKey) {
	slices.SortStableFunc(tasks, func(a, b Task) int {
		var c int
		switch key {
		case SortStart:
			c = compareDates(a.Start, b.Start)
		case SortDue:
			c = compareDates(a.Due, b.Due)
		case SortPriority:
			c = cmp.Compare(a.Priority, b.Priority)
		case SortDuration:
			c = compareUnsetLast(a.DurationMin, b.DurationMin)
		}
		if c != 0 {
			return c
		}
		if c = compareNames(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

func compareNames(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// compareDates orders earliest first, with unset dates last.
func compareDates(a, b Date) int {
	switch {
	case a.IsZero() && b.IsZero():
		return 0
	case a.IsZero():
		return 1
	case b.IsZero():
		return -1
	}
	return a.Compare(b)
}

// compareUnsetLast orders smallest first, with 0 (unset) last.
func compareUnsetLast(a, b int) int {
	switch {
	case a == 0 && b == 0:
		return 0
	case a == 0:
		return 1
	case b == 0:
		return -1
	}
	return cmp.Compare(a, b)
}
