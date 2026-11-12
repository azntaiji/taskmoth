# taskmoth — Implementation Plan

A keyboard-driven TUI task manager written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Targets macOS and Ubuntu. The UI is fully designed in Figma; this plan implements those designs phase by phase.

Source brief: `CREATE-PLAN.md`. Where this plan and the Figma file disagree, **Figma wins for visuals** and the operator decides behaviour (see gates).

---

## 1. Rules for the coding agent

1. **Figma MCP is mandatory.** Before any work in a phase that touches UI, confirm the Figma MCP is reachable (`whoami`, then `create_session` on file key `uRWYa0RRojip50i2lWNdKx`). **If it is not available, STOP and tell the operator.** Do not implement UI from this document's descriptions or from memory.
2. **Pull the design before building it.** For every screen/dialog, call `get_design_context` (with screenshot) on the node listed in §3 before writing its code. The screenshot is the visual target. Full-screen frames return very large payloads; if a response is sparse or truncated, fetch child nodes individually.
3. **Gates are hard stops.** A `GATE` is a decision that belongs to the operator. When you reach one: stop, present the question with the options and recommendation given here, wait for an answer, then record it in §8 (Decision log) before continuing. Never pick a default and move on, even if the recommendation looks obvious. Gates listed at the start of a phase must be resolved before that phase's code is written.
4. **Raise new gates.** If you hit an undocumented decision that changes user-visible behaviour, data format, or a key binding, treat it as a gate.
5. **One phase at a time.** Finish a phase's exit criteria, show the operator the result (run the app, or rendered golden output next to the Figma screenshot), and get a go-ahead before starting the next.
6. **Don't expand scope.** Time tracking and analytics (§7) are explicitly out. No recurring tasks, subtasks, labels, sync, or mouse support unless a gate adds them.
7. **Tests accompany code.** Every phase lists what must be tested. `go build ./...`, `go vet ./...`, and `go test ./...` must pass on macOS and Linux before a phase is called done.

---

## 2. What the designs establish

Observed directly from the Figma frames; treat as requirements unless a gate overrides.

**Grid.** Designed on an 80×24 cell terminal (cell 12×24 px, JetBrains Mono Nerd Font). All spacing tokens are whole cells.

**Tokens** (from Figma variables — re-pull with `get_variable_defs`, do not hardcode from this table):

| Token | Value | Token | Value |
|---|---|---|---|
| `bg/default` | `#22212c` | `selection/bg` | `#9282f7` |
| `text/default` | `#ffffff` | `text/muted` | `#504c67` |
| `text/success` | `#8aff80` | `key/key` / `key/label` | `#80ffea` / `#504c67` |
| `badge/error` (P1, overdue) | `#ffffff` on `#ff9580` | `badge/warning` (P2, today) | `#22212c` on `#ffff80` |
| `badge/neutral` (P3, future date) | `#22212c` on `#504c67` | `badge/info` (view badge) | `#ffffff` on `#ff80bf` |
| `badge/mode` | `#f8f8f2` on `#22212c` | `statusbar` | `#f8f8f2` on `#504c67` |
| `scrollbar/track` / `thumb` | `#504c67` / `#ffffff` | `attr/dim-opacity` | 50% |

**List view anatomy** (Today, project list, search results):
- Row 0 header: `taskmoth ▸ <View>` left; `tues · 2026-11-10 · 15:26` right, muted. The clock is live.
- Task rows are two lines plus a blank spacer: line 1 = selection marker `▶`, checkbox `[ ]`/`[x]`, priority badge, name (truncated with `…`), right-aligned due badge; line 2 (muted) = `#project  ~duration  s:start-date`, each shown only if set.
- Selected row: full-width `selection/bg` across both lines. Completed rows: green `[x]`, name and badges dimmed.
- P4 renders as plain text with no badge background. Tasks with no priority show no badge.
- Due badge: `today` (warning), `overdue` (error), or a `YYYY-MM-DD` date (neutral; error-coloured if past — see G4.3).
- Scrollbar in the last column.
- Status bar: view badge, summary (`4/6 tasks · ~8h total · ~3.75h remaining`), right-aligned sort indicator (`s:priority`).
- Two rows of key hints at the bottom.

