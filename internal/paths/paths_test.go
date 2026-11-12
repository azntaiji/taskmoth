package paths

import (
	"errors"
	"path/filepath"
	"testing"
)

func fakeEnv(vars map[string]string, home string, homeErr error) Env {
	return Env{
		Getenv:  func(k string) string { return vars[k] },
		HomeDir: func() (string, error) { return home, homeErr },
	}
}

func TestResolve(t *testing.T) {
	home := filepath.FromSlash("/home/moth")
	tests := []struct {
		name       string
		vars       map[string]string
		override   string
		wantData   string
		wantConfig string
	}{
		{
			name:       "defaults",
			wantData:   filepath.Join(home, ".local", "share", "taskmoth"),
			wantConfig: filepath.Join(home, ".config", "taskmoth"),
		},
		{
			name:       "xdg vars set",
			vars:       map[string]string{"XDG_DATA_HOME": "/xdg/data", "XDG_CONFIG_HOME": "/xdg/config"},
			wantData:   filepath.Join("/xdg/data", "taskmoth"),
			wantConfig: filepath.Join("/xdg/config", "taskmoth"),
		},
		{
			name:       "relative xdg vars ignored",
			vars:       map[string]string{"XDG_DATA_HOME": "rel/data", "XDG_CONFIG_HOME": "rel/config"},
			wantData:   filepath.Join(home, ".local", "share", "taskmoth"),
			wantConfig: filepath.Join(home, ".config", "taskmoth"),
		},
		{
			name:       "data override wins",
			vars:       map[string]string{"XDG_DATA_HOME": "/xdg/data"},
			override:   "/custom/place",
			wantData:   filepath.FromSlash("/custom/place"),
			wantConfig: filepath.Join(home, ".config", "taskmoth"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(fakeEnv(tt.vars, home, nil), tt.override)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got.DataDir != tt.wantData {
				t.Errorf("DataDir = %q, want %q", got.DataDir, tt.wantData)
			}
			if got.ConfigDir != tt.wantConfig {
				t.Errorf("ConfigDir = %q, want %q", got.ConfigDir, tt.wantConfig)
			}
		})
	}
}

func TestResolveRelativeOverrideIsMadeAbsolute(t *testing.T) {
	got, err := Resolve(fakeEnv(nil, "/home/moth", nil), "rel/dir")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !filepath.IsAbs(got.DataDir) {
		t.Errorf("DataDir %q is not absolute", got.DataDir)
	}
}

func TestResolveNoHome(t *testing.T) {
	if _, err := Resolve(fakeEnv(nil, "", errors.New("no home")), ""); err == nil {
		t.Error("expected error when home directory is unavailable")
	}
	if _, err := Resolve(fakeEnv(nil, "", nil), ""); err == nil {
		t.Error("expected error when home directory is empty")
	}
}
