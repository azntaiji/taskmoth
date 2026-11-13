package main

// Developer commands (--seed, --list). --list is a temporary way to inspect
// the database until the list views arrive in Phase 3.

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/azntaiji/taskmoth/internal/seed"
	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

// runDev opens the store and runs --seed and/or --list.
func runDev(o options, dataDir string) (err error) {
	ctx := context.Background()
	s, err := store.OpenDir(ctx, dataDir, store.Options{})
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() {
		if cerr := s.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing database: %w", cerr)
		}
	}()
	if o.seed {
		if err := runSeed(ctx, os.Stdout, s, dataDir, time.Now()); err != nil {
			return err
		}
	}
	if o.list != "" {
		return runList(ctx, os.Stdout, s, o.list, time.Now())
	}
	return nil
}

func runSeed(ctx context.Context, w io.Writer, s store.Store, dataDir string, now time.Time) error {
	p, t, err := seed.Run(ctx, s, task.DateOf(now))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Seeded %d projects and %d tasks into %s\n", p, t, dataDir)
	return err
}

// listViews are the view names --list accepts (plus "project:<name>").
var listViews = []string{"today", "upcoming", "overdue", "inbox", "completed", "trash"}

func runList(ctx context.Context, w io.Writer, s store.Store, view string, now time.Time) error {
	today := task.DateOf(now)
	all, err := s.ListTasks(ctx)
	if err != nil {
		return err
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return err
	}
	projectName := map[task.ID]string{}
	for _, p := range projects {
		projectName[p.ID] = p.Name
	}

	var keep func(task.Task) bool
	switch v := strings.ToLower(view); {
	case v == "today":
		keep = func(t task.Task) bool { return task.InToday(t, today) }
	case v == "upcoming":
		keep = func(t task.Task) bool { return task.InUpcoming(t, today) }
	case v == "overdue":
		keep = func(t task.Task) bool { return task.InOverdue(t, today) }
	case v == "inbox":
		keep = task.InInbox
	case v == "completed":
		keep = task.InCompleted
	case v == "trash":
		keep = task.InTrash
	case strings.HasPrefix(v, "project:"):
		p, err := s.ProjectByName(ctx, strings.TrimPrefix(v, "project:"))
		if err != nil {
			return fmt.Errorf("project %q: %w", strings.TrimPrefix(v, "project:"), err)
		}
		keep = func(t task.Task) bool { return task.InProject(t, p.ID) }
	default:
		return fmt.Errorf("unknown view %q; use one of %s, or project:<name>", view, strings.Join(listViews, ", "))
	}

	tasks := task.Filter(all, keep)
	task.Sort(tasks, task.SortPriority)
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s): %d tasks\n", view, today, len(tasks))
	for _, t := range tasks {
		box := "[ ]"
		if t.Done() {
			box = "[x]"
		}
		var meta []string
		if n := projectName[t.ProjectID]; n != "" {
			meta = append(meta, "#"+n)
		}
		if t.DurationMin > 0 {
			meta = append(meta, fmt.Sprintf("~%dm", t.DurationMin))
		}
		if !t.Start.IsZero() {
			meta = append(meta, "s:"+t.Start.String())
		}
		due := ""
		switch task.DueStateOf(t, today) {
		case task.DueToday:
			due = "today"
		case task.DueOverdue:
			due = "overdue " + t.Due.String()
		case task.DueUpcoming:
			due = t.Due.String()
		}
		fmt.Fprintf(&b, "  %s P%d %-45s %-40s %s\n", box, t.Priority, t.Name, strings.Join(meta, "  "), due)
	}
	_, err = io.WriteString(w, b.String())
	return err
}
