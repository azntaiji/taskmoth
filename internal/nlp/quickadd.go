package nlp

import (
	"strings"
	"unicode"

	"github.com/azntaiji/taskmoth/internal/task"
)

// Kind is what a recognised token sets.
type Kind int

// Token kinds.
const (
	KindProject Kind = iota
	KindPriority
	KindDuration
	KindStart
	KindDue
	numKinds
)

// Span is a recognised token, in rune offsets into the input [Start, End).
type Span struct {
	Start, End int
	Kind       Kind
}

// Result is a parsed quick-add line.
type Result struct {
	Name        string    // what's left once recognised tokens are removed
	Project     string    // normalized project name ("" = none given)
	Priority    int       // 1–4, or 0 if none was given
	Start, Due  task.Date // zero = none
	DurationMin int       // 0 = none
	// Spans are the tokens that count (the last of each kind), for
	// highlighting. Earlier tokens of the same kind stay in the name.
	Spans []Span
}

// Has reports whether a token of kind k was recognised.
func (r Result) Has(k Kind) bool {
	for _, s := range r.Spans {
		if s.Kind == k {
			return true
		}
	}
	return false
}

type word struct {
	text       string
	start, end int // rune offsets
}

func splitWords(rs []rune) []word {
	var ws []word
	for i := 0; i < len(rs); {
		if unicode.IsSpace(rs[i]) {
			i++
			continue
		}
		j := i
		for j < len(rs) && !unicode.IsSpace(rs[j]) {
			j++
		}
		ws = append(ws, word{text: string(rs[i:j]), start: i, end: j})
		i = j
	}
	return ws
}

// maxPhrase is the most words a date phrase can span
// ("2 weeks before 20 nov 2027").
const maxPhrase = 7

type match struct {
	kind        Kind
	first, last int // word indices
	project     string
	priority    int
	minutes     int
	date        task.Date
}

