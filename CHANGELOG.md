# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-03-07

### Added
- Configuration file support (`.lazyenvrc` in project root or `~/.config/lazyenv/config.toml`)
- 15 built-in color themes (catppuccin, dracula, gruvbox, nord, tokyo-night, and more)
- Theme background color support via Bubble Tea v2 `View.BackgroundColor`
- `--list-themes` flag to list available built-in themes
- `--no-theme-bg` flag and `no-theme-bg` config option to disable theme background
- `--show-config` flag to display effective configuration
- `-s` / `--sort` flag for sort order (position or alphabetical)
- Configurable file detection patterns (`[files]` section in config)
- Configurable layout dimensions (`[layout]` section in config)
- Configurable color overrides (`[colors]` section in config)
- Blocking error alert for invalid configuration (e.g. unknown theme)

### Fixed
- Config file search now uses current working directory instead of scan directory
- ANSI color reset leaking in file list when using styled themes
- Status bar text now uses theme colors for visibility on all backgrounds
- Exclude `.bak` files from env file detection

### Improved
- Centralized configuration constants into dedicated config package
- Extracted file detection patterns into configurable FileConfig
- Themes organized in dedicated subpackage with one file per theme

## [0.2.0] - 2026-03-01

### Added
- Completeness matrix view for multi-file variable comparison with inline add (`m`)
- Yank to clipboard: copy value (`y`) or KEY=value (`Y`)
- Peek original value for modified/new variables (`p`)
- Per-variable indicators: modified (`*`), new (`+`), deleted (`-`), duplicate (`D`), empty (`○`), placeholder (`…`)
- Combined modified and issue indicators on the same variable

### Fixed
- Preserve GitWarning flag across save, reset, and compare save operations
- Restore cursor position when exiting compare mode
- Show save/reset hints in file panel status bar
- Show matrix hotkey in status bar hints
- Reset peek state when switching files
- Set meaningful default version for source builds

### Improved
- Organize imports into stdlib, external, internal groups
- Rewrite README with updated features, install instructions, and keybindings
- Update help panel with all current keybindings and indicators

### Testing
- Increase parser test coverage from 82% to 93%
- Increase TUI test coverage from 44% to 79%

## [0.1.4] - 2026-03-01

### Added
- Automatic `.bak` backup before the first save of each session
- `-B`/`--no-backup` flag to disable backup

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
