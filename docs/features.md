# Features

## Browse and edit

Two-panel layout: file list on the left, variables on the right. Navigate with arrow keys or vim bindings (`hjkl`). Edit values inline, rename keys, add new variables, delete existing ones. A value with line breaks or other control characters is drawn on one row, with a visible stand-in for each: `↵` for a newline, `⇥` for a tab, `↩` for a carriage return, `␡` for DEL, the Unicode control pictures (`␀` … `␟`, `␛` for Escape) for the other C0 controls, and `�` for C1 controls and bytes that are not valid UTF-8. The file keeps the original characters.

## File management

Create, duplicate, rename, and delete `.env` files without leaving the TUI. Press `N` to create, `C` to duplicate, `R` to rename, or `D` to delete (with confirmation). Unsaved changes must be saved or reset before renaming or deleting.

Duplicate (`C`) copies the file on disk byte for byte, so unsaved edits in the source are not included. Template (`T`) writes the file as currently shown, unsaved edits included: comments, blank lines and order stay, while each variable becomes `KEY=`, its value and inline comment stripped — perfect for generating `.env.example` files.

!!! tip "Quick environment setup"
    Duplicate an existing file to bootstrap a new environment: `.env` → `.env.staging`, then compare and tweak the differences.

## Compare and sync

Side-by-side diff between any two `.env` files. Copy values in either direction, edit inline, and save — all without leaving the TUI. Copying from the side that lacks a key deletes the key from the other side, so both files match.

!!! tip "Jump between differences"
    Use `n` / `N` to jump to the next or previous difference without scrolling manually.

## Completeness matrix

Multi-file grid view showing which variables exist in which files. Spot missing entries at a glance, add them inline with `a`, or delete them from a specific file with `d`.

!!! example "Use case"
    You have `.env`, `.env.staging`, and `.env.production`. The matrix instantly shows you which variables are missing from which environment.

## Change tracking

Distinct indicators show the state of each variable:

| Marker | Meaning |
| ------ | ------- |
| `+` | Added |
| `*` | Modified |
| `-` | Deleted |
| `D` | Duplicate key |
| `○` | Empty value |
| `…` | Placeholder value |

