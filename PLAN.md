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

**Tokens.** Figma has two colour layers: "Primitives / Palette", the 16 ANSI slots with a "Dracula" mode, and "Semantic / Color", tokens that alias slots. `get_variable_defs` returns only resolved hex values. To see which slot each token points to, read the collections with a read-only `use_figma` script. The code mirrors both layers in `internal/ui/theme/palette.go` and `colors.go` (G1.7). Resolved Dracula values, for reference only:

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

**Dialogs.** 60 cells wide, square-cornered box border in cyan with a purple title inset in the top border, muted help text, input area on a `bg/input` field with a `>` prompt and block cursor, key hints on the last line. Overlaid centred on the current view, which is rendered **dimmed** behind it (node `32:13454`). The Keyboard Shortcuts dialog and the Single Task frame use a muted border instead.

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

- **G4.1 Upcoming layout.** Which tasks appear is settled by G2.1c-upcoming. Remaining: group by day with headers, or a flat list with date badges? Default sort? *Recommend flat, sorted by the earlier of start and due date.*
- **G4.2 All / Completed.** Does All include completed tasks? Does Completed show completion date in place of the due badge, and does `space` there un-complete? Default sort for Completed (*recommend most recently completed first*).
- **G4.3 Overdue badge style.** The word `overdue` (Today frame) or the red date (project frame)? *Assumed: date-relative views use words, other views use dates* — confirm.
- **G4.4 Deleting a project** (`x` in Goto Project): delete its tasks, or move them to Inbox? What about its completed tasks? *Recommend move open tasks to Inbox, with the confirmation dialog stating the count.*
- **G4.5 Key hints.** Add `g c` to the bottom hints (currently absent)? How should hint rows degrade when they overflow the width, as the Goto Project hint line does?
- **G4.6 Empty states.** Copy for each empty view and for "no projects yet".
- **G4.7 Trash access.** G2.3 chose a trash, which the designs don't include. How do you view trashed tasks (e.g. a `g` chord to a Trash list view), restore them, and permanently delete one? Does the Delete dialog's wording change to "Move to trash?"
- **G4.8 Emptying the trash.** Never automatically, after N days (e.g. 30), or only on an explicit "empty trash" action? Is permanent deletion confirmed with a dialog?

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

- **G8.1 Distribution.** GoReleaser with GitHub Releases, Homebrew tap, `.deb`, `go install` only? Target architectures (*recommend darwin/arm64, darwin/amd64, linux/amd64, linux/arm64*). (The versioning scheme was decided early, at G8.1a.)
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

