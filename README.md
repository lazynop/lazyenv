# lazyenv

TUI for managing `.env` files — written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Browse, compare, edit and validate environment variables from your terminal.

**[Documentation](https://lazyenv-4bb2c3.gitlab.io/)** | **[Releases](https://gitlab.com/traveltoaiur/lazyenv/-/releases)**

## Features

- Two-panel layout with file list and variables, inline editing
- Side-by-side diff between two files with bidirectional copy
- Completeness matrix showing which variables exist across files
- Change tracking: new, modified, deleted, duplicate, empty, placeholder
- Secret masking, gitignore check, automatic backup
- Clipboard support (OSC 52), search and sort
- Round-trip fidelity: saves preserve comments, blank lines, quoting
- TOML configuration with 15 built-in color themes

## Install

Download a binary from [Releases](https://gitlab.com/traveltoaiur/lazyenv/-/releases) (Linux, macOS, Windows, FreeBSD — amd64/arm64).

Or install from source (requires Go 1.22+):

```
go install gitlab.com/traveltoaiur/lazyenv@latest
```

## Quick start

```bash
lazyenv                  # scan current directory for .env files
lazyenv path/to/dir      # scan a specific directory
lazyenv -r               # scan recursively into subdirectories
```

See the **[full documentation](https://lazyenv-4bb2c3.gitlab.io/)** for [usage & flags](https://lazyenv-4bb2c3.gitlab.io/usage/), [configuration](https://lazyenv-4bb2c3.gitlab.io/configuration/), and [keybindings](https://lazyenv-4bb2c3.gitlab.io/keybindings/).

## License

[MIT](LICENSE)
