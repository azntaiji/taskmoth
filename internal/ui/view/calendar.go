package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	w "github.com/azntaiji/taskmoth/internal/ui/widget"
)

// Calendar is the Calendar view (decisions G14.1–G14.9): one line per day
// from today on, each with a bar of the tasks due that day, under weekly
// headers, and an overdue row at the top when anything is overdue. It
// scrolls forward without end.
type Calendar struct {
	today   task.Date
	loads   map[task.Date]task.DayLoad
	overdue task.DayLoad

	// sel is the selected day; the zero Date selects the overdue row.
	sel     task.Date
	topLine int
	height  int
}

// NewCalendar returns an empty Calendar.
func NewCalendar() *Calendar { return &Calendar{loads: map[task.Date]task.DayLoad{}} }

// SetTasks recounts every day from all tasks. A selection that has fallen
// into the past (at midnight) moves to today.
func (c *Calendar) SetTasks(all []task.Task, now time.Time) {
	c.today = task.DateOf(now)
	c.loads = task.LoadByDay(all)
	c.overdue = task.OverdueLoad(all, c.today)
	switch {
	case c.sel.IsZero() && !c.hasOverdue(), !c.sel.IsZero() && c.sel.Before(c.today):
		c.sel = c.today
	}
	c.scrollToSelection()
}

// SetHeight sets the body height in terminal rows.
func (c *Calendar) SetHeight(h int) {
	c.height = h
	c.scrollToSelection()
}

func (c *Calendar) hasOverdue() bool { return c.overdue.Due > 0 }

// Selected returns the selected day, or overdue = true on the overdue row.
func (c *Calendar) Selected() (day task.Date, overdue bool) { return c.sel, c.sel.IsZero() }

// Select selects day d (today if d is in the past).
func (c *Calendar) Select(d task.Date) {
	if d.Before(c.today) {
		d = c.today
	}
	c.sel = d
	c.scrollToSelection()
}

// Today selects today.
func (c *Calendar) Today() { c.Select(c.today) }

// Top selects the first row: the overdue row if there is one, else today.
func (c *Calendar) Top() {
	if c.hasOverdue() {
		c.sel = task.Date{}
		c.scrollToSelection()
		return
	}
	c.Today()
}

// Move moves the selection by n days. Moving up from today reaches the
// overdue row.
func (c *Calendar) Move(n int) {
	if c.sel.IsZero() {
		if n > 0 {
			c.Select(c.today.AddDays(n - 1))
		}
		return
	}
	if n < 0 && c.hasOverdue() && task.DaysBetween(c.today, c.sel) < -n {
		c.Top()
		return
	}
	c.Select(c.sel.AddDays(n))
}

// Week moves the selection a week forward (d = 1) or back (d = -1).
func (c *Calendar) Week(d int) {
	if c.sel.IsZero() {
		c.sel = c.today
	}
	c.Select(c.sel.AddDays(7 * d))
}

// Page moves the selection by about a screenful in direction d (±1).
func (c *Calendar) Page(d int) { c.Move(d * max(c.height-4, 1)) }

// --- layout ---
//
// The body starts with a blank line. The overdue row, if any, comes next,
// then each week: a blank line, a header and its days (from today on).

// base is the line of the first week's blank line.
func (c *Calendar) base() int {
	if c.hasOverdue() {
		return 2
	}
	return 0
}

// line is the line holding day d (on or after today).
func (c *Calendar) line(d task.Date) int {
	weeks := task.DaysBetween(task.WeekStart(c.today), task.WeekStart(d)) / 7
	return c.base() + 2*(weeks+1) + task.DaysBetween(c.today, d)
}

// selLine is the selection's line, and the first line to keep on screen with
// it (its week header when it's the first listed day of the week).
func (c *Calendar) selLine() (line, first int) {
	if c.sel.IsZero() {
		return 1, 0
	}
	line = c.line(c.sel)
	first = line
	if c.sel == c.today || c.sel.Weekday() == time.Monday {
		first = line - 2
	}
	return line, first
}

// lines is how far the view currently reaches: four weeks past the
// selection, and at least a screenful.
func (c *Calendar) lines() int {
	end := c.today
	if !c.sel.IsZero() {
		end = c.sel
	}
	end = task.WeekStart(end).AddDays(7*4 + 6)
	return max(c.line(end)+1, c.height)
}

func (c *Calendar) scrollToSelection() {
	if c.height <= 0 {
		c.topLine = 0
		return
	}
	line, first := c.selLine()
	if first < c.topLine {
		c.topLine = first
	}
	if line >= c.topLine+c.height {
		c.topLine = line - c.height + 1
	}
	c.topLine = max(min(c.topLine, c.lines()-c.height), 0)
}

// --- rendering ---

const (
	calLabelWidth  = 18 // "thurs · 2026-11-26"
	calCountsWidth = 16 // "99 done / 99 due"
	calGap         = 3
	calMinBar      = 5
	calMaxBar      = 20
)

// calBarWidth is the bar's column at body width bw.
func calBarWidth(bw int) int {
	rest := w.MarkerWidth + calLabelWidth + calGap + calGap + calCountsWidth + calGap + 7 // "~12.5h"
	return max(min(bw-rest, calMaxBar), calMinBar)
}

// LoadText is "1 done / 5 due" (G14.2, G14.12).
func LoadText(l task.DayLoad) string { return fmt.Sprintf("%d done / %d due", l.Done, l.Due) }

