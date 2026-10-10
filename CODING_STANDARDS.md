# Coding Standards

How code in lazyenv is written: the conventions every change follows, the pitfalls that have bitten us, and checklists for the change types that span many files. `just check` is the mechanical gate (formatting, `go vet`, `gocyclo` + `ineffassign`, race-enabled tests); this file covers what tooling can't check. For build and test commands see `just --list` and [CONTRIBUTING.md](CONTRIBUTING.md).

## Every change

### Conventions

- **English everywhere**: UI strings, status and error messages, comments, identifiers, docs.
- **Packages** live under `internal/`; the module exports nothing.
- **Errors** are wrapped with context: `fmt.Errorf("reading %s: %w", path, err)`.
- **Imports** come in three groups: stdlib, third-party, `github.com/lazynop/lazyenv/...`.
- **Tests** sit next to the code (`foo_test.go`, same package) and use testify (`assert`, `require`).
- **Charm v2 stack**: imports are `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`. v1 snippets (`github.com/charmbracelet/...`, `View() string` on a `tea.Model`) don't compile here.
- **Naming**: TUI sub-models end in `Model`, Bubble Tea messages in `Msg`.
- **Receivers**: the top-level `tea.Model`s (`App`, `ThemePreviewModel`) use value receivers; each handler takes a copy and returns the updated model. Helpers that mutate that copy in place (`flashMessage`, `updateLayout`) take `*App`. Sub-models (`FileListModel`, `VarListModel`, `DiffViewModel`, …) are not `tea.Model`s: `App` owns them and calls their pointer-receiver methods. Input is routed by `AppMode` first, then by `Focus` (files or vars panel).
- **Status-bar messages**: transient messages go through `a.flashMessage(msg)` or `a.flashError(msg)`, which set the text and schedule its auto-clear with the matching timeout. Call `a.statusBar.SetMessage` directly only for a message that must persist until the next state change (e.g. the reorder menu prompt).

### Architecture invariants

- **Round-trip fidelity**: an `EnvFile` keeps every original line as a `RawLine`; writes re-emit untouched lines as they were, so comments, blank lines and formatting survive edits.
- **Atomic writes**: every disk write goes through `parser.WriteFile` or `parser.WriteFileAtomic` (temp file in the same directory, fsync, rename).
- **Shell semantics**: with duplicate keys the last occurrence wins (`EnvFile.VarByKey`, diffs, session snapshots).
- **One background**: the terminal background is painted once, via `tea.View.BackgroundColor` in `App.View`, not per panel.

### Pitfalls

- **Nested ANSI with Lip Gloss**: render each styled segment on its own and concatenate the results; wrap a whole line in one `style.Render` only when it holds plain text. Wrapping already-rendered text lets the inner reset (`\e[0m`) cancel the outer style's foreground, which goes unnoticed on themes without a custom background. Reference: the non-cursor rows in `internal/tui/filelist.go` ("Style each segment individually").
- **Panel widths**: minimum panel widths come from named constants in `internal/config/config.go` (e.g. `config.FileListMinWidth`), in every TUI file, `theme_preview.go` included. A new minimum gets a new constant there.

## Checklists by change type

Each checklist is complete: a change of that type touches every item.

### Checklist: config option

1. `internal/config/config.go`: field on the right struct with a kebab-case `toml:"..."` tag (only `[secrets]` keys are snake_case), and its default in `DefaultConfig()`.
2. `internal/config/load.go` → `validate()`: validation, if the value can be invalid.
3. `main.go` → `cli` struct: Kong flag typed as a pointer (`*bool`, `*string`, `*int`) so `nil` means "not given"; add `negatable:""` to a bool that defaults to true.
4. `main.go` → `applyCLIOverrides()`: `if cli.X != nil { cfg.X = *cli.X }`.
5. `internal/config/config_test.go`: tests for the field.
6. `examples/config/full.toml`: the option, documented.
7. `docs/configuration.md`: the options table. For a flag, also both tables in `docs/usage.md` (flags; flag → config key).

### Checklist: theme color

Runtime precedence: explicit `[colors]` > `theme` preset > built-in dark/light defaults, picked from the terminal background (`tea.BackgroundColorMsg`).

1. `internal/config/config.go` → `ColorConfig`: field with `toml:"kebab-name,omitempty"`.
2. `internal/config/themes/themes.go` → `Colors`: the same field; the two structs stay aligned.
3. `internal/config/themes/*.go`: a value in **every** theme file.
4. `internal/config/theme.go` → `LookupTheme()` (field copy) and `resolveColors()` (`if colors.X != "" { base.X = colors.X }`).
5. `internal/config/load.go` → `validate()`: an entry in the color list.
6. `internal/tui/styles.go` → `BuildTheme()`: light/dark defaults via `resolve(colors.X, light, dark)` and the styles that use it.
7. `internal/tui/theme_preview.go`: the showcase list in `renderShowcase()`, plus `resolvedColors` / `resolveThemeColors()` if the preview renders with the color.
8. `internal/config/config_test.go` → `TestAllThemesHaveAllColors`: `assert.NotEmpty` for the field.
9. `examples/config/full.toml` and the colors table in `docs/configuration.md`.

