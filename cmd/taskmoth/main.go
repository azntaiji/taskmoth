// Command taskmoth is a keyboard-driven terminal task manager.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/paths"
	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/ui"
	"github.com/azntaiji/taskmoth/internal/ui/gallery"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

type options struct {
	dataDir string
	ascii   bool
	gallery bool
	seed    bool
	list    string
}

func main() {
	fs := flag.NewFlagSet("taskmoth", flag.ContinueOnError)
	var o options
	fs.StringVar(&o.dataDir, "data", "", "data directory (default $XDG_DATA_HOME/taskmoth or ~/.local/share/taskmoth)")
	fs.BoolVar(&o.ascii, "ascii", false, "draw with ASCII instead of Unicode symbols (also TASKMOTH_ASCII=1)")
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

	if o.seed || o.list != "" {
		return runDev(o, dirs.DataDir)
	}

	t := theme.New(o.ascii)
	if o.gallery {
		_, err = tea.NewProgram(gallery.New(t)).Run()
		return err
	}
	return runApp(t, dirs.DataDir)
}

// runApp opens the database and runs the TUI.
func runApp(t theme.Theme, dataDir string) (err error) {
	s, err := store.OpenDir(context.Background(), dataDir, store.Options{})
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() {
		if cerr := s.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing database: %w", cerr)
		}
	}()
	_, err = tea.NewProgram(ui.NewApp(t, s, time.Now)).Run()
	return err
}
