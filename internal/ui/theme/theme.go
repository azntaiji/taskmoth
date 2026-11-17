// Package theme holds taskmoth's design tokens and glyph sets.
//
// Colours mirror the two layers of Figma file uRWYa0RRojip50i2lWNdKx:
// a Palette of the 16 ANSI slots (palette.go, "Primitives / Palette") and
// semantic tokens that point at slots (colors.go, "Semantic / Color").
// The default palette sends ANSI indices, so the terminal's colour scheme
// decides the colours (decision G9.2); Dracula sends 24-bit RGB (G1.7).
//
// The dimmed theme draws the view behind a dialog. With an RGB palette every
// colour is pre-blended 50% toward bg/default, matching Figma's
// attr/dim-opacity (decision G1.4); with the ANSI palette, whose RGB values
// are unknown, text is drawn with the faint attribute instead (G9.3).
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// DimOpacity is Figma's attr/dim-opacity (50%).
const DimOpacity = 0.5

// Glyphs holds every non-ASCII symbol the UI draws, so the ASCII fallback
// (decisions G1.3a/G1.3b) can swap them.
type Glyphs struct {
	Marker     string // selection marker
	Breadcrumb string // header separator
	Up, Down   string // key hints
	Left       string
	Right      string
	Dot        string // status/header separator
	H, V       string // box-drawing lines
	TL, TR     string // corners
	BL, BR     string
	Thumb      string // scrollbar thumb
	Track      string // scrollbar track
	Ellipsis   string // truncation tail
	Times      string // dimensions, e.g. 80×24
}

// UnicodeGlyphs are the symbols used in the Figma designs.
func UnicodeGlyphs() Glyphs {
	return Glyphs{
		Marker: "▶", Breadcrumb: "▸",
		Up: "↑", Down: "↓", Left: "←", Right: "→",
		Dot: "·",
		H:   "─", V: "│",
		TL: "┌", TR: "┐", BL: "└", BR: "┘",
		Thumb: "█", Track: "│",
		Ellipsis: "…", Times: "×",
	}
}

// ASCIIGlyphs are the one-cell-per-glyph replacements from decision G1.3b.
// The truncation tail is "...", which still fits the truncated width.
func ASCIIGlyphs() Glyphs {
	return Glyphs{
		Marker: ">", Breadcrumb: ">",
		Up: "^", Down: "v", Left: "<", Right: ">",
		Dot: "-",
		H:   "-", V: "|",
		TL: "+", TR: "+", BL: "+", BR: "+",
		Thumb: "#", Track: "|",
		Ellipsis: "...", Times: "x",
	}
}

// Theme is everything a widget needs to draw itself.
type Theme struct {
	C       Colors // semantic tokens resolved against palette
	G       Glyphs
	palette Palette
	dimmed  bool
}

// New returns the normal theme on palette p, with ASCII glyphs if ascii is
// set.
func New(p Palette, ascii bool) Theme {
	g := UnicodeGlyphs()
	if ascii {
		g = ASCIIGlyphs()
	}
	return Theme{C: Semantic(p), G: g, palette: p}
}

// Palette returns the palette the theme was resolved against.
func (t Theme) Palette() Palette { return t.palette }

// IsDimmed reports whether this is the dimmed theme.
func (t Theme) IsDimmed() bool { return t.dimmed }

// ANSI reports whether the theme uses the terminal's ANSI colours.
func (t Theme) ANSI() bool { return t.palette.ANSI }

// Dimmed returns a copy of t for drawing the view behind a dialog. With an
// RGB palette every colour is blended 50% toward bg/default: it dims the
// palette slots and re-resolves the semantic tokens, so every token is
// covered, and base/bg blends with itself and is unchanged. With the ANSI
// palette the colours stay and Style adds the faint attribute (G9.3).
// Dimming an already dimmed theme is a no-op.
func (t Theme) Dimmed() Theme {
	if t.dimmed {
		return t
	}
	if t.palette.ANSI {
		t.dimmed = true
		return t
	}
	p := t.palette
	bg := p.Bg()
	for i := range p.Slots {
		p.Slots[i] = Blend(p.Slots[i], bg, DimOpacity)
	}
	return Theme{C: Semantic(p), G: t.G, palette: p, dimmed: true}
}

