# Features

## Browse and edit

Two-panel layout: file list on the left, variables on the right. Navigate with arrow keys or vim bindings (`hjkl`). Edit values inline, add new variables, delete existing ones.

## Compare and sync

Side-by-side diff between any two `.env` files. Copy values in either direction, edit inline, and save — all without leaving the TUI.

## Completeness matrix

Multi-file grid view showing which variables exist in which files. Spot missing entries at a glance and add them inline.

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

## Peek original values

Toggle inline display of the original value before your edits, so you always know what changed.

## Clipboard support

Yank values or full `KEY=value` lines to the clipboard using OSC 52 — works over SSH too.

## Secret masking

Sensitive keys (passwords, tokens, secrets) are automatically detected and masked. Toggle visibility with `Ctrl+S`.

## Gitignore check

Warns when `.env` files are not covered by `.gitignore`, so you don't accidentally commit secrets.

## Automatic backup

Creates a `.bak` copy before the first save of each session. Disable with `-B` or `no-backup = true` in config.

## Round-trip fidelity

Saves preserve comments, blank lines, quoting style, and key ordering — your files stay exactly as you wrote them.

## Search and sort

Filter variables by name or value with `/`. Toggle between original position and alphabetical sorting with `o`.
