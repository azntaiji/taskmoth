package view

import (
	"hash/fnv"

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

// InboxNote is the day's note for an empty Inbox (G6.12).
func InboxNote(today task.Date) string { return dailyPick(inboxNotes, today) }

// TodayDoneNote is the day's note for a finished Today list (G6.14).
func TodayDoneNote(today task.Date) string { return dailyPick(todayDoneNotes, today) }

// dailyPick picks the day's note: the same all day, a new one each day,
// chosen by hashing the date so the order looks random.
func dailyPick(notes []string, today task.Date) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(today.String()))
	return notes[h.Sum32()%uint32(len(notes))]
}