- **Task time tracking** — start/stop timers per task, actual vs estimated duration. Implication now: keep duration stored as minutes and keep `completed_at`. Storage plan (G2.4): a separate `time_entries(id, task_id → tasks.id, started_at, ended_at)` table added by a new migration, many entries per task, not a column on `tasks`.
- **Task and project analytics** — completion rates, time per project, trends. Implication now: keep timestamps on tasks and prefer a queryable store.
- **Recurring tasks** — e.g. `every monday`, `every 3 months`, Todoist's `every!` (repeat from completion). Implication now: quick add leaves recurring phrases in the name (G5.1e).
- **Times of day** — due and start times (`tomorrow at 4pm`). Dates are date-only today (G2.1b), so this needs a schema change.
- **Theme support** — user-selectable colour themes (e.g. files in `~/.config/taskmoth/themes/` mapping Figma token names to colours), possibly including a light theme and a theme that follows the terminal's own ANSI palette. Implication now: every colour goes through `theme.Colors` (named after its Figma variable); views never construct colours. A terminal-palette theme can't use pre-blended dimming (G1.4) because its RGB values are unknown, so it would need the faint attribute instead.

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
| G1.1 | **Fluid layout, minimum 80×24.** Header, rows and status bar stretch to full width; the list shows more rows when the terminal is taller; dialogs stay 60 cells wide and centred; below 80×24, show a "terminal too small" message. | 2026-11-12 | |
| G1.2 | **Automatic colour downsampling** (Lip Gloss) on 256- and 16-colour terminals. No light theme. | 2026-11-12 | |
| G1.3 | **Plain Unicode glyphs with an ASCII fallback.** Use only the symbols in the designs, no Nerd Font icons. An opt-in setting swaps them for ASCII. | 2026-11-12 | The fallback's trigger (env var, flag or config) and the ASCII character for each glyph are open sub-decisions to confirm before the glyph table is built. |
| G1.4 | **Pre-blended dimming.** Draw the view behind a dialog with a second theme whose colours are mixed 50% toward `bg/default`, not with the ANSI faint attribute. | 2026-11-12 | The operator compared both options in a terminal demo. This matches Figma's `attr/dim-opacity` of 50%. |
| G1.3a | **ASCII fallback is turned on by `--ascii` or `TASKMOTH_ASCII=1`.** | 2026-11-12 | No config file is needed for this. |
| G1.3b | **ASCII uses one character per glyph**, so layouts stay the same width: `▶`→`>`, `▸`→`>`, `↑`/`↓`→`^`/`v`, `←`/`→`→`<`/`>`, `·`→`-`, `─`→`-`, `│`→`\|`, corners→`+`, scrollbar thumb→`#`, scrollbar track→`\|`. | 2026-11-12 | |
| G1.5 | **Proportional scrollbar thumb, hidden when everything fits.** The thumb's size reflects visible/total and it moves between the first and last track cells (as in Figma). When the whole list fits, only the track shows. | 2026-11-12 | Raised during Phase 1. Figma shows a fixed 3-row thumb, which is a static mock. |
| G1.6 | **The ASCII truncation tail is `...`**, counted inside the truncated width. | 2026-11-12 | Raised during Phase 1. G1.3b did not cover `…`. |
| G1.7 | **The theme mirrors Figma's two colour layers, drawn as 24-bit RGB from the Dracula palette.** `theme.Palette` holds the 16 ANSI slots plus `base/fg` and `base/bg`. `theme.Semantic` resolves every semantic token to a slot exactly as Figma aliases it. Colours are sent as RGB, not ANSI indices, so the app looks the same in every terminal and dimming can blend real colours (G1.4). | 2026-11-12 | Supersedes the Phase 1 approach of copying resolved hex values. A palette that follows the terminal's own ANSI colours is a future theme (§7) and would need faint-attribute dimming. |
| G2.1a | **Priority is required, P1–P4, defaulting to P4.** There is no "unset". Tasks created without `!p` are P4. | 2026-11-12 | The Inbox design's missing badge is a property of that view's one-line row style, not of the data. |
| G2.1b | **Start and due dates are date-only**: local calendar dates (`YYYY-MM-DD`). "Today" and "overdue" use the local date. | 2026-11-12 | |
| G2.1c | **Start dates are shown and sortable, and also decide list membership.** **Today** = open tasks that are due today, overdue, or have a start date of today or earlier. **Upcoming** = open tasks whose start or due date falls between tomorrow and 7 days ahead. A task can be in both. | 2026-11-12 | The operator asked for start-date behaviour, then chose from the options presented. Long-running started tasks stay in Today until done. This supersedes the "which tasks" parts of G3.1 and G4.1. |
| G2.1d | **Task names are single-line.** | 2026-11-12 | Edit Name's `shift + enter newline` hint is to be dropped (confirm at G6.2). |
| G2.2 | **Project names are lowercase slugs** (`a–z`, `0–9`, `-`, `_`), unique, and displayed as `#name`. Typed capitals are lowercased. **Pickers list projects A–Z.** | 2026-11-12 | |
| G2.3 | **Completing records `completed_at`; un-completing clears it. Deleting a task moves it to a trash** (`deleted_at`) rather than removing it. *The trash became the Archive at G4.4, and the column became `archived_at`.* | 2026-11-12 | The trash isn't in the designs. How to view, restore and empty it is gated at G4.7 and G4.8. |
| G2.4 | **Schema approved with UUIDv7 IDs.** One SQLite file, `<data dir>/taskmoth.db`, opened with WAL, foreign keys and a busy timeout; version kept in `PRAGMA user_version`. **`projects`**: `id`, `name` (unique slug), `created_at`, `updated_at`. **`tasks`**: `id`, `name` (single line), `description` (`''` = none), `project_id` (NULL = Inbox; `ON DELETE RESTRICT` until G4.4), `priority` (1–4, default 4), `start_date` and `due_date` (local `YYYY-MM-DD`), `duration_minutes`, `created_at`, `updated_at`, `completed_at`, `deleted_at`. IDs are UUIDv7 text; timestamps are UTC RFC 3339. | 2026-11-12 | Time tracking is not in the schema. It will be its own `time_entries` table (many per task), added by a later migration (§7). |
| G3.1 | **Completed tasks stay listed, dimmed with `[x]`, until the end of the day they were completed**, in Today, Inbox and project lists. After that they appear only in Completed. A completed task stays in a list only if it would otherwise belong there by its dates or project. The summary `4/6 tasks` = open / listed. | 2026-11-12 | Which open tasks are in Today was settled at G2.1c. |
| G3.2 | **The status bar shows a pink view badge** (info tone) with the view name, then `N/M tasks · ~total · ~remaining` (`total` = all listed durations, `remaining` = open ones' durations), and the current sort at the right (`s:priority`). No modes. | 2026-11-12 | The overlay frame's `NORMAL` badge is superseded. |
| G3.3a | **Default sort is priority.** `s` cycles priority → due → start → duration → name → priority. Directions are fixed: P1 first, earliest date first, shortest first, A–Z. Tasks missing the field sort last. | 2026-11-12 | Supersedes the brief's "Name (default)". |
| G3.3b | **Each view remembers its own sort, across restarts**, stored in a new `settings` table (schema migration 2, an approved change to G2.4). Inbox sorts like the others but shows no sort indicator, as designed. | 2026-11-12 | `settings(key TEXT PRIMARY KEY, value TEXT NOT NULL)`. |
| G3.3c | **Completed tasks sort below open ones**: the open tasks first, then the completed tasks, each group in the chosen order. **Ties break A–Z by name, then by creation order.** | 2026-11-12 | Raised during Phase 3. The Today frame's P2 pair ("Practice…" above "Pack…") will appear swapped relative to the design. |
| G3.4 | **Movement keys:** ↑/↓ and j/k move, pgup/pgdn scroll a page, home/end and `g g` / `G` jump to top/bottom. **`g` chords have no timeout.** `esc` cancels a pending `g`, and the status bar shows `g…` while waiting. | 2026-11-12 | |
| G3.5 | **Inbox follows G3.1:** completed Inbox tasks stay listed until the end of the day. | 2026-11-12 | |
| G3.6 | **Header clock** as designed: lowercase weekday (`mon tues wed thurs fri sat sun`) · `YYYY-MM-DD` · 24-hour `HH:MM`, updated every minute. | 2026-11-12 | Raised during Phase 3. |
| G3.7 | **Duration display:** under 60 min, `~45m`; otherwise hours with up to 2 decimals, trailing zeros dropped (`~1.5h`, `~3.75h`, `~1.67h`, `~8h`). Used in rows and totals. | 2026-11-12 | Raised during Phase 3. |
| G3.8 | **Empty Today and Inbox** show one muted line: Today, "Nothing due or started today."; Inbox, "Inbox is empty."; followed by the hint "q to add a task". The status bar shows `0 tasks`. | 2026-11-12 | The Phase 3 part of G4.6, pulled forward. |
| G3.9 | **Store errors appear in the status bar** as `Error: …`, in place of the summary, until the next key press. | 2026-11-12 | Approved at the Phase 3 review. |
| G3.10 | **`g` chords for views not built yet** show "That view arrives in a later phase." in the status bar. This is temporary and is removed as each view lands. | 2026-11-12 | Approved at the Phase 3 review. |
| G3.11 | **Key hint rows show the full designed hints**, including keys not working yet. Phase 7 syncs the hints with the key map. | 2026-11-12 | Approved at the Phase 3 review. |
| G4.1 | **Upcoming is grouped by day.** A task is listed once: under its due date if that falls in the window (tomorrow to +7), otherwise under its start date. Each day starts with a header row in bold accent text, like `wed · 2026-11-11`; days with no tasks are omitted. Within a day the view's sort applies (default priority, `s` cycles, remembered). | 2026-11-12 | Not in the designs. Supersedes the PLAN default of a flat list. |
| G4.2 | **All = every open task**, plus today's completions (G3.1), with the default sort priority. **Completed = every completed task**, newest completion first; its badge shows the completion date instead of the due date, and `space` reopens a task. | 2026-11-12 | |
| G4.2a | **The Completed and Archive views have their own sort cycles.** Completed: completed → priority → due → start → duration → name. Archive: archived → priority → … → name. The `completed` and `archived` keys sort newest first. Each view remembers its choice (G3.3b). | 2026-11-12 | |
| G4.3 | **Due badges:** Today and Overdue use the words `today` / `overdue`. Upcoming, All and project lists show the date, coloured by state: yellow (warning) if today, red (error) if past, grey (neutral) if future. | 2026-11-12 | Matches the Today and project frames. |
| G4.4 | **The trash is renamed Archive (supersedes G2.3's trash), and projects can be archived.** `x` archives a task. Archiving a project also archives its unarchived tasks, all with the project's timestamp. Unarchiving a project restores it and the tasks archived with it; tasks archived individually before stay archived. Archived projects are hidden from the pickers. A new project can't take an archived project's name ("An archived project is named #work — unarchive it instead."). | 2026-11-12 | Schema migration 3 (approved): rename `tasks.deleted_at` → `archived_at`; add `projects.archived_at`. |
| G4.4a | **Unarchiving a single task whose project is archived moves it to the Inbox**, and the status bar says "Moved to Inbox (#work is archived)". Archived projects never contain open tasks. | 2026-11-12 | |
| G4.5 | **The bottom hint rows stay as designed, with `x archive`.** `g c` and `g x` are listed only in the Keyboard Shortcuts dialog (`?`). Rows that overflow are cut at the edge. | 2026-11-12 | |
| G4.5a | **In the Archive view, hint row 1 is** `u unarchive  x delete forever  s sort  ? shortcuts  / search`. Row 2 is unchanged. | 2026-11-12 | |
| G4.6 | **Empty states:** Upcoming, "Nothing starting or due in the next 7 days."; Overdue, "Nothing overdue."; All, "No open tasks."; Completed, "No completed tasks yet."; Archive, "The archive is empty."; a project, "#name has no open tasks."; Goto Project with no projects, "No projects yet." plus `n new`. Completed and Archive show no "q to add a task" hint. | 2026-11-12 | |
| G4.7 | **`g x` opens an Archive view** listing archived tasks newest first: two-line rows, with the badge showing the archive date. `u` unarchives. **Archived projects are listed in Goto Project (`g p`) under a muted "Archived" heading** below the active projects; `u` unarchives one there. | 2026-11-12 | Replaces the Trash view. |
| G4.8 | **Permanent deletion only happens from the archive**, after a second confirmation, and is never automatic. `x` on an archived task asks "Delete forever?". `x` on an archived project asks "Delete #work and its N archived tasks forever?". | 2026-11-12 | |
| G4.9 | **The archive confirmation dialog** reads `Archive "Get eggs"?` with `[ cancel ] [ archive ]`. Cancel starts selected; ←/→ switches, enter confirms the selected button, esc cancels. The "delete forever" dialogs follow the same pattern with `[ delete ]`. | 2026-11-12 | Resolves the Delete frame (40:14869) showing both buttons highlighted. |
| G4.10 | **G3.1's end-of-day rule also applies to Upcoming and Overdue**: tasks completed today stay listed, dimmed, until midnight. | 2026-11-12 | Approved at the Phase 4 review. |
| G4.11 | **Archived-name error wording** is "#work is archived; unarchive it instead." The G4.4 wording didn't fit the 56-cell dialog body. | 2026-11-12 | Approved at the Phase 4 review. |
| G4.12 | **Goto Project's hint row follows the selection.** With an archived project selected it shows `↑ / ↓ select  u unarchive  x delete forever  esc cancel`. The dialog opens on the project being viewed. | 2026-11-12 | Approved at the Phase 4 review. |
| G4.13 | **Status-bar notices** after project and archive actions: "Created #x.", "Archived #x.", "Unarchived #x.", "Deleted #x forever.", "Moved to Inbox (#x is archived)." | 2026-11-12 | Approved at the Phase 4 review. |
| G4.14 | **Completed and Archive rows show grey date badges** (completion or archive date). Their summaries are "N completed" and "N archived". Project and All lists show "N tasks · ~total · ~remaining", as in the project frame. | 2026-11-12 | Approved at the Phase 4 review. G4.3 stands: project lists show today's due date as a yellow date, not the word. |
| G5.1 | **The quick-add grammar in Appendix A is approved as written**, including all 46 examples, which become the parser's test table. | 2026-11-12 | A weekday matching today means next week's. |
| G5.1a | **Quick add is a Todoist-style hybrid.** Bare date phrases set the due date and bare `p1`–`p4` set the priority, as in Todoist. The design's sigils also work: `@` due, `s:` start, `~` duration, `#` project, `!p1`–`!p4` priority. Recognised tokens are highlighted as you type. | 2026-11-12 | Based on Todoist's help docs (Task Quick Add; Introduction to dates and time); no reverse engineering was needed. |
| G5.1b | **A value is the longest valid phrase after its sigil** (`@in 2 days` takes three words; `@tomorrow buy milk` takes one). A sigil without a valid value stays literal text. A backslash before a word keeps it literal (`\friday`). | 2026-11-12 | |
| G5.1d | **`ctrl+t` toggles the recognised token at or just before the cursor** between parsed (highlighted) and plain text. The hint row shows `ctrl+t keep as text`. | 2026-11-12 | The keyboard version of Todoist's click-to-unhighlight. |
| G5.1e | **Times and recurring phrases aren't stored** (G2.1b; tasks don't repeat). `tomorrow at 4pm` sets the due date to tomorrow, and `at 4pm` stays in the name. `every monday` isn't recognised and stays in the name. | 2026-11-12 | Recurring tasks and times of day were added to the roadmap (§7). |
| G5.1c | **Date rules:** numerics are US month/day (`11/20`, `11/20/2027`; `27/1` reads as Jan 27 because 27 can't be a month). A bare weekday is the next one. `next <weekday>` follows Todoist: the one in the following week if this week's hasn't passed. `next week` = next Monday, `next month` = the 1st of next month, `weekend` = the coming Saturday. The full phrase list is Appendix A. | 2026-11-12 | |
| G5.1f | **Short words only count after a sigil.** Bare recognition covers full words and phrases (`tomorrow`, `friday`, `november 20`, `in 3 days`) and month+day (`may 5`). Abbreviations and lone month names (`tom`, `sun`, `wed`, `may`) need `@` or `s:`. | 2026-11-12 | |
| G5.2 | **Project autocomplete plus did-you-mean.** Typing `#mu` lists matching projects under the input (fuzzy, so `#musc` still lists `#music`); `tab` completes the highlighted one. Saving with an unknown `#name` asks "Create #musc? Did you mean #music?" (`tab` uses the suggestion, `enter` creates the new project, `esc` keeps editing). An archived project's name is refused with the G4.11 message. | 2026-11-12 | Guards against near-duplicate projects. |
| G5.3 | **A hand-written parser** in `internal/nlp`: deterministic, table-tested and dependency-free, shared with the edit dialogs' date and duration fields. | 2026-11-12 | |
| G5.4 | **Add Task feedback:** a muted preview line under the input, e.g. `→ #oneofftasks · due 2026-11-12 · s:2026-11-11 · ~30m · P2`. In a project's list the project defaults to that project, otherwise the Inbox, unless `#` is typed. `enter` adds the task and closes the dialog; the new task is selected if it's in the current view; the status bar says `Added "name".` | 2026-11-12 | The preview line isn't in the design (40:16708). |
| G5.5 | **Approved at the Phase 5 review:**<br>• Recognised tokens are highlighted in `text/link`; the did-you-mean line is `text/warning` and errors are `text/error`.<br>• `ctrl+t` toggles a match by inserting or removing visible backslashes.<br>• The Add Task dialog grows below the input for the preview line (which wraps to two lines when long) and up to four suggestions.<br>• Prompt hints are shortened to fit (`tab #x  enter create  esc edit`; `esc hide list`).<br>• Did-you-mean suggests projects within 2 edits, or whose name contains the typed letters in order.<br>• `q` works in every view; from Completed and the Archive, tasks go to the Inbox.<br>• After adding, the status bar shows `Added "name".` and, when a project was created, `Created #x.` | 2026-11-12 | Known limitation: a word with trailing punctuation (`tomorrow.`) isn't recognised. |
| G8.1a | **Versioning**, pulled forward from G8.1: SemVer 2.0.0 released as git tags `vX.Y.Z`. The README shows a version badge from the latest tag. Until 1.0.0 the minor version tracks completed phases (first release `v0.5.0` = Phases 0–5) and fixes bump the patch. 1.0.0 is the Phase 8 release. | 2026-11-12 | The distribution parts of G8.1 are still open. |

---

## Appendix A. Quick-add grammar (G5.1, approved 2026-11-12)

Matching is case-insensitive and works on whole words. The examples assume today is **Tue 2026-11-10**, and weeks start on Monday.

### Tokens

| Field | Forms | Notes |
|---|---|---|
| Project | `#name` | `name` is `[A-Za-z0-9_-]+`, lowercased (`#Work` → `#work`). `#` must start a word, so `C#` is literal. Autocomplete and did-you-mean (G5.2) apply. |
| Priority | `p1`–`p4` bare, or `!p1`–`!p4` | Whole word. `p5` and `p0` stay as text. With no priority, the task is P4. |
| Duration | `~` + duration | `~30m`, `~30min`, `~30 minutes`, `~1.5h`, `~2h`, `~2 hours`, `~1h30m`, `~1h 30m`, `~90` (a bare number means minutes). Stored as whole minutes, rounded. `~` is required. |
| Start | `s:` + date phrase | |
| Due | `@` + date phrase, or a bare date phrase | Bare phrases are limited by G5.1f. |
| Literal | `\` + word | The backslash is removed and the word stays text (`\friday`, `\#1`). |

- **Repeated fields:** if a field appears more than once, the **last** occurrence counts and the earlier ones stay in the name.
- **Values:** a value is the longest valid phrase (G5.1b). A sigil with no valid value stays as text.
- **The name** is what's left once recognised tokens are removed, with whitespace collapsed. An empty name is an error ("Enter a task name.").

### Date phrases

- **ISO:** `2026-11-20`.
- **US numerics:** `11/20`, `11/20/2027`, `11/20/27`. A first number over 12 reads as day/month (`27/1`). Invalid dates (`2/30`) stay as text.
- **Relative days:** `today`, `tomorrow`, `yesterday`. The short forms `tod` and `tom` need a sigil.
- **Weekdays:** `monday`…`sunday` mean the next one after today (today's own weekday means next week's; use `today` for today). The short forms `mon`…`sun` need a sigil. `this friday` is the same as `friday`.
- **`next <weekday>`:** the one in the following week if this week's hasn't passed, otherwise the next upcoming one (Todoist's rule).
- **Periods:** `next week` is next Monday, `next month` the 1st of next month, `next year` January 1st. `weekend` and `this weekend` are the coming Saturday; `next weekend` is the Saturday of next week.
- **Offsets:** `in N days|weeks|months|years` (N is digits, or `a`/`an` for 1), `N days|weeks|months before|after <date>`.
- **Month and day:** `nov 20`, `november 20`, `20 nov`, `20th november`, `nov 20 2027`. A past date without a year means next year's. A lone month (`may`) is the 1st of that month and needs a sigil.
- **Ordinal day:** `27th` is the next 27th (this month if it hasn't passed).
- **Named points:** `mid <month>` is the 15th. `end of month` (or `eom`, sigil only) is the last day of this month. `end of <month>` is the last day of that month.
- **Clearing:** `no date` / `no due date` means no date. It's mainly for the edit dialogs.
- **Not recognised; stays in the name:** times (`at 4pm`, `16:00`, `4pm`; G5.1e) and recurring phrases (`every monday`, `daily`).

### Examples (Tue 2026-11-10)

| # | Input | Name | Project | Due | Start | Dur | Pri |
|---|---|---|---|---|---|---|---|
| 1 | `Change the laundry #oneofftasks @in 2 days ~30m !p2 s:tomorrow` (design sample) | Change the laundry | oneofftasks | 2026-11-12 | 2026-11-11 | 30 | P2 |
| 2 | `Buy milk` | Buy milk | Inbox | – | – | – | P4 |
| 3 | `Buy milk tomorrow` | Buy milk | Inbox | 2026-11-11 | – | – | P4 |
| 4 | `Call mom tomorrow at 4pm` | Call mom at 4pm | | 2026-11-11 | | | |
| 5 | `Submit report friday` | Submit report | | 2026-11-13 | | | |
| 6 | `Submit report next friday` | Submit report | | 2026-11-20 | | | |
| 7 | `Plan sprint next monday` | Plan sprint | | 2026-11-16 | | | |
| 8 | `Water plants tuesday` | Water plants | | 2026-11-17 | | | |
| 9 | `Clean garage next week` | Clean garage | | 2026-11-16 | | | |
| 10 | `Pay rent next month` | Pay rent | | 2026-12-01 | | | |
| 11 | `Hike this weekend` | Hike | | 2026-11-14 | | | |
| 12 | `Visit gran next weekend` | Visit gran | | 2026-11-21 | | | |
| 13 | `Renew passport in 3 weeks` | Renew passport | | 2026-12-01 | | | |
| 14 | `Dentist in a week` | Dentist | | 2026-11-17 | | | |
| 15 | `File taxes 2027-04-15` | File taxes | | 2027-04-15 | | | |
| 16 | `Gift for sam 11/20` | Gift for sam | | 2026-11-20 | | | |
| 17 | `Concert 27/1` | Concert | | 2027-01-27 | | | |
| 18 | `Party nov 20` | Party | | 2026-11-20 | | | |
| 19 | `Trip 20 november` | Trip | | 2026-11-20 | | | |
| 20 | `Renew lease jan 5` | Renew lease | | 2027-01-05 | | | |
| 21 | `Pay card 27th` | Pay card | | 2026-11-27 | | | |
| 22 | `Budget end of month` | Budget | | 2026-11-30 | | | |
| 23 | `Review mid december` | Review | | 2026-12-15 | | | |
| 24 | `Order cake 2 days before nov 20` | Order cake | | 2026-11-18 | | | |
| 25 | `Fix login bug p1` | Fix login bug | | – | | | P1 |
| 26 | `Fix login bug !p3` | Fix login bug | | | | | P3 |
| 27 | `Read ~1.5h` | Read | | | | 90 | |
| 28 | `Read ~1h30m` | Read | | | | 90 | |
| 29 | `Read ~2 hours` | Read | | | | 120 | |
| 30 | `Walk ~10 minutes` | Walk | | | | 10 | |
| 31 | `Prep slides s:monday @friday` | Prep slides | | 2026-11-13 | 2026-11-16 | | |
| 32 | `Watch Tomorrowland` | Watch Tomorrowland | | – | | | |
| 33 | `Buy sun cream` | Buy sun cream | | – | | | |
| 34 | `Run @sun` | Run | | 2026-11-15 | | | |
| 35 | `Call dad @tom` | Call dad | | 2026-11-11 | | | |
| 36 | `May 5k race` | May 5k race | | – | | | |
| 37 | `Pay may 5 bill` | Pay bill | | 2027-05-05 | | | |
| 38 | `Learn C# basics` | Learn C# basics | Inbox | | | | |
| 39 | `Email @ 5` | Email @ 5 | | – | | | |
| 40 | `Fix \friday bug` | Fix friday bug | | – | | | |
| 41 | `Plan #work #home` | Plan #work | home | | | | |
| 42 | `Groceries tomorrow friday` | Groceries tomorrow | | 2026-11-13 | | | |
| 43 | `Report @2026-02-30` | Report @2026-02-30 | | – | | | |
| 44 | `#Work standup` | standup | work | | | | |
| 45 | `p5 test` | p5 test | | | | | P4 |
| 46 | `#work` | (error: "Enter a task name.") | | | | | |

Blank cells mean the same as row 2: Inbox, no dates, no duration, P4. In a project's list, the project defaults to that project unless `#` is used (G5.4).
