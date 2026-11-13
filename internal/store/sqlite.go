package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/azntaiji/taskmoth/internal/task"
)

// FileName is the database file inside the data directory.
const FileName = "taskmoth.db"

//go:embed migrations/*.sql
var migrationFS embed.FS

// timeLayout is fixed-width UTC RFC 3339 with milliseconds, so stored
// timestamps sort correctly as text.
const timeLayout = "2006-01-02T15:04:05.000Z"

// SQLite is the SQLite-backed Store.
type SQLite struct {
	db   *sql.DB
	opts Options
}

var _ Store = (*SQLite)(nil)

// OpenDir opens (creating if needed) the database in dataDir.
func OpenDir(ctx context.Context, dataDir string, opts Options) (*SQLite, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	return Open(ctx, filepath.Join(dataDir, FileName), opts)
}

// Open opens (creating if needed) the database file at path and migrates it
// to the latest schema.
func Open(ctx context.Context, path string, opts Options) (*SQLite, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Set("_txlock", "immediate") // take the write lock at BEGIN, so busy_timeout applies
	db, err := sql.Open("sqlite", path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	s := &SQLite{db: db, opts: opts.withDefaults()}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close() // already returning the migration error
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *SQLite) Close() error { return s.db.Close() }

type migration struct {
	version int
	name    string
	sql     string
}

func migrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		num, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(num)
		if !ok || err != nil {
			return nil, fmt.Errorf("migration %q: name must start with a number", e.Name())
		}
		b, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: v, name: e.Name(), sql: string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	for i, m := range out {
		if m.version != i+1 {
			return nil, fmt.Errorf("migrations must be numbered 1, 2, 3…; found %s", m.name)
		}
	}
	return out, nil
}

// SchemaVersion returns the database's schema version (PRAGMA user_version).
func (s *SQLite) SchemaVersion(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	return v, err
}

func (s *SQLite) migrate(ctx context.Context) error {
	ms, err := migrations()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	var current int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return err
	}
	latest := len(ms)
	if current > latest {
		return fmt.Errorf("%w (schema %d, this build knows up to %d)", ErrNewerSchema, current, latest)
	}
	for _, m := range ms[current:] {
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	if current == latest {
		return nil
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", latest)); err != nil {
		return err
	}
	return tx.Commit()
}

// --- encoding ---

func encTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(timeLayout)
}

func decTime(s sql.NullString) (time.Time, error) {
	if !s.Valid {
		return time.Time{}, nil
	}
	return time.Parse(timeLayout, s.String)
}

