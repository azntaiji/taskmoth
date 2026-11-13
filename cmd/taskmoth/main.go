// Command taskmoth is a keyboard-driven terminal task manager.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/lock"
	"github.com/azntaiji/taskmoth/internal/paths"
	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/ui"
	"github.com/azntaiji/taskmoth/internal/ui/gallery"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

type options struct {
	dataDir string
	ascii   bool
	version bool
	gallery bool
	seed    bool
	list    string
}

// devFlags work but are left out of --help (decision G8.3).
var devFlags = map[string]bool{"gallery": true, "seed": true, "list": true}

// usage prints --help: the user-facing flags only.
func usage(fs *flag.FlagSet) func() {
	return func() {
		w := fs.Output()
		_, _ = fmt.Fprint(w, "Usage: taskmoth [flags]\n\nA keyboard-driven task manager for the terminal.\n\n")
		fs.VisitAll(func(f *flag.Flag) {
			if devFlags[f.Name] {
				return
			}
			name, help := flag.UnquoteUsage(f)
			_, _ = fmt.Fprintf(w, "  --%-12s %s\n", strings.TrimSpace(f.Name+" "+name), help)
		})
		_, _ = fmt.Fprint(w, "\nIn the app, press ? for every key. Quit with ctrl+q.\n")
	}
}

func main() {
	fs := flag.NewFlagSet("taskmoth", flag.ContinueOnError)
	var o options
	fs.Usage = usage(fs)
	fs.StringVar(&o.dataDir, "data", "", "data `dir`ectory (default $XDG_DATA_HOME/taskmoth or ~/.local/share/taskmoth)")
	fs.BoolVar(&o.ascii, "ascii", false, "draw with ASCII instead of Unicode symbols (also TASKMOTH_ASCII=1)")
	fs.BoolVar(&o.version, "version", false, "print the version and exit")
	fs.BoolVar(&o.gallery, "gallery", false, "show the widget gallery (development)")
	fs.BoolVar(&o.seed, "seed", false, "load the sample data from the designs into an empty data directory (development)")
	fs.StringVar(&o.list, "list", "", "print a view and exit: "+strings.Join(listViews, ", ")+", or project:<name> (development)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		// The flag package has already printed the error and usage.
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}
	o.ascii = o.ascii || envTrue(os.Getenv("TASKMOTH_ASCII"))
	if o.version {
		bi, _ := debug.ReadBuildInfo()
		fmt.Println(versionString(version, commit, date, bi))
		return
	}

	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "taskmoth:", err)
		os.Exit(1)
	}
}

// envTrue reports whether an environment value switches an option on.
func envTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func run(o options) error {
	dirs, err := paths.Resolve(paths.Env{Getenv: os.Getenv, HomeDir: os.UserHomeDir}, o.dataDir)
	if err != nil {
		return fmt.Errorf("resolving directories: %w", err)
	}

	t := theme.New(o.ascii)
	if o.gallery {
		_, err = tea.NewProgram(gallery.New(t)).Run()
		return err
	}
	if o.list != "" && !o.seed {
		return runDev(o, dirs.DataDir) // read-only: no lock (G8.4)
	}

	release, err := lockData(dirs.DataDir)
	if err != nil {
		return err
	}
	defer func() { _ = release() }()
	if o.seed {
		return runDev(o, dirs.DataDir)
	}
	return runApp(t, dirs.DataDir)
}

// lockData takes the data directory's single-instance lock (G8.4).
func lockData(dataDir string) (func() error, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	l, err := lock.Acquire(filepath.Join(dataDir, lock.FileName))
	if errors.Is(err, lock.ErrLocked) {
		return nil, fmt.Errorf("already running with %s; close the other taskmoth first", filepath.Join(dataDir, store.FileName))
	}
	if err != nil {
		return nil, err
	}
	return l.Release, nil
}

// openError explains why the database couldn't be opened (G8.5), naming the
// file and the newest backup, if any.
func openError(dataDir string, err error) error {
	path := filepath.Join(dataDir, store.FileName)
	var msg string
	switch {
	case errors.Is(err, store.ErrCorrupt):
		msg = fmt.Sprintf("%s can't be used: %v.", path, err)
	case errors.Is(err, store.ErrNewerSchema):
		msg = fmt.Sprintf("%s was written by a newer taskmoth (%v). Update taskmoth to open it.", path, err)
	default:
		return fmt.Errorf("opening %s: %w", path, err)
	}
	msg += " It has not been changed."
	if b, _ := store.Backups(path); len(b) > 0 {
		msg += " The newest backup is " + b[0] + "."
	}
	return errors.New(msg)
}

// runApp opens the database and runs the TUI.
func runApp(t theme.Theme, dataDir string) (err error) {
	s, err := store.OpenDir(context.Background(), dataDir, store.Options{})
	if err != nil {
		return openError(dataDir, err)
	}
	defer func() {
		if cerr := s.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing database: %w", cerr)
		}
	}()
	_, err = tea.NewProgram(ui.NewApp(t, s, time.Now)).Run()
	return err
}
