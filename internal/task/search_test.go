package task

import (
	"reflect"
	"testing"
	"time"
)

func TestParseQuery(t *testing.T) {
	cases := []struct {
		in   string
		want Query
	}{
		{"", Query{}},
		{"Buy  MILK", Query{Words: []string{"buy", "milk"}}},
		{"#Work !p1 report", Query{Words: []string{"report"}, Project: "work", Priority: 1}},
		{"#inbox milk", Query{Words: []string{"milk"}, Inbox: true}},
		{"# !p !", Query{}}, // half typed: ignored
		{"!p5 !px", Query{Words: []string{"!p5", "!px"}}}, // not priorities: plain words
	}
	for _, c := range cases {
		if got := ParseQuery(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseQuery(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestQueryMatch(t *testing.T) {
	milk := Task{Name: "Buy oat milk", Description: "From the corner shop", Priority: P2, ProjectID: "g"}
	inbox := Task{Name: "Call the bank", Priority: P4}
	cases := []struct {
		q             string
		t             Task
		project       string
		match, inName bool
	}{
		{"", milk, "groceries", true, true},
		{"milk buy", milk, "groceries", true, true},     // any order
		{"MILK", milk, "groceries", true, true},         // case-insensitive
		{"milk corner", milk, "groceries", true, false}, // one word only in the description
		{"grocer", milk, "groceries", true, false},      // project name
		{"milk eggs", milk, "groceries", false, false},  // every word must match
		{"#gro milk", milk, "groceries", true, true},    // project prefix while typing
		{"#work milk", milk, "groceries", false, false},
		{"#gro", inbox, "", false, false},
		{"#inbox", inbox, "", true, true},
		{"#inbox", milk, "groceries", false, false},
		{"!p2", milk, "groceries", true, true},
		{"!p1 milk", milk, "groceries", false, false},
	}
	for _, c := range cases {
		m, n := ParseQuery(c.q).Match(c.t, c.project)
		if m != c.match || n != c.inName {
			t.Errorf("%q vs %q: match=%v nameHit=%v, want %v %v", c.q, c.t.Name, m, n, c.match, c.inName)
		}
	}
}

func TestSortSearch(t *testing.T) {
	done := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	rows := []Task{
		{ID: "1", Name: "Notes about milk", Priority: P1},                 // not a name hit; P1 ties go A–Z
		{ID: "2", Name: "Buy milk", Priority: P3},                         // name hit
		{ID: "3", Name: "Milk the cows", Priority: P2, CompletedAt: done}, // completed: last
		{ID: "4", Name: "Almond milk", Priority: P3},                      // name hit, A–Z before "Buy"
		{ID: "5", Name: "Errand", Priority: P1},                           // not a name hit
	}
	inName := map[ID]bool{"2": true, "3": true, "4": true}
	SortSearch(rows, func(t Task) bool { return inName[t.ID] })
	var got []ID
	for _, r := range rows {
		got = append(got, r.ID)
	}
	if want := []ID{"4", "2", "5", "1", "3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}
