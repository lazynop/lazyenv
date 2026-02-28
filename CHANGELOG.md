# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed
- Upgraded to Bubble Tea v2, Lipgloss v2, Bubbles v2 (`charm.land` imports)
- Adaptive light/dark theme via terminal background detection (`BackgroundColorMsg`)
- Simplified GitLab CI pipeline to check + release stages (GoReleaser handles builds)
- Replaced Makefile with justfile

### Added
- Initial release, work in progress
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
- Test suites with testify for parser, model, util, and scanner packages (56 tests)
- GoReleaser configuration for automated multi-platform releases
- `justfile` as task runner (build, test, check, fmt, release-snapshot)
