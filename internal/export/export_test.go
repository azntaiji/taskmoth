package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/taskmoth/internal/task"
)

var (
	now     = time.Date(2026, 11, 17, 9, 30, 0, 0, time.FixedZone("PST", -8*3600))
	created = time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	done    = time.Date(2026, 11, 12, 18, 15, 0, 0, time.UTC)
)

func sample() ([]task.Project, []task.Task) {
	projects := []task.Project{
		{ID: "p2", Name: "work", CreatedAt: created, UpdatedAt: created},
		{ID: "p1", Name: "chores", CreatedAt: created, UpdatedAt: created, ArchivedAt: done},
	}
	tasks := []task.Task{
		{ID: "t2", Name: "Ship export", Description: "CSV, then JSON\nwith \"quotes\", commas", ProjectID: "p2",
			Priority: 1, Due: task.Date{Year: 2026, Month: 11, Day: 20}, DurationMin: 90, CreatedAt: created, UpdatedAt: created,
			Repeat: task.Rule{Every: 1, Unit: task.Month, MonthDay: 20}, SeriesID: "t0"},
		{ID: "t1", Name: "Buy milk", Priority: 4, Start: task.Date{Year: 2026, Month: 11, Day: 11},
			CreatedAt: created, UpdatedAt: done, CompletedAt: done, ArchivedAt: done},
	}
	return projects, tasks
}

func TestCSV(t *testing.T) {
	projects, tasks := sample()
	var b bytes.Buffer
	if err := Write(&b, "csv", projects, tasks, now); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&b).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		CSVHeader,
		{"t1", "Buy milk", "", "", "4", "2026-11-11", "", "", "2026-11-10T08:00:00Z", "2026-11-12T18:15:00Z", "2026-11-12T18:15:00Z", "2026-11-12T18:15:00Z", "", ""},
		{"t2", "Ship export", "CSV, then JSON\nwith \"quotes\", commas", "work", "1", "", "2026-11-20", "90", "2026-11-10T08:00:00Z", "2026-11-10T08:00:00Z", "", "", "every 20th", "t0"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows:\n%q\nwant:\n%q", rows, want)
	}
}

func TestJSON(t *testing.T) {
	projects, tasks := sample()
	var b bytes.Buffer
	if err := Write(&b, "json", projects, tasks, now); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"version":     1.0,
		"exported_at": "2026-11-17T17:30:00Z",
		"projects": []any{
			map[string]any{"id": "p1", "name": "chores", "created_at": "2026-11-10T08:00:00Z", "updated_at": "2026-11-10T08:00:00Z", "archived_at": "2026-11-12T18:15:00Z"},
			map[string]any{"id": "p2", "name": "work", "created_at": "2026-11-10T08:00:00Z", "updated_at": "2026-11-10T08:00:00Z", "archived_at": nil},
		},
		"tasks": []any{
			map[string]any{"id": "t1", "name": "Buy milk", "description": "", "project_id": nil, "project": nil,
				"priority": 4.0, "start": "2026-11-11", "due": nil, "duration_min": nil,
				"created_at": "2026-11-10T08:00:00Z", "updated_at": "2026-11-12T18:15:00Z",
				"completed_at": "2026-11-12T18:15:00Z", "archived_at": "2026-11-12T18:15:00Z", "repeat": nil, "series_id": nil},
			map[string]any{"id": "t2", "name": "Ship export", "description": "CSV, then JSON\nwith \"quotes\", commas",
				"project_id": "p2", "project": "work", "priority": 1.0, "start": nil, "due": "2026-11-20", "duration_min": 90.0,
				"created_at": "2026-11-10T08:00:00Z", "updated_at": "2026-11-10T08:00:00Z",
				"completed_at": nil, "archived_at": nil, "repeat": "every 20th", "series_id": "t0"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestEmptyJSONHasEmptyArrays(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, "json", nil, nil, now); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"projects": []`, `"tasks": []`} {
		if !strings.Contains(b.String(), s) {
			t.Errorf("missing %s in:\n%s", s, b.String())
		}
	}
}

func TestWriteDoesNotReorderInput(t *testing.T) {
	projects, tasks := sample()
	if err := Write(&bytes.Buffer{}, "csv", projects, tasks, now); err != nil {
		t.Fatal(err)
	}
	if tasks[0].ID != "t2" || projects[0].ID != "p2" {
		t.Error("Write sorted the caller's slices")
	}
}

func TestParseFormat(t *testing.T) {
	for in, want := range map[string]string{"csv": "csv", " JSON ": "json"} {
		if got, err := ParseFormat(in); err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseFormat("xml"); err == nil || !strings.Contains(err.Error(), "csv or json") {
		t.Errorf("ParseFormat(xml) error = %v", err)
	}
	if err := Write(&bytes.Buffer{}, "xml", nil, nil, now); err == nil {
		t.Error("Write accepted xml")
	}
}