// DayLoadLine is the due-date dialogs' line, "mon · 2026-11-23 · 1 done / 5
// due" (G14.12).
func DayLoadLine(t theme.Theme, d task.Date, l task.DayLoad) string {
	return Day(t, d) + " " + t.G.Dot + " " + LoadText(l)
}

// Body renders the list area: height rows of width cells, with the
// scrollbar in the last column.
func (c *Calendar) Body(t theme.Theme, width int) []string {
	base := t.Base()
	bw := BodyWidth(width)
	lines := make([]string, c.height)
	set := func(line int, s string) {
		if y := line - c.topLine; y >= 0 && y < c.height {
			lines[y] = s
		}
	}
	if c.hasOverdue() {
		set(1, c.row(t, bw, "overdue", c.overdue, c.sel.IsZero(), true, false))
	}
	// Days from the first visible week to the last visible line.
	end := c.topLine + c.height
	for d := c.today; c.line(d)-2 < end; d = d.AddDays(1) {
		if d == c.today || d.Weekday() == time.Monday {
			label := "week of " + task.WeekStart(d).String()
			set(c.line(d)-1, w.Blank(base, w.MarkerWidth)+
				t.Style(t.C.TextAccent, t.C.Bg).Bold(true).Render(w.Truncate(t, label, bw-w.MarkerWidth)))
		}
		set(c.line(d), c.row(t, bw, Day(t, d), c.loads[d], d == c.sel, false, d == c.today))
	}

	bar := w.Scrollbar(t, c.height, w.Scroll{Total: c.lines(), Visible: c.height, Offset: c.topLine})
	out := make([]string, c.height)
	for i := range out {
		out[i] = w.Fit(lines[i], bw, base) + base.Render(" ") + bar[i]
	}
	return out
}

// row draws one day (or the overdue row): marker, label, bar, counts and
// the estimate still remaining (G14.2, G14.3).
func (c *Calendar) row(t theme.Theme, bw int, label string, l task.DayLoad, selected, overdue, today bool) string {
	bg, fg := t.C.Bg, t.C.Text
	if selected {
		bg, fg = t.C.SelectionBg, t.C.SelectionFg
	}
	fill := t.Style(fg, bg)
	gap := w.Blank(fill, calGap)

	labelStyle := t.Style(fg, bg)
	if overdue && !selected {
		labelStyle = t.Style(t.C.TextError, bg)
	}
	if today {
		labelStyle = labelStyle.Bold(true)
	}
	s := w.Marker(t, selected, bg) + w.Fit(labelStyle.Render(w.Truncate(t, label, calLabelWidth)), calLabelWidth, fill) + gap

	barW := calBarWidth(bw)
	if l.Due == 0 {
		return w.Fit(s+t.Style(t.FadeOn(fg, bg), bg).Render(t.G.Dot), bw, fill)
	}
	s += w.Fit(calBar(t, barW, l, overdue, selected), barW, fill) + gap

	counts := LoadText(l)
	if overdue {
		counts = fmt.Sprintf("%d due", l.Due) // overdue tasks are open by definition
	}
	s += w.Fit(t.Style(fg, bg).Render(counts), calCountsWidth, fill)
	if l.RemainingMin > 0 {
		s += gap + t.Style(t.FadeOn(fg, bg), bg).Render(Duration(l.RemainingMin))
	}
	return w.Fit(s, bw, fill)
}

// calBar is one cell per task, open ones first and completed ones dimmed after
// them, ending in "+" when they don't fit in width cells (G14.3).
func calBar(t theme.Theme, width int, l task.DayLoad, overdue, selected bool) string {
	open, done := l.Open(), l.Done
	more := false
	if open+done > width {
		more = true
		room := width - 1
		open = min(open, room)
		done = room - open
	}
	col, bg := t.C.Key, t.C.Bg
	if overdue {
		col = t.C.TextError
	}
	if selected {
		col, bg = t.C.SelectionFg, t.C.SelectionBg
	}
	s := t.Style(col, bg).Render(strings.Repeat(t.G.Bar, open))
	if done > 0 {
		s += t.Style(t.FadeOn(col, bg), bg).Render(strings.Repeat(t.G.Bar, done))
	}
	if more {
		s += t.Style(col, bg).Render("+")
	}
	return s
}

// WeekLoad sums the week (Monday to Sunday) holding the selected day.
func (c *Calendar) WeekLoad() (start task.Date, l task.DayLoad) {
	start = task.WeekStart(c.today)
	if !c.sel.IsZero() {
		start = task.WeekStart(c.sel)
	}
	for i := range 7 {
		l = l.Add(c.loads[start.AddDays(i)])
	}
	return start, l
}

// StatusBar renders the status bar: the selected week's totals (G14.4), or
// the overdue totals on the overdue row. pending replaces the right-hand
// text and message the summary, as in the lists.
func (c *Calendar) StatusBar(t theme.Theme, width int, pending, message string) string {
	dot := " " + t.G.Dot + " "
	var msg string
	if c.sel.IsZero() {
		msg = fmt.Sprintf("%d overdue", c.overdue.Due)
		if c.overdue.RemainingMin > 0 {
			msg += dot + Duration(c.overdue.RemainingMin) + " remaining"
		}
	} else {
		start, l := c.WeekLoad()
		msg = "week of " + start.String() + dot + LoadText(l)
		if l.RemainingMin > 0 {
			msg += dot + Duration(l.RemainingMin) + " remaining"
		}
	}
	if message != "" {
		msg = message
	}
	return w.StatusBar(t, width, "Calendar", theme.ToneInfo, msg, pending)
}
