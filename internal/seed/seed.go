// Package seed loads the sample projects and tasks shown in the Figma
// designs, for development and demos (taskmoth --seed).
package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

// DesignToday is the date the designs were drawn for ("tues · 2026-11-10").
// Seeded dates keep their distance from it, so a seed run on any day looks
// like the designs did on theirs.
var DesignToday = task.Date{Year: 2026, Month: 11, Day: 10}

// ErrNotEmpty is returned when seeding a store that already has data.
var ErrNotEmpty = errors.New("store already has tasks or projects; seed only into an empty data directory")

// Projects are the projects listed in the Goto Project dialog (40:14667).
var Projects = []string{"oneofftasks", "reminders", "groceries", "parenting", "work", "homelab-tech", "hobbies-games", "movies-tv"}

type sample struct {
	name, project string
	priority      int
	start, due    int // days from DesignToday; noDate when unset
	minutes       int
	done          bool
	description   string
}

const noDate = 1 << 30

// samples are the tasks visible in the Today, Inbox, project list, search and
// single-task frames.
var samples = []sample{
	// Today (32:9525)
	{"File insurance reimbursements", "oneofftasks", 1, 0, 0, 120, false, ""},
	{"Practice meditating", "reminders", 2, noDate, 0, 30, false, ""},
	{"Pack Azalea's books in backpack", "parenting", 2, noDate, 0, 60, false, ""},
	{"Get eggs", "groceries", 3, -1, -1, 15, false,
		"I need to goto trader joe's to get eggs. Also get the rest of the stuff on the list.\n\nReach out to Hope if she needs anything that isn't on the list."},
	{"Finish creating task management TUI app. Then put it into production", "homelab-tech", 3, -3, -1, 210, true, ""},
	{"Watch Tales of the Loop", "movies-tv", 4, 0, 0, 45, true, ""},
	// Inbox (40:18095)
	{"Try Godot exercise", "", 4, noDate, noDate, 0, false, ""},
	{"Citizen sleeper", "", 4, noDate, noDate, 0, false, ""},
	{"Sable", "", 4, noDate, noDate, 0, false, ""},
	{"Season a letter to the future", "", 4, noDate, noDate, 0, false, ""},
	{"Pacific drive", "", 4, noDate, noDate, 0, false, ""},
	{"More graphic novels", "", 4, noDate, noDate, 0, false, ""},
	// #oneofftasks (40:17686)
	{"Pay credit card balances", "oneofftasks", 1, 0, 0, 30, false, ""},
	{"Move some PFML money to the taxes vault?", "oneofftasks", 2, noDate, 10, 15, false, ""},
	{"Water plants", "oneofftasks", 2, noDate, 8, 5, false, ""},
	{"Shave, clean CPAP, cut nails", "oneofftasks", 2, -3, -3, 60, false, ""},
	{"Put out the garbage", "oneofftasks", 3, noDate, 5, 15, false, ""},
	{"Get oliver a gift card", "oneofftasks", 4, 0, 4, 45, false, ""},
	// Search (40:16937)
	{"Prep for thanksgiving", "oneofftasks", 2, 4, 5, 60, false, ""},
	{"Go see wildwood", "movies-tv", 3, noDate, 12, 120, false, ""},
}

// Run fills an empty store with the sample data, with dates shifted so that
// DesignToday maps to today.
func Run(ctx context.Context, s store.Store, today task.Date) (projects, tasks int, err error) {
	ts, err := s.ListTasks(ctx)
	if err != nil {
		return 0, 0, err
	}
	ps, err := s.ListProjects(ctx)
	if err != nil {
		return 0, 0, err
	}
	if len(ts) > 0 || len(ps) > 0 {
		return 0, 0, ErrNotEmpty
	}

	ids := map[string]task.ID{}
	for _, name := range Projects {
		p, err := s.CreateProject(ctx, name)
		if err != nil {
			return 0, 0, fmt.Errorf("project %s: %w", name, err)
		}
		ids[name] = p.ID
	}

	shift := func(days int) task.Date {
		if days == noDate {
			return task.Date{}
		}
		return today.AddDays(days)
	}
	for _, x := range samples {
		t, err := s.CreateTask(ctx, task.Task{
			Name: x.name, Description: x.description, ProjectID: ids[x.project], Priority: x.priority,
			Start: shift(x.start), Due: shift(x.due), DurationMin: x.minutes,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("task %q: %w", x.name, err)
		}
		if x.done {
			if _, err := s.SetDone(ctx, t.ID, true); err != nil {
				return 0, 0, err
			}
		}
	}
	return len(Projects), len(samples), nil
}
