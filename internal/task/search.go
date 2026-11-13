package task

import (
	"cmp"
	"slices"
	"strings"
)

// Query is a parsed search (decisions G7.1b, G7.1c).
type Query struct {
	// Words must each appear, case-insensitively, in the task's name,
	// description or project name.
	Words []string
	// Project, if set, keeps tasks whose project name starts with it
	// (lower case, so "#wor" already narrows to #work while typing).
	Project string
	// Inbox keeps only tasks with no project (#inbox).
	Inbox bool
	// Priority, if 1–4, keeps only tasks of that priority (!p1–!p4).
	Priority int
}

// ParseQuery splits a search into words and filters. `#name` filters by
// project and `!p1`–`!p4` by priority, as in Add Task. A bare `#` or `!p`
// (half typed) is ignored rather than matching nothing.
func ParseQuery(s string) Query {
	var q Query
	for _, f := range strings.Fields(strings.ToLower(s)) {
		switch {
		case f == "#inbox":
			q.Inbox = true
		case strings.HasPrefix(f, "#"):
			q.Project = NormalizeProjectName(f[1:])
		case len(f) == 3 && f[:2] == "!p" && f[2] >= '1' && f[2] <= '4':
			q.Priority = int(f[2] - '0')
		case f == "!p" || f == "!":
			// still being typed
		default:
			q.Words = append(q.Words, f)
		}
	}
	return q
}

// Match reports whether t matches q, given the name of t's project ("" for
// the Inbox). nameHit is true when the name alone contains every word; those
// tasks are listed first (G7.1d).
func (q Query) Match(t Task, project string) (match, nameHit bool) {
	project = strings.ToLower(project)
	switch {
	case q.Inbox && t.ProjectID != "":
		return false, false
	case q.Project != "" && (t.ProjectID == "" || !strings.HasPrefix(project, q.Project)):
		return false, false
	case q.Priority != 0 && t.Priority != q.Priority:
		return false, false
	}
	name := strings.ToLower(t.Name)
	desc := strings.ToLower(t.Description)
	nameHit = true
	for _, w := range q.Words {
		inName := strings.Contains(name, w)
		if !inName && !strings.Contains(desc, w) && !strings.Contains(project, w) {
			return false, false
		}
		nameHit = nameHit && inName
	}
	return true, nameHit
}

// SortSearch orders search results (G7.1d): open before completed, then
// tasks whose name matches every word, then priority, then A–Z and creation
// order.
func SortSearch(tasks []Task, nameHit func(Task) bool) {
	slices.SortStableFunc(tasks, func(a, b Task) int {
		if a.Done() != b.Done() {
			if a.Done() {
				return 1
			}
			return -1
		}
		if ha, hb := nameHit(a), nameHit(b); ha != hb {
			if ha {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(a.Priority, b.Priority); c != 0 {
			return c
		}
		if c := compareNames(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}
