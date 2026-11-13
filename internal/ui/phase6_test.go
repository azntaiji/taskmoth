package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/taskmoth/internal/store"
	"github.com/azntaiji/taskmoth/internal/task"
)

const eggsDescription = "I need to goto trader joe's to get eggs. Also get the rest of the stuff on the list.\n\nReach out to Hope if she needs anything that isn't on the list."

// eggsApp is the design store with "Get eggs" described as in the Single
// Task frame (29:7123) and selected in Today.
func eggsApp(t *testing.T) (*App, store.Store) {
	t.Helper()
	clk := &clock{}
	s := designStore(t, clk)
	eggs := findTask(t, s, "Get eggs")
	eggs.Description = eggsDescription
	if _, err := s.UpdateTask(context.Background(), eggs); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "j", "j", "j")
	if sel, _ := a.current.Selected(); sel.Name != "Get eggs" {
		t.Fatalf("selected %q", sel.Name)
	}
	return a, s
}

func clearInput(a *App) { press(a, "end", "ctrl+u") }

func TestTaskViewMatchesDesign(t *testing.T) {
	a, _ := eggsApp(t)
	press(a, "e")
	if a.taskView == nil {
		t.Fatal("e should open the Single Task view")
	}
	assertScreen(t, a, 80, 24)
	lines := screen(a)
	want := map[int]string{
		0:  "taskmoth ▸ Task",
		2:  "│ Get eggs",
		4:  "│ #groceries",
		6:  "│  P3    overdue   s:2026-11-09  ~15m", // badge padding + 2-cell gaps (29:7123); G6.1 adds duration
		8:  "│ Description",
		9:  "│ I need to goto trader joe's to get eggs. Also get the rest of the",
		12: "│ Reach out to Hope if she needs anything that isn't on the list.",
		22: " ↑ / ↓ scroll  space done  esc return to list  x archive  n name",
		23: " p priority  v project  r start date  t due date  y duration  d description",
	}
	for row, prefix := range want {
		if !strings.HasPrefix(lines[row], prefix) {
			t.Errorf("row %d = %q\n      want prefix %q", row, lines[row], prefix)
		}
	}
	golden(t, "task_view", a.View().Content)

	press(a, "esc")
	if a.taskView != nil {
		t.Fatal("esc should return to the list")
	}
	if sel, _ := a.current.Selected(); sel.Name != "Get eggs" {
		t.Errorf("selection lost: %q", sel.Name)
	}
}

func TestTaskViewDoneAndArchive(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "e", "space")
	if a.taskView != nil {
		t.Fatal("completing should return to the list (G6.13)")
	}
	if sel, _ := a.current.Selected(); sel.Name != "Get eggs" || !findTask(t, s, "Get eggs").Done() {
		t.Errorf("the list should keep the completed task selected, got %q", sel.Name)
	}
	press(a, "e")
	if !strings.Contains(strings.Join(screen(a), "\n"), "Completed 2026-11-10") {
		t.Error("completed tasks show their completion date (G6.1)")
	}
	press(a, "space")
	if a.taskView == nil {
		t.Error("reopening should stay in the task view")
	}
	if findTask(t, s, "Get eggs").Done() {
		t.Error("space should reopen")
	}
	press(a, "x")
	if c, ok := a.topDialog().(*confirmDialog); !ok || c.action != "archive" {
		t.Fatal("x should ask to archive (G6.4)")
	}
	press(a, "right", "enter")
	if a.taskView != nil {
		t.Error("archiving should return to the list")
	}
	if !findTask(t, s, "Get eggs").Archived() {
		t.Error("task not archived")
	}
}

func TestReadOnlyArchivedTask(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "x", "right", "enter", "g", "x", "e")
	if a.taskView == nil || !a.taskView.readOnly {
		t.Fatal("e in the Archive should open a read-only view (G6.7)")
	}
	if !strings.HasPrefix(screen(a)[22], " ↑ / ↓ scroll  u unarchive  x delete forever  esc return to list") {
		t.Errorf("hints = %q", screen(a)[22])
	}
	press(a, "n", "p", "space") // all ignored
	if a.topDialog() != nil || findTask(t, s, "Get eggs").Done() {
		t.Error("archived tasks are read-only")
	}
	press(a, "u")
	if a.taskView != nil || findTask(t, s, "Get eggs").Archived() {
		t.Error("u should unarchive and return to the list")
	}
}

