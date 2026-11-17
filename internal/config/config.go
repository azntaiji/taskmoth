// Package config reads taskmoth's settings file,
// $XDG_CONFIG_HOME/taskmoth/config.toml (decision G9.1).
//
// The file is optional; a missing file or setting means the default. A
// malformed file or an unknown setting is an error, so a typo doesn't
// silently do nothing.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the settings file's name in the config directory.
const FileName = "config.toml"

// Config holds the settings. A zero value means "not set".
type Config struct {
	// Theme names the colour theme: "default" or "dracula" (G9.5).
	Theme string `toml:"theme"`
}

// Load reads the settings file at path. A missing file returns the zero
// Config and no error.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	md, err := toml.Decode(string(data), &c)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if extra := md.Undecoded(); len(extra) > 0 {
		keys := make([]string, len(extra))
		for i, k := range extra {
			keys[i] = k.String()
		}
		return Config{}, fmt.Errorf("%s: unknown setting %s", path, strings.Join(keys, ", "))
	}
	return c, nil
}
