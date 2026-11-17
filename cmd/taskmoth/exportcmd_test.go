package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azntaiji/taskmoth/internal/export"
	"github.com/azntaiji/taskmoth/internal/seed"
	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

func TestCheckExportFlags(t *testing.T) {
	cases := []struct {
		name    string
		o       options
		want    string
		wantErr bool
	}{
		{"none", options{}, "", false},
		{"csv", options{export: "CSV"}, "csv", false},
		{"json to file", options{export: "json", output: "x.json"}, "json", false},
		{"unknown format", options{export: "xml"}, "", true},
		{"output alone", options{output: "x.csv"}, "", true},
		{"with list", options{export: "csv", list: "today"}, "", true},
	}
	for _, c := range cases {
		err := checkExportFlags(&c.o)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: error %v, wantErr %v", c.name, err, c.wantErr)
			continue
		}
		if err == nil && c.o.export != c.want {
			t.Errorf("%s: format %q, want %q", c.name, c.o.export, c.want)
		}
	}
}

// TestExportStore exports the sample data and checks every task is there,
// including completed and archived ones (G10.2).
func TestExportStore(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory(store.Options{})
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	if _, _, err := seed.Run(ctx, s, task.DateOf(now)); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListTasks(ctx)
	projects, _ := s.ListProjects(ctx)

	var b bytes.Buffer
	if err := exportStore(ctx, &b, s, "csv", now); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&b).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(all)+1 {
		t.Errorf("CSV has %d rows, want %d tasks + header", len(rows), len(all))
	}

	b.Reset()
	if err := exportStore(ctx, &b, s, "json", now); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Projects []any `json:"projects"`
		Tasks    []any `json:"tasks"`
	}
	if err := json.Unmarshal(b.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Tasks) != len(all) || len(doc.Projects) != len(projects) {
		t.Errorf("JSON has %d tasks and %d projects, want %d and %d", len(doc.Tasks), len(doc.Projects), len(all), len(projects))
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.csv")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	// A failed write leaves the old file alone and no temporary file behind.
	boom := errors.New("boom")
	if err := writeFileAtomic(path, func(w io.Writer) error {
		_, _ = io.WriteString(w, "partial")
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("error %v, want boom", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Errorf("file is %q after a failed write, want \"old\"", b)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("%d files in the directory, want 1", len(entries))
	}

	if err := writeFileAtomic(path, func(w io.Writer) error {
		return export.Write(w, "csv", nil, nil, time.Time{})
	}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) == "old" {
		t.Error("file wasn't replaced")
	}
}
