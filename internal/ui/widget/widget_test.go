package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

var (
	uni   = theme.New(false)
	ascii = theme.New(true)
)

func TestTruncateIsWidthAware(t *testing.T) {
	tests := []struct {
		name string
		th   theme.Theme
		in   string
		w    int
		want string
	}{
		{"fits", uni, "Get eggs", 8, "Get eggs"},
		{"ascii fits", uni, "Get eggs", 10, "Get eggs"},
		{"cut", uni, "Finish creating task", 10, "Finish cr…"},
		{"cut ascii tail", ascii, "Finish creating task", 10, "Finish ..."},
		{"wide chars cut on cell boundary", uni, "保険の書類を出す", 7, "保険の…"},
		{"emoji", uni, "Buy 🎂 cake today", 8, "Buy 🎂 …"},
		{"zero width", uni, "abc", 0, ""},
		{"tail wider than room", ascii, "abcdef", 2, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.th, tt.in, tt.w)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.in, tt.w, got, tt.want)
			}
			if w := ansi.StringWidth(got); w > tt.w {
				t.Errorf("result is %d cells, limit %d", w, tt.w)
			}
		})
	}
}

func TestHeader(t *testing.T) {
	h := Header(uni, 80, "Today", "tues · 2026-11-10 · 15:26")
	assertBlock(t, h, 80, 1)
	plain := ansi.Strip(h)
	if !strings.HasPrefix(plain, "taskmoth ▸ Today") {
		t.Errorf("header left = %q", plain)
	}
	if !strings.HasSuffix(plain, "tues · 2026-11-10 · 15:26") {
		t.Errorf("clock not right-aligned: %q", plain)
	}
	golden(t, "header", h)

	// Too narrow for the clock: it is dropped and the section truncated.
	h = Header(uni, 20, "A very long project name", "tues · 2026-11-10 · 15:26")
	assertBlock(t, h, 20, 1)
	if got := ansi.Strip(h); got != "taskmoth ▸ A very l…" {
		t.Errorf("narrow header = %q", got)
	}

	if got := ansi.Strip(Header(ascii, 30, "Inbox", "")); !strings.HasPrefix(got, "taskmoth > Inbox") {
		t.Errorf("ascii header = %q", got)
	}
}

func TestStatusBar(t *testing.T) {
	s := StatusBar(uni, 80, "Today", theme.ToneInfo, "4/6 tasks · ~8h total · ~3.75h remaining", "s:priority")
	assertBlock(t, s, 80, 1)
	plain := ansi.Strip(s)
	// Figma: badge at 12px, message at 120px → padding, " Today ", two-cell gap.
	if !strings.HasPrefix(plain, "  Today   4/6 tasks") {
		t.Errorf("status left = %q", plain)
	}
	if !strings.HasSuffix(plain, "s:priority ") {
		t.Errorf("status right = %q", plain)
	}
	golden(t, "statusbar", s)

	// Narrow: the right text is dropped before the message is cut.
	s = StatusBar(uni, 30, "Inbox", theme.ToneInfo, "6 tasks to organize", "s:priority")
	assertBlock(t, s, 30, 1)
}

func todayHints() [][]Hint {
	return [][]Hint{
		{{"q", "add"}, {"space", "done"}, {"e", "edit/view"}, {"x", "delete"}, {"s", "sort"}, {"?", "shortcuts"}, {"/", "search"}},
		{{"g i", "inbox"}, {"g t", "today"}, {"g u", "upcoming"}, {"g p", "projects"}, {"g o", "overdue"}, {"g a", "all"}},
	}
}

func TestKeyHints(t *testing.T) {
	rows := KeyHints(uni, 80, todayHints()...)
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	for _, r := range rows {
		assertBlock(t, r, 80, 1)
	}
	if got := ansi.Strip(rows[0]); !strings.HasPrefix(got, " q add  space done  e edit/view") {
		t.Errorf("row 1 = %q", got)
	}
	golden(t, "keyhints", strings.Join(rows, "\n"))

	// Overflowing rows are cut at the edge, as in the Goto Project design.
	long := KeyHints(uni, 30, []Hint{{"↑ / ↓", "select"}, {"enter", "select"}, {"esc", "cancel"}})
	assertBlock(t, long[0], 30, 1)
	if got := ansi.Strip(long[0]); got != " ↑ / ↓ select  enter select   " {
		t.Errorf("cut row = %q", got)
	}

	// Figma: the last hint in row 1 ends at 852px = 71 cells.
	if HintsWidth(todayHints()[0]) != 71 {
		t.Errorf("HintsWidth = %d", HintsWidth(todayHints()[0]))
	}
}

