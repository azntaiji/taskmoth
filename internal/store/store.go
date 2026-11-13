// Package store persists tasks and projects. Store is implemented by SQLite
// (the app's real storage, decision G0.2) and by Memory (for UI tests). Both
// pass the same contract tests.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/azntaiji/taskmoth/internal/task"
)

// Errors returned by every Store. Validation failures wrap task.ErrInvalid.
var (
	ErrNotFound         = errors.New("not found")
	ErrDuplicateProject = errors.New("a project with that name already exists")
	ErrUnknownProject   = errors.New("project does not exist")
	ErrProjectInUse     = errors.New("project still has tasks") // until G4.4 decides otherwise
	ErrNotTrashed       = errors.New("task is not in the trash")
	ErrNewerSchema      = errors.New("database was created by a newer taskmoth")
)

// Store is taskmoth's persistence API.
type Store interface {
	// ListTasks returns every task, including completed and trashed ones, in
	// creation order. Views filter with the task package.
	ListTasks(ctx context.Context) ([]task.Task, error)
	GetTask(ctx context.Context, id task.ID) (task.Task, error)
	// CreateTask assigns the ID and timestamps. Priority 0 means the default
	// (P4); CompletedAt and DeletedAt are ignored.
	CreateTask(ctx context.Context, t task.Task) (task.Task, error)
	// UpdateTask saves the editable fields (name, description, project,
	// priority, dates, duration). Completion and trash state are unchanged.
	UpdateTask(ctx context.Context, t task.Task) (task.Task, error)
	// SetDone marks a task completed or open. Completing an already completed
	// task keeps its original completion time.
	SetDone(ctx context.Context, id task.ID, done bool) (task.Task, error)
	// Trash moves a task to the trash; Restore takes it out.
	Trash(ctx context.Context, id task.ID) (task.Task, error)
	Restore(ctx context.Context, id task.ID) (task.Task, error)
	// Purge permanently deletes a task. It must be in the trash.
	Purge(ctx context.Context, id task.ID) error

	// ListProjects returns projects A–Z (decision G2.2).
	ListProjects(ctx context.Context) ([]task.Project, error)
	GetProject(ctx context.Context, id task.ID) (task.Project, error)
	ProjectByName(ctx context.Context, name string) (task.Project, error)
	// CreateProject normalizes and validates the name ("#Work" → "work").
	CreateProject(ctx context.Context, name string) (task.Project, error)
	// DeleteProject fails with ErrProjectInUse while any task, including
	// completed and trashed ones, belongs to it.
	DeleteProject(ctx context.Context, id task.ID) error

	Close() error
}

// Options customizes a store. Tests use them to fix the clock and IDs.
type Options struct {
	Now   func() time.Time
	NewID func() (task.ID, error)
}

func (o Options) withDefaults() Options {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.NewID == nil {
		o.NewID = NewID
	}
	return o
}

// NewID returns a new UUIDv7 ID (decision G2.4).
func NewID() (task.ID, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return task.ID(u.String()), nil
}

// prepareNew validates a task for creation and fills in defaults.
func prepareNew(t task.Task) (task.Task, error) {
	if t.Priority == 0 {
		t.Priority = task.DefaultPriority
	}
	t.CompletedAt, t.DeletedAt = time.Time{}, time.Time{}
	return t, t.Validate()
}
