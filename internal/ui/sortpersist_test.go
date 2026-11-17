package ui

import (
	"context"
	"testing"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

// TestEverySortSurvivesRestart checks G3.3b end to end on SQLite: each view,
// and a project list, keeps its own sort while switching views and after
// the app is restarted on the same database.
func TestEverySortSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: designNow}
	dir := t.TempDir()
	open := func() *store.SQLite {
		s, err := store.OpenDir(ctx, dir, store.Options{Now: clk.Now})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := open()
	if _, err := s.CreateProject(ctx, "work"); err != nil {
		t.Fatal(err)
	}

	chords := map[view.Kind]string{
		view.Today: "t", view.Inbox: "i", view.Upcoming: "u", view.Overdue: "o",
		view.All: "a", view.Someday: "s", view.Archive: "x",
	}
	a := newApp(t, s, clk, 80, 24)
	want := map[view.Kind]task.SortKey{}
	n := 1
	for k, key := range chords {
		press(a, "g", key)
		for range 1 + n%(len(a.current.Spec.Cycle)-1) { // never back to the default
			press(a, "s")
		}
		if a.current.Sort == a.current.Spec.Cycle[0] {
			t.Fatalf("%s: still on the default sort", a.current.Spec.Key)
		}
		want[k] = a.current.Sort
		n++
	}
	openProject(t, a, "work")
	press(a, "s", "s")
	wantProject := a.current.Sort

	// Within the session: each view kept its own choice.
	for k, key := range chords {
		press(a, "g", key)
		if a.current.Sort != want[k] {
			t.Errorf("%s: sort %v after switching views, want %v", a.current.Spec.Key, a.current.Sort, want[k])
		}
	}

	// Across a restart.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = open()
	defer func() { _ = s.Close() }()
	b := newApp(t, s, clk, 80, 24)
	for k, key := range chords {
		press(b, "g", key)
		if b.current.Sort != want[k] {
			t.Errorf("%s: sort %v after restart, want %v", b.current.Spec.Key, b.current.Sort, want[k])
		}
	}
	openProject(t, b, "work")
	if b.current.Sort != wantProject {
		t.Errorf("#work: sort %v after restart, want %v", b.current.Sort, wantProject)
	}
}
