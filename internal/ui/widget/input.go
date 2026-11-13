package widget

import (
	"image/color"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// PromptWidth is the width of the "> " prompt column.
const PromptWidth = 2

// Input is an editable text field (Figma "Input", node 11:800, and the
// shaded multi-line fields in the edit dialogs, e.g. node 15:5973). Text
// soft-wraps at word boundaries; the "> " prompt appears on the first visual
// row only.
//
// Input handles cursor movement, insertion, deletion and paste. It does not
// bind a key to "new line": which key that is remains an open decision (G6.2
// in PLAN.md), so callers call InsertNewline themselves.
type Input struct {
	Placeholder string
	// Highlights colour parts of the value; the caller keeps them in sync.
	Highlights []Highlight
	// Multiline keeps newlines from pastes and allows InsertNewline.
	Multiline bool

	value   []rune
	cursor  int // index into value, 0..len(value)
	focused bool
	width   int // total width in cells, including the prompt
	height  int // visible rows
	top     int // first visible visual row
}

// Highlight colours runes [Start, End) of the value, e.g. tokens recognised
// by the quick-add parser (decision G5.1a).
type Highlight struct {
	Start, End int
	Fg         color.Color
}

// Cursor returns the cursor position as a rune offset into the value.
func (in *Input) Cursor() int { return in.cursor }

// SetValueCursor replaces the text and places the cursor.
func (in *Input) SetValueCursor(s string, cursor int) {
	in.SetValue(s)
	in.cursor = min(max(cursor, 0), len(in.value))
	in.scrollToCursor()
}

// SetValue replaces the text and moves the cursor to the end.
func (in *Input) SetValue(s string) {
	s = CleanText(s, in.Multiline)
	in.value = []rune(s)
	in.cursor = len(in.value)
	in.scrollToCursor()
}

// Value returns the current text.
func (in *Input) Value() string { return string(in.value) }

// Focus shows the cursor and accepts keys.
func (in *Input) Focus() { in.focused = true }

// Blur hides the cursor.
func (in *Input) Blur() { in.focused = false }

// Focused reports whether the input has focus.
func (in *Input) Focused() bool { return in.focused }

// SetSize sets the total width (including the prompt) and visible rows.
func (in *Input) SetSize(width, height int) {
	in.width, in.height = width, max(height, 1)
	in.scrollToCursor()
}

// InsertNewline inserts a line break at the cursor if the input is
// multi-line.
func (in *Input) InsertNewline() {
	if in.Multiline {
		in.insert([]rune{'\n'})
	}
}

// Update applies a key press or paste. It reports whether the message was
// used, so callers can handle the rest (enter, esc, …) themselves.
func (in *Input) Update(msg tea.Msg) bool {
	if !in.focused {
		return false
	}
	switch msg := msg.(type) {
	case tea.PasteMsg:
		in.insert([]rune(CleanText(msg.Content, in.Multiline)))
		return true
	case tea.KeyPressMsg:
		return in.handleKey(msg)
	}
	return false
}

func (in *Input) handleKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "left", "ctrl+b":
		in.cursor = max(in.cursor-1, 0)
	case "right", "ctrl+f":
		in.cursor = min(in.cursor+1, len(in.value))
	case "home", "ctrl+a":
		in.cursor = in.lineStart()
	case "end", "ctrl+e":
		in.cursor = in.lineEnd()
	case "alt+left", "ctrl+left":
		in.cursor = in.wordLeft()
	case "alt+right", "ctrl+right":
		in.cursor = in.wordRight()
	case "up":
		in.moveVertical(-1)
	case "down":
		in.moveVertical(1)
	case "backspace", "ctrl+h":
		if in.cursor > 0 {
			in.deleteRange(in.cursor-1, in.cursor)
		}
	case "delete", "ctrl+d":
		if in.cursor < len(in.value) {
			in.deleteRange(in.cursor, in.cursor+1)
		}
	case "alt+backspace", "ctrl+w":
		in.deleteRange(in.wordLeft(), in.cursor)
	case "ctrl+u":
		in.deleteRange(in.lineStart(), in.cursor)
	case "ctrl+k":
		in.deleteRange(in.cursor, in.lineEnd())
	default:
		if msg.Text == "" || msg.Mod.Contains(tea.ModCtrl) || msg.Mod.Contains(tea.ModAlt) ||
			msg.Mod.Contains(tea.ModSuper) || msg.Mod.Contains(tea.ModMeta) {
			return false
		}
		in.insert([]rune(CleanText(msg.Text, false)))
		return true
	}
	in.scrollToCursor()
	return true
}