**Inbox variant.** One line per task (checkbox + name only), blank spacer between, status `6 tasks to organize`, no sort indicator.

**Dialogs.** 60 cells wide, square-cornered box border in cyan with a purple title inset in the top border, muted help text, input area on a `statusbar/bg`-coloured field with a `>` prompt and block cursor, key hints on the last line. Overlaid centred on the current view, which is rendered **dimmed** behind it (node `32:13454`). The Keyboard Shortcuts dialog and the Single Task frame use a muted border instead.

**Add-task syntax shown in the design:** `#project`, `s:<start>`, `@<due>`, `~<duration>`, `!p1`–`!p4`; e.g. `Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow`. Tasks with no `#project` go to Inbox.

**Design inconsistencies to resolve with the operator** (each is covered by a gate below):
- `q` is "add task", so there is no quit key anywhere in the designs.
- Brief says default sort is Name; every frame shows `s:priority`.
- The overlay example frame shows a `NORMAL` mode badge and a `4/6` counter on the right of the status bar; the standalone Today frame shows a `Today` badge and `s:priority`.
- Single Task view shows priority, due and start but not duration.
- Edit Duration dialog's sample input reads `tomorrow` (looks like a copy from the due-date dialog).
- Edit Name offers `shift + enter newline`, implying multi-line names; list rows show names on one line.
- Today contains an overdue task and completed tasks; overdue shows as the word `overdue` in Today but as a red date in the project list.
- Bottom hints list `g a` but not `g c`; the Goto Project hint line is clipped (`x delet`).

---

## 3. Figma node map

File: `https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth` — file key `uRWYa0RRojip50i2lWNdKx`.

| Screen / dialog | Node | Phase |
|---|---|---|
| Today list view | `32:9525` | 3 |
| Inbox list view | `40:18095` | 3 |
| Dialog-over-view example (dimming) | `32:13454` | 1 |
| Goto Project dialog | `40:14667` | 4 |
| Project list view | `40:17686` | 4 |
| Create New Project dialog | `40:14812` | 4 |
| Delete confirmation dialog | `40:14869` | 4 |
| Add Task dialog | `40:16708` | 5 |
| Single Task view | `29:7123` | 6 |
| Edit Name | `40:16847` | 6 |
| Edit Description | `15:5973` | 6 |
| Edit Project | `28:6404` | 6 |
| Edit Start date | `40:15310` | 6 |
| Edit Due date | `40:15226` | 6 |
| Edit Priority | `15:6168` | 6 |
| Edit Duration | `40:16639` | 6 |
| Search | `40:16937` | 7 |
| Keyboard Shortcuts dialog | `40:15079` | 7 |

Derived views with no frame of their own: **Upcoming** and **Overdue** are variations of Today; **All** and **Completed** are variations of the project list.

---

## 4. Proposed architecture

Subject to Phase 0 gates.

```
cmd/taskmoth/        main: flags, config/data paths, tea.Program
internal/task/       domain types, validation, sort comparators, view filters (pure)
internal/nlp/        natural-language parser for quick-add, dates, durations (pure)
internal/store/      persistence behind a Store interface
internal/ui/theme/   design tokens -> lipgloss styles, dimmed variant
internal/ui/widget/  header, statusbar, keyhints, badge, scrollbar, dialog frame, text input
internal/ui/view/    list view (configurable), single task view, search
internal/ui/dialog/  one file per dialog
internal/ui/app.go   root model: active view, dialog stack, key chords, clock tick
```

Principles:
- `task` and `nlp` have no UI or storage imports and take the current time as a parameter, so everything date-related is deterministic in tests.
- One **configurable list view** drives all seven lists (filter, row style one-line/two-line, status summary, due-badge style, sort availability) rather than seven models.
- The root model owns a **dialog stack**; the top dialog receives keys, the view underneath is rendered with the dimmed theme and the dialog composited over it.
- All persistence goes through `tea.Cmd`s; the UI never blocks on I/O.
- UI tests render at 80×24 and compare against golden files.

---

## 5. Phases

### Phase 0 — Bootstrap and foundational decisions

**Gates (resolve all before writing code):**