// Fade applies Figma's 50% dim opacity to a text colour drawn on
// bg/default, as the component "dim" variants do (completed tasks, the
// header clock).
func (t Theme) Fade(c color.Color) color.Color {
	return t.FadeOn(c, t.C.Bg)
}

// FadeOn applies the 50% dim opacity to a text colour drawn on bg, e.g. task
// metadata on a selected row. With the ANSI palette the colour is marked
// faded, and Style draws it with the faint attribute (G9.3).
func (t Theme) FadeOn(c, bg color.Color) color.Color {
	if t.palette.ANSI {
		if _, ok := c.(faded); ok {
			return c
		}
		if c == termBg {
			// Drawn with reverse, where faint isn't reliable; the selection
			// bar's second line stays readable (G9.6).
			return c
		}
		return faded{c}
	}
	return Blend(c, bg, DimOpacity)
}

// FadePair applies the dim opacity to a filled element, such as a badge or
// a button. The faint attribute can't dim a background, so with the ANSI
// palette a filled pair becomes the neutral badge, and a pair on the page
// background (or on slot 0) just has its text faded (G9.3).
func (t Theme) FadePair(p Pair) Pair {
	if !t.palette.ANSI {
		return Pair{Fg: t.Fade(p.Fg), Bg: t.Fade(p.Bg)}
	}
	if p.Bg == termBg || p.Bg == t.palette.Color(Black) {
		// On the page background, or on slot 0 (the P4 badge), which
		// stands in for it: fading the text is enough.
		return Pair{Fg: t.Fade(p.Fg), Bg: p.Bg}
	}
	return t.C.Badges[ToneNeutral]
}

// Style returns a style with the given foreground and background. It also
// handles the ANSI palette's special colours: the terminal's default
// foreground and background are left unset (SGR 39/49), base/bg used as a
// text colour is drawn with the reverse attribute (G9.4), and faded colours
// and the dimmed theme add the faint attribute (G9.3).
func (t Theme) Style(fg, bg color.Color) lipgloss.Style {
	st := lipgloss.NewStyle()
	faint := t.dimmed && t.palette.ANSI
	if f, ok := fg.(faded); ok {
		fg, faint = f.c, true
	}
	if f, ok := bg.(faded); ok {
		bg = f.c
	}
	switch {
	case fg == termBg && bg != termBg:
		// Reverse swaps the colours: bg is set as the foreground so it
		// becomes the background, and the terminal's default background
		// becomes the text colour.
		if bg != termFg {
			st = st.Foreground(bg)
		}
		st = st.Reverse(true)
	default:
		if fg != termFg && fg != termBg {
			st = st.Foreground(fg)
		}
		if bg != termFg && bg != termBg {
			st = st.Background(bg)
		}
	}
	if faint {
		st = st.Faint(true)
	}
	return st
}

// Base returns default text on the default background.
func (t Theme) Base() lipgloss.Style {
	return t.Style(t.C.Text, t.C.Bg)
}

// termColor is the terminal's default foreground or background, the ANSI
// palette's base/fg and base/bg (G9.4). Style leaves it unset.
type termColor struct{ bg bool }

// RGBA is required by color.Color; the real value is unknown.
func (termColor) RGBA() (r, g, b, a uint32) { return 0, 0, 0, 0xffff }

var (
	termFg color.Color = termColor{bg: false}
	termBg color.Color = termColor{bg: true}
)

// faded is a text colour at the dim opacity in an ANSI theme; Style draws
// it with the faint attribute (G9.3).
type faded struct{ c color.Color }

// RGBA returns the underlying colour's value.
func (f faded) RGBA() (r, g, b, a uint32) { return f.c.RGBA() }

// Blend mixes a toward b: f=0 returns a, f=1 returns b. Alpha is ignored; the
// result is opaque.
func Blend(a, b color.Color, f float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	mix := func(x, y uint32) uint8 {
		v := float64(x>>8)*(1-f) + float64(y>>8)*f
		return uint8(v + 0.5)
	}
	return color.RGBA{R: mix(ar, br), G: mix(ag, bg), B: mix(ab, bb), A: 0xff}
}
