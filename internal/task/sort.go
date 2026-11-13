package task

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// SortKey is a list sort order (PLAN.md "Sorting").
type SortKey int

// Sort keys.
const (
	SortName SortKey = iota
	SortStart
	SortDue
	SortPriority
	SortDuration
	SortCompleted // newest completion first (Completed view, G4.2a)
	SortArchived  // newest archived first (Archive view, G4.2a)
)

var sortNames = map[SortKey]string{
	SortName: "name", SortStart: "start", SortDue: "due", SortPriority: "priority",
	SortDuration: "duration", SortCompleted: "completed", SortArchived: "archived",
}

// String returns the key's name as shown in the status bar ("s:priority").
func (k SortKey) String() string { return sortNames[k] }

// ParseSortKey parses a name produced by SortKey.String.
func ParseSortKey(s string) (SortKey, bool) {
	for k, n := range sortNames {
		if n == s {
			return k, true
		}
	}
	return SortPriority, false
}

// Sort cycles: the order `s` steps through in each kind of view. The first
// key is the view's default.
var (
	// DefaultCycle is used by every list except Completed and Archive (G3.3a).
	DefaultCycle = []SortKey{SortPriority, SortDue, SortStart, SortDuration, SortName}
	// CompletedCycle is the Completed view's (G4.2a).
	CompletedCycle = append([]SortKey{SortCompleted}, DefaultCycle...)
	// ArchiveCycle is the Archive view's (G4.2a).
	ArchiveCycle = append([]SortKey{SortArchived}, DefaultCycle...)
)

// DefaultSort is the default of DefaultCycle.
const DefaultSort = SortPriority

// Next returns the key after k in cycle; a key not in the cycle restarts it.
func Next(cycle []SortKey, k SortKey) SortKey {
	i := slices.Index(cycle, k)
	return cycle[(i+1)%len(cycle)]
}

// Sort orders tasks in place by key. Open tasks come before completed ones
// (decision G3.3c) except under SortCompleted and SortArchived, which order
// purely by their date. Within each group, every order is total: ties fall
// back to name A–Z, then ID (creation order, since IDs are UUIDv7).
//
// Directions are fixed (decision G3.3a): names A–Z (case-insensitive), dates
// earliest first, P1 first, durations shortest first, completion and archive
// times newest first. Tasks without the date or duration sort after those
// with one.
func Sort(tasks []Task, key SortKey) {
	slices.SortStableFunc(tasks, func(a, b Task) int {
		if key != SortCompleted && key != SortArchived && a.Done() != b.Done() {
			if a.Done() {
				return 1
			}
			return -1
		}
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
		case SortCompleted:
			c = newestFirst(a.CompletedAt, b.CompletedAt)
		case SortArchived:
			c = newestFirst(a.ArchivedAt, b.ArchivedAt)
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

// newestFirst orders later times first, with unset times last.
func newestFirst(a, b time.Time) int {
	switch {
	case a.IsZero() && b.IsZero():
		return 0
	case a.IsZero():
		return 1
	case b.IsZero():
		return -1
	}
	return b.Compare(a)
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