- **G0.1 Bubble Tea version.** The brief names `github.com/charmbracelet/bubbletea` (v1). v2 lives at `charm.land/bubbletea/v2` with Lip Gloss v2 and Bubbles v2. *Recommend v2*: it has layer compositing for dialog overlays and keyboard enhancements that make `shift+enter` detectable (G6.2). Check the current stable release and docs before asking; state what you found.
- **G0.2 Storage.** (a) SQLite via pure-Go `modernc.org/sqlite` — no cgo, clean cross-builds, real queries, ready for future analytics; (b) single JSON file — simplest, hand-editable; (c) one markdown/text file per task. *Recommend (a).*
- **G0.3 Data and config location.** *Recommend* XDG on both platforms (`$XDG_DATA_HOME/taskmoth`, default `~/.local/share/taskmoth`; config in `~/.config/taskmoth`) with a `--data` flag override, versus `~/Library/Application Support` on macOS.
- **G0.4 Quit key.** `q` is taken by "add". Options: `ctrl+c` only; `ctrl+c` and `ctrl+q`; `Q`; `:q`. *Recommend `ctrl+c` + `ctrl+q`*, and ask whether it should appear in the Keyboard Shortcuts dialog (it is not in the design).
- **G0.5 Module path and repo.** Go module path, licence, and whether to `git init` (the directory is not yet a repository).
- **G0.6 Minimum Go version** and whether to add an `AGENTS.md` capturing these rules for future agent sessions.

**Work:** module, directory skeleton, `main` that opens an empty alt-screen program and exits on the quit key, Makefile or `justfile` (`build`, `test`, `lint`, `run`), `golangci-lint` config, CI workflow running build/vet/test on `macos-latest` and `ubuntu-latest`.

**Exit:** binary builds and runs on both OSes; CI green; decisions logged.

---

### Phase 1 — Design system and overlay compositing

**Gates:**

- **G1.1 Terminal size behaviour.** Designs are 80×24. Options: (a) fluid — widths stretch, list shows more rows, dialogs stay 60 wide and centred, with a "terminal too small" message below a minimum; (b) fixed 80×24 canvas centred in larger terminals. *Recommend (a)*, minimum 80×24.
- **G1.2 Colour fallback.** Truecolor is assumed. In 256/16-colour terminals: automatic downsampling (Lip Gloss default) or a hand-tuned ANSI palette? *Recommend automatic.* Also confirm there is no light theme.
- **G1.3 Glyphs.** `▶ ▸ ↑ ↓ ← → ·` and box-drawing are plain Unicode and need no Nerd Font. Confirm no Nerd-Font-only glyphs are wanted, and whether an ASCII fallback is needed.
- **G1.4 Dimming.** Terminals have no opacity. *Recommend* re-rendering the background view with a second theme whose colours are pre-blended 50% toward `bg/default`, rather than the ANSI faint attribute (inconsistent across terminals). Show the operator both against node `32:13454`.

**Work:**
- `theme`: tokens pulled from Figma variables into one file; normal and dimmed style sets.
- Widgets, each built from its Figma component: header with live clock, status bar, key-hint rows, badges (priority, due, view/mode), checkbox, selection marker, scrollbar, dialog frame with inset title, single-line and multi-line text inputs with `>` prompt and placeholder.
- Overlay compositor: centre a dialog over a dimmed view.
- A hidden `--gallery` dev flag (or test-only harness) that renders every widget.

**Tests:** golden render per widget; overlay golden matching `32:13454` layout; width-aware truncation with wide/emoji characters.

**Exit:** operator has compared the gallery to Figma and approved.

---

### Phase 2 — Domain model and storage

**Gates:**

- **G2.1 Field semantics.**
  - Priority unset vs P4: are they distinct? (Design shows Inbox tasks with no badge and a `P4` text badge elsewhere.) *Recommend distinct, unset sorts last.*
  - Dates: date-only, or date + optional time? *Recommend date-only for v1* — every design shows `YYYY-MM-DD`.
  - Start date: purely informational, or does a future start date hide the task from Today/Upcoming? *Recommend informational.*
  - Name: single line or multi-line (see G6.2)?
