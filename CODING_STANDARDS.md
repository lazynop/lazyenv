# Coding Standards

How code in lazyenv is written: the conventions every change follows, the pitfalls that have bitten us, and checklists for the change types that span many files. `just check` is the mechanical gate (gofmt, gci import grouping, `go vet`, `gocyclo`, `ineffassign`, `deadcode`, `exhaustive`, race-enabled tests); this file covers what tooling can't check, and names the test that guards a rule where one exists. For build and test commands see `just --list` and [CONTRIBUTING.md](CONTRIBUTING.md).

## Every change

### Conventions

- **English everywhere**: UI strings, status and error messages, comments, identifiers, docs.
- **Packages** live under `internal/`; the module exports nothing.
- **Errors** are wrapped with context: `fmt.Errorf("reading %s: %w", path, err)`.
- **Tests** sit next to the code (`foo_test.go`, same package) and use testify (`assert`, `require`).
- **Charm v2 stack**: imports are `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`. v1 snippets (`github.com/charmbracelet/...`, `View() string` on a `tea.Model`) don't compile here.
- **Naming**: TUI sub-models end in `Model`, Bubble Tea messages in `Msg`.
- **Receivers**: the top-level `tea.Model`s (`App`, `ThemePreviewModel`) use value receivers; each handler takes a copy and returns the updated model. Helpers that mutate that copy in place (`flashMessage`, `updateLayout`) take `*App`. Sub-models (`FileListModel`, `VarListModel`, `DiffViewModel`, …) are not `tea.Model`s: `App` owns them and calls their pointer-receiver methods. Input is routed by `AppMode` first, then by `Focus` (files or vars panel).
- **Status-bar messages**: transient messages go through `a.flashMessage(msg)` or `a.flashError(msg)`, which set the text and schedule its auto-clear with the matching timeout. Call `a.statusBar.SetMessage` directly only for a message that must persist until the next state change (e.g. the reorder menu prompt).

### Architecture invariants

- **Round-trip fidelity**: an `EnvFile` keeps every original line as a `RawLine`; writes re-emit untouched lines as they were, so comments, blank lines and formatting survive edits.
- **Atomic writes**: every disk write goes through `parser.WriteFile` or `parser.WriteFileAtomic` (temp file in the same directory, fsync, rename).
- **Re-parse via `reparse`**: an in-memory file is refreshed from disk only through `reparse` (`internal/tui/app_reparse.go`), which carries over runtime-only state such as `GitWarning`. `TestParseFileOnlyThroughReparse` fails on a direct `parser.ParseFile` elsewhere in `internal/tui`.
- **Shell semantics**: with duplicate keys the last occurrence wins (`EnvFile.VarByKey`, diffs, session snapshots).
- **One background**: the terminal background is painted once, via `tea.View.BackgroundColor` in `App.View`, not per panel.

### Pitfalls

- **Nested ANSI with Lip Gloss**: render each styled segment on its own and concatenate the results; wrap a whole line in one `style.Render` only when it holds plain text. Wrapping already-rendered text lets the inner reset (`\e[0m`) cancel the outer style's foreground, which goes unnoticed on themes without a custom background. Reference: the non-cursor rows in `internal/tui/filelist.go` ("Style each segment individually").
- **Panel widths**: minimum panel widths come from named constants in `internal/config/config.go` (e.g. `config.FileListMinWidth`), in every TUI file, `theme_preview.go` included. A new minimum gets a new constant there, and an entry in `layoutMinimums` (`docs_test.go`) so the minimum `docs/configuration.md` states is checked.
- **Panel chrome**: visible rows and columns, and the row offsets mouse clicks are mapped with, come from the chrome constants in `internal/tui/chrome.go` (`panelChromeHeight`, `panelContentOffsetY`, …), never from raw numbers. A constant counts only rows that are actually drawn: the compare view once reserved a header row it never rendered, which wasted panel height and shifted clicks by one row.

## Checklists by change type

Each checklist lists the steps only you can remember. The steps a test or lint catches are named below the list: those checks fail until the step is done.

### Checklist: config option

1. `internal/config/config.go`: field on the right struct with a kebab-case `toml:"..."` tag (only `[secrets]` keys are snake_case), and its default in `DefaultConfig()`.
2. `internal/config/load.go` → `validate()`: validation, if the value can be invalid.
3. `main.go` → `cli` struct: Kong flag typed as a pointer (`*bool`, `*string`, `*int`) so `nil` means "not given"; add `negatable:""` to a bool that defaults to true.
4. `main.go` → `applyCLIOverrides()`: `if cli.X != nil { cfg.X = *cli.X }`.
5. `internal/config/config_test.go`: tests for the field.

