package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
)

// clock is a controllable time source.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newClock() *clock {
	return &clock{t: time.Date(2026, 11, 10, 15, 26, 0, 0, time.UTC)}
}

type factory func(t *testing.T, opts Options) Store

func implementations() map[string]factory {
	return map[string]factory{
		"memory": func(t *testing.T, opts Options) Store { return NewMemory(opts) },
		"sqlite": func(t *testing.T, opts Options) Store {
			s, err := OpenDir(context.Background(), t.TempDir(), opts)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := s.Close(); err != nil {
					t.Error(err)
				}
			})
			return s
		},
	}
}

// TestContract runs the same behaviour checks against every Store.
func TestContract(t *testing.T) {
	for name, open := range implementations() {
		t.Run(name, func(t *testing.T) {
			t.Run("tasks", func(t *testing.T) { testTasks(t, open) })
			t.Run("completion and trash", func(t *testing.T) { testLifecycle(t, open) })
			t.Run("projects", func(t *testing.T) { testProjects(t, open) })
			t.Run("errors", func(t *testing.T) { testErrors(t, open) })
		})
	}
}

func testTasks(t *testing.T, open factory) {
	ctx := context.Background()
	clk := newClock()
	s := open(t, Options{Now: clk.Now})
	proj, err := s.CreateProject(ctx, "groceries")
	if err != nil {
		t.Fatal(err)
	}

	in := task.Task{
		Name: "Get eggs", Description: "Trader Joe's\n\nand milk", ProjectID: proj.ID,
		Priority: task.P3, Start: task.Date{Year: 2026, Month: 11, Day: 9},
		Due: task.Date{Year: 2026, Month: 11, Day: 10}, DurationMin: 15,
		CompletedAt: clk.Now(), // ignored on create
	}
	created, err := s.CreateTask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || !created.CreatedAt.Equal(clk.Now()) || !created.UpdatedAt.Equal(clk.Now()) {
		t.Errorf("ID/timestamps not assigned: %+v", created)
	}
	if created.Done() {
		t.Error("CompletedAt should be ignored on create")
	}

	got, err := s.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Errorf("round trip:\n got  %+v\n want %+v", got, created)
	}

	// Defaults: priority 0 → P4; empty optional fields stay empty.
	inbox, err := s.CreateTask(ctx, task.Task{Name: "Sable"})
	if err != nil {
		t.Fatal(err)
	}
	if inbox.Priority != task.P4 || inbox.ProjectID != "" || !inbox.Due.IsZero() || inbox.DurationMin != 0 {
		t.Errorf("defaults: %+v", inbox)
	}

	// Update changes editable fields and UpdatedAt only.
	clk.Advance(time.Hour)
	got.Name, got.Priority, got.ProjectID, got.Due, got.DurationMin = "Get eggs and milk", task.P1, "", task.Date{}, 0
	updated, err := s.UpdateTask(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Get eggs and milk" || updated.Priority != task.P1 || updated.ProjectID != "" ||
		!updated.Due.IsZero() || updated.DurationMin != 0 {
		t.Errorf("update not applied: %+v", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.Equal(clk.Now()) {
		t.Errorf("timestamps after update: created %v updated %v", updated.CreatedAt, updated.UpdatedAt)
	}

	all, err := s.ListTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != created.ID || all[1].ID != inbox.ID {
		t.Errorf("ListTasks should return both in creation order, got %d", len(all))
	}
}

func testLifecycle(t *testing.T, open factory) {
	ctx := context.Background()
	clk := newClock()
	s := open(t, Options{Now: clk.Now})
	tk, err := s.CreateTask(ctx, task.Task{Name: "Watch Tales of the Loop"})
	if err != nil {
		t.Fatal(err)
	}

	clk.Advance(time.Minute)
	doneAt := clk.Now()
	tk, err = s.SetDone(ctx, tk.ID, true)
	if err != nil || !tk.CompletedAt.Equal(doneAt) {
		t.Fatalf("SetDone: %v %v", tk.CompletedAt, err)
	}
	clk.Advance(time.Minute)
	if tk, _ = s.SetDone(ctx, tk.ID, true); !tk.CompletedAt.Equal(doneAt) {
		t.Error("completing again should keep the original completion time")
	}
	if tk, _ = s.SetDone(ctx, tk.ID, false); tk.Done() {
		t.Error("SetDone(false) should reopen")
	}

	if err := s.Purge(ctx, tk.ID); !errors.Is(err, ErrNotTrashed) {
		t.Errorf("Purge of untrashed task: %v", err)
	}
	if tk, err = s.Trash(ctx, tk.ID); err != nil || !tk.Trashed() {
		t.Fatalf("Trash: %v", err)
	}
	if tk, err = s.Restore(ctx, tk.ID); err != nil || tk.Trashed() {
		t.Fatalf("Restore: %v", err)
	}
	if _, err = s.Trash(ctx, tk.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Purge(ctx, tk.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, err := s.GetTask(ctx, tk.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("purged task still found: %v", err)
	}
}

func testProjects(t *testing.T, open factory) {
	ctx := context.Background()
	s := open(t, Options{})
	for _, n := range []string{"work", "#Groceries", "homelab-tech"} {
		if _, err := s.CreateProject(ctx, n); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	ps, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range ps {
		names = append(names, p.Name)
	}
	if fmt.Sprint(names) != "[groceries homelab-tech work]" {
		t.Errorf("projects not A–Z / normalized: %v", names)
	}

	g, err := s.ProjectByName(ctx, "#GROCERIES")
	if err != nil || g.Name != "groceries" {
		t.Fatalf("ProjectByName: %+v %v", g, err)
	}
	if got, err := s.GetProject(ctx, g.ID); err != nil || got != g {
		t.Errorf("GetProject: %+v %v", got, err)
	}

	tk, _ := s.CreateTask(ctx, task.Task{Name: "eggs", ProjectID: g.ID})
	if err := s.DeleteProject(ctx, g.ID); !errors.Is(err, ErrProjectInUse) {
		t.Errorf("deleting a project with tasks: %v", err)
	}
	// Trashed tasks still block deletion (they can be restored).
	if _, err := s.Trash(ctx, tk.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(ctx, g.ID); !errors.Is(err, ErrProjectInUse) {
		t.Errorf("deleting a project with trashed tasks: %v", err)
	}
	if err := s.Purge(ctx, tk.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(ctx, g.ID); err != nil {
		t.Errorf("deleting an empty project: %v", err)
	}
	if _, err := s.GetProject(ctx, g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted project still found: %v", err)
	}
}

func testErrors(t *testing.T, open factory) {
	ctx := context.Background()
	s := open(t, Options{})
	if _, err := s.CreateProject(ctx, "work"); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		err  error
		want error
	}{
		{"duplicate project", second(s.CreateProject(ctx, "#Work")), ErrDuplicateProject},
		{"invalid project name", second(s.CreateProject(ctx, "my project")), task.ErrInvalid},
		{"empty task name", second(s.CreateTask(ctx, task.Task{Name: " "})), task.ErrInvalid},
		{"multi-line task name", second(s.CreateTask(ctx, task.Task{Name: "a\nb"})), task.ErrInvalid},
		{"bad priority", second(s.CreateTask(ctx, task.Task{Name: "a", Priority: 9})), task.ErrInvalid},
		{"unknown project on create", second(s.CreateTask(ctx, task.Task{Name: "a", ProjectID: "nope"})), ErrUnknownProject},
		{"get missing task", second(s.GetTask(ctx, "nope")), ErrNotFound},
		{"update missing task", second(s.UpdateTask(ctx, task.Task{ID: "nope", Name: "a", Priority: 1})), ErrNotFound},
		{"complete missing task", second(s.SetDone(ctx, "nope", true)), ErrNotFound},
		{"trash missing task", second(s.Trash(ctx, "nope")), ErrNotFound},
		{"purge missing task", s.Purge(ctx, "nope"), ErrNotFound},
		{"missing project by name", second(s.ProjectByName(ctx, "nope")), ErrNotFound},
		{"delete missing project", s.DeleteProject(ctx, "nope"), ErrNotFound},
	}
	tk, _ := s.CreateTask(ctx, task.Task{Name: "a"})
	tk.ProjectID = "nope"
	checks = append(checks, struct {
		name string
		err  error
		want error
	}{"unknown project on update", second(s.UpdateTask(ctx, tk)), ErrUnknownProject})

	for _, c := range checks {
		if !errors.Is(c.err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, c.err, c.want)
		}
	}
}

func second[T any](_ T, err error) error { return err }

// --- SQLite-specific ---

func TestSQLiteMigratesAndReopens(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := OpenDir(ctx, dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.SchemaVersion(ctx)
	if err != nil || v != 1 {
		t.Fatalf("schema version = %d, %v", v, err)
	}
	p, _ := s.CreateProject(ctx, "work")
	tk, err := s.CreateTask(ctx, task.Task{Name: "persist me", ProjectID: p.ID, Due: task.Date{Year: 2026, Month: 12, Day: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = OpenDir(ctx, dir, Options{}) // reopening is a no-op migration
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	got, err := s.GetTask(ctx, tk.ID)
	if err != nil || got != tk {
		t.Errorf("after reopen: %+v %v", got, err)
	}
}

func TestSQLiteRefusesNewerSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), FileName)
	s, err := Open(ctx, path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA user_version = 99"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, path, Options{}); !errors.Is(err, ErrNewerSchema) {
		t.Errorf("opening a newer schema: %v", err)
	}
}

func TestSQLiteConstraintsBackUpValidation(t *testing.T) {
	ctx := context.Background()
	s, err := OpenDir(ctx, t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	now := encTime(time.Now())
	bad := []struct {
		name, sql string
		args      []any
	}{
		{"priority 0", "INSERT INTO tasks (id,name,priority,created_at,updated_at) VALUES ('a','x',0,?,?)", []any{now, now}},
		{"newline in name", "INSERT INTO tasks (id,name,created_at,updated_at) VALUES ('b','x' || char(10) || 'y',?,?)", []any{now, now}},
		{"bad date", "INSERT INTO tasks (id,name,due_date,created_at,updated_at) VALUES ('c','x','11/20/2026',?,?)", []any{now, now}},
		{"zero minutes", "INSERT INTO tasks (id,name,duration_minutes,created_at,updated_at) VALUES ('d','x',0,?,?)", []any{now, now}},
		{"capital project", "INSERT INTO projects (id,name,created_at,updated_at) VALUES ('e','Work',?,?)", []any{now, now}},
		{"text priority (STRICT)", "INSERT INTO tasks (id,name,priority,created_at,updated_at) VALUES ('f','x','high',?,?)", []any{now, now}},
	}
	for _, b := range bad {
		if _, err := s.db.ExecContext(ctx, b.sql, b.args...); err == nil {
			t.Errorf("%s: database accepted it", b.name)
		}
	}
}

// TestSQLiteConcurrentWriters opens two stores on one file, as two taskmoth
// windows would, and writes from both at once.
func TestSQLiteConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	a, err := OpenDir(ctx, dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	b, err := OpenDir(ctx, dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()

	const perStore = 50
	var wg sync.WaitGroup
	errs := make(chan error, 2*perStore)
	for _, s := range []Store{a, b} {
		for i := 0; i < perStore; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tk, err := s.CreateTask(ctx, task.Task{Name: fmt.Sprintf("task %d", i)})
				if err == nil {
					_, err = s.SetDone(ctx, tk.ID, true)
				}
				errs <- err
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	all, err := a.ListTasks(ctx)
	if err != nil || len(all) != 2*perStore {
		t.Errorf("got %d tasks, want %d (%v)", len(all), 2*perStore, err)
	}
}

func TestNewIDIsSortableUUIDv7(t *testing.T) {
	a, _ := NewID()
	time.Sleep(2 * time.Millisecond)
	b, _ := NewID()
	if len(a) != 36 || a[14] != '7' {
		t.Errorf("not a UUIDv7: %s", a)
	}
	if a >= b {
		t.Errorf("IDs should sort by creation time: %s >= %s", a, b)
	}
}
