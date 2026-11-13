package widget

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Overlay draws dialog centred over base, a full-screen view of width×height
// cells. Render base with theme.Dimmed() so it reads as dimmed behind the
// dialog (decision G1.4; see Figma node 32:13454).
func Overlay(base, dialog string, width, height int) string {
	dw, dh := lipgloss.Width(dialog), lipgloss.Height(dialog)
	x := max((width-dw)/2, 0)
	y := max((height-dh)/2, 0)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
	).Render()
}

// Screen pads or cuts lines to exactly width×height cells on bg/default.
func Screen(t theme.Theme, width, height int, lines []string) string {
	base := t.Base()
	out := make([]string, height)
	for i := range out {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out[i] = Fit(l, width, base)
	}
	return strings.Join(out, "\n")
}

// MinWidth and MinHeight are the smallest supported terminal size
// (decision G1.1).
const (
	MinWidth  = 80
	MinHeight = 24
)

// TooSmall renders the message shown when the terminal is below the minimum
// size.
func TooSmall(t theme.Theme, width, height int) string {
	x := t.G.Times
	msg := []string{
		"Terminal too small",
		fmt.Sprintf("taskmoth needs %d%s%d; this is %d%s%d.", MinWidth, x, MinHeight, width, x, height),
	}
	base := t.Base()
	muted := t.Style(t.Fade(t.C.Text), t.C.Bg)
	block := lipgloss.JoinVertical(lipgloss.Center,
		base.Bold(true).Render(Truncate(t, msg[0], width)),
		muted.Render(Truncate(t, msg[1], width)))
	return lipgloss.Place(max(width, 1), max(height, 1), lipgloss.Center, lipgloss.Center, block,
		lipgloss.WithWhitespaceStyle(base))
}