Caught by `TestExampleDocumentsEveryKey`: the key documented in `examples/config/full.toml`. Caught by `TestConfigurationDocumentsDefaults`: the option's row, with its default, in a `docs/configuration.md` options table. Caught by `TestUsageDocumentsEveryFlag`: a flag's rows in both `docs/usage.md` tables (flags; flag → config key).

### Checklist: theme color

Runtime precedence: explicit `[colors]` > `theme` preset > built-in dark/light defaults, picked from the terminal background (`tea.BackgroundColorMsg`).

1. `internal/config/config.go` → `ColorConfig`: field with `toml:"kebab-name,omitempty"`.
2. `internal/tui/styles.go` → `BuildTheme()`: light/dark defaults via `resolve(colors.X, light, dark)` and the styles that use it.
3. `internal/tui/theme_preview.go`: the showcase list in `renderShowcase()`, plus `resolvedColors` / `resolveThemeColors()` if the preview renders with the color.

Caught by the build (`LookupTheme` converts one struct into the other): the same field at the same position in `internal/config/themes/themes.go` → `Colors`. Caught by `TestColorFieldsWiredEverywhere`: a valid value in every theme file (`internal/config/themes/*.go`), the override in `resolveColors()` and the entry in the `validate()` color list (`internal/config/load.go`). Caught by `TestExampleDocumentsEveryKey`: the key in `examples/config/full.toml`. Caught by `TestConfigurationDocumentsThemeColors`: the row in the `docs/configuration.md` colors table, with the default-dark and default-light values.

### Checklist: scrollable view

For a new view, window or `AppMode` with scrollable content:

1. `internal/tui/app_mouse.go`: move the mode out of the no-op cases into its own case, with hit detection in `handleMouseClick()` and scrolling in `handleMouseWheel()`.
2. The view's model: `SetCursor(index int)` for click positioning.

Caught by `just lint` (`exhaustive`): both switches name every `AppMode`, so a new mode fails until it is placed in one of their cases.

### Checklist: read-only guard

`--read-only` (`config.ReadOnly`) blocks every write while navigation, search, peek, sorting and toggles keep working. Guard the mutating keys only, with one of the existing patterns:

- **Mixed handler** (mutating and non-mutating keys in one switch): before the switch, route the mutating keys (`key.Matches(msg, a.keys.Edit, ...)`) to a sub-handler whose first statement is the `a.readOnlyFlash()` check. Examples: `handleNormalVarEdit`, `handleComparingEdit`.
- **Key-group guard**: `if a.config.ReadOnly && key.Matches(msg, ...) { return a, a.readOnlyFlash(), true }` ahead of the switch (see `handleNormalFileAction`).
- **Mutation-only entry point** (a flow that only writes, e.g. `startReorder`): the `readOnlyFlash()` check as its first statement.

Caught by `TestReadOnlyNoKeyChangesFiles` (`internal/tui/app_readonly_property_test.go`): it presses every `KeyMap` binding in every reachable mode under `--read-only` and fails if a file changes on disk or in memory. A new mode a key can reach fails it until classified there: in `mutatingModes`, or as a start state in `readOnlyStartStates`.

### Checklist: SessionStats hooks

`SessionStats` (`internal/tui/session_stats.go`) prints a summary of disk-level changes on exit. Baseline is the disk state at startup, final is the disk state at quit; unsaved in-memory edits never appear. Every write, create, rename or delete of a `.env` file goes through the `App` methods in `internal/tui/app_disk.go` (`saveFile`, `createFile`, `duplicateFile`, `templateFile`, `renameFile`, `deleteFile`), which call the matching `Record*` hook right after the disk call succeeds. The initial load is recorded by the `FilesLoadedMsg` handler (`RecordInitialLoad`).

Caught by `TestDiskWritesGoThroughChokepoint`: a direct `parser.WriteFile`, `parser.WriteFileAtomic` or `parser.CreateBackup` (allowed only in `backupIfNeeded`, since backups are not tracked), or an `os` function that creates, writes, truncates, renames, links or deletes files (`os.WriteFile`, `os.Create`, `os.OpenFile`, `os.Rename`, `os.Remove`, `os.RemoveAll`, …; the full list is in the test), anywhere else in `internal/tui`. `os.OpenFile` counts even for reads: read with `os.Open` or `os.ReadFile`.

