package widget

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func key(s string) tea.KeyPressMsg {
	switch s {
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "ctrl+w":
		return tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func typeText(in *Input, s string) {
	for _, r := range s {
		if r == ' ' {
			in.Update(key("space"))
			continue
		}
		in.Update(key(string(r)))
	}
}

func newInput(w, h int) *Input {
	in := &Input{Placeholder: "enter your text here..."}
	in.SetSize(w, h)
	in.Focus()
	return in
}

func TestInputEditing(t *testing.T) {
	in := newInput(40, 1)
	typeText(in, "Get eggs")
	if in.Value() != "Get eggs" {
		t.Fatalf("value = %q", in.Value())
	}
	in.Update(key("home"))
	typeText(in, "> ")
	in.Update(key("end"))
	in.Update(key("backspace"))
	if in.Value() != "> Get egg" {
		t.Errorf("value = %q", in.Value())
	}
	in.Update(key("ctrl+w"))
	if in.Value() != "> Get " {
		t.Errorf("after ctrl+w: %q", in.Value())
	}
	in.Update(key("left"))
	in.Update(key("delete"))
	if in.Value() != "> Get" {
		t.Errorf("after delete: %q", in.Value())
	}
	if in.Update(key("enter")) {
		t.Error("enter should be left to the caller")
	}
	if in.Update(tea.KeyPressMsg{Code: 'a', Text: "a", Mod: tea.ModCtrl}) {
		t.Error("ctrl+a with text should not insert")
	}

	in.Blur()
	if in.Update(key("x")) {
		t.Error("blurred input should ignore keys")
	}
}

func TestInputPaste(t *testing.T) {
	in := newInput(40, 1)
	in.Update(tea.PasteMsg{Content: "line one\nline two"})
	if in.Value() != "line one line two" {
		t.Errorf("single-line paste = %q", in.Value())
	}
	ml := newInput(40, 3)
	ml.Multiline = true
	ml.Update(tea.PasteMsg{Content: "line one\r\nline two"})
	if ml.Value() != "line one\nline two" {
		t.Errorf("multi-line paste = %q", ml.Value())
	}
	in.InsertNewline()
	if strings.Contains(in.Value(), "\n") {
		t.Error("single-line input accepted a newline")
	}
}

func TestLayoutWraps(t *testing.T) {
	text := []rune("I need to goto trader joe's to get eggs. Also get the rest of the stuff on the list.\n\nReach out")
	segs := layout(text, 51)
	var got []string
	for _, s := range segs {
		got = append(got, string(text[s.start:s.end]))
	}
	want := []string{
		"I need to goto trader joe's to get eggs. Also get ",
		"the rest of the stuff on the list.",
		"",
		"Reach out",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("layout:\n%q\nwant\n%q", got, want)
	}

	// A word longer than the row is hard-broken.
	long := []rune("abcdefghij")
	if n := len(layout(long, 4)); n != 3 {
		t.Errorf("hard break rows = %d, want 3", n)
	}
	// Wide characters never overflow the row.
	wide := []rune("保険保険保険")
	for _, s := range layout(wide, 5) {
		if w := ansi.StringWidth(string(wide[s.start:s.end])); w > 5 {
			t.Errorf("row %q is %d cells", string(wide[s.start:s.end]), w)
		}
	}
}

func TestInputVerticalMovementAndScroll(t *testing.T) {
	in := newInput(20, 2) // 17 text cells per row
	in.Multiline = true
	in.SetValue("aaaa bbbb cccc dddd eeee ffff")
	// rows: "aaaa bbbb cccc " | "dddd eeee ffff"
	row, col := in.cursorRow(layout(in.value, in.textWidth()))
	if row != 1 || col != 14 {
		t.Fatalf("cursor at row %d col %d", row, col)
	}
	in.Update(key("up"))
	if in.cursor != 14 {
		t.Errorf("after up cursor = %d, want 14", in.cursor)
	}
	in.Update(key("down"))
	if in.cursor != len(in.value) {
		t.Errorf("after down cursor = %d", in.cursor)
	}

	in.SetValue("one\ntwo\nthree\nfour")
	if in.top != 2 {
		t.Errorf("top = %d, want 2 so the cursor row is visible", in.top)
	}
	v := ansi.Strip(in.View(uni, true))
	if v != "  three             \n  four              " {
		t.Errorf("scrolled view:\n%q", v)
	}
}

func TestInputView(t *testing.T) {
	in := newInput(54, 3)
	view := in.View(uni, true)
	assertBlock(t, view, 54, 3)
	if got := ansi.Strip(strings.Split(view, "\n")[0]); !strings.HasPrefix(got, ">  enter your text here...") {
		t.Errorf("placeholder row = %q", got)
	}

	in.SetValue("File insurance reimbursements")
	view = in.View(uni, true)
	assertBlock(t, view, 54, 3)
	golden(t, "input_field", view)

	in.Multiline = true
	in.SetSize(54, 5)
	in.SetValue("I need to goto trader joe's to get eggs. Also get the rest of the stuff on the list.\n\nReach out to Hope if she needs anything that isn't on the list.")
	view = in.View(uni, true)
	assertBlock(t, view, 54, 5)
	lines := strings.Split(ansi.Strip(view), "\n")
	if !strings.HasPrefix(lines[0], "> I need") || !strings.HasPrefix(lines[1], "  the rest") {
		t.Errorf("multi-line view:\n%s", ansi.Strip(view))
	}
	golden(t, "input_multiline", view)

	single := newInput(54, 1)
	single.SetValue("2026-10-02")
	view = single.View(uni, false)
	assertBlock(t, view, 54, 1)
	golden(t, "input_single", view)
}
