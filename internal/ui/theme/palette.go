package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
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

// Palette is one mode of the Figma "Primitives / Palette" collection: an RGB
// value for each ANSI slot, plus which slots base/fg and base/bg point to.
//
// taskmoth draws palette colours as 24-bit RGB rather than as ANSI indices
// (decision G1.7), so the app looks the same in every terminal and the
// dimmed theme can blend real colours.
type Palette struct {
	Name   string
	Slots  [numSlots]color.Color
	BaseFg Slot // base/fg
	BaseBg Slot // base/bg
}

// Color returns the colour of a slot.
func (p Palette) Color(s Slot) color.Color { return p.Slots[s] }

// Fg returns base/fg.
func (p Palette) Fg() color.Color { return p.Slots[p.BaseFg] }

// Bg returns base/bg.
func (p Palette) Bg() color.Color { return p.Slots[p.BaseBg] }

// Dracula is the "Dracula" mode of the Figma palette and taskmoth's default.
func Dracula() Palette {
	c := lipgloss.Color
	return Palette{
		Name: "Dracula",
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
