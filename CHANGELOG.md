# Changelog

All notable changes to this project will be documented in this file.

## [0.1.4] - 2026-03-01

### Added
- Automatic `.bak` backup before the first save of each session
- `-B`/`--no-backup` flag to disable backup
- GitLab Pages job for API documentation (gomarkdoc + pandoc)
- `just docs` recipe for local documentation generation

## [0.1.3] - 2026-03-01

### Added
- Visual warning (`!`) in file list for `.env` files not covered by `.gitignore`
- `-G`/`--no-git-check` flag to disable gitignore check
- Silent fallback when git is not installed or not in a git repo

### Improved
- Modernize codebase for Go 1.26 (`SplitSeq`, range over int, etc.)

## [0.1.2] - 2026-02-28

### Added
- Reset command (`r`) in normal mode to discard changes and reload from disk

### Fixed
- Recursive scan (`-r`) without explicit path now correctly defaults to current directory

### Improved
- Replaced `flag` with Kong for CLI argument parsing
- Kong `existingdir` type for path validation

### Testing
- Test for `ScanDir` with dot path in recursive mode

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
