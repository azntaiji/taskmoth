package seed

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

func names(ts []task.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Name
	}
	slices.Sort(out)
	return out
}

func TestSeedMatchesDesigns(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory(store.Options{})
	np, nt, err := Run(ctx, s, DesignToday)
	if err != nil {
		t.Fatal(err)
	}
	if np != 8 || nt != 20 {
		t.Errorf("seeded %d projects, %d tasks", np, nt)
	}
	all, _ := s.ListTasks(ctx)

	// Today on the design date: everything due/overdue/started, open only.
	today := names(task.Filter(all, func(x task.Task) bool { return task.InToday(x, DesignToday) }))
	want := []string{
		"File insurance reimbursements", "Get eggs", "Get oliver a gift card", "Pack Azalea's books in backpack",
		"Pay credit card balances", "Practice meditating", "Shave, clean CPAP, cut nails",
	}
	if !slices.Equal(today, want) {
		t.Errorf("Today = %v\nwant    %v", today, want)
	}

	inbox := task.Filter(all, task.InInbox)
	if len(inbox) != 6 {
		t.Errorf("Inbox has %d tasks, want 6 (design 40:18095)", len(inbox))
	}
	done := names(task.Filter(all, task.InCompleted))
	if !slices.Equal(done, []string{"Finish creating task management TUI app. Then put it into production", "Watch Tales of the Loop"}) {
		t.Errorf("Completed = %v", done)
	}
	for _, x := range all {
		if x.Name == "Get eggs" && x.Description == "" {
			t.Error("Get eggs should carry its description from the single task view")
		}
	}
}

func TestSeedShiftsDates(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory(store.Options{})
	later := DesignToday.AddDays(30)
	if _, _, err := Run(ctx, s, later); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListTasks(ctx)
	for _, x := range all {
		if x.Name == "Water plants" && x.Due != later.AddDays(8) {
			t.Errorf("Water plants due %s, want %s", x.Due, later.AddDays(8))
		}
	}
}

func TestSeedRefusesNonEmptyStore(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory(store.Options{})
	if _, err := s.CreateProject(ctx, "existing"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Run(ctx, s, DesignToday); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("err = %v, want ErrNotEmpty", err)
	}
}
