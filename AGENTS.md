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
- **Versioning follows Semantic Versioning 2.0.0, released as git tags `vX.Y.Z`.** The README's version badge reads the latest tag, so never hardcode a version in README.md.
  - **Until 1.0.0:** each completed phase bumps the minor version (Phase 6 → `v0.6.0`), and bug fixes bump the patch (`v0.6.1`). 1.0.0 is the Phase 8 release.
  - **Tagging:** after a phase or fix is committed and pushed, propose the next tag to the operator. Create and push it (`git tag -a vX.Y.Z -m …` then `git push origin vX.Y.Z`) only when they confirm. Never move or delete an existing tag. Pushing a tag publishes a public GitHub Release (`release.yml`), so run `make release-check` before proposing one.

## Layout

```
cmd/taskmoth/        entry point and flags (--data, --ascii, --version; hidden: --gallery, --seed, --list)
internal/paths/      XDG data/config directory resolution
internal/lock/       single-instance lock on the data directory (flock; G8.4)
internal/task/       domain model, validation, view filters, sort orders (pure)
internal/nlp/        quick-add parser (PLAN.md Appendix A), ParseDate, ParseDuration (pure)
internal/store/      Store interface; SQLite implementation + embedded migrations, integrity
                     check (G8.5) and pre-migration backups (G8.2);
                     in-memory implementation for UI tests (both pass the same contract tests)
internal/seed/       sample data from the designs (taskmoth --seed)
internal/ui/         root Bubble Tea model (app.go): store, clock, key chords, view switching;
                     dialogs.go: dialog stack, Goto Project, Create New Project, confirmations;
                     addtask.go: Add Task (quick add, autocomplete, did-you-mean);
                     taskview.go: Single Task view; edit.go: the field edit dialogs;
                     keymap.go: the key map (dispatch, hint rows and the shortcuts dialog all
                     come from it); search.go: the Search view; shortcuts.go: Keyboard Shortcuts
internal/ui/view/    list views: one configurable List + a Spec per view (Today, Inbox, Upcoming,
                     Overdue, All, Completed, Archive, projects); duration/clock/day formatting
internal/ui/theme/   Figma colour tokens, the dimmed theme, glyph sets (Unicode/ASCII)
internal/ui/widget/  widgets built from the Figma Components page (header, status bar,
                     key hints, badges, checkbox, marker, scrollbar, frame, input, overlay)
internal/ui/gallery/ dev gallery (taskmoth --gallery) for comparing widgets with Figma
```

## UI conventions

- **Colours follow Figma's two layers.** `theme.Palette` (`palette.go`) holds the 16 ANSI slots, and `theme.Semantic` (`colors.go`) maps each semantic token to a slot, mirroring Figma's aliases. Widgets and views use only `theme.Colors` fields, each named after its Figma semantic variable. They never use palette slots or hex values directly. To add a token, read its alias in Figma, add the field and its slot in `Semantic`, then add its expected value in `TestSemanticMatchesFigma`. `get_variable_defs` shows only resolved hex; read the alias with a read-only `use_figma` script.
- **Glyphs come from `theme.Glyphs`.** Never write `▶`, `│`, `…` and so on directly, so that `--ascii` / `TASKMOTH_ASCII=1` keeps working.
- **Dimming:** draw the view behind a dialog with `theme.Dimmed()`, then composite it with `widget.Overlay`. A component's own "dim" variant (completed tasks, the header clock) uses `theme.Fade` / `FadeOn`.
- **Widths are terminal cells, not bytes or runes.** One Figma cell is 12×24 px. Truncate plain text with `widget.Truncate` before styling it, and lay out with `Fit` / `Spread`.
- **Every rendered line must be exactly the target width.** The golden tests assert this.
- **Keys live in the key map** (`internal/ui/keymap.go`). Add a list key there, with its hint and shortcuts-dialog labels, rather than in a `switch`; hint rows are built with `hintFor`/`listHints`, and `TestHintsMatchKeyMap` checks them.
- **The quick-add grammar is PLAN.md Appendix A.** `TestAppendixA` is that table row for row. Change the grammar only through a gate, and update the appendix and the test together.

## Data conventions

- **Schema changes are new migrations.** Add `internal/store/migrations/NNNN_name.sql`, numbered consecutively. Never edit a migration that has shipped. Changing the schema is a gate (decision G2.4 set version 1).
- **Every `Store` implementation must pass the contract tests** in `internal/store/store_test.go`. Add new behaviour there, so SQLite and Memory stay in sync.
- **Dates and times:** dates are `task.Date` (local, `YYYY-MM-DD`). Timestamps are stored as UTC. Never call `time.Now()` in `task`; pass `today` in. The store's clock is injectable through `store.Options`.
- **Don't experiment on the real database.** For `--seed` and `--list`, always pass `--data` pointing to a scratch directory.

PLAN.md §4 describes the full target layout.

## Commands

```
make build   # bin/taskmoth
make run     # go run ./cmd/taskmoth  (ARGS="--data /tmp/tm")
make gallery # widget gallery; tab / shift+tab between pages, ctrl+q quits
make test    # go test ./...
make golden  # rewrite golden files after an intended visual change, then review the diff
make vet
make lint    # golangci-lint v2 via go run (no install needed)
make fmt
make check   # build + vet + test + lint (must pass before a phase is done)
make release-check # goreleaser check + a snapshot build of every release archive into dist/
```

CI (`.github/workflows/ci.yml`) runs build, vet, and race-enabled tests on `ubuntu-latest` and `macos-latest`, plus lint. Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which tests and then publishes a GitHub Release with GoReleaser (`.goreleaser.yaml`). Pinned tool versions: golangci-lint and GoReleaser in the Makefile, GoReleaser again in `release.yml`; keep them in step.

## Before reporting work done

- `make check` passes.
- Every gate you resolved is in the PLAN.md decision log.
- AGENTS.md (layout and commands) and PLAN.md are updated if anything they describe changed.
- README.md is kept up-to-date with: Shields badges, an overview of the application, key features, commands, and a section for developers/contributing. 
