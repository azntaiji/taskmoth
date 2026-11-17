package main

import (
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/config"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

func TestPickTheme(t *testing.T) {
	const path = "/home/moth/.config/taskmoth/config.toml"
	tests := []struct {
		name    string
		cfg     string
		env     string
		want    string
		wantErr string
	}{
		{name: "nothing set", want: theme.DefaultName},
		{name: "config file", cfg: "dracula", want: theme.DraculaName},
		{name: "env wins over file", cfg: "dracula", env: "default", want: theme.DefaultName},
		{name: "case and spaces", env: " Dracula ", want: theme.DraculaName},
		{name: "unknown in file", cfg: "solarized", wantErr: path + `: unknown theme "solarized"; use default or dracula`},
		{name: "unknown in env", env: "nord", wantErr: `TASKMOTH_THEME: unknown theme "nord"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := pickTheme(config.Config{Theme: tt.cfg}, tt.env, path)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || p.Name != tt.want {
				t.Fatalf("pickTheme = %q, %v; want %q", p.Name, err, tt.want)
			}
		})
	}
}
