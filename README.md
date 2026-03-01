# lazyenv

TUI for managing `.env` files — written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Browse, compare, edit and validate environment variables from your terminal.

## Features

- **Browse and edit** — two-panel layout with file list and variables, inline editing, add, delete
- **Compare and sync** — side-by-side diff between two files with bidirectional copy and inline editing
- **Completeness matrix** — multi-file grid view showing which variables exist where, with inline add for missing entries
- **Change tracking** — distinct indicators for new (`+`), modified (`*`), deleted (`-`), duplicate (`D`), empty (`○`), and placeholder (`…`) variables
- **Peek original values** — toggle inline display of the original value before edits
- **Clipboard support** — yank values or full `KEY=value` lines to clipboard (OSC 52)
- **Secret masking** — auto-detects sensitive keys and masks their values
- **Gitignore check** — warns when `.env` files are not covered by `.gitignore`
- **Automatic backup** — creates a `.bak` copy before the first save of each session
- **Round-trip fidelity** — saves preserve comments, blank lines, quoting, and ordering
- **Search and sort** — filter variables by name or value, toggle alphabetical sorting

## Install

### From releases

Download the latest binary from [GitLab Releases](https://gitlab.com/traveltoaiur/lazyenv/-/releases). Builds are available for Linux, macOS, Windows, and FreeBSD (amd64/arm64).

### From source

Requires Go 1.26+.

```
go install gitlab.com/traveltoaiur/lazyenv@latest
```

### Build locally

```
just build        # build to bin/lazyenv
just run          # build + run
just run -r       # build + run with flags
just test         # run tests
just check        # fmt + vet + tests
just clean        # remove build artifacts
```

## Usage

```
lazyenv [path] [flags]
```

| Flag | Description |
| ---- | ---------------------------------------- |
| `-r` | Scan subdirectories recursively |
| `-a` | Show secrets in cleartext at startup |
| `-B` | Disable `.bak` backup before first save |
| `-G` | Disable `.gitignore` check |
| `-v` | Show version |
| `-h` | Show help |

## Keybindings

### Navigation

| Key | Action |
| ----------- | ----------------------------- |
| `↑/↓` `j/k` | Navigate items |
| `←/→` `h/l` | Switch panels (files / vars) |
| `Enter` | Select file |

### Editing

| Key | Action |
| ----- | ---------------------------------- |
| `e` | Edit variable value |
| `a` | Add new variable |
| `d` | Delete variable (with confirmation) |
| `w` | Save changes |
| `r` | Reset file to saved state |

### Tools

| Key | Action |
| -------- | ------------------------------------ |
| `y` | Copy value to clipboard |
| `Y` | Copy `KEY=value` to clipboard |
| `p` | Peek original value (toggle) |
| `c` | Compare two files (diff view) |
| `m` | Completeness matrix (multi-file) |
| `/` | Search variables |
| `o` | Toggle sort (position / alphabetical) |
| `Ctrl+S` | Toggle secret masking |

### General

| Key | Action |
| ---------- | ------------------- |
| `?` | Show/hide help |
| `q` `Ctrl+C` | Quit |
| `Esc` | Back / cancel |

## Documentation

API docs: [pkg.go.dev/gitlab.com/traveltoaiur/lazyenv](https://pkg.go.dev/gitlab.com/traveltoaiur/lazyenv)

## License

[MIT](LICENSE)
