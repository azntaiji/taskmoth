# taskmoth

[![Version](https://img.shields.io/github/v/tag/azntaiji/taskmoth?sort=semver&label=version)](https://github.com/azntaiji/taskmoth/tags)
[![CI](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml/badge.svg)](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/azntaiji/taskmoth)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Ubuntu-lightgrey)

A keyboard-driven task manager for the terminal, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Status: in development.** You can add tasks with natural-language quick add, view and edit every field, and browse, sort, complete, archive and organize tasks across every list view (phases 0–6 of [PLAN.md](PLAN.md)). Search and the shortcuts overlay come next.

## Features

Implemented:

- **Natural-language quick add** (`q`), Todoist-style:
  - Example: `Change the laundry #chores tomorrow ~30m p2 s:monday`.
  - Recognised parts are highlighted as you type, with a preview line of the result.
  - Project names autocomplete, and a typo like `#chors` asks "Did you mean #chores?".
- **Task view and editing**:
  - `e` opens a task with its description.
  - Single keys edit any field, from the task view or straight from a list: name, description, project, start date, due date, duration and priority.
  - Dates and durations accept natural language (`next monday`, `1.5h`).
- **List views**:
  - **Today**: due, overdue, or already started.
  - **Upcoming**: the next 7 days, grouped by day.
  - **Overdue**, **Inbox**, **All**, **Completed**, the **Archive**, and one list per **project**.
- **Rows and status**:
  - Two-line task rows with priority and due badges, duration and start date.
  - A live clock in the header.
  - A status bar with task count, total and remaining time, and the current sort.
- **Projects** named as `#slugs`:
  - Pick one with `g p`, and create new ones there.
  - Archiving a project archives its tasks too, and unarchiving brings them back.
- **Archive instead of delete**: `x` archives a task, and `g x` shows the archive. Permanent deletion only happens from the archive, after a confirmation.
- **Keyboard driven**: arrows or `j`/`k`, `space` to complete, `s` to sort (each view remembers its own sort), and `g` chords to switch views.
- **Search** (`/`): matches words in any order across names, descriptions and project names, with `#project` and `!p1`–`!p4` filters. Results update as you type, best match first.
- **Keyboard Shortcuts** dialog (`?`), generated from the same key map as the hint rows.
- **Completed tasks stay visible** (dimmed) until the end of the day.
- **Local SQLite database** in `~/.local/share/taskmoth` (or `$XDG_DATA_HOME/taskmoth`) on macOS and Linux.
- **Tasks** with a name, description, project, start date, due date, estimated duration and priority (P1–P4).
- **Themed UI**: colours follow the Figma design's ANSI-palette tokens (Dracula by default). There is a Unicode/ASCII glyph switch.

Planned (see [PLAN.md](PLAN.md)):

- Recurring tasks, times of day, user themes, time tracking, analytics.

## Install

From source (Go 1.26+):

```sh
go install github.com/azntaiji/taskmoth/cmd/taskmoth@latest
```

## Usage

```sh
taskmoth                    # start the app
taskmoth --data DIR         # use DIR for the database instead of the default
taskmoth --ascii            # ASCII symbols for minimal fonts/terminals (or TASKMOTH_ASCII=1)
```

Quit with `ctrl+q` or `ctrl+c` (`q` is "add task"). Press `?` for every key.

| Key | Action |
|---|---|
| `↑`/`↓`, `k`/`j` | move the selection |
| `pgup`/`pgdn` | move by a page |
| `home`/`end`, `g g`/`G` | jump to top/bottom |
| `q` | add a task (quick add; see below) |
| `e` | open the selected task (`esc` returns to the list) |
| `n` `d` `v` `r` `t` `y` `p` | edit name, description, project, start date, due date, duration, priority |
| `space` | complete / reopen the selected task |
| `s` | cycle the sort (priority → due → start → duration → name) |
| `x` | archive the selected task (in the Archive: delete forever) |
| `u` | unarchive (in the Archive and in Goto Project) |
| `g t` `g i` `g u` `g o` `g a` `g c` `g x` | Today, Inbox, Upcoming, Overdue, All, Completed, Archive |
| `g p` | Goto Project: `enter` open, `n` new project, `x` archive, `u` unarchive |
| `/` | search (see below) |
| `?` | keyboard shortcuts (`esc` or `?` closes) |
| `esc` | cancel a pending `g`, close a dialog, or go back |

Dialogs use `←`/`→` to choose a button and `enter` to confirm. In edit dialogs, an empty value clears the field (`no date` also clears a date), and Edit Project's "Inbox" entry removes the project. In Edit Description, `shift+enter` (where the terminal supports it), `alt+enter` or `ctrl+j` adds a line break.

### Search

`/` opens Search with the query input in the status bar. Every word must appear, in any order, in a task's name, description or project name; `#work` keeps tasks in projects starting with "work", `#inbox` keeps tasks with no project, and `!p1`–`!p4` filter by priority. Archived tasks aren't searched; completed ones are listed last.

While typing, `↑`/`↓` move the selection. `enter` moves to the results, where the usual list keys work (`e`, `space`, `x`, `p`, …); `/` or `esc` returns to the input, and `esc` there goes back to the previous view. The last query is kept until you quit.

### Quick add

Type the task name. Anything taskmoth recognises is highlighted and shown in the preview line.

| Write | Sets |
|---|---|
| `tomorrow`, `friday`, `next week`, `in 3 days`, `nov 20`, `27th`, `end of month`, `2 days before nov 20`, `11/20`, `2027-04-15` | due date (the `@` prefix is optional: `@tom`, `@sun`) |
| `s:monday`, `s:tomorrow` | start date |
| `#project` | project (`tab` completes; an unknown name asks before creating it) |
| `p1`–`p4` or `!p1`–`!p4` | priority (default P4) |
| `~30m`, `~1.5h`, `~1h30m`, `~2 hours` | estimated duration |

If something is recognised by mistake ("Pay may 5 bill"), put the cursor on it and press `ctrl+t` to keep it as text, or type `\` before the word. Times and repeats (`at 4pm`, `every monday`) stay in the name. The full grammar is [PLAN.md Appendix A](PLAN.md#appendix-a-quick-add-grammar-g51-approved-2026-11-12).

## Development

Requirements: Go 1.26+ and make. There is no cgo; SQLite is pure Go (`modernc.org/sqlite`).

```sh
make build     # bin/taskmoth
make run       # go run ./cmd/taskmoth (ARGS="--data /tmp/tm")
make gallery   # widget gallery
make test      # go test ./...
make golden    # rewrite golden render files after an intended visual change
make lint      # golangci-lint v2 (fetched with go run)
make check     # build + vet + test + lint: must pass before a change is done
```

The code is laid out as:

```
cmd/taskmoth/        entry point and flags
internal/task/       domain model, view filters, sorting (pure; no UI or storage)
internal/nlp/        quick-add parser, date phrases, durations (pure)
internal/store/      SQLite store, migrations, in-memory store for tests
internal/seed/       sample data from the designs
internal/ui/view/    list views (Today, Inbox, …)
internal/ui/theme/   colour tokens (palette → semantic), dimmed theme, glyphs
internal/ui/widget/  widgets built from the Figma Components page
internal/ui/gallery/ developer gallery
```

### Contributing

- Read [AGENTS.md](AGENTS.md) for working conventions and [PLAN.md](PLAN.md) for the phased plan. Any change to user-visible behaviour, the data format or key bindings is a decision that gets recorded in the PLAN.md decision log.
- The UI is implemented from Figma designs, and each screen's node is listed in PLAN.md §3. Colours come only from `theme.Colors`, and glyphs only from `theme.Glyphs`.
- Changes to the database schema go in a new numbered file in `internal/store/migrations/`. Never edit a released migration.
- CI runs build, vet, race-enabled tests and lint on macOS and Ubuntu.
- Versions follow [Semantic Versioning 2.0.0](https://semver.org/) and are released as git tags (`vX.Y.Z`); the version badge shows the latest tag. Until 1.0.0, each completed phase bumps the minor version (Phase 5 → 0.5.0) and fixes bump the patch.

## License

[MIT](LICENSE)
