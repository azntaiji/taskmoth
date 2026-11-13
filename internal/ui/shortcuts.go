package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// shortcutsDialog is the Keyboard Shortcuts dialog (Figma 40:15079), opened
// with `?`. Its columns are generated from the key map (G7.3b); `esc` or `?`
// closes it (G7.3a).
type shortcutsDialog struct{}

func (d *shortcutsDialog) update(_ *App, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		return true, nil
	}
	return false, nil
}

// columnGap is the space between columns (Figma: 24px).
const columnGap = 2

// shortcutColumn renders one key group: a bold title, then a hint per line.
func shortcutColumn(t theme.Theme, g keyGroup) []string {
	key := t.Style(t.C.Key, t.C.Bg).Bold(true)
	label := t.Style(t.C.KeyLabel, t.C.Bg)
	out := []string{t.Style(t.C.Text, t.C.Bg).Bold(true).Render(g.title)}
	for _, b := range g.bindings {
		out = append(out, key.Render(b.key(t.G))+t.Base().Render(" ")+label.Render(b.long))
	}
	return out
}

// shortcutRows lays the columns out side by side, starting a new row of
// columns when the next one wouldn't fit in maxWidth.
func shortcutRows(t theme.Theme, maxWidth int) []string {
	base := t.Base()
	var out, row []string
	rowWidth := 0
	flush := func() {
		if len(out) > 0 && len(row) > 0 {
			out = append(out, "")
		}
		out = append(out, row...)
		row, rowWidth = nil, 0
	}
	for _, g := range keyGroups {
		col := shortcutColumn(t, g)
		cw := 0
		for _, l := range col {
			cw = max(cw, w.Width(l))
		}
		gap := columnGap
		if rowWidth == 0 {
			gap = 0
		}
		if rowWidth > 0 && rowWidth+gap+cw > maxWidth {
			flush()
			gap = 0
		}
		for len(row) < len(col) {
			row = append(row, w.Blank(base, rowWidth))
		}
		for i := range row {
			cell := ""
			if i < len(col) {
				cell = col[i]
			}
			row[i] = w.Fit(row[i], rowWidth, base) + w.Blank(base, gap) + w.Fit(cell, cw, base)
		}
		rowWidth += gap + cw
	}
	flush()
	return out
}

func (d *shortcutsDialog) view(a *App, t theme.Theme) string {
	rows := shortcutRows(t, a.width-4)
	bw := 0
	for _, r := range rows {
		bw = max(bw, w.Width(r))
	}
	body := append([]string{""}, rows...)
	body = append(body, "", w.HintRow(t, bw, []w.Hint{hintFor(t.G, "esc", "close")}), "")
	return w.Frame(t, w.FrameOptions{Width: bw + 4, Title: "Keyboard Shortcuts"}, body)
}
