package theme

import "image/color"

// Tone selects a badge colour pair (Figma "Badge" tone variants).
type Tone int

// Badge tones.
const (
	ToneNeutral Tone = iota
	ToneInfo
	ToneSuccess
	ToneWarning
	ToneError
	ToneMode
	ToneLink
	numTones
)

// Pair is a foreground/background colour pair.
type Pair struct {
	Fg, Bg color.Color
}

// Colors holds the resolved semantic tokens (Figma collection
// "Semantic / Color"). The comment on each field names its Figma variable.
// Widgets and views only use these, never palette slots or raw colours.
type Colors struct {
	Bg      color.Color // bg/default
	BgInput color.Color // bg/input (shaded input fields)

	Text        color.Color // text/default
	TextTitle   color.Color // text/title
	TextMuted   color.Color // text/muted
	TextAccent  color.Color // text/accent
	TextLink    color.Color // text/link
	TextSuccess color.Color // text/success
	TextWarning color.Color // text/warning
	TextError   color.Color // text/error
	TextInfo    color.Color // text/info
	// text/disabled has no value in Figma yet, so it is not defined here.

	SelectionBg color.Color // selection/bg
	SelectionFg color.Color // selection/fg

	BorderDefault color.Color // border/default
	BorderFocused color.Color // border/focused

	ScrollTrack color.Color // scrollbar/track
	ScrollThumb color.Color // scrollbar/thumb

	Key      color.Color // key/key
	KeyLabel color.Color // key/label

	Status      Pair // statusbar/fg, statusbar/bg
	Reverse     Pair // reverse/fg, reverse/bg (cursor, buttons)
	SearchMatch Pair // search-match/fg, search-match/bg

	Badges [numTones]Pair // badge/<tone>/fg, badge/<tone>/bg
}

// Semantic resolves the semantic tokens against a palette, following the
// aliases in the Figma "Semantic / Color" collection exactly.
func Semantic(p Palette) Colors {
	s := p.Color
	fg, bg := p.Fg(), p.Bg()
	return Colors{
		Bg:      bg,
		BgInput: s(BrightBlack),

		Text:        fg,
		TextTitle:   fg,
		TextMuted:   s(BrightBlack),
		TextAccent:  s(Blue),
		TextLink:    s(Cyan),
		TextSuccess: s(Green),
		TextWarning: s(Yellow),
		TextError:   s(Red),
		TextInfo:    s(Magenta),

		SelectionBg: s(Blue),
		SelectionFg: s(BrightWhite),

		BorderDefault: s(BrightBlack),
		BorderFocused: s(Cyan),

		ScrollTrack: s(BrightBlack),
		ScrollThumb: fg,

		Key:      s(Cyan),
		KeyLabel: s(BrightBlack),

		Status:      Pair{Fg: s(White), Bg: s(BrightBlack)},
		Reverse:     Pair{Fg: bg, Bg: fg},
		SearchMatch: Pair{Fg: s(Black), Bg: s(Yellow)},

		Badges: [numTones]Pair{
			ToneNeutral: {Fg: bg, Bg: s(BrightBlack)},
			ToneInfo:    {Fg: s(BrightWhite), Bg: s(Magenta)},
			ToneSuccess: {Fg: bg, Bg: s(Green)},
			ToneWarning: {Fg: bg, Bg: s(Yellow)},
			ToneError:   {Fg: s(BrightWhite), Bg: s(Red)},
			ToneMode:    {Fg: s(White), Bg: s(Black)},
			ToneLink:    {Fg: s(BrightWhite), Bg: s(Cyan)},
		},
	}
}

// Default returns the semantic colours resolved against the Dracula palette.
func Default() Colors { return Semantic(Dracula()) }
