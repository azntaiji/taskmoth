// Package export writes every task and project as CSV or JSON (decisions
// G10.1–G10.4). It has no UI or storage imports: the caller loads the data and
// passes the export time in.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
)

// Formats are the names --export accepts.
var Formats = []string{"csv", "json"}

// FormatVersion is the JSON document's "version". Bump it if a field changes
// meaning or is removed; adding fields doesn't need a bump.
const FormatVersion = 1

// CSVHeader is the CSV's first row (G10.3).
var CSVHeader = []string{
	"id", "name", "description", "project", "priority", "start", "due", "duration_min",
	"created_at", "updated_at", "completed_at", "archived_at", "repeat", "series_id",
}

// ParseFormat normalizes a format name, or explains which ones exist.
func ParseFormat(s string) (string, error) {
	f := strings.ToLower(strings.TrimSpace(s))
	if !slices.Contains(Formats, f) {
		return "", fmt.Errorf("unknown export format %q; use %s", s, strings.Join(Formats, " or "))
	}
	return f, nil
}

// Write writes every task (open, completed and archived) and, for JSON, every
// project to w. Tasks are in creation order and projects by name.
func Write(w io.Writer, format string, projects []task.Project, tasks []task.Task, now time.Time) error {
	projects = slices.Clone(projects)
	slices.SortFunc(projects, func(a, b task.Project) int { return strings.Compare(a.Name, b.Name) })
	tasks = slices.Clone(tasks)
	slices.SortFunc(tasks, func(a, b task.Task) int { return strings.Compare(string(a.ID), string(b.ID)) })
	names := make(map[task.ID]string, len(projects))
	for _, p := range projects {
		names[p.ID] = p.Name
	}

	switch format {
	case "csv":
		return writeCSV(w, tasks, names)
	case "json":
		return writeJSON(w, projects, tasks, names, now)
	}
	_, err := ParseFormat(format)
	return err
}

func writeCSV(w io.Writer, tasks []task.Task, names map[task.ID]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(CSVHeader); err != nil {
		return err
	}
	for _, t := range tasks {
		dur := ""
		if t.DurationMin > 0 {
			dur = strconv.Itoa(t.DurationMin)
		}
		row := []string{
			string(t.ID), t.Name, t.Description, names[t.ProjectID], strconv.Itoa(t.Priority),
			t.Start.String(), t.Due.String(), dur,
			stamp(t.CreatedAt), stamp(t.UpdatedAt), stamp(t.CompletedAt), stamp(t.ArchivedAt),
			t.Repeat.String(), string(t.SeriesID),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

type jsonDoc struct {
	Version    int           `json:"version"`
	ExportedAt string        `json:"exported_at"`
	Projects   []jsonProject `json:"projects"`
	Tasks      []jsonTask    `json:"tasks"`
}

type jsonProject struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  *string `json:"created_at"`
	UpdatedAt  *string `json:"updated_at"`
	ArchivedAt *string `json:"archived_at"`
}

type jsonTask struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ProjectID   *string `json:"project_id"`
	Project     *string `json:"project"`
	Priority    int     `json:"priority"`
	Start       *string `json:"start"`
	Due         *string `json:"due"`
	DurationMin *int    `json:"duration_min"`
	CreatedAt   *string `json:"created_at"`
	UpdatedAt   *string `json:"updated_at"`
	CompletedAt *string `json:"completed_at"`
	ArchivedAt  *string `json:"archived_at"`
	// Repeat is the rule in words, as quick add reads it ("every mon");
	// SeriesID links a series' occurrences (G15.1).
	Repeat   *string `json:"repeat"`
	SeriesID *string `json:"series_id"`
}

func writeJSON(w io.Writer, projects []task.Project, tasks []task.Task, names map[task.ID]string, now time.Time) error {
	doc := jsonDoc{
		Version:    FormatVersion,
		ExportedAt: stamp(now),
		Projects:   make([]jsonProject, 0, len(projects)),
		Tasks:      make([]jsonTask, 0, len(tasks)),
	}
	for _, p := range projects {
		doc.Projects = append(doc.Projects, jsonProject{
			ID: string(p.ID), Name: p.Name,
			CreatedAt: null(stamp(p.CreatedAt)), UpdatedAt: null(stamp(p.UpdatedAt)), ArchivedAt: null(stamp(p.ArchivedAt)),
		})
	}
	for _, t := range tasks {
		jt := jsonTask{
			ID: string(t.ID), Name: t.Name, Description: t.Description, Priority: t.Priority,
			ProjectID: null(string(t.ProjectID)), Start: null(t.Start.String()), Due: null(t.Due.String()),
			CreatedAt: null(stamp(t.CreatedAt)), UpdatedAt: null(stamp(t.UpdatedAt)),
			CompletedAt: null(stamp(t.CompletedAt)), ArchivedAt: null(stamp(t.ArchivedAt)),
			Repeat: null(t.Repeat.String()), SeriesID: null(string(t.SeriesID)),
		}
		if t.ProjectID != "" {
			jt.Project = null(names[t.ProjectID])
		}
		if t.DurationMin > 0 {
			d := t.DurationMin
			jt.DurationMin = &d
		}
		doc.Tasks = append(doc.Tasks, jt)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// stamp formats a timestamp as RFC 3339 in UTC, or "" when unset.
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// null turns "" into a JSON null.
func null(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
