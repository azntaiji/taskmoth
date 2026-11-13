// Package task is taskmoth's domain model: tasks, projects, validation, the
// list-view filters and the sort orders. It has no UI or storage imports, and
// anything date-dependent takes "today" as a parameter.
package task

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ID identifies a task or project. IDs are UUIDv7 strings (decision G2.4),
// assigned by the store; they sort in creation order.
type ID string

// Priorities (decision G2.1a: every task has one; P4 is the default).
const (
	P1 = 1
	P2 = 2
	P3 = 3
	P4 = 4

	DefaultPriority = P4
)

// Task is one to-do item. Zero values mean "not set" for the optional fields.
type Task struct {
	ID          ID
	Name        string // required, single line (G2.1d)
	Description string // "" = none
	ProjectID   ID     // "" = Inbox
	Priority    int    // 1–4
	Start       Date   // zero = none
	Due         Date   // zero = none
	DurationMin int    // estimated minutes; 0 = none

	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt time.Time // zero = open (G2.3)
	ArchivedAt  time.Time // zero = not archived (G4.4)
}

// Done reports whether the task is completed.
func (t Task) Done() bool { return !t.CompletedAt.IsZero() }

// Archived reports whether the task is archived (decision G4.4).
func (t Task) Archived() bool { return !t.ArchivedAt.IsZero() }

// Project groups tasks. Names are lowercase slugs shown as #name (G2.2).
// Archiving a project archives its tasks with the same timestamp (G4.4).
type Project struct {
	ID         ID
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt time.Time // zero = active
}

// Archived reports whether the project is archived.
func (p Project) Archived() bool { return !p.ArchivedAt.IsZero() }

// ErrInvalid wraps every validation failure.
var ErrInvalid = errors.New("invalid")

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

// Validate checks the task's own fields (not whether its project exists).
func (t Task) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return invalid("task name is required")
	}
	if strings.ContainsAny(t.Name, "\r\n") {
		return invalid("task name must be a single line")
	}
	if t.Priority < P1 || t.Priority > P4 {
		return invalid("priority must be 1–4, got %d", t.Priority)
	}
	if t.DurationMin < 0 {
		return invalid("duration must not be negative")
	}
	return nil
}

// NormalizeProjectName lowercases a project name and drops a leading '#',
// so "#HomeLab-Tech" becomes "homelab-tech".
func NormalizeProjectName(s string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(s), "#"))
}

// ValidateProjectName checks an already-normalized name: one or more of
// a–z, 0–9, '-' and '_'.
func ValidateProjectName(name string) error {
	if name == "" {
		return invalid("project name is required")
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return invalid("project name %q may only contain a–z, 0–9, - and _", name)
		}
	}
	return nil
}
