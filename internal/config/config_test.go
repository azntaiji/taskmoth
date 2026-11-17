package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingFileIsDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), FileName))
	if err != nil || c != (Config{}) {
		t.Fatalf("Load = %+v, %v; want zero, nil", c, err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name, body string
		want       Config
		wantErr    string
	}{
		{name: "empty", body: ""},
		{name: "comments only", body: "# theme = \"dracula\"\n"},
		{name: "theme", body: "theme = \"dracula\"\n", want: Config{Theme: "dracula"}},
		{name: "unknown key", body: "them = \"dracula\"\n", wantErr: "unknown setting them"},
		{name: "wrong type", body: "theme = 3\n", wantErr: "config.toml"},
		{name: "malformed", body: "theme = \n", wantErr: "config.toml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(write(t, tt.body))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("Load = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}
