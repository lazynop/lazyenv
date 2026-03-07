# lazyenv

**TUI for managing `.env` files** — written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Browse, compare, edit and validate environment variables from your terminal.

## Why lazyenv?

Working with `.env` files is tedious: scattered across projects, easy to forget a variable, painful to compare between environments. lazyenv gives you a fast, keyboard-driven interface to manage them all.

- **See everything at a glance** — two-panel layout with files on the left, variables on the right
- **Compare environments** — side-by-side diff with bidirectional copy
- **Catch mistakes early** — detects duplicates, empty values, placeholders, and missing `.gitignore` coverage
- **Looks good** — 17 built-in color themes, or bring your own

## Quick start

```bash
# Install
go install gitlab.com/traveltoaiur/lazyenv@latest

# Run in the current directory
lazyenv

# Run in a specific directory, recursively
lazyenv ./services -r
```

Check the [Installation](installation.md) page for more options.
