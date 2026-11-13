package task

import (
	"fmt"
	"time"
)

// Date is a calendar date with no time or time zone (decision G2.1b). The
// zero value means "no date".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateOf returns the calendar date of t in t's own location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, m, d}
}

// ParseDate parses YYYY-MM-DD. The empty string parses to the zero Date.
func ParseDate(s string) (Date, error) {
	if s == "" {
		return Date{}, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: want YYYY-MM-DD", s)
	}
	return DateOf(t), nil
}

// IsZero reports whether d is unset.
func (d Date) IsZero() bool { return d == Date{} }

// String formats d as YYYY-MM-DD, or "" when unset.
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// time returns midnight UTC on d; only used for arithmetic.
func (d Date) time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// AddDays returns d shifted by n days.
func (d Date) AddDays(n int) Date { return DateOf(d.time().AddDate(0, 0, n)) }

// Compare returns -1, 0 or +1 as d is before, equal to or after e.
func (d Date) Compare(e Date) int { return d.time().Compare(e.time()) }

// Before reports whether d is before e.
func (d Date) Before(e Date) bool { return d.Compare(e) < 0 }

// After reports whether d is after e.
func (d Date) After(e Date) bool { return d.Compare(e) > 0 }
