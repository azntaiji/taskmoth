// Package gallery is a developer screen (taskmoth --gallery) that renders
// every Phase 1 widget, plus mock screens assembled from them, so they can be
// compared with the Figma designs. The task rows here are a rough mock for
// context; the real list views are built in Phase 3.
package gallery

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

type page struct {
	title string
	figma string // node to compare against
	view  func(m *Model, t theme.Theme) string
	input *w.Input // the live input on this page, if any
}

// Model is the gallery's Bubble Tea model.
type Model struct {
	theme  theme.Theme
	width  int
	height int
	page   int
	pages  []page

	desc *w.Input // Edit Description page
	date *w.Input // Edit Start date page
}

// New returns the gallery.
func New(t theme.Theme) *Model {
	desc := &w.Input{Multiline: true}
	desc.SetValue("I need to goto trader joe's to get eggs. Also get the rest of the stuff on the list.\n\nReach out to Hope if she needs anything that isn't on the list.")
	desc.Focus()
	date := &w.Input{Placeholder: "yourprojectname"}
	date.SetValue("2026-10-02")
	date.Focus()

	m := &Model{theme: t, desc: desc, date: date}
	m.pages = []page{
		{"Components", "Components page (0:1)", (*Model).components, nil},
		{"Scrollbars", "29:7311", (*Model).scrollbars, nil},
		{"Today (mock rows)", "32:9525", func(m *Model, t theme.Theme) string { return m.today(t) }, nil},
		{"Dialog over dimmed view", "32:13454", (*Model).overlay, nil},
		{"Edit Description (live input)", "15:5973", (*Model).editDescription, desc},
		{"Edit Start date (live input)", "40:15310", (*Model).editStartDate, date},
		{"Keyboard Shortcuts frame", "40:15079", (*Model).shortcuts, nil},
	}
	return m
}