func TestEditTextFields(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "e")

	// Name (40:16847): single line, no newline hint; empty is an error.
	press(a, "n")
	golden(t, "edit_name", a.View().Content)
	if strings.Contains(strings.Join(screen(a), "\n"), "newline") {
		t.Error("Edit Name has no newline (G2.1d, G6.2)")
	}
	clearInput(a)
	press(a, "enter")
	if !strings.Contains(strings.Join(screen(a), "\n"), "Enter a task name.") {
		t.Error("empty name should show an error in the dialog")
	}
	typeText(a, "Get eggs and milk")
	press(a, "enter")
	if findTask(t, s, "Get eggs and milk").ID == "" || a.topDialog() != nil {
		t.Fatal("name not saved")
	}

	// Description (15:5973): newline keys, then clearing (G6.2, G6.3).
	press(a, "d")
	golden(t, "edit_description", a.View().Content)
	if !strings.Contains(strings.Join(screen(a), "\n"), "ctrl+j newline") {
		t.Error("without enhanced keys the hint is ctrl+j")
	}
	a.Update(tea.KeyboardEnhancementsMsg{Flags: 1})
	if !strings.Contains(strings.Join(screen(a), "\n"), "shift + enter newline") {
		t.Error("with enhanced keys the hint is shift + enter")
	}
	press(a, "ctrl+j")
	typeText(a, "P.S.")
	press(a, "alt+enter")
	typeText(a, "Thanks")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs and milk").Description; got != eggsDescription+"\nP.S.\nThanks" {
		t.Errorf("description = %q", got)
	}
	press(a, "d")
	for range 200 {
		press(a, "backspace")
	}
	press(a, "enter")
	if got := findTask(t, s, "Get eggs and milk").Description; got != "" {
		t.Errorf("description should be cleared, got %q", got)
	}
	if !strings.Contains(strings.Join(screen(a), "\n"), "No description. Press d to add one.") {
		t.Error("empty description message")
	}
}

func TestEditDatesAndDuration(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "e")

	press(a, "r") // start (40:15310), opens with the current value
	golden(t, "edit_start", a.View().Content)
	if addDialogValue(a) != "2026-11-09" {
		t.Errorf("start dialog opened with %q", addDialogValue(a))
	}
	clearInput(a)
	typeText(a, "someday soon")
	press(a, "enter")
	if !strings.Contains(strings.Join(screen(a), "\n"), "Can't read that date.") {
		t.Error("bad date should show an error")
	}
	clearInput(a)
	typeText(a, "next monday")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").Start.String(); got != "2026-11-16" {
		t.Errorf("start = %s", got)
	}
	// Start (11-16) is now after due (11-09): saved, with a warning (G6.5).
	if !strings.Contains(strings.Join(screen(a), "\n"), "Start date is after the due date.") {
		t.Error("expected the start-after-due warning in the status bar (G6.10)")
	}

	press(a, "t") // due (40:15226)
	clearInput(a)
	typeText(a, "nov 20")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").Due.String(); got != "2026-11-20" {
		t.Errorf("due = %s", got)
	}
	press(a, "t")
	clearInput(a)
	typeText(a, "no date")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").Due; !got.IsZero() {
		t.Errorf("'no date' should clear the due date, got %s", got)
	}
	press(a, "r")
	clearInput(a)
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").Start; !got.IsZero() {
		t.Errorf("empty input should clear the start date, got %s", got)
	}

	press(a, "y") // duration (40:16639): opens with "15m", not "tomorrow" (G6.4)
	golden(t, "edit_duration", a.View().Content)
	if addDialogValue(a) != "15m" {
		t.Errorf("duration dialog opened with %q", addDialogValue(a))
	}
	clearInput(a)
	typeText(a, "2 days")
	press(a, "enter")
	if !strings.Contains(strings.Join(screen(a), "\n"), "Can't read that duration.") {
		t.Error("bad duration should show an error")
	}
	clearInput(a)
	typeText(a, "1h30m")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").DurationMin; got != 90 {
		t.Errorf("duration = %d", got)
	}
	press(a, "y")
	if addDialogValue(a) != "1h30m" {
		t.Errorf("duration text = %q", addDialogValue(a))
	}
	clearInput(a)
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").DurationMin; got != 0 {
		t.Errorf("duration should be cleared, got %d", got)
	}
}

// addDialogValue is the text in the open text-edit dialog.
func addDialogValue(a *App) string {
	if d, ok := a.topDialog().(*textEditDialog); ok {
		return d.input.Value()
	}
	return "<no text dialog>"
}

func TestEditPriority(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "e", "p")
	d, ok := a.topDialog().(*priorityDialog)
	if !ok || d.sel != 3 {
		t.Fatal("Edit Priority should open on the current priority (G6.9)")
	}
	golden(t, "edit_priority", a.View().Content)
	press(a, "left", "left", "left", "enter") // clamps at P1
	if got := findTask(t, s, "Get eggs").Priority; got != 1 {
		t.Errorf("priority = %d", got)
	}
	press(a, "p", "4") // a number key selects and saves
	if got := findTask(t, s, "Get eggs").Priority; got != 4 || a.topDialog() != nil {
		t.Errorf("priority = %d", got)
	}
	press(a, "p", "esc")
	if got := findTask(t, s, "Get eggs").Priority; got != 4 {
		t.Error("esc should cancel")
	}
}

