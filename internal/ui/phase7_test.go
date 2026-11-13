package ui

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/azntaiji/taskmoth/internal/task"
	"github.com/azntaiji/taskmoth/internal/ui/theme"
	"github.com/azntaiji/taskmoth/internal/ui/view"
)

func rowNames(a *App) []string {
	var out []string
	for _, r := range a.current.Rows() {
		out = append(out, r.Name)
	}
	return out
}

// TestSearchMatchesDesign: Search (Figma 40:16937) replaces the status bar
// with the query input and lists matches as standard rows.
func TestSearchMatchesDesign(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "/")
	assertScreen(t, a, 80, 24)
	lines := screen(a)
	if !strings.HasPrefix(lines[0], "taskmoth ▸ Search") {
		t.Errorf("header = %q", lines[0])
	}
	if !strings.HasPrefix(lines[21], "  Search   >  search for a task...") ||
		!strings.HasSuffix(lines[21], "enter results  esc exit search ") {
		t.Errorf("search bar = %q", lines[21])
	}
	if strings.Contains(lines[22], "s sort") {
		t.Error("Search has no sort (G7.1d)")
	}
	golden(t, "search", a.View().Content)

	typeText(a, "the")
	if got := rowNames(a); len(got) != 3 || got[0] != "Season a letter to the future" {
		t.Errorf("results for \"the\" = %v", got)
	}
	if !a.current.Rows()[2].Done() {
		t.Error("completed matches come last, dimmed (G7.1a)")
	}
	golden(t, "search_results", a.View().Content)
}

func TestSearchFilters(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "/")
	typeText(a, "#groc")
	if got := rowNames(a); !slices.Equal(got, []string{"Get eggs"}) {
		t.Errorf("#groc = %v", got)
	}
	press(a, "ctrl+u")
	typeText(a, "!p1")
	if got := rowNames(a); !slices.Equal(got, []string{"File insurance reimbursements"}) {
		t.Errorf("!p1 = %v", got)
	}
	press(a, "ctrl+u")
	typeText(a, "#inbox sable")
	if got := rowNames(a); !slices.Equal(got, []string{"Sable"}) {
		t.Errorf("#inbox sable = %v", got)
	}
	press(a, "ctrl+u")
	typeText(a, "zzz")
	if !strings.Contains(strings.Join(screen(a), "\n"), "No matching tasks.") {
		t.Error("no-match message missing")
	}
}

// TestSearchKeys walks G7.2a and G7.2b.
func TestSearchKeys(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "g", "i", "/")
	typeText(a, "x e q s ?")
	if a.topDialog() != nil || a.taskView != nil {
		t.Fatal("letters must type while the input has focus")
	}
	press(a, "ctrl+u")
	typeText(a, "p")
	first := selectedName(a)
	press(a, "down")
	if selectedName(a) == first {
		t.Error("down should move the selection while typing")
	}

	press(a, "enter") // focus the results
	if a.search.input.Focused() {
		t.Fatal("enter should focus the results")
	}
	if bar := screen(a)[21]; !strings.HasSuffix(bar, "esc edit query ") || strings.Contains(bar, "exit search") {
		t.Errorf("results-focused bar = %q (G7.4)", bar)
	}
	press(a, "x")
	if c, ok := a.topDialog().(*confirmDialog); !ok || c.action != "archive" {
		t.Fatal("list keys work in the results")
	}
	press(a, "esc", "s")
	if a.current.Sort != a.current.Spec.Cycle[0] {
		t.Error("s does nothing in Search")
	}
	press(a, "e", "esc")
	if !a.searching() || a.search.input.Focused() {
		t.Error("returning from a task keeps the results focused")
	}
	press(a, "/")
	if !a.search.input.Focused() {
		t.Error("/ refocuses the input")
	}
	press(a, "enter", "esc")
	if !a.search.input.Focused() || !a.searching() {
		t.Error("esc in the results goes back to the input")
	}
	press(a, "esc")
	if a.searching() || a.current.Spec.Kind != view.Inbox {
		t.Error("esc in the input returns to the previous view")
	}
	press(a, "/")
	if a.search.input.Value() != "p" {
		t.Errorf("query = %q, want it kept (G7.2b)", a.search.input.Value())
	}
}

// TestShortcutsDialog: `?` opens Figma 40:15079 from lists, Search results
// and the task view; esc or ? closes it (G7.3).
func TestShortcutsDialog(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "?")
	if _, ok := a.topDialog().(*shortcutsDialog); !ok {
		t.Fatal("? should open Keyboard Shortcuts")
	}
	assertScreen(t, a, 80, 24)
	text := strings.Join(screen(a), "\n")
	for _, want := range []string{"Keyboard Shortcuts", "Goto Lists", "g x archive", "Within Lists", "j / k select task",
		"Selected Tasks", "x archive", "d edit description", "General", "ctrl+q quit", "esc close"} {
		if !strings.Contains(text, want) {
			t.Errorf("dialog lacks %q", want)
		}
	}
	golden(t, "shortcuts", a.View().Content)
	press(a, "j", "q")
	if _, ok := a.topDialog().(*shortcutsDialog); !ok {
		t.Error("other keys are ignored")
	}
	press(a, "?")
	if a.topDialog() != nil {
		t.Error("? closes it")
	}
	press(a, "e", "?")
	if _, ok := a.topDialog().(*shortcutsDialog); !ok {
		t.Error("? works in the task view")
	}
	press(a, "esc")
	if a.topDialog() != nil || a.taskView == nil {
		t.Error("esc closes only the dialog")
	}
}

