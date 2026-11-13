package theme

import (
	"fmt"
	"image/color"
	"reflect"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("%02x%02x%02x", r>>8, g>>8, b>>8)
}

func TestBlend(t *testing.T) {
	white := lipgloss.Color("#ffffff")
	bg := lipgloss.Color("#22212c")
	tests := []struct {
		f    float64
		want string
	}{
		{0, "ffffff"},
		{1, "22212c"},
		{0.5, "919096"}, // (255+34)/2=144.5→145, (255+33)/2=144, (255+44)/2=149.5→150
	}
	for _, tt := range tests {
		if got := hex(Blend(white, bg, tt.f)); got != tt.want {
			t.Errorf("Blend(white, bg, %v) = %s, want %s", tt.f, got, tt.want)
		}
	}
}

// colourFields flattens every colour in Colors (including pairs and badges)
// into name → colour, so tests cover tokens added later automatically.
func colourFields(c Colors) map[string]color.Color {
	out := map[string]color.Color{}
	colorType := reflect.TypeFor[color.Color]()
	var walk func(prefix string, v reflect.Value)
	walk = func(prefix string, v reflect.Value) {
		switch {
		case v.Type() == colorType:
			out[prefix] = v.Interface().(color.Color)
		case v.Kind() == reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(prefix+"."+v.Type().Field(i).Name, v.Field(i))
			}
		case v.Kind() == reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(fmt.Sprintf("%s[%d]", prefix, i), v.Index(i))
			}
		}
	}
	walk("C", reflect.ValueOf(c))
	return out
}

// TestSemanticMatchesFigma checks every token against the value Figma
// resolves it to in the Dracula palette (file uRWYa0RRojip50i2lWNdKx,
// collection "Semantic / Color"). It also fails if a token is left nil.
func TestSemanticMatchesFigma(t *testing.T) {
	c := Default()
	want := map[string]string{
		"C.Bg": "22212c", "C.BgInput": "504c67",
		"C.Text": "ffffff", "C.TextTitle": "ffffff", "C.TextMuted": "504c67", "C.TextAccent": "9282f7",
		"C.TextLink": "80ffea", "C.TextSuccess": "8aff80", "C.TextWarning": "ffff80",
		"C.TextError": "ff9580", "C.TextInfo": "ff80bf",
		"C.SelectionBg": "9282f7", "C.SelectionFg": "ffffff",
		"C.BorderDefault": "504c67", "C.BorderFocused": "80ffea",
		"C.ScrollTrack": "504c67", "C.ScrollThumb": "ffffff",
		"C.Key": "80ffea", "C.KeyLabel": "504c67",
		"C.Status.Fg": "f8f8f2", "C.Status.Bg": "504c67",
		"C.Reverse.Fg": "22212c", "C.Reverse.Bg": "ffffff",
		"C.SearchMatch.Fg": "22212c", "C.SearchMatch.Bg": "ffff80",
		"C.Badges[0].Fg": "22212c", "C.Badges[0].Bg": "504c67", // neutral
		"C.Badges[1].Fg": "ffffff", "C.Badges[1].Bg": "ff80bf", // info
		"C.Badges[2].Fg": "22212c", "C.Badges[2].Bg": "8aff80", // success
		"C.Badges[3].Fg": "22212c", "C.Badges[3].Bg": "ffff80", // warning
		"C.Badges[4].Fg": "ffffff", "C.Badges[4].Bg": "ff9580", // error
		"C.Badges[5].Fg": "f8f8f2", "C.Badges[5].Bg": "22212c", // mode
		"C.Badges[6].Fg": "ffffff", "C.Badges[6].Bg": "80ffea", // link
	}
	got := colourFields(c)
	for name, col := range got {
		if col == nil {
			t.Errorf("%s is nil", name)
			continue
		}
		w, ok := want[name]
		if !ok {
			t.Errorf("%s = %s has no expected value in this test; add it from Figma", name, hex(col))
			continue
		}
		if hex(col) != w {
			t.Errorf("%s = %s, want %s", name, hex(col), w)
		}
	}
	for name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("expected token %s is missing from Colors", name)
		}
	}
}

func TestDimmedBlendsEveryColourButBg(t *testing.T) {
	n := New(false)
	d := n.Dimmed()
	if !d.IsDimmed() || n.IsDimmed() {
		t.Fatal("IsDimmed wrong")
	}
	if hex(d.C.Bg) != hex(n.C.Bg) {
		t.Errorf("bg changed: %s -> %s", hex(n.C.Bg), hex(d.C.Bg))
	}
	normal, dimmed := colourFields(n.C), colourFields(d.C)
	for name, c := range normal {
		want := hex(Blend(c, n.C.Bg, DimOpacity))
		if got := hex(dimmed[name]); got != want {
			t.Errorf("%s: dimmed %s, want %s", name, got, want)
		}
	}
	if hex(d.Dimmed().C.Text) != hex(d.C.Text) {
		t.Error("dimming twice should be a no-op")
	}
}

func TestPaletteMatchesANSIOrder(t *testing.T) {
	p := Dracula()
	if p.Fg() != p.Color(BrightWhite) || p.Bg() != p.Color(Black) {
		t.Error("base/fg should be bright/15-white and base/bg normal/00-black")
	}
	if BrightWhite != 15 || Black != 0 || BrightBlack != 8 {
		t.Error("slots must use ANSI indices")
	}
	for i, c := range p.Slots {
		if c == nil {
			t.Errorf("slot %d unset", i)
		}
	}
}

func TestGlyphSetsAreOneCellWide(t *testing.T) {
	for name, g := range map[string]Glyphs{"unicode": UnicodeGlyphs(), "ascii": ASCIIGlyphs()} {
		for field, s := range map[string]string{
			"Marker": g.Marker, "Breadcrumb": g.Breadcrumb, "Up": g.Up, "Down": g.Down,
			"Left": g.Left, "Right": g.Right, "Dot": g.Dot, "H": g.H, "V": g.V,
			"TL": g.TL, "TR": g.TR, "BL": g.BL, "BR": g.BR, "Thumb": g.Thumb, "Track": g.Track,
		} {
			if w := ansi.StringWidth(s); w != 1 {
				t.Errorf("%s %s = %q is %d cells wide, want 1", name, field, s, w)
			}
		}
	}
}

func TestASCIIGlyphsArePureASCII(t *testing.T) {
	g := ASCIIGlyphs()
	for _, s := range []string{g.Marker, g.Breadcrumb, g.Up, g.Down, g.Left, g.Right, g.Dot,
		g.H, g.V, g.TL, g.TR, g.BL, g.BR, g.Thumb, g.Track, g.Ellipsis, g.Times} {
		for _, r := range s {
			if r > 0x7e {
				t.Errorf("ASCII glyph %q contains non-ASCII rune %U", s, r)
			}
		}
	}
}