func TestEditProject(t *testing.T) {
	a, s := eggsApp(t)
	ctx := context.Background()
	press(a, "e", "v")
	d, ok := a.topDialog().(*editProjectDialog)
	if !ok {
		t.Fatal("v should open Edit Project")
	}
	if d.options(a)[d.sel].Name != "groceries" {
		t.Errorf("should open on the current project, got %q", d.options(a)[d.sel].Name)
	}
	golden(t, "edit_project", a.View().Content)

	// "Inbox" at the top removes the project (G6.3).
	d.sel = 0
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").ProjectID; got != "" {
		t.Errorf("project should be cleared, got %q", got)
	}
	if !strings.Contains(strings.Join(screen(a), "\n"), "│ Inbox") {
		t.Error("an Inbox task's project line shows Inbox (G6.10)")
	}

	// n creates a project, returns selected, and enter moves the task (G6.8).
	press(a, "v", "n")
	typeText(a, "errands")
	press(a, "enter")
	d, ok = a.topDialog().(*editProjectDialog)
	if !ok || d.options(a)[d.sel].Name != "errands" {
		t.Fatal("should return to Edit Project with #errands selected")
	}
	press(a, "enter")
	e, _ := s.ProjectByName(ctx, "errands")
	if got := findTask(t, s, "Get eggs").ProjectID; got != e.ID {
		t.Error("task should move into #errands")
	}
}

func TestEditKeysFromLists(t *testing.T) {
	a, s := eggsApp(t)
	press(a, "y") // from the list, on the selected row (G6.6)
	if _, ok := a.topDialog().(*textEditDialog); !ok {
		t.Fatal("y in a list should open Edit Duration")
	}
	clearInput(a)
	typeText(a, "45m")
	press(a, "enter")
	if got := findTask(t, s, "Get eggs").DurationMin; got != 45 {
		t.Errorf("duration = %d", got)
	}
	if a.taskView != nil {
		t.Error("editing from a list stays in the list")
	}
	// Not in the Archive.
	press(a, "x", "right", "enter", "g", "x", "y", "n", "p")
	if a.topDialog() != nil {
		t.Error("edit keys don't apply in the Archive view")
	}
	// The `g t` chord still switches views rather than editing the due date.
	press(a, "g", "t")
	if a.topDialog() != nil {
		t.Error("g t must not open Edit Due date")
	}
}

func TestTaskViewScrollsLongDescription(t *testing.T) {
	clk := &clock{t: designNow}
	s := store.NewMemory(store.Options{Now: clk.Now})
	long := strings.Repeat("A long line of description text that wraps. ", 60)
	if _, err := s.CreateTask(context.Background(), task.Task{Name: "Long", Description: long, Due: date("2026-11-10")}); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "e")
	assertScreen(t, a, 80, 24)
	first := screen(a)[2]
	press(a, "pgdown")
	if screen(a)[2] == first {
		t.Error("pgdown should scroll the task view")
	}
	press(a, "end")
	if !strings.Contains(strings.Join(screen(a), "\n"), "wraps.") {
		t.Error("end should show the last lines")
	}
	press(a, "home")
	if screen(a)[2] != first {
		t.Error("home should return to the top")
	}
	assertScreen(t, a, 80, 24)
}

// Terminals often paste newlines as a bare \r, which moved the cursor to
// column 0 and broke the task view and Edit Description frames.
func TestDescriptionWithCarriageReturns(t *testing.T) {
	clk := &clock{t: designNow}
	s := store.NewMemory(store.Options{Now: clk.Now})
	para := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 6)
	stored := para + "\r" + para + "\r\n" + para
	if _, err := s.CreateTask(context.Background(), task.Task{Name: "Long", Description: stored, Due: date("2026-11-10")}); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	press(a, "e")
	assertScreen(t, a, 80, 24)
	if strings.Contains(a.View().Content, "\r") {
		t.Error("the task view drew a carriage return")
	}

	press(a, "d")
	clearInput(a)
	for range 5 {
		clearInput(a)
		press(a, "backspace")
	}
	a.Update(tea.PasteMsg{Content: para + "\r" + para})
	assertScreen(t, a, 80, 24)
	if strings.Contains(a.View().Content, "\r") {
		t.Error("Edit Description drew a carriage return")
	}
	press(a, "enter")
	if got := findTask(t, s, "Long").Description; got != strings.TrimSpace(para+"\n"+para) {
		t.Errorf("description = %q", got)
	}
}
