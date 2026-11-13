package widget

import (
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// ButtonTone is a Figma Button tone (node 10:614).
type ButtonTone int

// Button tones.
const (
	ButtonDefault     ButtonTone = iota // [ cancel ]
	ButtonDestructive                   // [ delete ], [ archive ]
	ButtonConfirm                       // [ save ]
)

// Button renders "[ label ]" (Figma Button, node 10:614). A focused button
// is filled: default is reverse (dark on white), destructive is
// badge/error, confirm is badge/success. An unfocused one is coloured text on
// the background: text/default, text/error or text/success. Disabled buttons
// are drawn at the 50% dim opacity.
func Button(t theme.Theme, label string, tone ButtonTone, focused, disabled bool) string {
	var fg, bg = t.C.Text, t.C.Bg
	switch {
	case focused && tone == ButtonDefault:
		fg, bg = t.C.Reverse.Fg, t.C.Reverse.Bg
	case focused && tone == ButtonDestructive:
		fg, bg = t.C.Badges[theme.ToneError].Fg, t.C.Badges[theme.ToneError].Bg
	case focused && tone == ButtonConfirm:
		fg, bg = t.C.Badges[theme.ToneSuccess].Fg, t.C.Badges[theme.ToneSuccess].Bg
	case tone == ButtonDestructive:
		fg = t.C.TextError
	case tone == ButtonConfirm:
		fg = t.C.TextSuccess
	}
	if disabled {
		fg, bg = t.Fade(fg), t.Fade(bg)
	}
	return t.Style(fg, bg).Render("[ " + label + " ]")
}
