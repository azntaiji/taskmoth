package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Set by GoReleaser with -ldflags "-X main.version=… -X main.commit=… -X
// main.date=…" (decision G8.6). Empty in other builds.
var version, commit, date string

// versionString formats --version: "taskmoth 1.0.0 (a1b2c3d, 2026-11-20)".
// Without ldflags it falls back to Go's build info: the module version for
// `go install …@v1.0.0`, otherwise "dev", plus the VCS revision and time
// when the build recorded them.
func versionString(version, commit, date string, bi *debug.BuildInfo) string {
	if bi != nil {
		if version == "" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch {
			case s.Key == "vcs.revision" && commit == "":
				commit = s.Value
			case s.Key == "vcs.time" && date == "" && version != "":
				date = s.Value
			}
		}
	}
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		version = "dev"
	}
	if len(commit) > 7 {
		commit = commit[:7]
	}
	if len(date) > 10 {
		date = date[:10] // RFC 3339 → YYYY-MM-DD
	}
	var extra []string
	for _, s := range []string{commit, date} {
		if s != "" {
			extra = append(extra, s)
		}
	}
	if len(extra) == 0 {
		return "taskmoth " + version
	}
	return fmt.Sprintf("taskmoth %s (%s)", version, strings.Join(extra, ", "))
}
