package view

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/azntaiji/taskmoth/internal/task"
)

// inboxNotes are the friendly lines shown under "Inbox is empty." (G6.12).
var inboxNotes = []string{
	"Nothing to sort. The moth approves.",
	"Inbox empty. The moth has nothing to flutter at.",
	"No tasks here. Just a moth and its thoughts.",
	"All sorted. The moth is off looking for a lamp.",
	"Empty inbox. Even the moths have left.",
	"Nothing to organize. The moth ate the last one.",
	"Inbox zero. A moth could nap in here.",
	"Clear as a porch light on a summer night.",
	"No tasks. The moth checked your sweaters too.",
	"Nothing here but a little moth dust.",
	"All tidy. The moth is drawn to your efficiency.",
	"Empty. The moth suggests finding a nice lamp.",
	"The moth circled twice and found nothing.",
	"Inbox clear. Mothing to see here.",
	"No loose ends. Not even a moth-eaten one.",
	"Inbox zero. Go stretch your legs.",
	"Every task has a home. So does the moth.",
	"Clean slate. What's next?",
	"All caught up. Drink some water.",
	"Quiet in here. Savour it.",
}

// somedayNotes are shown under an empty Someday list (G7.9).
var somedayNotes = []string{
	"Everything has a due date. Very decisive.",
	"No someday tasks. The moth lives in the now.",
	"Nothing undated. The moth admires your planning.",
	"Someday is empty. Today is looking good.",
	"Every task has a date. The moth checked twice.",
	"No loose ends floating around like moths.",
	"Nothing for someday. Maybe take a nap today?",
	"All scheduled. The moth has nothing to wonder about.",
	"Someday has flown off like a moth at dawn.",
	"No vague plans here. Just a moth and a lamp.",
	"Nothing undated. Your calendar thanks you.",
	"The moth looked for someday and found today.",
	"Empty. Every task knows when it's due.",
	"No drifting tasks. The moth approves.",
	"Someday? Not today. Not any day, apparently.",
	"Nothing waiting for a someday. Nice.",
	"Every task is pinned down. Like a moth collection, but kinder.",
	"No undated tasks. Very tidy wings.",
	"All dated. The moth is mildly impressed.",
	"Nothing here. Go enjoy a someday-free day.",
}

// projectNotes are shown under an empty project list (G7.9); %s is the
// project's name.
var projectNotes = []string{
	"#%s is all clear. The moth approves.",
	"Nothing left in #%s. Time to flutter elsewhere.",
	"#%s: empty. The moth checked under the lamp.",
	"All done in #%s. Rest your wings.",
	"#%s has nothing to do. Enviable.",
	"No open tasks in #%s. Moth-ivation achieved.",
	"#%s is quiet. Just a moth and its thoughts.",
	"Empty project, clear mind.",
	"#%s is tidy. Not a moth-eaten task in sight.",
	"Nothing open in #%s. Go find a nice lamp.",
	"#%s: done and dusted.",
	"The moth circled #%s twice and found nothing.",
	"All clear in #%s. Put the kettle on.",
	"#%s is empty. Mothing to see here.",
	"No tasks in #%s. The moth is doing a little dance.",
	"#%s: handled.",
	"Nothing left in #%s. Future you says thanks.",
	"#%s has flown the coop. Or the lamp.",
	"Empty. The moth suggests a new project, or a nap.",
	"#%s is clear. Well earned.",
}

// todayDoneNotes replace Today's status bar summary once every task in it
// is done (G6.14). They must fit the status bar at 80 columns.
var todayDoneNotes = []string{
	"All done! The moth is proud of you.",
	"Today: handled. Go find a nice lamp.",
	"Every box ticked. Moth-ivation: 100%.",
	"Done! Time to flutter off for the day.",
	"All clear. The moth is doing a little dance.",
	"Finished. Lights out, moths in.",
	"That's a wrap. Not a moth-eaten one.",
	"All done. The moth salutes you.",
	"Nothing left. Rest your wings.",
	"Done for today. Head for the porch light.",
	"Every task done. You're the moth's knees.",
	"All done. The moth would bring you a sweater.",
	"Today's list: moth-balled.",
	"You finished the lot. Very moth-ure.",
	"Done! Moth-er would be proud.",
	"All done. Go enjoy the evening.",
	"That's everything. Put the kettle on.",
	"Done and dusted. Time to rest.",
	"All checked off. Future you says thanks.",
	"Clean sweep today. Take a bow.",
}

// MothArt is drawn under the empty Inbox's note (G6.15). It is plain
// ASCII, so it needs no --ascii variant.
var MothArt = []string{
	`       \   /`,
	`   .-.  \ /  .-.`,
	`  (   '-(o)-'   )`,
	`   '-.  |=|  .-'`,
	`  (   '-|=|-'   )`,
	`   '-.__\_/__.-'`,
}

// InboxNote is the day's note for an empty Inbox (G6.12).
func InboxNote(today task.Date) string { return dailyPick(inboxNotes, today) }

// SomedayNote is the day's note for an empty Someday list (G7.9).
func SomedayNote(today task.Date) string { return dailyPick(somedayNotes, today) }

// ProjectNote is the day's note for project name's empty list (G7.9).
func ProjectNote(name string, today task.Date) string {
	n := dailyPick(projectNotes, today)
	if strings.Contains(n, "%s") {
		return fmt.Sprintf(n, name)
	}
	return n
}

// TodayDoneNote is the day's note for a finished Today list (G6.14).
func TodayDoneNote(today task.Date) string { return dailyPick(todayDoneNotes, today) }

// dailyPick picks the day's note: the same all day, a new one each day,
// chosen by hashing the date so the order looks random.
func dailyPick(notes []string, today task.Date) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(today.String()))
	return notes[h.Sum32()%uint32(len(notes))]
}