func (in *Input) insert(rs []rune) {
	v := make([]rune, 0, len(in.value)+len(rs))
	v = append(v, in.value[:in.cursor]...)
	v = append(v, rs...)
	v = append(v, in.value[in.cursor:]...)
	in.value = v
	in.cursor += len(rs)
	in.scrollToCursor()
}

func (in *Input) deleteRange(a, b int) {
	if a >= b {
		return
	}
	in.value = append(in.value[:a:a], in.value[b:]...)
	in.cursor = a
	in.scrollToCursor()
}

func (in *Input) lineStart() int {
	i := in.cursor
	for i > 0 && in.value[i-1] != '\n' {
		i--
	}
	return i
}

func (in *Input) lineEnd() int {
	i := in.cursor
	for i < len(in.value) && in.value[i] != '\n' {
		i++
	}
	return i
}

func (in *Input) wordLeft() int {
	i := in.cursor
	for i > 0 && unicode.IsSpace(in.value[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(in.value[i-1]) {
		i--
	}
	return i
}

func (in *Input) wordRight() int {
	i := in.cursor
	for i < len(in.value) && unicode.IsSpace(in.value[i]) {
		i++
	}
	for i < len(in.value) && !unicode.IsSpace(in.value[i]) {
		i++
	}
	return i
}

// textWidth is the number of cells available for text: the width minus the
// prompt and one cell kept free so the cursor fits after a full row.
func (in *Input) textWidth() int { return max(in.width-PromptWidth-1, 1) }

// segment is one visual row: value[start:end].
type segment struct {
	start, end int
	lastOfLine bool // the segment ends a hard line (or the text)
}

// layout wraps the value into visual rows of at most w cells, breaking after
// spaces where possible.
func layout(value []rune, w int) []segment {
	var segs []segment
	lineStart := 0
	for i := 0; i <= len(value); i++ {
		if i < len(value) && value[i] != '\n' {
			continue
		}
		segs = wrapLine(segs, value, lineStart, i, w)
		lineStart = i + 1
	}
	return segs
}

func wrapLine(segs []segment, value []rune, a, b, w int) []segment {
	start := a
	for {
		width, lastBreak, i := 0, -1, start
		for i < b {
			rw := runeWidth(value[i])
			if width+rw > w {
				break
			}
			width += rw
			if value[i] == ' ' {
				lastBreak = i + 1
			}
			i++
		}
		if i >= b {
			return append(segs, segment{start: start, end: b, lastOfLine: true})
		}
		end := i
		if lastBreak > start {
			end = lastBreak
		} else if end == start {
			end = start + 1 // a single rune wider than the row
		}
		segs = append(segs, segment{start: start, end: end})
		start = end
	}
}

func runeWidth(r rune) int { return max(ansi.StringWidth(string(r)), 0) }

// cursorRow returns the visual row holding the cursor and its column in cells.
func (in *Input) cursorRow(segs []segment) (row, col int) {
	for i, s := range segs {
		if in.cursor < s.end || (in.cursor == s.end && s.lastOfLine) {
			if in.cursor >= s.start {
				return i, ansi.StringWidth(string(in.value[s.start:in.cursor]))
			}
		}
	}
	last := len(segs) - 1
	return last, ansi.StringWidth(string(in.value[segs[last].start:segs[last].end]))
}

func (in *Input) moveVertical(d int) {
	segs := layout(in.value, in.textWidth())
	row, col := in.cursorRow(segs)
	target := row + d
	if target < 0 || target >= len(segs) {
		return
	}
	s := segs[target]
	i, w := s.start, 0
	for i < s.end && w+runeWidth(in.value[i]) <= col {
		w += runeWidth(in.value[i])
		i++
	}
	// Don't land after a soft-wrap space, which belongs to the row's end.
	if i == s.end && !s.lastOfLine && i > s.start {
		i--
	}
	in.cursor = i
}

func (in *Input) scrollToCursor() {
	if in.height <= 0 {
		return
	}
	row, _ := in.cursorRow(layout(in.value, in.textWidth()))
	if row < in.top {
		in.top = row
	}
	if row >= in.top+in.height {
		in.top = row - in.height + 1
	}
}

// View renders the input as height rows of width cells. field shades the
// whole area with the input background, as in the multi-line edit dialogs;
// without it the input sits on bg/default, as in the date dialogs.
func (in *Input) View(t theme.Theme, field bool) string {
	bg := t.C.Bg
	if field {
		bg = t.C.BgInput
	}
	fill := t.Style(t.C.Text, bg)
	prompt := t.Style(t.C.TextAccent, bg).Render(">") + fill.Render(" ")
	cursor := t.Style(t.C.Reverse.Fg, t.C.Reverse.Bg)
	textStyle := fill
	if !in.focused {
		textStyle = t.Style(t.C.TextMuted, bg)
	}
	w := max(in.width, PromptWidth+1)

	rows := make([]string, in.height)
	if len(in.value) == 0 {
		first := prompt
		if in.focused {
			first += cursor.Render(" ")
		} else {
			first += fill.Render(" ")
		}
		ph := Truncate(t, in.Placeholder, w-Width(first))
		rows[0] = Fit(first+t.Style(t.C.TextMuted, bg).Render(ph), w, fill)
		for i := 1; i < len(rows); i++ {
			rows[i] = Blank(fill, w)
		}
		return strings.Join(rows, "\n")
	}

	segs := layout(in.value, in.textWidth())
	curRow, _ := in.cursorRow(segs)
	top := min(max(in.top, 0), max(len(segs)-1, 0))

	for i := range rows {
		r := top + i
		if r >= len(segs) {
			rows[i] = Blank(fill, w)
			continue
		}
		s := segs[r]
		lead := fill.Render("  ")
		if r == 0 {
			lead = prompt
		}
		// Group runs of runes that share a style: plain text (-1), the
		// cursor (-2), or highlight number h (h >= 0).
		key := func(k int) int {
			if in.focused && k == in.cursor {
				return -2
			}
			if in.focused {
				for h, hl := range in.Highlights {
					if k >= hl.Start && k < hl.End {
						return h
					}
				}
			}
			return -1
		}
		styleOf := func(k int) lipgloss.Style {
			switch k {
			case -1:
				return textStyle
			case -2:
				return cursor
			}
			return t.Style(in.Highlights[k].Fg, bg)
		}
		var b strings.Builder
		for k := s.start; k < s.end; {
			j, kk := k+1, key(k)
			for j < s.end && key(j) == kk {
				j++
			}
			b.WriteString(styleOf(kk).Render(string(in.value[k:j])))
			k = j
		}
		if in.focused && r == curRow && in.cursor == s.end {
			b.WriteString(cursor.Render(" "))
		}
		rows[i] = Fit(lead+b.String(), w, fill)
	}
	return strings.Join(rows, "\n")
}

// CleanText makes text safe to draw. Line breaks (\r\n, \r — terminals
// often paste newlines as \r — and \n) become \n, or a space when multiline
// is false; tabs become a space; other control characters are dropped. A
// stray \r or escape would move the terminal's cursor and break the layout.
func CleanText(s string, multiline bool) string {
	nl := "\n"
	if !multiline {
		nl = " "
	}
	s = strings.NewReplacer("\r\n", nl, "\r", nl, "\n", nl, "\t", " ").Replace(s)
	return strings.Map(func(r rune) rune {
		if r != '\n' && (r < 0x20 || (r >= 0x7f && r < 0xa0)) {
			return -1
		}
		return r
	}, s)
}
