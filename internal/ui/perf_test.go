package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

// bigStore is a SQLite store with n tasks across 20 projects, about half
// due within the next weeks (Phase 8: 5,000-task performance check).
func bigStore(b testing.TB, clk *clock, n int) store.Store {
	b.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(b.TempDir(), store.FileName), store.Options{Now: clk.Now})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = s.Close() })
	var projects []task.ID
	for i := range 20 {
		p, err := s.CreateProject(ctx, fmt.Sprintf("project-%02d", i))
		if err != nil {
			b.Fatal(err)
		}
		projects = append(projects, p.ID)
	}
	words := []string{"call", "email", "write", "review", "buy", "fix", "plan", "read", "clean", "book"}
	for i := range n {
		tk := task.Task{
			Name:        fmt.Sprintf("%s item %d about %s", words[i%10], i, words[(i/10)%10]),
			Description: "Some notes for the task, long enough to search through.",
			Priority:    i%4 + 1,
			DurationMin: (i % 6) * 15,
		}
		if i%3 != 0 {
			tk.ProjectID = projects[i%20]
		}
		if i%2 == 0 {
			tk.Due = date("2026-11-10").AddDays(i%30 - 5)
		}
		if _, err := s.CreateTask(ctx, tk); err != nil {
			b.Fatal(err)
		}
	}
	return s
}

func BenchmarkReload5000(b *testing.B) {
	clk := &clock{t: designNow}
	a := newApp(b, bigStore(b, clk, 5000), clk, 120, 40)
	b.ResetTimer()
	for range b.N {
		run(a, a.load(false)) // what every edit triggers
	}
}

func BenchmarkRenderAndMove5000(b *testing.B) {
	clk := &clock{t: designNow}
	a := newApp(b, bigStore(b, clk, 5000), clk, 120, 40)
	press(a, "g", "a")
	b.ResetTimer()
	for i := range b.N {
		key := "j"
		if i%200 >= 100 {
			key = "k"
		}
		a.Update(keyText(key))
		_ = a.View()
	}
}

func BenchmarkSearchKeystroke5000(b *testing.B) {
	clk := &clock{t: designNow}
	a := newApp(b, bigStore(b, clk, 5000), clk, 120, 40)
	press(a, "/")
	b.ResetTimer()
	for i := range b.N {
		k := "e"
		if i%2 == 1 {
			k = "backspace"
		}
		_, cmd := a.Update(keyText(k))
		run(a, cmd)
		_ = a.View()
	}
}

// TestEveryScreenAtEverySize resizes through small, odd and large terminals
// with each kind of screen open and checks every frame is exactly the
// terminal's size (Phase 8: resize handling).
func TestEveryScreenAtEverySize(t *testing.T) {
	sizes := [][2]int{{80, 24}, {81, 25}, {97, 31}, {120, 40}, {200, 60}, {79, 24}, {80, 23}, {40, 10}, {1, 1}}
	screens := map[string][]string{
		"today": nil, "upcoming": {"g", "u"}, "inbox": {"g", "i"}, "someday": {"g", "s"},
		"search": {"/", "e"}, "search results": {"/", "e", "enter"},
		"task view": {"e"}, "add task": {"q"}, "edit description": {"e", "d"},
		"edit project": {"v"}, "goto project": {"g", "p"}, "shortcuts": {"?"}, "confirm": {"x"},
	}
	for name, keys := range screens {
		clk := &clock{}
		a := newApp(t, designStore(t, clk), clk, 80, 24)
		press(a, keys...)
		for _, sz := range sizes {
			a.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			lines := screen(a)
			if len(lines) != sz[1] {
				t.Errorf("%s at %dx%d: %d lines", name, sz[0], sz[1], len(lines))
				continue
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w != sz[0] {
					t.Errorf("%s at %dx%d: line %d is %d wide", name, sz[0], sz[1], i, w)
					break
				}
			}
		}
	}
}