func TestBadgesAndMarks(t *testing.T) {
	for p := 0; p <= 4; p++ {
		for _, dim := range []bool{false, true} {
			if w := Width(PriorityMark(uni, p, dim, uni.C.Bg)); w != PriorityWidth {
				t.Errorf("PriorityMark(%d, dim=%v) is %d cells", p, dim, w)
			}
		}
	}
	if got := ansi.Strip(PriorityMark(uni, 2, false, uni.C.Bg)); got != " P2 " {
		t.Errorf("P2 = %q", got)
	}
	if DueLabel(uni, DueNone, "x", false) != "" {
		t.Error("DueNone should render nothing")
	}
	if got := ansi.Strip(DueLabel(uni, DueUpcoming, "2026-11-20", false)); got != " 2026-11-20 " {
		t.Errorf("upcoming = %q", got)
	}
	if Width(Checkbox(uni, true, uni.C.Bg)) != CheckboxWidth || Width(Checkbox(uni, false, uni.C.Bg)) != CheckboxWidth {
		t.Error("checkbox width")
	}
	if Width(Marker(uni, true, uni.C.Bg)) != MarkerWidth || Width(Marker(uni, false, uni.C.Bg)) != MarkerWidth {
		t.Error("marker width")
	}
	if got := ansi.Strip(Marker(ascii, true, ascii.C.Bg)); got != "> " {
		t.Errorf("ascii marker = %q", got)
	}

	var b strings.Builder
	for tone := theme.ToneNeutral; tone <= theme.ToneMode; tone++ {
		b.WriteString(Badge(uni, tone, "text", false) + " " + Badge(uni, tone, "text", true) + "\n")
	}
	for p := 1; p <= 4; p++ {
		b.WriteString(PriorityMark(uni, p, false, uni.C.Bg) + " " + PriorityMark(uni, p, true, uni.C.Bg) + "\n")
	}
	b.WriteString(DueLabel(uni, DueUpcoming, "2026-11-20", false) + " " +
		DueLabel(uni, DueToday, "today", false) + " " + DueLabel(uni, DueOverdue, "overdue", false) + "\n")
	b.WriteString(Marker(uni, true, uni.C.Bg) + Checkbox(uni, false, uni.C.Bg) + Checkbox(uni, true, uni.C.Bg))
	golden(t, "badges", b.String())
}

func TestScrollbarGeometry(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		s        Scroll
		size, at int
	}{
		{"fits: no thumb (G1.5)", 18, Scroll{Total: 6, Visible: 20}, 0, 0},
		{"exactly fits: no thumb", 18, Scroll{Total: 20, Visible: 20}, 0, 0},
		{"top", 18, Scroll{Total: 60, Visible: 20, Offset: 0}, 6, 0},
		{"bottom", 18, Scroll{Total: 60, Visible: 20, Offset: 40}, 6, 12},
		{"middle", 18, Scroll{Total: 60, Visible: 20, Offset: 20}, 6, 6},
		{"offset clamped", 18, Scroll{Total: 60, Visible: 20, Offset: 99}, 6, 12},
		{"tiny thumb is at least one cell", 10, Scroll{Total: 1000, Visible: 5, Offset: 0}, 1, 0},
		{"empty", 10, Scroll{}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, at := thumbGeometry(tt.n, tt.s)
			if size != tt.size || at != tt.at {
				t.Errorf("got size %d at %d, want size %d at %d", size, at, tt.size, tt.at)
			}
		})
	}
}

func TestScrollbarKeepsTrackEnds(t *testing.T) {
	rows := Scrollbar(uni, 13, Scroll{Total: 30, Visible: 10, Offset: 0})
	if len(rows) != 13 {
		t.Fatalf("rows = %d", len(rows))
	}
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = ansi.Strip(r)
	}
	got := strings.Join(plain, "")
	// 11 inner cells, thumb round(11×10/30) = 4, at the top.
	if got != "│████││││││││" {
		t.Errorf("scrollbar = %q", got)
	}
}

func gotoProjectBody(th theme.Theme, bw int) []string {
	base := th.Base()
	muted := th.Style(th.Fade(th.C.Text), th.C.Bg)
	sel := th.Style(th.C.Text, th.C.SelectionBg)
	projects := []string{"#oneofftasks", "#reminders", "#groceries", "#parenting", "#work", "#homelab-tech", "#hobbies-games", "#movies-tv"}
	body := []string{"", muted.Render("Select a project to open its list:"), ""}
	for i, p := range projects {
		if i == 2 {
			body = append(body, Fit(Marker(th, true, th.C.SelectionBg)+sel.Render(p), bw, sel))
			continue
		}
		body = append(body, Marker(th, false, th.C.Bg)+base.Render(p))
	}
	body = append(body, "")
	body = append(body, HintRow(th, bw, []Hint{
		{th.G.Up + " / " + th.G.Down, "select"}, {"enter", "select"}, {"esc", "cancel"}, {"n", "new"}, {"x", "delete"},
	}))
	return body
}

