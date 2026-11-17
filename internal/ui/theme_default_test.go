package ui

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// sgrParams lists every SGR parameter sequence in s, e.g. "1;35".
func sgrParams(s string) []string {
	var out []string
	for _, m := range sgr.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

func hasParam(seqs []string, p string) bool {
	for _, s := range seqs {
		if slices.Contains(strings.Split(s, ";"), p) {
			return true
		}
	}
	return false
}

// TestDefaultThemeUsesANSIColours renders screens with the default theme
// (G9.2–G9.4): only ANSI colour codes, no 24-bit or 256-colour ones, and
// the faint attribute for dimming. The goldens hold the rest.
func TestDefaultThemeUsesANSIColours(t *testing.T) {
	th := theme.New(theme.ANSI(), false)
	screens := map[string]func(a *App){
		"today_default":           func(*App) {},
		"task_view_default":       func(a *App) { press(a, "e") },
		"confirm_archive_default": func(a *App) { press(a, "x") },
	}
	for name, setup := range screens {
		t.Run(name, func(t *testing.T) {
			clk := &clock{}
			a := newAppTheme(t, th, designStore(t, clk), clk, 80, 24)
			setup(a)
			assertScreen(t, a, 80, 24)
			out := a.View().Content
			seqs := sgrParams(out)
			for _, s := range seqs {
				if strings.Contains(s, "38;") || strings.Contains(s, "48;") {
					t.Fatalf("extended colour %q in the default theme", s)
				}
			}
			if !hasParam(seqs, "2") {
				t.Error("no faint text: completed tasks and dimmed views should use it")
			}
			golden(t, name, out)
		})
	}
}