- **Nil-safe**: `a.sessionStats` is `nil` in read-only mode or with `session-summary = false`, and every `Record*` method handles a nil receiver, so call it unconditionally.
- **New kind of disk operation** (e.g. moving a file between directories): add an `App` method in `app_disk.go` and a `Record*` method, handle it in `Summary()` / `formatLiveRow` / `formatDeletedRow`, and test it in `session_stats_test.go`.
- **New create path** outside `confirmCreateFile` / `confirmDuplicateFile` / `confirmTemplateFile`: pick the matching `createKind`, or add one.

### Checklist: renames the compiler can't see

`go build` and gopls track Go identifiers; names that live in strings are invisible to them. When renaming a TOML key, CLI flag, theme name, keybinding or user-facing message, search for the old string separately in:

- struct tags (`toml:"..."`, Kong `name:"..."`) and `key.WithKeys(...)` / help text in `internal/tui/keys.go`;
- tests and their fixtures;
- `examples/config/`, `justfile`, `docs/`, `README.md`.

Caught by `TestUsageDocumentsEveryFlag` and `TestKeybindingsDocumentsEveryKey`: a flag or key still documented under its old name in `docs/usage.md` or `docs/keybindings.md`.

## Documentation map

| File | Holds | Update when |
| --- | --- | --- |
| `README.md` | Project showcase, links to the docs site | The headline feature list changes |
| `CHANGELOG.md` | Release history, in English | At release time; product changes only, not CI or tooling |
| `docs/usage.md` | CLI flags, flag → config key map, `just` recipes | A flag or recipe changes |
| `docs/features.md` | Detailed feature descriptions | A feature is added or changed |
| `docs/keybindings.md` | Hotkeys per screen | A keybinding is added or changed |
| `docs/configuration.md` | TOML options, themes, colors, layout | An option, theme or color is added |
| `docs/index.md` | Docs landing page | A major feature lands |
| `docs/installation.md` | Install instructions | Requirements or distribution channels change |
| `CODING_STANDARDS.md` | This file | A convention, pitfall or checklist changes |
| `examples/config/full.toml` | Reference config users copy; its comments state defaults and limits | An option, default or limit changes |
| `viewHelp` in `internal/tui/app_views.go` | In-app help screen | A key, or the screens it works in, changes |
| `modeHints` in `internal/tui/statusbar.go` | Status-bar key hints per mode | A key or a mode changes |
| `help:` tags of the `cli` struct in `main.go` | `--help` output | A flag changes |

- **A behaviour change updates every surface above that describes the old behaviour, in the same change.** Find them by searching the surfaces for the terms the change touches (the key, flag, option, message, default), not only through the "Update when" column: a description of the old behaviour can sit anywhere.
- `docs/` is published as the docs site, and every Markdown file in it goes public. `.github/workflows/docs.yml` builds it from release tags only, so the site describes the latest release: a doc change on `main` goes live with the release that ships its code.
- `docs_test.go` checks the docs' facts against the code: flags (`TestUsageDocumentsEveryFlag`), keys (`TestKeybindingsDocumentsEveryKey`, `TestHelpScreenNamesBoundKeys`), options, defaults and minimums (`TestConfigurationDocumentsDefaults`), default colors (`TestConfigurationDocumentsThemeColors`), the theme count and Go version (`TestDocsStateCurrentCounts`), and the code spans in this file (`TestCodingStandardsNamesExistingCode`). A span here that names something outside the repo on purpose goes in that test's `notInRepo` list. Descriptions of behaviour are beyond any test: the rule above covers them.
- `docs/changelog.md` is generated in CI from `CHANGELOG.md`; edit `CHANGELOG.md` only.
- When the TUI's look changes significantly, regenerate the README demo GIF with `just demo` and the project page media with `just page-media`, then copy `dist/page-media/` into lazynop.dev's `site/lazyenv/media/`.

## Versioning

The git tag (`vX.Y.Z`) is the only source of the version. GoReleaser injects it at link time (`-X main.version=...` in `.goreleaser.yml`); `main.go` keeps `version = "dev"` permanently (`TestVersionIsDevSentinel` enforces it), so any build that reports `dev` is visibly unreleased. Releases happen by tagging, never by editing that literal.
