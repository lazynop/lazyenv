# Features

## Browse and edit

Two-panel layout: file list on the left, variables on the right. Navigate with arrow keys or vim bindings (`hjkl`). Edit values inline, rename keys, add new variables, delete existing ones.

## File management

Create, duplicate, rename, and delete `.env` files without leaving the TUI. Press `N` to create, `C` to duplicate, `R` to rename, or `D` to delete (with confirmation). Unsaved changes must be saved or reset before renaming or deleting.

Both duplicate (`C`) and template (`T`) preserve comments, blank lines, and file structure. Duplicate copies values as-is; template strips them, perfect for generating `.env.example` files.

!!! tip "Quick environment setup"
    Duplicate an existing file to bootstrap a new environment: `.env` → `.env.staging`, then compare and tweak the differences.

## Compare and sync

Side-by-side diff between any two `.env` files. Copy values in either direction, edit inline, and save — all without leaving the TUI.

!!! tip "Jump between differences"
    Use `n` / `N` to jump to the next or previous difference without scrolling manually.

## Completeness matrix

Multi-file grid view showing which variables exist in which files. Spot missing entries at a glance and add them inline.

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

Toggle inline display of the original value before your edits, so you always know what changed.

## Clipboard support

Yank values or full `KEY=value` lines to the clipboard using OSC 52.

!!! tip "Works over SSH"
    OSC 52 clipboard works through SSH sessions and tmux — no local clipboard tool needed.

## Secret masking

Sensitive keys (passwords, tokens, secrets) are automatically detected and masked. Toggle visibility with `Ctrl+S`.

Detection uses key name patterns (e.g. `*_PASSWORD`, `*_TOKEN`, `*_SECRET`), known value prefixes (`sk-`, `ghp_`, `Bearer`), and entropy-based heuristic for random-looking values.

Customize detection with the [`[secrets]`](configuration.md#secrets) config section — mark keys as safe, add extra patterns, or disable the value heuristic entirely.

!!! warning "Detection is heuristic"
    Always review your files before sharing. Use `safe_patterns` and `extra_patterns` to fine-tune detection for your project.

## Gitignore check

Warns when `.env` files are not covered by `.gitignore`, so you don't accidentally commit secrets.

## Automatic backup

Creates a `.bak` copy before the first save of each session. Disable with `-B` or `no-backup = true` in config.

## Round-trip fidelity

Saves preserve comments, blank lines, quoting style, and key ordering — your files stay exactly as you wrote them.

## Search and sort

Filter variables by name or value with `/`. Toggle between original position and alphabetical sorting with `o`.

## Mouse support

Click to select files, variables, diff entries, and matrix cells. Scroll wheel navigates the panel under the mouse cursor. Disable with `--no-mouse` or `no-mouse = true` in config.

## Theme preview

Browse all built-in themes interactively with `lazyenv --themes`. The two-panel view shows every theme's full color palette — swatch, name, and hex value — updating live as you navigate. Press Enter to get the config snippet for your chosen theme.
