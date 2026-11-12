// Package paths resolves taskmoth's data and config directories.
//
// taskmoth follows the XDG Base Directory spec on every OS, including macOS
// (decision G0.3 in PLAN.md):
//
//	data:   $XDG_DATA_HOME/taskmoth   (default ~/.local/share/taskmoth)
//	config: $XDG_CONFIG_HOME/taskmoth (default ~/.config/taskmoth)
package paths

import (
	"errors"
	"path/filepath"
)

const appName = "taskmoth"

// Paths holds the resolved directories. Nothing is created on disk.
type Paths struct {
	DataDir   string
	ConfigDir string
}

// Env supplies the values Resolve reads, so it can be tested without touching
// the real environment.
type Env struct {
	Getenv  func(string) string
	HomeDir func() (string, error)
}

// Resolve returns taskmoth's directories. A non-empty dataOverride (the
// --data flag) replaces the data directory.
func Resolve(env Env, dataOverride string) (Paths, error) {
	var p Paths

	if dataOverride != "" {
		abs, err := filepath.Abs(dataOverride)
		if err != nil {
			return Paths{}, err
		}
		p.DataDir = abs
	} else {
		dir, err := xdgDir(env, "XDG_DATA_HOME", ".local/share")
		if err != nil {
			return Paths{}, err
		}
		p.DataDir = dir
	}

	dir, err := xdgDir(env, "XDG_CONFIG_HOME", ".config")
	if err != nil {
		return Paths{}, err
	}
	p.ConfigDir = dir

	return p, nil
}

// xdgDir returns $<envVar>/taskmoth, falling back to ~/<fallback>/taskmoth.
// Per the XDG spec, a relative value in the variable is ignored.
func xdgDir(env Env, envVar, fallback string) (string, error) {
	if v := env.Getenv(envVar); v != "" && filepath.IsAbs(v) {
		return filepath.Join(v, appName), nil
	}
	home, err := env.HomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("cannot determine home directory")
	}
	return filepath.Join(home, filepath.FromSlash(fallback), appName), nil
}