// Parse parses a quick-add line (PLAN.md Appendix A, decisions G5.1a–f).
func Parse(input string, today task.Date) Result {
	ws := splitWords([]rune(input))
	lower := make([]string, len(ws))
	for i, w := range ws {
		lower[i] = strings.ToLower(w.text)
	}

	var matches []match
	escaped := map[int]bool{}
	for i := 0; i < len(ws); {
		m, ok := matchAt(ws, lower, i, today)
		switch {
		case strings.HasPrefix(ws[i].text, `\`) && len(ws[i].text) > 1:
			escaped[i] = true
			i++
		case ok:
			matches = append(matches, m)
			i = m.last + 1
		default:
			i++
		}
	}

	// The last token of each kind wins (Appendix A); earlier ones stay text.
	var winners [numKinds]*match
	for i := range matches {
		winners[matches[i].kind] = &matches[i]
	}
	var r Result
	consumed := map[int]bool{}
	for _, m := range winners {
		if m == nil {
			continue
		}
		for i := m.first; i <= m.last; i++ {
			consumed[i] = true
		}
		r.Spans = append(r.Spans, Span{Start: ws[m.first].start, End: ws[m.last].end, Kind: m.kind})
		switch m.kind {
		case KindProject:
			r.Project = m.project
		case KindPriority:
			r.Priority = m.priority
		case KindDuration:
			r.DurationMin = m.minutes
		case KindStart:
			r.Start = m.date
		case KindDue:
			r.Due = m.date
		}
	}

	var name []string
	for i, w := range ws {
		switch {
		case consumed[i]:
		case escaped[i]:
			name = append(name, w.text[1:])
		default:
			name = append(name, w.text)
		}
	}
	r.Name = strings.Join(name, " ")
	return r
}

// matchAt tries every token form at word i.
func matchAt(ws []word, lower []string, i int, today task.Date) (match, bool) {
	w := lower[i]
	switch {
	case strings.HasPrefix(w, `\`):
		return match{}, false
	case strings.HasPrefix(w, "#"):
		name := task.NormalizeProjectName(w)
		if name != "" && task.ValidateProjectName(name) == nil {
			return match{kind: KindProject, first: i, last: i, project: name}, true
		}
	case len(w) == 2 && w[0] == 'p' || len(w) == 3 && strings.HasPrefix(w, "!p"):
		if p := w[len(w)-1] - '0'; p >= 1 && p <= 4 {
			return match{kind: KindPriority, first: i, last: i, priority: int(p)}, true
		}
	case strings.HasPrefix(w, "~"):
		for n := min(4, len(ws)-i); n >= 1; n-- {
			phrase := strings.Join(append([]string{w[1:]}, lower[i+1:i+n]...), " ")
			if mins, ok := ParseDuration(phrase); ok {
				return match{kind: KindDuration, first: i, last: i + n - 1, minutes: mins}, true
			}
		}
	case strings.HasPrefix(w, "s:"):
		return datePhrase(lower, i, w[2:], KindStart, true, today)
	case strings.HasPrefix(w, "@"):
		return datePhrase(lower, i, w[1:], KindDue, true, today)
	default:
		return datePhrase(lower, i, w, KindDue, false, today)
	}
	return match{}, false
}

// datePhrase finds the longest date phrase starting at word i, whose first
// word (after any sigil) is head.
func datePhrase(lower []string, i int, head string, kind Kind, sigil bool, today task.Date) (match, bool) {
	if head == "" {
		return match{}, false
	}
	for n := min(maxPhrase, len(lower)-i); n >= 1; n-- {
		words := append([]string{head}, lower[i+1:i+n]...)
		if d, ok := parseDate(words, today, sigil); ok {
			return match{kind: kind, first: i, last: i + n - 1, date: d}, true
		}
	}
	return match{}, false
}

// ToggleAt implements ctrl+t (decision G5.1d): if the cursor (a rune offset)
// is in or just after a recognised token, every word of that token gets a
// backslash so it stays text; if it is on an escaped word, the backslashes
// of that run of escaped words are removed. It returns the new text and
// cursor, or ok=false if there was nothing to toggle.
func ToggleAt(input string, cursor int, today task.Date) (string, int, bool) {
	rs := []rune(input)
	ws := splitWords(rs)

	// Un-escape a run of escaped words around the cursor.
	for i, w := range ws {
		if cursor < w.start || cursor > w.end || !strings.HasPrefix(w.text, `\`) {
			continue
		}
		lo, hi := i, i
		for lo > 0 && strings.HasPrefix(ws[lo-1].text, `\`) {
			lo--
		}
		for hi < len(ws)-1 && strings.HasPrefix(ws[hi+1].text, `\`) {
			hi++
		}
		var at []int
		for j := lo; j <= hi; j++ {
			at = append(at, ws[j].start)
		}
		return removeAt(rs, at, cursor)
	}

	// Escape the recognised token at or just before the cursor.
	for _, sp := range Parse(input, today).Spans {
		if cursor < sp.Start || cursor > sp.End {
			continue
		}
		var at []int
		for _, w := range ws {
			if w.start >= sp.Start && w.end <= sp.End {
				at = append(at, w.start)
			}
		}
		out := make([]rune, 0, len(rs)+len(at))
		shift := 0
		for i, r := range rs {
			for _, a := range at {
				if a == i {
					out = append(out, '\\')
					if i < cursor {
						shift++
					}
				}
			}
			out = append(out, r)
		}
		return string(out), cursor + shift, true
	}
	return input, cursor, false
}

// removeAt deletes the rune at each offset in at (all backslashes).
func removeAt(rs []rune, at []int, cursor int) (string, int, bool) {
	drop := map[int]bool{}
	for _, a := range at {
		drop[a] = true
	}
	out := make([]rune, 0, len(rs))
	shift := 0
	for i, r := range rs {
		if drop[i] {
			if i < cursor {
				shift++
			}
			continue
		}
		out = append(out, r)
	}
	return string(out), cursor - shift, true
}
