// Package view holds taskmoth's screens: the list views (Today, Inbox, and
// in later phases the rest) and their text formatting.
package view

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
)

// Duration formats minutes as in the designs (decision G3.7): ~45m under an
// hour, otherwise hours with up to two decimals and no trailing zeros
// (~1.5h, ~3.75h, ~8h).
func Duration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("~%dm", minutes)
	}
	h := math.Round(float64(minutes)/60*100) / 100
	return "~" + strconv.FormatFloat(h, 'f', -1, 64) + "h"
}

var weekdays = [...]string{"sun", "mon", "tues", "wed", "thurs", "fri", "sat"}

// Clock formats the header clock (decision G3.6): "tues · 2026-11-10 · 15:26".
func Clock(t theme.Theme, now time.Time) string {
	dot := " " + t.G.Dot + " "
	return weekdays[now.Weekday()] + dot + now.Format(time.DateOnly) + dot + now.Format("15:04")
}

// Day formats an Upcoming day header (decision G4.1): "wed · 2026-11-11".
func Day(t theme.Theme, d task.Date) string {
	wd := time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC).Weekday()
	return weekdays[wd] + " " + t.G.Dot + " " + d.String()
}
