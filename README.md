# lazyenv

[![CI](https://github.com/lazynop/lazyenv/actions/workflows/check.yml/badge.svg)](https://github.com/lazynop/lazyenv/actions/workflows/check.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lazynop/lazyenv)](https://goreportcard.com/report/github.com/lazynop/lazyenv)
[![GoDoc](https://pkg.go.dev/badge/github.com/lazynop/lazyenv)](https://pkg.go.dev/github.com/lazynop/lazyenv)
[![GitHub tag](https://img.shields.io/github/tag/lazynop/lazyenv.svg)](https://github.com/lazynop/lazyenv/releases/latest)
[![GitHub Downloads](https://img.shields.io/github/downloads/lazynop/lazyenv/total)](https://github.com/lazynop/lazyenv/releases)
![GitHub repo size](https://img.shields.io/github/repo-size/lazynop/lazyenv)

TUI for managing `.env` files — written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Browse, compare, edit and validate environment variables from your terminal.

**[Documentation](https://lazynop.github.io/lazyenv/)** | **[Releases](https://github.com/lazynop/lazyenv/releases)**

## Features

- Two-panel layout with file list and variables, inline editing
- Side-by-side diff between two files with bidirectional copy
- Completeness matrix showing which variables exist across files
- Change tracking: new, modified, deleted, duplicate, empty, placeholder
- Secret masking, gitignore check, automatic backup
- Clipboard support (OSC 52), search and sort
- Round-trip fidelity: saves preserve comments, blank lines, quoting
- TOML configuration: layout, colors, file patterns, behaviors
- Mouse support: click to select, scroll wheel to navigate (disable with `--no-mouse`)
- 56 built-in color themes with interactive preview (`--themes`)

## Install

Download a binary from [Releases](https://github.com/lazynop/lazyenv/releases) (Linux, macOS, Windows, FreeBSD — amd64/arm64).

Or install from source (requires Go 1.26+):

```
go install github.com/lazynop/lazyenv@latest
```

## Quick start

```bash
lazyenv                  # scan current directory for .env files
lazyenv path/to/dir      # scan a specific directory
lazyenv -r               # scan recursively into subdirectories
```

See the **[full documentation](https://lazynop.github.io/lazyenv/)** for [usage & flags](https://lazynop.github.io/lazyenv/usage/), [configuration](https://lazynop.github.io/lazyenv/configuration/), and [keybindings](https://lazynop.github.io/lazyenv/keybindings/).

## License

[MIT](LICENSE)