// pageWith returns the index of the page that edits in.
func (m *Model) pageWith(in *w.Input) int {
	for i, p := range m.pages {
		if p.input == in {
			return i
		}
	}
	return -1
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		case "tab":
			m.page = (m.page + 1) % len(m.pages)
			return m, nil
		case "shift+tab":
			m.page = (m.page + len(m.pages) - 1) % len(m.pages)
			return m, nil
		case "ctrl+j":
			// Gallery-only newline key; the real binding is gate G6.2.
			if in := m.pages[m.page].input; in != nil {
				in.InsertNewline()
			}
			return m, nil
		}
		if in := m.pages[m.page].input; in != nil {
			in.Update(msg)
		}
	case tea.PasteMsg:
		if in := m.pages[m.page].input; in != nil {
			in.Update(msg)
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() tea.View {
	var content string
	switch {
	case m.width == 0:
		content = ""
	case m.width < w.MinWidth || m.height < w.MinHeight:
		content = w.TooSmall(m.theme, m.width, m.height)
	default:
		p := m.pages[m.page]
		content = p.view(m, m.theme)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	p := m.pages[m.page]
	v.WindowTitle = fmt.Sprintf("taskmoth gallery %d/%d: %s (Figma %s)", m.page+1, len(m.pages), p.title, p.figma)
	return v
}

// --- pages ---

func (m *Model) components(t theme.Theme) string {
	W := m.width
	base := t.Base()
	label := t.Style(t.Fade(t.C.Text), t.C.Bg)
	p := m.pages[m.page]

	var lines []string
	lines = append(lines, w.Header(t, W, "Gallery", fmt.Sprintf("%d/%d %s", m.page+1, len(m.pages), p.title)))
	lines = append(lines, label.Render(" tab / shift+tab: pages · ctrl+q: quit · window title names the Figma node to compare"))
	lines = append(lines, "")

	row := func(name, content string) string {
		return label.Render(fmt.Sprintf(" %-13s", name)) + content
	}
	var tones, dims string
	names := []string{"neutral", "info", "success", "warning", "error", "mode"}
	for i, n := range names {
		tones += w.Badge(t, theme.Tone(i), n, false) + base.Render(" ")
		dims += w.Badge(t, theme.Tone(i), n, true) + base.Render(" ")
	}
	lines = append(lines, row("Badge", tones))
	lines = append(lines, row("Badge dim", dims))

	var pri, priDim string
	for p := 1; p <= 4; p++ {
		pri += w.PriorityMark(t, p, false, t.C.Bg) + base.Render(" ")
		priDim += w.PriorityMark(t, p, true, t.C.Bg) + base.Render(" ")
	}
	lines = append(lines, row("Priority", pri+base.Render("   ")+label.Render("dim ")+priDim))
	lines = append(lines, row("DueLabel",
		w.DueLabel(t, w.DueUpcoming, "2026-11-20", false)+base.Render(" ")+
			w.DueLabel(t, w.DueToday, "today", false)+base.Render(" ")+
			w.DueLabel(t, w.DueOverdue, "overdue", false)+base.Render("  ")+
			w.DueLabel(t, w.DueToday, "today", true)+base.Render(" ")+
			w.DueLabel(t, w.DueOverdue, "overdue", true)))
	lines = append(lines, row("Marker/Check",
		w.Marker(t, true, t.C.Bg)+w.Checkbox(t, false, t.C.Bg)+base.Render("  ")+
			w.Marker(t, false, t.C.Bg)+w.Checkbox(t, true, t.C.Bg)+base.Render("  ")+
			w.Marker(t, true, t.C.SelectionBg)+w.Checkbox(t, false, t.C.SelectionBg)+
			t.Style(t.C.Text, t.C.SelectionBg).Render("selected ")))
	lines = append(lines, row("Divider", w.Divider(t, W-15)))
	lines = append(lines, "")
	lines = append(lines, label.Render(" StatusBar"))
	lines = append(lines, w.StatusBar(t, W, "Today", theme.ToneInfo, "4/6 tasks · ~8h total · ~3.75h remaining", "s:priority"))
	lines = append(lines, w.StatusBar(t, W, "NORMAL", theme.ToneMode, "6 tasks · ~6h total · ~3.75h remaining", "4/6"))
	lines = append(lines, "")
	lines = append(lines, label.Render(" KeyHints"))
	lines = append(lines, w.KeyHints(t, W, todayHints...)...)
	return w.Screen(t, W, m.height, lines)
}

// scrollbars shows the Figma Scrollbar variants (29:7311) side by side.
func (m *Model) scrollbars(t theme.Theme) string {
	base := t.Base()
	label := t.Style(t.Fade(t.C.Text), t.C.Bg)
	h := m.height - 3
	variants := []struct {
		name string
		s    w.Scroll
	}{
		{"top", w.Scroll{Total: 60, Visible: 20, Offset: 0}},
		{"middle", w.Scroll{Total: 60, Visible: 20, Offset: 20}},
		{"bottom", w.Scroll{Total: 60, Visible: 20, Offset: 40}},
		{"top, focused", w.Scroll{Total: 60, Visible: 20, Offset: 0, Focused: true}},
		{"middle, focused", w.Scroll{Total: 60, Visible: 20, Offset: 20, Focused: true}},
		{"bottom, focused", w.Scroll{Total: 60, Visible: 20, Offset: 40, Focused: true}},
		{"all visible", w.Scroll{Total: 6, Visible: 20}},
	}
	const colW = 11
	var head strings.Builder
	bars := make([][]string, len(variants))
	for i, v := range variants {
		head.WriteString(w.Fit(label.Render(" "+w.Truncate(t, v.name, colW-2)), colW, base))
		bars[i] = w.Scrollbar(t, h, v.s)
	}
	lines := []string{w.Header(t, m.width, "Gallery", "Scrollbars"), head.String(), ""}
	for r := 0; r < h; r++ {
		var b strings.Builder
		for i := range variants {
			b.WriteString(w.Blank(base, colW/2) + bars[i][r] + w.Blank(base, colW-colW/2-1))
		}
		lines = append(lines, b.String())
	}
	return w.Screen(t, m.width, m.height, lines)
}

var todayHints = [][]w.Hint{
	{{Key: "q", Label: "add"}, {Key: "space", Label: "done"}, {Key: "e", Label: "edit/view"}, {Key: "x", Label: "delete"},
		{Key: "s", Label: "sort"}, {Key: "?", Label: "shortcuts"}, {Key: "/", Label: "search"}},
	{{Key: "g i", Label: "inbox"}, {Key: "g t", Label: "today"}, {Key: "g u", Label: "upcoming"},
		{Key: "g p", Label: "projects"}, {Key: "g o", Label: "overdue"}, {Key: "g a", Label: "all"}},
}

type mockTask struct {
	done     bool
	priority int
	name     string
	meta     []string
	due      w.DueState
	dueText  string
}

var mockToday = []mockTask{
	{false, 1, "File insurance reimbursements", []string{"#oneofftasks", "~2h", "s:2026-11-10"}, w.DueToday, "today"},
	{false, 2, "Practice meditating", []string{"#reminders", "~30m"}, w.DueToday, "today"},
	{false, 2, "Pack Azalea's books in backpack", []string{"#parenting", "~1h"}, w.DueToday, "today"},
	{false, 3, "Get eggs", []string{"#groceries", "~15m", "s:2026-11-09"}, w.DueOverdue, "overdue"},
	{true, 3, "Finish creating task management TUI app. Then put it into production", []string{"#homelab-tech", "~3.5h", "s:2026-11-07"}, w.DueOverdue, "overdue"},
	{true, 4, "Watch Tales of the Loop", []string{"#movies-tv", "~45m", "s:2026-11-10"}, w.DueToday, "today"},
}

// mockRow approximates Figma TaskItemv2 (14:1199) for context only.
func mockRow(t theme.Theme, width int, task mockTask, selected bool) []string {
	bg := t.C.Bg
	if selected {
		bg = t.C.SelectionBg
	}
	fill := t.Style(t.C.Text, bg)
	nameColor := t.C.Text
	if task.done {
		nameColor = t.FadeOn(t.C.Text, bg)
	}
	due := w.DueLabel(t, task.due, task.dueText, task.done)
	nameW := width - 10 - w.Width(due) - 2

	l1 := w.Marker(t, selected, bg) + w.Checkbox(t, task.done, bg) +
		w.PriorityMark(t, task.priority, task.done, bg) + fill.Render(" ") +
		t.Style(nameColor, bg).Render(w.Truncate(t, task.name, nameW))
	l1 = w.Spread(l1, fill.Render(" ")+due, width, fill)

	meta := t.Style(t.FadeOn(t.C.Text, bg), bg)
	l2 := w.Blank(fill, 10)
	for _, s := range task.meta {
		l2 += meta.Render(" " + s + " ")
	}
	return []string{l1, w.Fit(l2, width, fill)}
}

func (m *Model) today(t theme.Theme) string {
	W, H := m.width, m.height
	bodyW := W - 2 // body, gap, scrollbar column
	bodyH := H - 4

	var body []string
	for i, task := range mockToday {
		body = append(body, "")
		body = append(body, mockRow(t, bodyW, task, i == 3)...)
	}
	bar := w.Scrollbar(t, bodyH, w.Scroll{Total: len(body), Visible: bodyH})
	base := t.Base()

	lines := []string{w.Header(t, W, "Today", "tues "+t.G.Dot+" 2026-11-10 "+t.G.Dot+" 15:26")}
	for i := 0; i < bodyH; i++ {
		l := ""
		if i < len(body) {
			l = body[i]
		}
		lines = append(lines, w.Fit(l, bodyW, base)+base.Render(" ")+bar[i])
	}
	lines = append(lines, w.StatusBar(t, W, "Today", theme.ToneInfo,
		"4/6 tasks "+t.G.Dot+" ~8h total "+t.G.Dot+" ~3.75h remaining", "s:priority"))
	lines = append(lines, w.KeyHints(t, W, todayHints...)...)
	return w.Screen(t, W, H, lines)
}

func (m *Model) dialogOver(t theme.Theme, dialog string) string {
	return w.Overlay(m.today(t.Dimmed()), dialog, m.width, m.height)
}

func (m *Model) overlay(t theme.Theme) string {
	const width = 60
	bw := w.FrameBodyWidth(width, true)
	base := t.Base()
	help := t.Style(t.Fade(t.C.Text), t.C.Bg)
	sel := t.Style(t.C.Text, t.C.SelectionBg)
	body := []string{"", help.Render("Select a project to open its list:"), ""}
	for i, p := range []string{"#oneofftasks", "#reminders", "#groceries", "#parenting", "#work", "#homelab-tech", "#hobbies-games", "#movies-tv"} {
		if i == 2 {
			body = append(body, w.Fit(w.Marker(t, true, t.C.SelectionBg)+sel.Render(p), bw, sel))
		} else {
			body = append(body, w.Marker(t, false, t.C.Bg)+base.Render(p))
		}
	}
	body = append(body, "", w.HintRow(t, bw, []w.Hint{
		{Key: t.G.Up + " / " + t.G.Down, Label: "select"}, {Key: "enter", Label: "select"},
		{Key: "esc", Label: "cancel"}, {Key: "n", Label: "new"}, {Key: "x", Label: "delete"},
	}))
	d := w.Frame(t, w.FrameOptions{Width: width, Title: "Goto Project", Focused: true,
		Scroll: &w.Scroll{Total: 12, Visible: 8}}, body)
	return m.dialogOver(t, d)
}

func (m *Model) editDescription(t theme.Theme) string {
	const width = 60
	bw := w.FrameBodyWidth(width, true)
	help := t.Style(t.Fade(t.C.Text), t.C.Bg)
	m.desc.SetSize(bw, 5)
	body := []string{"", help.Render("Edit task description:"), ""}
	body = append(body, w.Lines(m.desc.View(t, true))...)
	body = append(body, "", w.HintRow(t, bw, []w.Hint{
		{Key: "ctrl+j", Label: "newline (gallery only)"}, {Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"},
	}))
	d := w.Frame(t, w.FrameOptions{Width: width, Title: "Edit Description", Focused: true,
		Scroll: &w.Scroll{Total: 30, Visible: 10}}, body)
	return m.dialogOver(t, d)
}

func (m *Model) editStartDate(t theme.Theme) string {
	const width = 60
	bw := w.FrameBodyWidth(width, false)
	help := t.Style(t.Fade(t.C.Text), t.C.Bg)
	m.date.SetSize(bw, 1)
	body := []string{
		"",
		help.Render("Enter date in YYYY-MM-DD format or use natural language."),
		help.Render("Examples: 2026-11-02, today, in 3 days, next monday"),
		"",
		m.date.View(t, false),
		"",
		w.HintRow(t, bw, []w.Hint{{Key: "enter", Label: "save"}, {Key: "esc", Label: "cancel"}}),
	}
	d := w.Frame(t, w.FrameOptions{Width: width, Title: "Edit Start date", Focused: true}, body)
	return m.dialogOver(t, d)
}

func (m *Model) shortcuts(t theme.Theme) string {
	const width = 57
	key := t.Style(t.C.Key, t.C.Bg).Bold(true)
	head := t.Base().Bold(true)
	lbl := t.Style(t.C.KeyLabel, t.C.Bg)
	base := t.Base()
	col := func(rows [][2]string, title string, wdt int) []string {
		out := []string{w.Fit(head.Render(title), wdt, base)}
		for _, r := range rows {
			out = append(out, w.Fit(key.Render(r[0])+base.Render(" ")+lbl.Render(r[1]), wdt, base))
		}
		return out
	}
	gotoCol := col([][2]string{{"g i", "inbox"}, {"g t", "today"}, {"g u", "upcoming"}, {"g p", "projects"},
		{"g o", "overdue"}, {"g a", "all"}, {"g c", "completed"}}, "Goto Lists", 15)
	listCol := col([][2]string{{t.G.Up + " / " + t.G.Down, "select task"}, {"pgup / pgdn", "scroll"},
		{"q", "add task"}, {"s", "cycle sorting"}, {"/", "search for task"}}, "Within Lists", 20)
	selCol := col([][2]string{{"space", "mark done"}, {"e", "edit/view task"}, {"x", "delete"},
		{"p", "edit priority"}, {"v", "edit project"}, {"r", "edit start date"}, {"t", "edit due date"},
		{"y", "edit duration"}, {"n", "edit name"}, {"d", "edit description"}}, "Selected Tasks", 18)

	// Figma 40:15079: columns at 24px, 204px and 444px → 15, 20 and 18 cells.
	body := []string{""}
	for i := 0; i < len(selCol); i++ {
		get := func(c []string, wdt int) string {
			if i < len(c) {
				return c[i]
			}
			return w.Blank(base, wdt)
		}
		body = append(body, get(gotoCol, 15)+get(listCol, 20)+get(selCol, 18))
	}
	body = append(body, "")
	d := w.Frame(t, w.FrameOptions{Width: width, Title: "Keyboard Shortcuts"}, body)
	return m.dialogOver(t, d)
}