- **G2.2 Projects.** Name rules (*recommend* lowercase, no spaces, `[a-z0-9-_]`, unique, shown with `#`); ordering in pickers (creation order as in the design, or alphabetical).
- **G2.3 Completion and deletion.** Store `completed_at`? (*Recommend yes — needed for Completed view and future analytics.*) Hard delete or soft delete/trash? (*Recommend hard delete behind the confirmation dialog.*)
- **G2.4 Schema sign-off.** Present the final schema/file format before implementing; it is expensive to change later. Include `created_at`/`updated_at` and a schema version with migrations.

**Work:** `task.Task`, `task.Project`, validation; `store.Store` interface with the chosen backend, migrations, and an in-memory fake for UI tests; view filters as pure functions of `(tasks, now)`; comparators for name, start, due, priority, duration with stable tie-breaking; a `--seed` dev command that loads the sample data visible in the designs.

**Tests:** table-driven filter and sort tests with a fixed clock (day boundaries, unset fields); store round-trip, migration, and concurrent-safety tests against a temp directory.

**Exit:** seeded database can be listed from a throwaway CLI command.

---

### Phase 3 — Core list views: Today and Inbox

**Gates:**

- **G3.1 Today's contents.** The design shows overdue and completed tasks in Today. Confirm: Today = due today **plus** overdue? Do completed tasks stay visible until the day ends, until the view is left, or never? What does the `4/6` in the summary count (*assumed: remaining/total*)?
- **G3.2 Status bar.** Which variant is canonical — `Today` view badge + `s:priority` (frame `32:9525`), or `NORMAL` mode badge + `4/6` (frame `32:13454`)? If modes are real, define them.
- **G3.3 Sorting.** Default sort: Name (brief) or Priority (designs)? Does `s` cycle in the brief's order (name → start → due → priority → duration)? Fixed direction per field or a reverse toggle? Where do unset values go? Is the choice remembered per view and across restarts? Does Inbox sort at all (design shows no indicator)?
- **G3.4 Movement keys.** Design lists `↑/↓` and `pgup/pgdn`. Add `j/k`, `home/end`? Note `g` is the chord prefix, so vim `gg`/`G` needs a decision.
- **G3.5 Inbox semantics.** Inbox = no project. Should completed Inbox tasks disappear immediately?

**Work:** configurable list view with two-line and one-line row styles; selection, scrolling with scrollbar, empty state; `space` toggles done; `s` cycles sort; status summary with duration totals (`~8h`, `~3.75h`, `~45m` formatting as designed); `g` chord handling with timeout for `g t` and `g i`; Today as startup view; day rollover refreshes "today".

**Tests:** goldens for both frames using seed data and a fixed clock; chord state machine; selection stability across toggle and re-sort; long-name truncation.

**Exit:** Today and Inbox match Figma side by side at 80×24.

---

### Phase 4 — Projects and remaining list views

**Gates:**

- **G4.1 Upcoming.** "Due within the next 7 days": include today? include overdue? Group by day with headers, or flat list with date badges? *Recommend tomorrow through +7 days, flat, sorted by due date by default.*
- **G4.2 All / Completed.** Does All include completed tasks? Does Completed show completion date in place of the due badge, and does `space` there un-complete? Default sort for Completed (*recommend most recently completed first*).
- **G4.3 Overdue badge style.** The word `overdue` (Today frame) or the red date (project frame)? *Assumed: date-relative views use words, other views use dates* — confirm.
- **G4.4 Deleting a project** (`x` in Goto Project): delete its tasks, or move them to Inbox? What about its completed tasks? *Recommend move open tasks to Inbox, with the confirmation dialog stating the count.*
- **G4.5 Key hints.** Add `g c` to the bottom hints (currently absent)? How should hint rows degrade when they overflow the width, as the Goto Project hint line does?
- **G4.6 Empty states.** Copy for each empty view and for "no projects yet".

**Work:** Goto Project dialog (`g p`) with scroll, `n` new, `x` delete; Create New Project dialog with validation errors; Delete confirmation dialog (reused for tasks, `x` in lists); project list view; Upcoming, Overdue (`g u`, `g o`) from the Today configuration; All, Completed (`g a`, `g c`) from the project-list configuration.

