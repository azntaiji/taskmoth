// Package widget renders taskmoth's building blocks from the Figma
// Components page. Widgets are pure functions of a theme and their inputs and
// return styled strings measured in terminal cells.
package widget

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Width returns the display width of s in cells, ignoring ANSI sequences.
func Width(s string) int { return ansi.StringWidth(s) }

// Truncate shortens plain text s to at most w cells, ending in the theme's
// ellipsis when it had to cut. Truncate before styling so the ellipsis keeps
// the surrounding style.
func Truncate(t theme.Theme, s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	tail := t.G.Ellipsis
	if ansi.StringWidth(tail) > w {
		tail = ""
	}
	return ansi.Truncate(s, w, tail)
}

// Blank returns n cells of spaces in style st.
func Blank(st lipgloss.Style, n int) string {
	if n <= 0 {
		return ""
	}
	return st.Render(strings.Repeat(" ", n))
}

// Fit makes the styled string s exactly w cells wide: shorter strings are
// padded with blanks in style fill, longer ones are cut (without an ellipsis;
// truncate plain text beforehand for that).
func Fit(s string, w int, fill lipgloss.Style) string {
	sw := ansi.StringWidth(s)
	switch {
	case sw == w:
		return s
	case sw < w:
		return s + Blank(fill, w-sw)
	default:
		return ansi.Truncate(s, w, "")
	}
}

// Spread places left and right in a line exactly w cells wide, filling the gap
// with style fill. If both don't fit, right is dropped and left is cut.
func Spread(left, right string, w int, fill lipgloss.Style) string {
	lw, rw := ansi.StringWidth(left), ansi.StringWidth(right)
	if lw+rw > w {
		return Fit(left, w, fill)
	}
	return left + Blank(fill, w-lw-rw) + right
}

// Lines splits a rendered block into its lines.
func Lines(s string) []string { return strings.Split(s, "\n") }
