package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrCorrupt means the database file is damaged or isn't a SQLite database
// (decision G8.5). taskmoth stops rather than touching it.
var ErrCorrupt = errors.New("the database file is damaged or isn't a taskmoth database")

// keepBackups is how many pre-migration backups are kept (decision G8.2).
const keepBackups = 3

// backupPrefix starts every backup's file name, after the database's own.
const backupPrefix = ".bak-v"

// check runs SQLite's quick integrity check (decision G8.5).
func (s *SQLite) check(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA quick_check")
	if err != nil {
		return corruptErr(err)
	}
	defer func() { _ = rows.Close() }()
	var problems []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return corruptErr(err)
		}
		if line != "ok" {
			problems = append(problems, line)
		}
	}
	if err := rows.Err(); err != nil {
		return corruptErr(err)
	}
	if len(problems) > 0 {
		if len(problems) > 3 {
			problems = append(problems[:3], "…")
		}
		return fmt.Errorf("%w (%s)", ErrCorrupt, strings.Join(problems, "; "))
	}
	return nil
}

// corruptErr reports SQLite's "not a database" and "malformed" errors as
// ErrCorrupt; other errors pass through.
func corruptErr(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() & 0xff {
		case sqlite3.SQLITE_NOTADB, sqlite3.SQLITE_CORRUPT:
			return fmt.Errorf("%w (%w)", ErrCorrupt, err)
		}
	}
	return err
}

// backup copies the database before a migration (decision G8.2), using
// VACUUM INTO, which writes a consistent copy including WAL contents. It
// then keeps only the newest keepBackups copies.
func (s *SQLite) backup(ctx context.Context, fromVersion int) error {
	if s.path == "" || strings.HasPrefix(s.path, ":memory:") {
		return nil
	}
	base := fmt.Sprintf("%s%s%d-%s", s.path, backupPrefix, fromVersion, s.opts.Now().Format("20060102"))
	dest := base
	for n := 2; ; n++ {
		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			break
		}
		dest = fmt.Sprintf("%s-%d", base, n)
	}
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		return fmt.Errorf("backing up before the upgrade: %w", err)
	}
	all, err := Backups(s.path)
	if err != nil {
		return err
	}
	for _, old := range all[min(keepBackups, len(all)):] {
		if err := os.Remove(old); err != nil {
			return fmt.Errorf("removing an old backup: %w", err)
		}
	}
	return nil
}

// Backups lists the pre-migration backups of the database at dbPath,
// newest first.
func Backups(dbPath string) ([]string, error) {
	matches, err := filepath.Glob(globEscape(dbPath) + backupPrefix + "*")
	if err != nil {
		return nil, err
	}
	type file struct {
		path string
		mod  time.Time
	}
	files := make([]file, 0, len(matches))
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			continue // removed meanwhile
		}
		files = append(files, file{m, fi.ModTime()})
	}
	slices.SortFunc(files, func(a, b file) int {
		if c := b.mod.Compare(a.mod); c != 0 {
			return c
		}
		return strings.Compare(b.path, a.path)
	})
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.path
	}
	return out, nil
}

// globEscape escapes glob metacharacters in a literal path.
func globEscape(p string) string {
	return strings.NewReplacer(`\`, `\\`, "*", `\*`, "?", `\?`, "[", `\[`).Replace(p)
}