func encDate(d task.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func encID(id task.ID) any {
	if id == "" {
		return nil
	}
	return string(id)
}

func encMinutes(m int) any {
	if m == 0 {
		return nil
	}
	return m
}

const taskColumns = `id, name, description, project_id, priority, start_date, due_date,
	duration_minutes, created_at, updated_at, completed_at, deleted_at`

type scanner interface{ Scan(...any) error }

func scanTask(r scanner) (task.Task, error) {
	var (
		t                               task.Task
		project, start, due             sql.NullString
		minutes                         sql.NullInt64
		created, updated, done, trashed sql.NullString
	)
	if err := r.Scan(&t.ID, &t.Name, &t.Description, &project, &t.Priority, &start, &due,
		&minutes, &created, &updated, &done, &trashed); err != nil {
		return task.Task{}, err
	}
	t.ProjectID = task.ID(project.String)
	t.DurationMin = int(minutes.Int64)
	var err error
	if t.Start, err = task.ParseDate(start.String); err != nil {
		return t, err
	}
	if t.Due, err = task.ParseDate(due.String); err != nil {
		return t, err
	}
	for _, f := range []struct {
		dst *time.Time
		src sql.NullString
	}{{&t.CreatedAt, created}, {&t.UpdatedAt, updated}, {&t.CompletedAt, done}, {&t.DeletedAt, trashed}} {
		if *f.dst, err = decTime(f.src); err != nil {
			return t, err
		}
	}
	return t, nil
}

func scanProject(r scanner) (task.Project, error) {
	var p task.Project
	var created, updated sql.NullString
	if err := r.Scan(&p.ID, &p.Name, &created, &updated); err != nil {
		return p, err
	}
	var err error
	if p.CreatedAt, err = decTime(created); err != nil {
		return p, err
	}
	p.UpdatedAt, err = decTime(updated)
	return p, err
}

// mapErr turns SQLite constraint errors into store errors.
func mapErr(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return ErrDuplicateProject // the only UNIQUE constraint is projects.name
		case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return ErrUnknownProject
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// --- tasks ---

// ListTasks implements Store.
func (s *SQLite) ListTasks(ctx context.Context) ([]task.Task, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+taskColumns+" FROM tasks ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }() // errors surface via rows.Err
	var out []task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTask implements Store.
func (s *SQLite) GetTask(ctx context.Context, id task.ID) (task.Task, error) {
	t, err := scanTask(s.db.QueryRowContext(ctx, "SELECT "+taskColumns+" FROM tasks WHERE id = ?", string(id)))
	return t, mapErr(err)
}

// CreateTask implements Store.
func (s *SQLite) CreateTask(ctx context.Context, t task.Task) (task.Task, error) {
	t, err := prepareNew(t)
	if err != nil {
		return task.Task{}, err
	}
	if t.ID, err = s.opts.NewID(); err != nil {
		return task.Task{}, err
	}
	now := s.opts.Now().UTC().Truncate(time.Millisecond)
	t.CreatedAt, t.UpdatedAt = now, now
	_, err = s.db.ExecContext(ctx, "INSERT INTO tasks ("+taskColumns+") VALUES (?,?,?,?,?,?,?,?,?,?,NULL,NULL)",
		string(t.ID), t.Name, t.Description, encID(t.ProjectID), t.Priority, encDate(t.Start), encDate(t.Due),
		encMinutes(t.DurationMin), encTime(t.CreatedAt), encTime(t.UpdatedAt))
	if err != nil {
		return task.Task{}, mapErr(err)
	}
	return t, nil
}

// UpdateTask implements Store.
func (s *SQLite) UpdateTask(ctx context.Context, t task.Task) (task.Task, error) {
	if err := t.Validate(); err != nil {
		return task.Task{}, err
	}
	now := s.opts.Now().UTC().Truncate(time.Millisecond)
	res, err := s.db.ExecContext(ctx, `UPDATE tasks SET name=?, description=?, project_id=?, priority=?,
		start_date=?, due_date=?, duration_minutes=?, updated_at=? WHERE id=?`,
		t.Name, t.Description, encID(t.ProjectID), t.Priority, encDate(t.Start), encDate(t.Due),
		encMinutes(t.DurationMin), encTime(now), string(t.ID))
	if err != nil {
		return task.Task{}, mapErr(err)
	}
	if err := oneRow(res); err != nil {
		return task.Task{}, err
	}
	return s.GetTask(ctx, t.ID)
}

func oneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// setTime updates one timestamp column (and updated_at) on a task.
func (s *SQLite) setTime(ctx context.Context, id task.ID, column string, value any, onlyIfNull bool) (task.Task, error) {
	now := encTime(s.opts.Now().UTC().Truncate(time.Millisecond))
	q := fmt.Sprintf("UPDATE tasks SET %s = ?, updated_at = ? WHERE id = ?", column)
	if onlyIfNull {
		q = fmt.Sprintf("UPDATE tasks SET %s = COALESCE(%s, ?), updated_at = ? WHERE id = ?", column, column)
	}
	res, err := s.db.ExecContext(ctx, q, value, now, string(id))
	if err != nil {
		return task.Task{}, mapErr(err)
	}
	if err := oneRow(res); err != nil {
		return task.Task{}, err
	}
	return s.GetTask(ctx, id)
}

// SetDone implements Store.
func (s *SQLite) SetDone(ctx context.Context, id task.ID, done bool) (task.Task, error) {
	if !done {
		return s.setTime(ctx, id, "completed_at", nil, false)
	}
	return s.setTime(ctx, id, "completed_at", encTime(s.opts.Now()), true)
}

// Trash implements Store.
func (s *SQLite) Trash(ctx context.Context, id task.ID) (task.Task, error) {
	return s.setTime(ctx, id, "deleted_at", encTime(s.opts.Now()), true)
}

// Restore implements Store.
func (s *SQLite) Restore(ctx context.Context, id task.ID) (task.Task, error) {
	return s.setTime(ctx, id, "deleted_at", nil, false)
}

// Purge implements Store.
func (s *SQLite) Purge(ctx context.Context, id task.ID) error {
	t, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if !t.Trashed() {
		return ErrNotTrashed
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ? AND deleted_at IS NOT NULL", string(id))
	return mapErr(err)
}

// --- projects ---

const projectColumns = "id, name, created_at, updated_at"

// ListProjects implements Store.
func (s *SQLite) ListProjects(ctx context.Context) ([]task.Project, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+projectColumns+" FROM projects ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }() // errors surface via rows.Err
	var out []task.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProject implements Store.
func (s *SQLite) GetProject(ctx context.Context, id task.ID) (task.Project, error) {
	p, err := scanProject(s.db.QueryRowContext(ctx, "SELECT "+projectColumns+" FROM projects WHERE id = ?", string(id)))
	return p, mapErr(err)
}

// ProjectByName implements Store.
func (s *SQLite) ProjectByName(ctx context.Context, name string) (task.Project, error) {
	p, err := scanProject(s.db.QueryRowContext(ctx, "SELECT "+projectColumns+" FROM projects WHERE name = ?",
		task.NormalizeProjectName(name)))
	return p, mapErr(err)
}

// CreateProject implements Store.
func (s *SQLite) CreateProject(ctx context.Context, name string) (task.Project, error) {
	name = task.NormalizeProjectName(name)
	if err := task.ValidateProjectName(name); err != nil {
		return task.Project{}, err
	}
	id, err := s.opts.NewID()
	if err != nil {
		return task.Project{}, err
	}
	now := s.opts.Now().UTC().Truncate(time.Millisecond)
	p := task.Project{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}
	_, err = s.db.ExecContext(ctx, "INSERT INTO projects ("+projectColumns+") VALUES (?,?,?,?)",
		string(p.ID), p.Name, encTime(now), encTime(now))
	if err != nil {
		return task.Project{}, mapErr(err)
	}
	return p, nil
}

// DeleteProject implements Store.
func (s *SQLite) DeleteProject(ctx context.Context, id task.ID) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", string(id))
	if err != nil {
		// ON DELETE RESTRICT reports SQLITE_CONSTRAINT_TRIGGER (1811) rather
		// than the plain foreign-key code, so match any constraint failure:
		// the only constraint a project delete can hit is tasks referencing it.
		var se *sqlite.Error
		if errors.As(err, &se) && se.Code()&0xff == sqlite3.SQLITE_CONSTRAINT {
			return ErrProjectInUse
		}
		return mapErr(err)
	}
	return oneRow(res)
}
