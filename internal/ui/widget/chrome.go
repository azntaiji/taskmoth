package widget

import (
	"strings"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Header renders the top row (Figma "Header", node 5:398):
//
//	taskmoth ▸ <section>                  <right>
//
// The app name is bold, the separator is bold and muted, and the right-hand
// text is drawn at the 50% dim opacity. Gaps are one cell.
func Header(t theme.Theme, width int, section, right string) string {
	base := t.Base()
	left := base.Bold(true).Render("taskmoth") +
		base.Render(" ") +
		t.Style(t.C.TextMuted, t.C.Bg).Bold(true).Render(t.G.Breadcrumb) +
		base.Render(" ")
	used := Width(left)

	rightW := Width(right)
	if used+1+rightW > width { // no room for the right text
		right, rightW = "", 0
	}
	room := width - used - rightW
	if rightW > 0 {
		room-- // keep at least one cell before the right text
	}
	left += base.Render(Truncate(t, section, room))

	r := ""
	if rightW > 0 {
		r = t.Style(t.Fade(t.C.Text), t.C.Bg).Render(right)
	}
	return Spread(left, r, width, base)
}

// StatusBar renders the status row (Figma "StatusBar", node 6:412): one cell
// of padding, a badge, a two-cell gap, a message, and right-aligned text, all
// on statusbar/bg.
func StatusBar(t theme.Theme, width int, badge string, tone theme.Tone, message, right string) string {
	bg := t.Style(t.C.Status.Fg, t.C.Status.Bg)
	inner := width - 2

	left := ""
	if badge != "" {
		left = Badge(t, tone, badge, false) + Blank(bg, 2)
	}
	r := ""
	if right != "" {
		r = bg.Render(right)
	}
	room := inner - Width(left) - Width(r)
	if r != "" {
		room--
	}
	if room < 0 {
		r, room = "", inner-Width(left)
	}
	left += bg.Render(Truncate(t, message, room))

	return Blank(bg, 1) + Spread(left, r, inner, bg) + Blank(bg, 1)
}

// Hint is one key hint: a key (or key sequence) and what it does.
type Hint struct {
	Key   string
	Label string
}

// KeyHints renders rows of key hints (Figma "KeyHints", node 7:484): one cell
// of padding at each side, the key bold in key/key, one cell, the label in
// key/label, and two cells between hints. Rows that are too long are cut at
// the edge, as in the designs.
func KeyHints(t theme.Theme, width int, rows ...[]Hint) []string {
	base := t.Base()
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		line := HintRow(t, max(width-2, 0), row)
		out = append(out, Fit(Blank(base, 1)+line+Blank(base, 1), width, base))
	}
	return out
}

// HintRow renders one row of hints exactly width cells wide with no outer
// padding, for use inside dialogs, whose frame already pads its body.
func HintRow(t theme.Theme, width int, row []Hint) string {
	base := t.Base()
	key := t.Style(t.C.Key, t.C.Bg).Bold(true)
	label := t.Style(t.C.KeyLabel, t.C.Bg)

	var b strings.Builder
	for i, h := range row {
		if i > 0 {
			b.WriteString(base.Render("  "))
		}
		b.WriteString(key.Render(h.Key))
		if h.Label != "" {
			b.WriteString(base.Render(" ") + label.Render(h.Label))
		}
	}
	return Fit(b.String(), width, base)
}

// HintsWidth returns how many cells a row of hints needs, without padding.
func HintsWidth(row []Hint) int {
	w := 0
	for i, h := range row {
		if i > 0 {
			w += 2
		}
		w += Width(h.Key)
		if h.Label != "" {
			w += 1 + Width(h.Label)
		}
	}
	return w
}

// Divider renders a muted horizontal rule (Figma "Spacer", decoration=divider).
func Divider(t theme.Theme, width int) string {
	return t.Style(t.C.TextMuted, t.C.Bg).Render(strings.Repeat(t.G.H, max(width, 0)))
}
