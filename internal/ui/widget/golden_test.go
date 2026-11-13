package widget

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var update = flag.Bool("update", false, "rewrite golden files")

// golden compares got with testdata/<name>.golden (raw ANSI, viewable with
// `cat`). A plain-text copy is written alongside as <name>.txt for review.
// Run `go test ./internal/ui/widget -update` to rewrite them.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		plain := filepath.Join("testdata", name+".txt")
		if err := os.WriteFile(plain, []byte(ansi.Strip(got)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run with -update): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("%s differs from golden.\n--- want (plain)\n%s\n--- got (plain)\n%s",
			name, ansi.Strip(string(want)), ansi.Strip(got))
	}
}

// assertBlock checks that every line of s is exactly w cells wide and that
// there are h lines.
func assertBlock(t *testing.T, s string, w, h int) {
	t.Helper()
	lines := strings.Split(s, "\n")
	if len(lines) != h {
		t.Errorf("got %d lines, want %d", len(lines), h)
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Errorf("line %d is %d cells wide, want %d: %q", i, got, w, ansi.Strip(l))
		}
	}
}
