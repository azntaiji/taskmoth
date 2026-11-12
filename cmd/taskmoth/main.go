// Command taskmoth is a keyboard-driven terminal task manager.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/paths"
	"github.com/azntaiji/taskmoth/internal/ui"
)

func main() {
	fs := flag.NewFlagSet("taskmoth", flag.ContinueOnError)
	dataDir := fs.String("data", "", "data directory (default $XDG_DATA_HOME/taskmoth or ~/.local/share/taskmoth)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		// The flag package has already printed the error and usage.
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if err := run(*dataDir); err != nil {
		fmt.Fprintln(os.Stderr, "taskmoth:", err)
		os.Exit(1)
	}
}

func run(dataDir string) error {

	// Resolved now so a bad --data value fails fast; the store that uses it
	// arrives in Phase 2.
	if _, err := paths.Resolve(paths.Env{Getenv: os.Getenv, HomeDir: os.UserHomeDir}, dataDir); err != nil {
		return fmt.Errorf("resolving directories: %w", err)
	}

	_, err := tea.NewProgram(ui.NewApp()).Run()
	return err
}
