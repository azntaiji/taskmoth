// Package theme holds taskmoth's design tokens and glyph sets.
//
// Colours mirror the two layers of Figma file uRWYa0RRojip50i2lWNdKx:
// a Palette of the 16 ANSI slots (palette.go, "Primitives / Palette") and
// semantic tokens that point at slots (colors.go, "Semantic / Color").
// Palette colours are drawn as 24-bit RGB (decision G1.7 in PLAN.md).
//
// The dimmed theme draws the view behind a dialog: every colour is pre-blended
// 50% toward bg/default, matching Figma's attr/dim-opacity (decision G1.4).
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

// New returns the normal theme on the default (Dracula) palette, with ASCII
// glyphs if ascii is set.
func New(ascii bool) Theme {
	return NewWithPalette(Dracula(), ascii)
}

// NewWithPalette returns the normal theme on palette p.
func NewWithPalette(p Palette, ascii bool) Theme {
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

// Dimmed returns a copy of t with every colour blended 50% toward bg/default,
// for drawing the view behind a dialog. It dims the palette slots and
// re-resolves the semantic tokens, so every token is covered. base/bg blends
// with itself and is unchanged. Dimming an already dimmed theme is a no-op.
func (t Theme) Dimmed() Theme {
	if t.dimmed {
		return t
	}
	p := t.palette
	bg := p.Bg()
	for i := range p.Slots {
		p.Slots[i] = Blend(p.Slots[i], bg, DimOpacity)
	}
	return Theme{C: Semantic(p), G: t.G, palette: p, dimmed: true}
}

// Fade applies Figma's 50% dim opacity to a colour drawn on bg/default, as
// the component "dim" variants do (completed tasks, the header clock).
func (t Theme) Fade(c color.Color) color.Color {
	return t.FadeOn(c, t.C.Bg)
}

// FadeOn applies the 50% dim opacity to a colour drawn on bg, e.g. task
// metadata on a selected row.
func (t Theme) FadeOn(c, bg color.Color) color.Color {
	return Blend(c, bg, DimOpacity)
}

// Style returns a style with the given foreground and background.
func (t Theme) Style(fg, bg color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(fg).Background(bg)
}

// Base returns default text on the default background.
func (t Theme) Base() lipgloss.Style {
	return t.Style(t.C.Text, t.C.Bg)
}

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
