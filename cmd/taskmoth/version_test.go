package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionString(t *testing.T) {
	vcs := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "a1b2c3d4e5f6"}, {Key: "vcs.time", Value: "2026-11-20T10:00:00Z"}}}
	cases := []struct {
		name                  string
		version, commit, date string
		bi                    *debug.BuildInfo
		want                  string
	}{
		{"goreleaser", "1.0.0", "a1b2c3d4e5", "2026-11-20T10:00:00Z", nil, "taskmoth 1.0.0 (a1b2c3d, 2026-11-20)"},
		{"go install", "", "", "", &debug.BuildInfo{Main: debug.Module{Version: "v1.0.0"}}, "taskmoth 1.0.0"},
		{"local build", "", "", "", vcs, "taskmoth dev (a1b2c3d)"},
		{"nothing known", "", "", "", nil, "taskmoth dev"},
	}
	for _, c := range cases {
		if got := versionString(c.version, c.commit, c.date, c.bi); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