Indicators are color-coded and [fully configurable](configuration.md#colors).

## Peek original values

Toggle with `p` to see what changed: while it is on, a modified variable under the cursor shows its original value on a `↳ was:` line below it, and a variable added this session shows `↳ new variable`. While secrets are masked, a secret's original value is masked too, until `Ctrl+S` reveals it. Other rows are unchanged.

## Clipboard support

Yank values or full `KEY=value` lines to the clipboard using OSC 52.

!!! tip "Works over SSH"
    OSC 52 clipboard works through SSH sessions and tmux — no local clipboard tool needed.

## Secret masking

Sensitive keys (passwords, tokens, secrets) are automatically detected and masked. Toggle visibility with `Ctrl+S`.

Detection uses key name patterns (e.g. `*_PASSWORD`, `*_TOKEN`, `*_SECRET`), known value prefixes (`sk-`, `ghp_`, `Bearer`), and entropy-based heuristic for random-looking values.

Detection runs again whenever a key or value changes (an edit, a rename, a copy in compare), so a variable is masked exactly as it will be after saving: a `ghp_` token pasted into a plain variable is masked, and a secret renamed to a plain key is shown unless its value looks like a secret too. The edit bar shows the value being edited in clear.

Customize detection with the [`[secrets]`](configuration.md#secrets) config section — mark keys as safe, add extra patterns, or disable the value heuristic entirely.

!!! warning "Detection is heuristic"
    Always review your files before sharing. Use `safe_patterns` and `extra_patterns` to fine-tune detection for your project.

## Gitignore check

Warns when `.env` files are not covered by `.gitignore`, so you don't accidentally commit secrets.

## Automatic backup

Before lazyenv first writes a file in a session, it copies it to a `.bak` file next to it (`.env` → `.env.bak`); later saves of the same file leave that backup alone, so each file gets one backup per session. Disable with `-B` or `no-backup = true` in config.

## Round-trip fidelity

Saves preserve comments, blank lines, quoting style, and key ordering — your files stay exactly as you wrote them.

## Search and sort

Filter variables by name or value with `/`: `Enter` keeps the filter, `Esc` clears it. While secrets are masked, the filter skips their values, so a secret matches only by name until `Ctrl+S` reveals it. Toggle between original position and alphabetical sorting with `o`.

## Variable grouping

Toggle prefix-based grouping with `g`. Variables sharing a non-empty prefix (everything before the first `_`) collapse under a header showing the prefix and group size:

```
▾ DB (4)
  DB_HOST     localhost
  DB_PORT     5432
  DB_USER     admin
  DB_NAME     app
▾ REDIS (2)
  REDIS_URL   redis://localhost
  REDIS_PORT  6379
▾ UNGROUPED (2)
  PORT        3000
  DEBUG       true
```

Press `Enter` or `Space` (or click) on any header — including `UNGROUPED` — to collapse it. The trailing `UNGROUPED` section holds variables with a unique prefix or no `_`; it only appears when at least one named group exists, so files where nothing groups stay in the linear view.

Groups follow file order by default. Toggling sort (`o`) reorders groups and variables within each group alphabetically; the `UNGROUPED` section stays pinned at the bottom. Search disables grouping in rendering and restores it when you clear the query. The toggle and per-group collapsed state live for the session — for a permanent on-disk reorder, use `O` (see below).

To start with grouping already enabled, set `group = true` in your config or pass `-g` / `--group` on the command line.

## Reorder on disk

While `o` only changes how variables are displayed, `O` rewrites the file on disk in a new order — like `gofmt` for `.env` files. Press `O`, choose a mode, and confirm:

- **`a` alphabetical** — variables sorted A→Z by key.
- **`g` grouped** — variables grouped by prefix (groups alphabetical, then by key within each group), mirroring the `g` view, with the ungrouped bucket last and a blank line between groups.

The reorder is **lossless for comments**: a comment block directly above a variable (no blank line between) travels with it, the file's leading and trailing comment blocks stay put, and stand-alone comments move up to just above the first variable. Blank lines between variables are not kept: alphabetical mode removes them, grouped mode leaves one between groups. The write goes through the same safe path as a manual save (a `.bak` backup on first write, atomic replace), so unsaved edits in the file are persisted as part of the reorder. Disabled in `--read-only` mode.

## Read-only mode

Disable all write operations with `--read-only` or `read-only = true` in config. Every key that would change a file is suppressed with a status bar notice: `e`, `E`, `a`, `d`, `w` and `O` in the main view, `N`, `C`, `D`, `R` and `T` in the file list, `←` / `→`, `e`, `E` and `w` in the diff view, and `a` / `d` in the matrix. A `[READ-ONLY]` badge is shown in the status bar. Navigation, search, sort, grouping, peek, yank, and browsing the diff and matrix views keep working.

Useful for safely inspecting production `.env` files without risk of accidental modifications.

## Session summary

When you exit lazyenv, a recap of disk-level changes is printed to stdout. It lists only files that actually changed on disk during the session — created, deleted, renamed, duplicated, templated, or with variables that differ from session start in value or in order — one line per file, sorted by path. A change of order, such as a reorder on disk, shows as `reordered`; adding or deleting a variable leaves the others in their order, so it is not a reorder.

Example:

```
Session summary:
  .env.backup — duplicated from .env.local, 0 added, 1 changed, 2 deleted
  .env.ci — reordered
  .env.dev (renamed from .env.staging) — 0 added, 2 changed, 0 deleted
  .env.fresh — new file (3 variables)
  .env.local — 2 added, 1 changed, 0 deleted
  .env.old — deleted
  .env.schema — from template .env.production (8 variables)
```

Paths include the directory you passed: with `lazyenv ./services`, a line reads `services/.env — …`. A file renamed several times shows a single rename from its original name, renaming a file back to its original name drops the rename, and a file created and then deleted in the same session is not listed. A duplicate counts from the source's contents on disk, so edits to the source not yet saved are not part of it.

Silent when nothing changed. Disable via `session-summary = false` in config or the `--no-session-summary` flag. `--read-only` disables it implicitly.

## Mouse support

Click to select files, variables, diff entries, and matrix cells. Scroll wheel navigates the panel under the mouse cursor. While an edit bar, a file-name bar, a prompt or a confirmation is open, the mouse is ignored, so it can't move the entry it acts on; during a search it keeps working. Disable with `--no-mouse` or `no-mouse = true` in config.

## Theme preview

Browse all built-in themes interactively with `lazyenv --themes`. The two-panel view shows every theme's full color palette — swatch, name, and hex value — updating live as you navigate. Press Enter to get the config snippet for your chosen theme.
