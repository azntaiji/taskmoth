package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Slot is one of the 16 ANSI palette entries (Figma collection
// "Primitives / Palette"). Semantic tokens refer to slots, never to colours
// directly.
type Slot int

// Palette slots, in ANSI index order (Figma variable in the comment).
const (
	Black         Slot = iota // normal/00-black
	Red                       // normal/01-red
	Green                     // normal/02-green
	Yellow                    // normal/03-yellow
	Blue                      // normal/04-blue
	Magenta                   // normal/05-magenta
	Cyan                      // normal/06-cyan
	White                     // normal/07-white
	BrightBlack               // bright/08-black
	BrightRed                 // bright/09-red
	BrightGreen               // bright/10-green
	BrightYellow              // bright/11-yellow
	BrightBlue                // bright/12-blue
	BrightMagenta             // bright/13-magenta
	BrightCyan                // bright/14-cyan
	BrightWhite               // bright/15-white
	numSlots
)

// Palette is one mode of the Figma "Primitives / Palette" collection: a
// colour for each ANSI slot, plus which slots base/fg and base/bg point to.
//
// There are two kinds (decision G9.2). An RGB palette such as Dracula sends
// 24-bit colours (G1.7), so it looks the same in every terminal and the
// dimmed theme can blend real colours. The ANSI palette ("default") sends
// each slot as its ANSI index, so the terminal's own colour scheme decides
// the colours; its base/fg and base/bg are the terminal's defaults (G9.4).
type Palette struct {
	Name   string
	Slots  [numSlots]color.Color
	BaseFg Slot // base/fg
	BaseBg Slot // base/bg
	// ANSI marks a palette of ANSI indices, whose RGB values are unknown:
	// base/fg and base/bg are the terminal defaults, and dimming uses the
	// faint attribute instead of blending (G9.3).
	ANSI bool
}

// Color returns the colour of a slot.
func (p Palette) Color(s Slot) color.Color { return p.Slots[s] }

// Fg returns base/fg: the terminal's default foreground in an ANSI palette.
func (p Palette) Fg() color.Color {
	if p.ANSI {
		return termFg
	}
	return p.Slots[p.BaseFg]
}

// Bg returns base/bg: the terminal's default background in an ANSI palette.
func (p Palette) Bg() color.Color {
	if p.ANSI {
		return termBg
	}
	return p.Slots[p.BaseBg]
}

// Theme names for the config file and TASKMOTH_THEME (decision G9.5).
const (
	DefaultName = "default"
	DraculaName = "dracula"
)

// Names lists the palettes taskmoth knows, default first.
func Names() []string { return []string{DefaultName, DraculaName} }

// ByName returns the named palette.
func ByName(name string) (Palette, bool) {
	switch name {
	case DefaultName:
		return ANSI(), true
	case DraculaName:
		return Dracula(), true
	}
	return Palette{}, false
}

// ANSI is the default palette: each slot is the terminal's own colour for
// that index (the "Index n · SGR …" in each Figma palette variable), sent as
// SGR 30–37/90–97 and 40–47/100–107 (decision G9.2).
func ANSI() Palette {
	p := Palette{Name: DefaultName, BaseFg: BrightWhite, BaseBg: Black, ANSI: true}
	for i := range p.Slots {
		p.Slots[i] = ansi.BasicColor(i) //nolint:gosec // i < 16
	}
	return p
}

// Dracula is the "Dracula" mode of the Figma palette, drawn as 24-bit RGB.
func Dracula() Palette {
	c := lipgloss.Color
	return Palette{
		Name: DraculaName,
		Slots: [numSlots]color.Color{
			Black:         c("#22212c"),
			Red:           c("#ff9580"),
			Green:         c("#8aff80"),
			Yellow:        c("#ffff80"),
			Blue:          c("#9282f7"),
			Magenta:       c("#ff80bf"),
			Cyan:          c("#80ffea"),
			White:         c("#f8f8f2"),
			BrightBlack:   c("#504c67"),
			BrightRed:     c("#ffaa99"),
			BrightGreen:   c("#a2ff99"),
			BrightYellow:  c("#ffff99"),
			BrightBlue:    c("#aa99ff"),
			BrightMagenta: c("#ff99cc"),
			BrightCyan:    c("#99ffee"),
			BrightWhite:   c("#ffffff"),
		},
		BaseFg: BrightWhite,
		BaseBg: Black,
	}
}
