# taskmoth

[![CI](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml/badge.svg)](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/azntaiji/taskmoth)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Ubuntu-lightgrey)

A keyboard-driven task manager for the terminal, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Status: in development.** All the list views, projects and the archive work: you can browse, sort, complete, archive and organize tasks (phases 0–4 of [PLAN.md](PLAN.md)). Adding and editing tasks, and search, come in later phases. Until then, try it with the sample data (see *Developer flags*).

## Features

Implemented:

- **List views**:
  - **Today**: due, overdue, or already started.
  - **Upcoming**: the next 7 days, grouped by day.
  - **Overdue**, **Inbox**, **All**, **Completed**, the **Archive**, and one list per **project**.
- **Rows and status**:
  - Two-line task rows with priority and due badges; the Inbox uses one-line rows.
  - A live clock in the header.
  - A status bar with task count, total and remaining time, and the current sort.
- **Projects** named as `#slugs`:
  - Pick one with `g p`, and create new ones there.
  - Archiving a project archives its tasks too, and unarchiving brings them back.
- **Archive instead of delete**: `x` archives a task, and `g x` shows the archive. Permanent deletion only happens from the archive, after a confirmation.
- **Keyboard driven**: arrows or `j`/`k`, `space` to complete, `s` to sort (each view remembers its own sort), and `g` chords to switch views.
- **Completed tasks stay visible** (dimmed) until the end of the day.
- **Local SQLite database** in `~/.local/share/taskmoth` (or `$XDG_DATA_HOME/taskmoth`) on macOS and Linux.
- **Tasks** with a name, description, project, start date, due date, estimated duration and priority (P1–P4).
- **Themed UI**: colours follow the Figma design's ANSI-palette tokens (Dracula by default). There is a Unicode/ASCII glyph switch.

Planned (see [PLAN.md](PLAN.md)):

- Adding and editing tasks, search, and a keyboard shortcuts overlay.
- **Natural-language quick add**, e.g. `Change the laundry #chores @in 2 days ~30m !p2 s:tomorrow`.
- Later: user themes, time tracking, analytics.

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

Quit with `ctrl+q` or `ctrl+c`. (`q` will be "add task".)

| Key | Action |
|---|---|
| `↑`/`↓`, `k`/`j` | move the selection |
| `pgup`/`pgdn` | move by a page |
| `home`/`end`, `g g`/`G` | jump to top/bottom |
| `space` | complete / reopen the selected task |
| `s` | cycle the sort (priority → due → start → duration → name) |
| `x` | archive the selected task (in the Archive: delete forever) |
| `u` | unarchive (in the Archive and in Goto Project) |
| `g t` `g i` `g u` `g o` `g a` `g c` `g x` | Today, Inbox, Upcoming, Overdue, All, Completed, Archive |
| `g p` | Goto Project: `enter` open, `n` new project, `x` archive, `u` unarchive |
| `esc` | cancel a pending `g` or close a dialog |

Dialogs use `←`/`→` to choose a button and `enter` to confirm. The bottom hint rows also show keys from the designs that later phases will make work (`q`, `e`, `?`, `/`).

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
- Versions follow [Semantic Versioning 2.0.0](https://semver.org/).

## License

[MIT](LICENSE)
