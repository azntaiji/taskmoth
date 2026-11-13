// Command taskmoth is a keyboard-driven terminal task manager.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/paths"
	"github.com/azntaiji/taskmoth/internal/ui"
	"github.com/azntaiji/taskmoth/internal/ui/gallery"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

type options struct {
	dataDir string
	ascii   bool
	gallery bool
}

func main() {
	fs := flag.NewFlagSet("taskmoth", flag.ContinueOnError)
	var o options
	fs.StringVar(&o.dataDir, "data", "", "data directory (default $XDG_DATA_HOME/taskmoth or ~/.local/share/taskmoth)")
	fs.BoolVar(&o.ascii, "ascii", false, "draw with ASCII instead of Unicode symbols (also TASKMOTH_ASCII=1)")
	fs.BoolVar(&o.gallery, "gallery", false, "show the widget gallery (development)")
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
	// Resolved now so a bad --data value fails fast; the store that uses it
	// arrives in Phase 2.
	if _, err := paths.Resolve(paths.Env{Getenv: os.Getenv, HomeDir: os.UserHomeDir}, o.dataDir); err != nil {
		return fmt.Errorf("resolving directories: %w", err)
	}

	t := theme.New(o.ascii)
	var model tea.Model = ui.NewApp(t)
	if o.gallery {
		model = gallery.New(t)
	}
	_, err := tea.NewProgram(model).Run()
	return err
}