### Checklist: scrollable view

For a new view, window or `AppMode` with scrollable content:

1. `internal/tui/app_mouse.go` → `handleMouseClick()`: a case for the mode, with hit detection.
2. `internal/tui/app_mouse.go` → `handleMouseWheel()`: a case for the mode.
3. The view's model: `SetCursor(index int)` for click positioning.

### Checklist: read-only guard

`--read-only` (`config.ReadOnly`) blocks every write while navigation, search, peek, sorting and toggles keep working. Guard the mutating keys only, with one of the existing patterns:

- **Mixed handler** (mutating and non-mutating keys in one switch): before the switch, route the mutating keys (`key.Matches(msg, a.keys.Edit, ...)`) to a sub-handler whose first statement is the `a.readOnlyFlash()` check. Examples: `handleNormalVarEdit`, `handleComparingEdit`.
- **Key-group guard**: `if a.config.ReadOnly && key.Matches(msg, ...) { return a, a.readOnlyFlash(), true }` ahead of the switch (see `handleNormalFileAction`).
- **Mutation-only entry point** (a flow that only writes, e.g. `startReorder`): the `readOnlyFlash()` check as its first statement.

Cover the new key with a `TestReadOnlyBlocks...` case in `internal/tui/app_readonly_test.go`.

### Checklist: SessionStats hooks

`SessionStats` (`internal/tui/session_stats.go`) prints a summary of disk-level changes on exit. Baseline is the disk state at startup, final is the disk state at quit; unsaved in-memory edits never appear. Every feature that writes, creates, renames or deletes a `.env` file calls the matching hook:

| Operation | Hook | Call it |
| --- | --- | --- |
| Save existing file | `RecordSave(f.Path, f.Vars)` | after `parser.WriteFile` succeeds, **before** the re-parse, with the vars just written |
| Create from scratch | `RecordCreateScratch(path, vars)` | after `parser.WriteFileAtomic` succeeds, before `finaliseNewFile` |
| Duplicate a file | `RecordCreateDuplicate(newPath, srcPath, srcVars)` | same |
| Template from a file | `RecordCreateTemplate(newPath, srcPath, keysOnlyVars)` | same, with every `Value` empty |
| Rename | `RecordRename(oldPath, newPath)` | after `os.Rename` succeeds |
| Delete | `RecordDelete(path)` | after `os.Remove` succeeds |
| Initial load | `RecordInitialLoad(f.Path, f.Vars)` | already wired in the `FilesLoadedMsg` handler |

- **Success path, right after the disk call**: place the hook after the disk operation's error return and before any post-processing that can fail (re-parse, `CheckGitIgnore`). The summary tracks the disk, not memory.
- **Nil-safe**: `a.sessionStats` is `nil` in read-only mode or with `session-summary = false`, and every `Record*` method handles a nil receiver, so call it unconditionally.
- **New kind of disk operation** (e.g. moving a file between directories): add a `Record*` method, handle it in `Summary()` / `formatLiveRow` / `formatDeletedRow`, and test it in `session_stats_test.go`.
- **New create path** outside `confirmCreateFile` / `confirmDuplicateFile` / `confirmTemplateFile`: pick the matching `createKind`, or add one.

### Checklist: renames the compiler can't see

`go build` and gopls track Go identifiers; names that live in strings are invisible to them. When renaming a TOML key, CLI flag, theme name, keybinding or user-facing message, search for the old string separately in:

- struct tags (`toml:"..."`, Kong `name:"..."`) and `key.WithKeys(...)` / help text in `internal/tui/keys.go`;
- tests and their fixtures;
- `examples/config/`, `justfile`, `docs/`, `README.md`.

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

- `docs/` is published as the docs site: every Markdown file in it goes public.
- `docs/changelog.md` is generated in CI from `CHANGELOG.md`; edit `CHANGELOG.md` only.
- When the TUI's look changes significantly, regenerate the README demo GIF from `docs/assets/demo.tape`; the run and compress commands are in its header (needs `vhs`, `ttyd`, `ffmpeg`, `gifsicle`).

## Versioning

The git tag (`vX.Y.Z`) is the only source of the version. GoReleaser injects it at link time (`-X main.version=...` in `.goreleaser.yml`); `main.go` keeps `version = "dev"` permanently, so any build that reports `dev` is visibly unreleased. Releases happen by tagging, never by editing that literal.
