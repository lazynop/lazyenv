# lazyenv

> **Work in progress** — not ready for production use.

TUI for managing `.env` files — written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Browse, compare, edit and validate environment variables from your terminal.

## Features

- **Gitignore warning** — shows `!` next to files not covered by `.gitignore` (disable with `-G`)
- **File scanning** — finds `.env`, `.env.*`, `*.env` in the current directory (or recursively with `-r`)
- **Two-panel layout** — file list on the left, variables on the right
- **Secret masking** — auto-detects keys like `*_PASSWORD`, `*_TOKEN`, `*_API_KEY` and masks their values
- **Inline validation** — warns on empty values, placeholders (`TODO`, `changeme`), and duplicate keys
- **Diff/compare** — side-by-side comparison between two env files with bidirectional copy, inline editing, difference filtering, and jump-to-next/prev diff
- **Inline editing** — edit, add, and delete variables without leaving the TUI
- **Round-trip fidelity** — saves preserve comments, blank lines, quoting, and ordering

## Build

Requires Go 1.22+.

```
make              # build to bin/lazyenv
make run          # build + run
make run ARGS=-r  # build + run with flags
make test         # run tests
make vet          # static analysis
make clean        # remove bin/
```

## Usage

```
lazyenv [path] [flags]
```

| Flag | Description |
|------|-------------|
| `-r` | Scan subdirectories recursively |
| `-a` | Show secrets in cleartext at startup |
| `-G` | Disable `.gitignore` check |
| `-v` | Show version |
| `-h` | Show help |

## Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` `j/k` | Navigate items |
| `←/→` `h/l` | Switch panels |
| `Enter` | Select file |
| `e` | Edit variable value |
| `a` | Add new variable |
| `d` | Delete variable |
| `w` | Save changes |
| `c` | Compare two files |
| `/` | Search variables |
| `o` | Toggle sort (position / alphabetical) |
| `Ctrl+S` | Toggle secret masking |
| `?` | Help |
| `q` | Quit |
