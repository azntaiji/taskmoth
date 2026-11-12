# AGENTS.md

taskmoth is a keyboard-driven TUI task manager in Go (Bubble Tea v2, Lip Gloss v2, Bubbles v2, SQLite via `modernc.org/sqlite`). It must work on macOS and Ubuntu.

`PLAN.md` is the source of truth for scope, phases, and decisions. Read it before starting work.

## Rules

- **Work one phase at a time.** Don't start a phase until the operator approves the previous one.
- **Gates are hard stops.** Ask the operator every `G*` question for a phase before writing that phase's code, and record each answer in PLAN.md §8 (Decision log). Treat any new decision that affects user-visible behaviour, the data format, or key bindings as a gate.
- **Use Figma first for UI work.** Confirm the Figma MCP is available and stop if it isn't. Pull `get_design_context` for the node listed in PLAN.md §3 before building a screen. File key: `uRWYa0RRojip50i2lWNdKx`.
- **Don't expand scope.** Time tracking and analytics are future features (PLAN.md §7).
- **Ship tests with code.** Keep the pure packages (`task`, `nlp`) free of UI and storage imports, and pass the current time in as a parameter rather than calling `time.Now()`.
- **Quit keys are `ctrl+c` and `ctrl+q`.** `q` is "add task".

## Layout

```
cmd/taskmoth/      entry point and flags (--data)
internal/paths/    XDG data/config directory resolution
internal/ui/       Bubble Tea models (root App in app.go)
```

PLAN.md §4 describes the full target layout.

## Commands

```
make build   # bin/taskmoth
make run     # go run ./cmd/taskmoth  (ARGS="--data /tmp/tm")
make test    # go test ./...
make vet
make lint    # golangci-lint v2 via go run (no install needed)
make fmt
make check   # build + vet + test + lint (must pass before a phase is done)
```

CI (`.github/workflows/ci.yml`) runs build, vet, and race-enabled tests on `ubuntu-latest` and `macos-latest`, plus lint.

## Before reporting work done

- `make check` passes.
- Every gate you resolved is in the PLAN.md decision log.
- AGENTS.md (layout and commands) and PLAN.md are updated if anything they describe changed.
