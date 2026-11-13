package widget

import (
	"strings"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Scroll describes a scrollable region for Scrollbar.
type Scroll struct {
	Total   int  // number of rows of content
	Visible int  // number of rows shown
	Offset  int  // index of the first row shown
	Focused bool // draws the track in border/focused
}

// Scrollbar renders a one-cell-wide scrollbar of the given height as one
// string per row (Figma "Scrollbar", node 29:7311). As in the design, the
// thumb travels between the first and last track cells, so one track cell is
// always visible above and below it. The thumb is proportional to
// Visible/Total and hidden when everything fits (decision G1.5), so a thumb
// always means there is more to scroll.
func Scrollbar(t theme.Theme, height int, s Scroll) []string {
	trackColor := t.C.ScrollTrack
	if s.Focused {
		trackColor = t.C.BorderFocused
	}
	track := t.Style(trackColor, t.C.Bg).Render(t.G.Track)
	thumb := t.Style(t.C.ScrollThumb, t.C.Bg).Render(t.G.Thumb)

	rows := make([]string, height)
	for i := range rows {
		rows[i] = track
	}
	inner := height - 2
	if inner < 1 {
		return rows
	}

	size, pos := thumbGeometry(inner, s)
	for i := 0; i < size; i++ {
		rows[1+pos+i] = thumb
	}
	return rows
}

// thumbGeometry returns the thumb's size and position within a track of n
// cells. Size is 0 when all content is visible.
func thumbGeometry(n int, s Scroll) (size, pos int) {
	if s.Total <= 0 || s.Visible >= s.Total {
		return 0, 0
	}
	size = (n*s.Visible + s.Total/2) / s.Total
	size = min(max(size, 1), n)

	maxOffset := s.Total - s.Visible
	off := min(max(s.Offset, 0), maxOffset)
	pos = ((n-size)*off + maxOffset/2) / maxOffset
	return size, pos
}

// FrameOptions configures Frame.
type FrameOptions struct {
	Width   int     // total width including borders
	Title   string  // inset in the top border; empty for none
	Focused bool    // focused: border/focused border and text/accent title
	Scroll  *Scroll // nil leaves the scrollbar column blank
}

// FrameBodyWidth is the width available to body lines in a frame of the
// given total width. With a scrollbar: border, padding, body, padding, gap,
// scrollbar, border. Without one (Figma showScrollbar=false, e.g. Keyboard
// Shortcuts): border, padding, body, padding, border.
func FrameBodyWidth(width int, scrollbar bool) int {
	if scrollbar {
		return width - 6
	}
	return width - 4
}

// Frame renders a bordered panel (Figma "Panel", node 7:797, and "Dialog",
// node 11:729) around body lines. Each body line should be
// FrameBodyWidth(o.Width, o.Scroll != nil) cells wide; shorter lines are
// padded and longer ones cut. The frame is len(body)+2 rows tall.
func Frame(t theme.Theme, o FrameOptions, body []string) string {
	borderColor, titleColor := t.C.BorderDefault, t.C.Text
	if o.Focused {
		borderColor, titleColor = t.C.BorderFocused, t.C.TextAccent
	}
	border := t.Style(borderColor, t.C.Bg)
	base := t.Base()
	w := max(o.Width, 8)
	bw := FrameBodyWidth(w, o.Scroll != nil)

	lines := make([]string, 0, len(body)+2)

	// Top edge: ┌─ Title ───┐
	top := border.Render(t.G.TL + t.G.H)
	if o.Title != "" {
		title := Truncate(t, o.Title, w-6)
		top += t.Style(titleColor, t.C.Bg).Bold(true).Render(" " + title + " ")
	}
	top += border.Render(strings.Repeat(t.G.H, max(w-1-Width(top), 0)) + t.G.TR)
	lines = append(lines, top)

	var bar []string
	if o.Scroll != nil {
		bar = Scrollbar(t, len(body), *o.Scroll)
	}
	for i, l := range body {
		right := base.Render(" ") // padding
		if bar != nil {
			right = base.Render("  ") + bar[i] // padding, gap, scrollbar
		}
		lines = append(lines,
			border.Render(t.G.V)+base.Render(" ")+Fit(l, bw, base)+right+border.Render(t.G.V))
	}

	lines = append(lines, border.Render(t.G.BL+strings.Repeat(t.G.H, w-2)+t.G.BR))
	return strings.Join(lines, "\n")
}