**Tests:** goldens for `40:14667`, `40:17686`, `40:14812`, `40:14869`; filter tests for each derived view; project delete behaviour.

**Exit:** every `g` chord lands on a correct, populated view.

---

### Phase 5 — Natural-language parsing and Add Task

**Gates:**

- **G5.1 Grammar.** Present a written grammar and a table of 30+ example inputs with expected results for sign-off before coding. Open points:
  - The design's sigils (`#`, `s:`, `@`, `~`, `!p2`) differ from Todoist's (bare `p1`, bare dates). Sigils only, or also Todoist-style bare recognition? *Recommend sigils only for v1 — unambiguous and matches the dialog's help text.*
  - Multi-word values (`@in 2 days`, `@next monday`): value runs until the next sigil or end of line?
  - Supported date phrases (minimum from the dialogs: ISO date, `today`, `tomorrow`, `in N days`, `next <weekday>`) and what else: weekday names, `mon`, `nov 20`, `11/20`, `in 2 weeks`, `eom`? Does a bare weekday mean the next occurrence?
  - Durations (from the dialog: `30m`, `1.5h`, `2 hours`, `10 minutes`); also `1h30m`? Stored as minutes?
  - Escaping a literal `#`, `@`, `~`, `!` in a task name.
  - Week start day and date-format ambiguity (`11/12`).
- **G5.2 Unknown `#project`.** Create it silently, ask, or reject? *Recommend create, and say so in the status bar.*
- **G5.3 Parser implementation.** Hand-written parser over a small fixed grammar (deterministic, fully testable, no dependency) versus a library such as `olebedev/when`. *Recommend hand-written.* If the operator wants Todoist parity, use the `rea` MCP to study Todoist's quick-add behaviour first and report findings.
- **G5.4 Feedback.** The design shows a plain text field. Add live highlighting of recognised tokens or a parsed preview line? Where do parse errors appear? Does adding a task while in a project view default to that project? Does the dialog stay open for rapid entry?

**Work:** `nlp.Parse(input, now) (Draft, []Diagnostic)`; shared date and duration parsers reused by the edit dialogs; Add Task dialog (`q`) with wrapping multi-line input; new task selected after creation if visible in the current view.

**Tests:** the approved example table as table-driven tests; fuzz test that the parser never panics and the remaining name text is preserved; dialog golden for `40:16708`.

**Exit:** the design's sample input produces the expected task.

---

### Phase 6 — Single Task view and edit dialogs

**Gates:**

- **G6.1 Single Task content.** Show duration in the meta row (absent in the design)? Show created/completed dates? Is the view scroll-only, with all edits via the dialog keys?
- **G6.2 Newline key.** `shift+enter` cannot be distinguished from `enter` in many terminals (Terminal.app, tmux without extended keys). Confirm a fallback such as `alt+enter` or `ctrl+j`, and whether hints should show it. Confirm whether task **names** may contain newlines at all (*recommend no — offer newline only in Description*).
- **G6.3 Clearing values.** How to unset an optional field: empty input + `enter`, or a dedicated key? How to unset priority and project (extra "none" option, `0` key, `backspace`)?
- **G6.4 Edit Duration sample text** reads `tomorrow` in the design — confirm it should be a duration such as `30m`.
- **G6.5 Validation.** Is start-after-due an error, a warning, or allowed? Where do dialog validation errors render?
- **G6.6 Edit keys from lists.** The shortcuts dialog lists `p v r t y n d` under "Selected Tasks": confirm they work directly on the selected row in list views as well as inside the Single Task view.

**Work:** Single Task view (`e`, `esc` returns to the originating list with selection preserved); dialogs for name, description, project (with `n` to create inline), start date, due date, priority (`←/→` and `1`–`4`), duration; `space` and `x` inside the task view.

**Tests:** golden per dialog and for `29:7123`; each dialog's save/cancel/clear paths; round-trip from list → task → edit → list.

**Exit:** every task field can be set, changed, and cleared from the keyboard.

---

### Phase 7 — Search, shortcuts dialog, polish

**Gates:**

