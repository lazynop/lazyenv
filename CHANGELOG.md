# Changelog

All notable changes to this project will be documented in this file.

## [0.1.1] - 2026-02-28

First public release of LazyEnv.

- Two-panel TUI layout with file list and variable viewer
- .env file scanner: finds `.env`, `.env.*`, `*.env` patterns
- Parser with round-trip fidelity (preserves comments, blank lines, quoting, ordering)
- Secret masking: auto-detects keys like `*_PASSWORD`, `*_TOKEN`, `*_API_KEY`
- Inline validation: warns on empty values, placeholders, and duplicate keys
- Side-by-side diff/compare between two env files
- Inline editing: edit, add, and delete variables
- Write-back with format preservation
- Keyboard navigation with vim-style bindings (hjkl)
- CLI flags: `-r` (recursive scan), `-a` (show secrets), `-v` (version), `-h` (help)
- Adaptive light/dark theme via terminal background detection
- Test suites with testify for parser, model, util, and scanner packages
- Bubble Tea v2, Lipgloss v2, Bubbles v2
- GoReleaser for automated multi-platform releases
- `justfile` as task runner
