package widget

import (
	"fmt"
	"image/color"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Badge renders bold text with one cell of padding on each side in a tone's
// colours (Figma "Badge", node 7:423). dim applies the 50% dim opacity.
func Badge(t theme.Theme, tone theme.Tone, text string, dim bool) string {
	p := t.C.Badges[tone]
	fg, bg := p.Fg, p.Bg
	if dim {
		fg, bg = t.Fade(fg), t.Fade(bg)
	}
	return t.Style(fg, bg).Bold(true).Render(" " + text + " ")
}

// PriorityWidth is the fixed width of the priority slot (Figma
// "PriorityMark", 48px).
const PriorityWidth = 4

// PriorityTone maps P1–P4 to badge tones: P1 error, P2 warning, P3 neutral,
// P4 mode (which shares the page background, so it reads as plain bold text).
func PriorityTone(p int) theme.Tone {
	switch p {
	case 1:
		return theme.ToneError
	case 2:
		return theme.ToneWarning
	case 3:
		return theme.ToneNeutral
	default:
		return theme.ToneMode
	}
}

// PriorityMark renders the four-cell priority slot (Figma "PriorityMark",
// node 7:1828). Priority 0 (unset) renders blank cells on bg.
func PriorityMark(t theme.Theme, priority int, dim bool, bg color.Color) string {
	if priority < 1 || priority > 4 {
		return Blank(t.Style(t.C.Text, bg), PriorityWidth)
	}
	return Badge(t, PriorityTone(priority), fmt.Sprintf("P%d", priority), dim)
}

// DueState is the state of a task's due date (Figma "DueLabel", node 10:247).
type DueState int

// Due states.
const (
	DueNone DueState = iota
	DueUpcoming
	DueToday
	DueOverdue
)

// DueLabel renders the due badge: neutral for upcoming, warning for today,
// error for overdue. text is what the badge says ("today", "overdue", or a
// date). DueNone renders nothing.
func DueLabel(t theme.Theme, state DueState, text string, dim bool) string {
	var tone theme.Tone
	switch state {
	case DueUpcoming:
		tone = theme.ToneNeutral
	case DueToday:
		tone = theme.ToneWarning
	case DueOverdue:
		tone = theme.ToneError
	default:
		return ""
	}
	return Badge(t, tone, text, dim)
}

// CheckboxWidth is the width of a checkbox including its trailing space.
const CheckboxWidth = 4

// Checkbox renders "[ ] " or "[x] " (Figma "Checkbox", node 10:283); done is
// drawn in text/success. bg is the row background.
func Checkbox(t theme.Theme, done bool, bg color.Color) string {
	if done {
		return t.Style(t.C.TextSuccess, bg).Render("[x] ")
	}
	return t.Style(t.C.Text, bg).Render("[ ] ")
}

// MarkerWidth is the width of the selection marker slot.
const MarkerWidth = 2

// Marker renders the two-cell selection marker (Figma "Marker", node 10:277).
func Marker(t theme.Theme, selected bool, bg color.Color) string {
	if !selected {
		return Blank(t.Style(t.C.Text, bg), MarkerWidth)
	}
	return t.Style(t.C.Text, bg).Render(t.G.Marker + " ")
}