func TestFrame(t *testing.T) {
	bw := FrameBodyWidth(60, true)
	if bw != 54 {
		t.Fatalf("body width = %d, want 54 (Figma: 648px body in a 720px dialog)", bw)
	}
	body := gotoProjectBody(uni, bw)
	f := Frame(uni, FrameOptions{Width: 60, Title: "Goto Project", Focused: true,
		Scroll: &Scroll{Total: 12, Visible: 8}}, body)
	assertBlock(t, f, 60, len(body)+2)
	lines := strings.Split(ansi.Strip(f), "\n")
	if !strings.HasPrefix(lines[0], "┌─ Goto Project ─") || !strings.HasSuffix(lines[0], "─┐") {
		t.Errorf("top edge = %q", lines[0])
	}
	if !strings.HasPrefix(lines[2], "│ Select a project") {
		t.Errorf("body line = %q", lines[2])
	}
	if !strings.HasPrefix(lines[len(lines)-1], "└") || !strings.HasSuffix(lines[len(lines)-1], "┘") {
		t.Errorf("bottom edge = %q", lines[len(lines)-1])
	}
	golden(t, "frame_focused", f)

	// Unfocused, untitled, no scrollbar (Single Task view style), ASCII.
	f = Frame(ascii, FrameOptions{Width: 30}, []string{"Get eggs"})
	assertBlock(t, f, 30, 3)
	want := "+----------------------------+\n| Get eggs                   |\n+----------------------------+"
	if got := ansi.Strip(f); got != want {
		t.Errorf("ascii frame:\n%s\nwant:\n%s", got, want)
	}
}

func TestOverlayCentresDialog(t *testing.T) {
	d := uni.Dimmed()
	var lines []string
	lines = append(lines, Header(d, 80, "Today", "tues · 2026-11-10 · 15:26"))
	for i := 0; i < 20; i++ {
		lines = append(lines, d.Base().Render(strings.Repeat("x", 80)))
	}
	lines = append(lines, StatusBar(d, 80, "Today", theme.ToneInfo, "6 tasks", "s:priority"))
	lines = append(lines, KeyHints(d, 80, todayHints()...)...)
	base := Screen(d, 80, 24, lines)
	assertBlock(t, base, 80, 24)

	dialog := Frame(uni, FrameOptions{Width: 60, Title: "Goto Project", Focused: true,
		Scroll: &Scroll{Total: 12, Visible: 8}}, gotoProjectBody(uni, 54))
	out := Overlay(base, dialog, 80, 24)
	assertBlock(t, out, 80, 24)

	plain := strings.Split(ansi.Strip(out), "\n")
	// 60×15 dialog in 80×24: x = 10, y = 4 (Figma: 120px, 108px → 10 cells, 4.5 rows).
	if !strings.HasPrefix(plain[4][10:], "┌─ Goto Project") {
		t.Errorf("dialog not at (10,4): row 4 = %q", plain[4])
	}
	if !strings.HasPrefix(plain[4], "xxxxxxxxxx") {
		t.Errorf("base should show left of the dialog: %q", plain[4])
	}
	golden(t, "overlay", out)
}

func TestScreenAndTooSmall(t *testing.T) {
	s := Screen(uni, 10, 3, []string{"abc", "a much longer line than ten"})
	assertBlock(t, s, 10, 3)

	ts := TooSmall(uni, 50, 10)
	assertBlock(t, ts, 50, 10)
	if !strings.Contains(ansi.Strip(ts), "taskmoth needs 80×24; this is 50×10.") {
		t.Errorf("too small = %q", ansi.Strip(ts))
	}
	if !strings.Contains(ansi.Strip(TooSmall(ascii, 50, 10)), "80x24") {
		t.Error("ascii too small should use x")
	}
}

func TestButton(t *testing.T) {
	if got := ansi.Strip(Button(uni, "cancel", ButtonDefault, true, false)); got != "[ cancel ]" {
		t.Errorf("label = %q", got)
	}
	// Figma 10:614: the delete frame's 120px buttons are 10 cells.
	if w := Width(Button(uni, "delete", ButtonDestructive, false, false)); w != 10 {
		t.Errorf("width = %d", w)
	}
	var b strings.Builder
	for _, tone := range []ButtonTone{ButtonDefault, ButtonDestructive, ButtonConfirm} {
		for _, f := range []bool{true, false} {
			for _, d := range []bool{false, true} {
				b.WriteString(Button(uni, "x", tone, f, d) + " ")
			}
		}
		b.WriteString("\n")
	}
	golden(t, "buttons", b.String())
}
