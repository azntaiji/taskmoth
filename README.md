# taskmoth

[![CI](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml/badge.svg)](https://github.com/azntaiji/taskmoth/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/azntaiji/taskmoth)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Ubuntu-lightgrey)

A keyboard-driven task manager for the terminal, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Status: in development.** The design system, widgets, data model and SQLite storage are done (phases 0–2 of [PLAN.md](PLAN.md)). The task list views arrive in Phase 3. Until then, `taskmoth` opens a placeholder screen.

## Features

Implemented:

- **Local SQLite database** in `~/.local/share/taskmoth` (or `$XDG_DATA_HOME/taskmoth`) on macOS and Linux.
- **Tasks** with a name, description, project, start date, due date, estimated duration and priority (P1–P4).
- **Projects** named as `#slugs`. Deleted tasks go to a **trash**.
- **Smart lists**:
  - **Today**: due, overdue, or already started.
  - **Upcoming**: starting or due in the next 7 days.
  - **Overdue**, **Inbox** and **Completed**.
- **Sorting** by name, start date, due date, priority or duration.
- **Themed UI**: colours follow the Figma design's ANSI-palette tokens (Dracula by default). There is a Unicode/ASCII glyph switch.

Planned (see [PLAN.md](PLAN.md)):

- The full list views, task editing dialogs, search, and a keyboard shortcuts overlay.
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

Developer flags:

```sh
taskmoth --gallery                       # widget gallery for comparing with the Figma designs
taskmoth --data /tmp/tm --seed           # load the sample data from the designs into an empty data dir
taskmoth --data /tmp/tm --list today     # print a view: today, upcoming, overdue, inbox,
                                         # completed, trash, or project:<name>
```

`--list` is temporary; it goes away once the list views exist.

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
