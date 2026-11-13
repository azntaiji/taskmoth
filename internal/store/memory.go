package store

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
)

// Memory is an in-memory Store for UI tests. It enforces the same rules as
// SQLite and passes the same contract tests.
type Memory struct {
	mu       sync.Mutex
	opts     Options
	tasks    map[task.ID]task.Task
	projects map[task.ID]task.Project
}

var _ Store = (*Memory)(nil)

// NewMemory returns an empty in-memory store.
func NewMemory(opts Options) *Memory {
	return &Memory{
		opts:     opts.withDefaults(),
		tasks:    map[task.ID]task.Task{},
		projects: map[task.ID]task.Project{},
	}
}

func (m *Memory) now() time.Time { return m.opts.Now().UTC().Truncate(time.Millisecond) }

// Close implements Store.
func (m *Memory) Close() error { return nil }

// ListTasks implements Store.
func (m *Memory) ListTasks(context.Context) ([]task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]task.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b task.Task) int { return strings.Compare(string(a.ID), string(b.ID)) })
	return out, nil
}

// GetTask implements Store.
func (m *Memory) GetTask(_ context.Context, id task.ID) (task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return task.Task{}, ErrNotFound
	}
	return t, nil
}

func (m *Memory) checkProject(id task.ID) error {
	if id == "" {
		return nil
	}
	if _, ok := m.projects[id]; !ok {
		return ErrUnknownProject
	}
	return nil
}

// CreateTask implements Store.
func (m *Memory) CreateTask(_ context.Context, t task.Task) (task.Task, error) {
	t, err := prepareNew(t)
	if err != nil {
		return task.Task{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkProject(t.ProjectID); err != nil {
		return task.Task{}, err
	}
	if t.ID, err = m.opts.NewID(); err != nil {
		return task.Task{}, err
	}
	t.CreatedAt = m.now()
	t.UpdatedAt = t.CreatedAt
	m.tasks[t.ID] = t
	return t, nil
}

// UpdateTask implements Store.
func (m *Memory) UpdateTask(_ context.Context, t task.Task) (task.Task, error) {
	if err := t.Validate(); err != nil {
		return task.Task{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.tasks[t.ID]
	if !ok {
		return task.Task{}, ErrNotFound
	}
	if err := m.checkProject(t.ProjectID); err != nil {
		return task.Task{}, err
	}
	cur.Name, cur.Description, cur.ProjectID, cur.Priority = t.Name, t.Description, t.ProjectID, t.Priority
	cur.Start, cur.Due, cur.DurationMin = t.Start, t.Due, t.DurationMin
	cur.UpdatedAt = m.now()
	m.tasks[t.ID] = cur
	return cur, nil
}

// modify applies f to a stored task and bumps UpdatedAt.
func (m *Memory) modify(id task.ID, f func(*task.Task, time.Time)) (task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return task.Task{}, ErrNotFound
	}
	now := m.now()
	f(&t, now)
	t.UpdatedAt = now
	m.tasks[id] = t
	return t, nil
}

// SetDone implements Store.
func (m *Memory) SetDone(_ context.Context, id task.ID, done bool) (task.Task, error) {
	return m.modify(id, func(t *task.Task, now time.Time) {
		switch {
		case !done:
			t.CompletedAt = time.Time{}
		case t.CompletedAt.IsZero():
			t.CompletedAt = now
		}
	})
}

// Trash implements Store.
func (m *Memory) Trash(_ context.Context, id task.ID) (task.Task, error) {
	return m.modify(id, func(t *task.Task, now time.Time) {
		if t.DeletedAt.IsZero() {
			t.DeletedAt = now
		}
	})
}

// Restore implements Store.
func (m *Memory) Restore(_ context.Context, id task.ID) (task.Task, error) {
	return m.modify(id, func(t *task.Task, _ time.Time) { t.DeletedAt = time.Time{} })
}

// Purge implements Store.
func (m *Memory) Purge(_ context.Context, id task.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return ErrNotFound
	}
	if !t.Trashed() {
		return ErrNotTrashed
	}
	delete(m.tasks, id)
	return nil
}

// ListProjects implements Store.
func (m *Memory) ListProjects(context.Context) ([]task.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]task.Project, 0, len(m.projects))
	for _, p := range m.projects {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b task.Project) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// GetProject implements Store.
func (m *Memory) GetProject(_ context.Context, id task.ID) (task.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[id]
	if !ok {
		return task.Project{}, ErrNotFound
	}
	return p, nil
}

// ProjectByName implements Store.
func (m *Memory) ProjectByName(_ context.Context, name string) (task.Project, error) {
	name = task.NormalizeProjectName(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return task.Project{}, ErrNotFound
}

// CreateProject implements Store.
func (m *Memory) CreateProject(_ context.Context, name string) (task.Project, error) {
	name = task.NormalizeProjectName(name)
	if err := task.ValidateProjectName(name); err != nil {
		return task.Project{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.projects {
		if p.Name == name {
			return task.Project{}, ErrDuplicateProject
		}
	}
	id, err := m.opts.NewID()
	if err != nil {
		return task.Project{}, err
	}
	now := m.now()
	p := task.Project{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}
	m.projects[id] = p
	return p, nil
}

// DeleteProject implements Store.
func (m *Memory) DeleteProject(_ context.Context, id task.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.projects[id]; !ok {
		return ErrNotFound
	}
	for _, t := range m.tasks {
		if t.ProjectID == id {
			return ErrProjectInUse
		}
	}
	delete(m.projects, id)
	return nil
}
