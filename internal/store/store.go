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
	// ErrArchivedProjectName: the name belongs to an archived project (G4.4).
	ErrArchivedProjectName = errors.New("an archived project has that name; unarchive it instead")
	ErrUnknownProject      = errors.New("project does not exist")
	ErrProjectArchived     = errors.New("project is archived")
	ErrNotArchived         = errors.New("not archived") // permanent deletion needs it archived first (G4.8)
	ErrNewerSchema         = errors.New("database was created by a newer taskmoth")
)

// Store is taskmoth's persistence API.
type Store interface {
	// ListTasks returns every task, including completed and archived ones, in
	// creation order. Views filter with the task package.
	ListTasks(ctx context.Context) ([]task.Task, error)
	GetTask(ctx context.Context, id task.ID) (task.Task, error)
	// CreateTask assigns the ID and timestamps. Priority 0 means the default
	// (P4); CompletedAt and ArchivedAt are ignored. The project, if any, must
	// exist and not be archived.
	CreateTask(ctx context.Context, t task.Task) (task.Task, error)
	// UpdateTask saves the editable fields (name, description, project,
	// priority, dates, duration). Completion and archive state are unchanged.
	// A changed project must exist and not be archived.
	UpdateTask(ctx context.Context, t task.Task) (task.Task, error)
	// SetDone marks a task completed or open. Completing an already completed
	// task keeps its original completion time.
	SetDone(ctx context.Context, id task.ID, done bool) (task.Task, error)
	// Archive archives a task (G4.4); archiving it again keeps the original
	// time. Unarchive restores it; if its project is archived, the task moves
	// to the Inbox (G4.4a).
	Archive(ctx context.Context, id task.ID) (task.Task, error)
	Unarchive(ctx context.Context, id task.ID) (task.Task, error)
	// DeleteForever permanently deletes an archived task (G4.8).
	DeleteForever(ctx context.Context, id task.ID) error

	// ListProjects returns every project, active and archived, A–Z (G2.2).
	ListProjects(ctx context.Context) ([]task.Project, error)
	GetProject(ctx context.Context, id task.ID) (task.Project, error)
	ProjectByName(ctx context.Context, name string) (task.Project, error)
	// CreateProject normalizes and validates the name ("#Work" → "work").
	CreateProject(ctx context.Context, name string) (task.Project, error)
	// ArchiveProject archives a project and every unarchived task in it, all
	// with the same timestamp (G4.4).
	ArchiveProject(ctx context.Context, id task.ID) (task.Project, error)
	// UnarchiveProject restores a project and the tasks archived with it,
	// i.e. those whose archive time equals the project's (G4.4).
	UnarchiveProject(ctx context.Context, id task.ID) (task.Project, error)
	// DeleteProject permanently deletes an archived project and all of its
	// tasks (G4.8).
	DeleteProject(ctx context.Context, id task.ID) error

	// GetSetting returns a stored setting, or ErrNotFound.
	GetSetting(ctx context.Context, key string) (string, error)
	// SetSetting stores a setting, replacing any previous value.
	SetSetting(ctx context.Context, key, value string) error

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
	t.CompletedAt, t.ArchivedAt = time.Time{}, time.Time{}
	return t, t.Validate()
}
