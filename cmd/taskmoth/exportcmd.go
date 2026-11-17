package main

// --export and --output (decisions G10.1–G10.4).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/azntaiji/taskmoth/internal/export"
	"github.com/azntaiji/taskmoth/internal/store"
)

// checkExportFlags validates --export and --output before anything is opened,
// normalizing the format name.
func checkExportFlags(o *options) error {
	if o.export == "" {
		if o.output != "" {
			return errors.New("--output needs --export csv or --export json")
		}
		return nil
	}
	f, err := export.ParseFormat(o.export)
	if err != nil {
		return err
	}
	o.export = f
	if o.seed || o.list != "" || o.gallery {
		return errors.New("--export can't be combined with --seed, --list or --gallery")
	}
	return nil
}

// runExport opens the database read-only in spirit (no lock, G10.4) and writes
// the export to --output or standard output.
func runExport(o options, dataDir string) (err error) {
	ctx := context.Background()
	s, err := store.OpenDir(ctx, dataDir, store.Options{})
	if err != nil {
		return openError(dataDir, err)
	}
	defer func() {
		if cerr := s.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing database: %w", cerr)
		}
	}()
	write := func(w io.Writer) error { return exportStore(ctx, w, s, o.export, time.Now()) }
	if o.output == "" {
		return write(os.Stdout)
	}
	return writeFileAtomic(o.output, write)
}

// exportStore loads every project and task and writes them in format.
func exportStore(ctx context.Context, w io.Writer, s store.Store, format string, now time.Time) error {
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return err
	}
	tasks, err := s.ListTasks(ctx)
	if err != nil {
		return err
	}
	return export.Write(w, format, projects, tasks, now)
}

// writeFileAtomic writes path through a temporary file in the same directory,
// so a failed export never leaves a half-written file or clobbers an old one.
// The file is readable only by its owner, like the data directory.
func writeFileAtomic(path string, write func(io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
	}()
	if err = write(f); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