// TestHintsMatchKeyMap: every hint row comes from the key map, every key in
// the map does something, and no key is bound twice.
func TestHintsMatchKeyMap(t *testing.T) {
	for _, th := range []theme.Theme{theme.New(false), theme.New(true)} {
		for k := view.Today; k <= view.Search; k++ {
			for _, row := range listHints(th.G, k) { // panics on an unbound key
				if len(row) == 0 {
					t.Errorf("view %d has an empty hint row", k)
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, g := range []keyGroup{listGroup, taskGroup, generalGroup} {
		for _, b := range g.bindings {
			for _, k := range b.keys {
				if seen[k] {
					t.Errorf("%q is bound twice", k)
				}
				seen[k] = true
			}
			if b.act == nil && !slices.Contains(b.keys, "esc") && !slices.Contains(b.keys, "ctrl+q") {
				t.Errorf("%v has no action", b.keys)
			}
		}
	}
	for _, f := range editFields {
		if _, ok := lookup(f.key, taskGroup); !ok {
			t.Errorf("edit key %q missing from Selected Tasks", f.key)
		}
	}
	for k := range fixedViews {
		if _, ok := lookup(k, gotoGroup); !ok {
			t.Errorf("g %s missing from Goto Lists", k)
		}
	}
}

// G7.5: a project's own list leaves #project off the second line.
func TestProjectRowsOmitProjectName(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	openProject(t, a, "oneofftasks")
	body := strings.Join(screen(a)[1:21], "\n")
	if strings.Contains(body, "#oneofftasks") {
		t.Error("project rows repeat the project name")
	}
	if !strings.Contains(body, "~2h  s:") {
		t.Error("the rest of the second line should stay")
	}
	press(a, "g", "a")
	if !strings.Contains(strings.Join(screen(a), "\n"), "#oneofftasks") {
		t.Error("other views still show the project")
	}
}

// G7.7: from a project's list, Add Task's help names that project.
func TestAddTaskHelpNamesDefaultProject(t *testing.T) {
	clk := &clock{}
	a := newApp(t, designStore(t, clk), clk, 80, 24)
	press(a, "q")
	if text := strings.Join(screen(a), " "); !strings.Contains(text, "sent to the Inbox") {
		t.Error("from Today, tasks go to the Inbox")
	}
	press(a, "esc")
	openProject(t, a, "groceries")
	press(a, "q")
	text := strings.Join(screen(a), " ")
	if strings.Contains(text, "Inbox") || !strings.Contains(text, "#groceries.") {
		t.Errorf("help should name #groceries:\n%s", strings.Join(screen(a), "\n"))
	}
}

// G7.6: Someday (g s) lists open tasks with no due date.
func TestSomedayView(t *testing.T) {
	clk := &clock{}
	s := designStore(t, clk)
	a := newApp(t, s, clk, 80, 24)
	press(a, "g", "s")
	assertScreen(t, a, 80, 24)
	if !strings.HasPrefix(screen(a)[0], "taskmoth ▸ Someday") {
		t.Errorf("header = %q", screen(a)[0])
	}
	if len(a.current.Rows()) == 0 {
		t.Fatal("the design data has undated tasks")
	}
	for _, r := range a.current.Rows() {
		if !r.Due.IsZero() || r.Archived() {
			t.Errorf("%q doesn't belong in Someday", r.Name)
		}
	}
	if !strings.Contains(screen(a)[21], "s:priority") {
		t.Error("Someday sorts like the other views")
	}
}

// G7.9: empty Someday and project lists show their own note and the moth.
func TestEmptySomedayAndProjectNotes(t *testing.T) {
	clk := &clock{t: designNow}
	s := projectStore(t, clk) // every task has a due date, so Someday is empty
	if _, err := s.CreateProject(context.Background(), "chores"); err != nil {
		t.Fatal(err)
	}
	a := newApp(t, s, clk, 80, 24)
	today := task.DateOf(designNow)

	press(a, "g", "s")
	assertScreen(t, a, 80, 24)
	if !strings.Contains(screen(a)[4], view.SomedayNote(today)) || !strings.Contains(screen(a)[6], view.MothArt[0]) {
		t.Errorf("empty Someday:\n%s", strings.Join(screen(a)[:12], "\n"))
	}

	openProject(t, a, "chores")
	assertScreen(t, a, 80, 24)
	if !strings.Contains(screen(a)[2], "#chores has no open tasks.") ||
		!strings.Contains(screen(a)[4], view.ProjectNote("chores", today)) || !strings.Contains(screen(a)[6], view.MothArt[0]) {
		t.Errorf("empty project:\n%s", strings.Join(screen(a)[:12], "\n"))
	}
	if view.ProjectNote("chores", today) == view.SomedayNote(today) || view.SomedayNote(today) == view.InboxNote(today) {
		t.Error("each list has its own note")
	}
}