- **G7.1 Search behaviour.** Scope (all open tasks, or include completed)? Fields (name only, or description and project too)? Substring or fuzzy? Results update per keystroke? Default result ordering? Does `#project` or `!p1` in the query filter?
- **G7.2 Search interaction.** While the input is focused, letter keys must type rather than act. Proposed: typing filters, `↑/↓` move selection, `enter` moves focus to results (enabling `e`, `space`, `x`, …), `/` refocuses input, `esc` leaves search. Confirm.
- **G7.3 Shortcuts dialog.** `?` opens it; which key closes it (design shows none)? Add the quit key and any keys approved in earlier gates?

**Work:** Search view (`/`) with the status bar replaced by the search input as designed; Keyboard Shortcuts dialog; consistency pass over every hint row against the actual key map (generate hints from one key-map definition so they cannot drift).

**Tests:** search matching tests; goldens for `40:16937` and `40:15079`; key-map/hint consistency test.

**Exit:** full walkthrough of every screen in §3 against Figma with the operator.

---

### Phase 8 — Hardening and release

**Gates:**

- **G8.1 Distribution.** GoReleaser with GitHub Releases, Homebrew tap, `.deb`, `go install` only? Target architectures (*recommend darwin/arm64, darwin/amd64, linux/amd64, linux/arm64*). Versioning scheme.
- **G8.2 Data safety.** Automatic backups before migrations? Export/import command (JSON)?

**Work:** manual test matrix — macOS (Terminal.app, iTerm2, Ghostty) and Ubuntu (GNOME Terminal, plus tmux and SSH); resize handling at all sizes; 256-colour check; graceful handling of a locked or corrupt data file; performance check with 5,000 tasks (render and search stay responsive); README with install, key reference, and quick-add syntax; `--version`; release pipeline per G8.1.

**Exit:** tagged release installable on both platforms.

---

## 6. Testing strategy

- **Pure logic** (`task`, `nlp`): table-driven, fixed clock, fuzzing on parsers.
- **Store:** temp-directory integration tests; migration tests from every prior schema version.
- **UI:** golden-file tests rendering models at 80×24 with seed data and a fixed clock (`teatest` or direct `View()` calls); a second size (e.g. 120×40) for fluid layout. Goldens are reviewed against Figma screenshots when first created and whenever they change.
- **CI:** build, vet, lint, test on macOS and Ubuntu runners for every push.

---

## 7. Future features (out of scope)

Not to be built now, but keep the door open:

- **Task time tracking** — start/stop timers per task, actual vs estimated duration. Implication now: keep duration stored as minutes and keep `completed_at`.
- **Task and project analytics** — completion rates, time per project, trends. Implication now: keep timestamps on tasks and prefer a queryable store.

---

## 8. Decision log

Record every gate outcome here before proceeding.

| Gate | Decision | Date | Notes |
|---|---|---|---|
| G0.1 | Bubble Tea **v2** (`charm.land/bubbletea/v2`), with Lip Gloss v2 and Bubbles v2 | 2026-11-12 | Latest stable at decision time: bubbletea v2.1.1, lipgloss v2.1.0, bubbles v2.2.1 (v1 was v1.3.10). |
| G0.2 | **SQLite** via pure-Go `modernc.org/sqlite` | 2026-11-12 | No cgo. Schema still needs sign-off at G2.4. |
| G0.3 | **XDG on both OSes**: data `$XDG_DATA_HOME/taskmoth` (default `~/.local/share/taskmoth`), config `$XDG_CONFIG_HOME/taskmoth` (default `~/.config/taskmoth`); `--data` flag overrides the data dir | 2026-11-12 | Same paths on macOS and Ubuntu. |
| G0.4 | Quit with **`ctrl+c` and `ctrl+q`** | 2026-11-12 | Must be listed in the Keyboard Shortcuts dialog (Phase 7). |
| G0.5 | Module `github.com/azntaiji/taskmoth`; **MIT** licence; local `git init` (no remote, no push) | 2026-11-12 | Repo already existed (operator's `Initial Commit` ab00710), so no init was needed. |
| G0.6 | `go 1.26` minimum in `go.mod`; add **AGENTS.md** | 2026-11-12 | Local toolchain is 1.27.3. |
