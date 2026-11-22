# Configuration

Create a `.lazyenvrc` file (TOML) in the directory you run lazyenv from, or `lazyenv/config.toml` in your user config directory for global settings (see [Config file location](#config-file-location)). CLI flags override config file values. Run `lazyenv --show-config` to see the effective configuration.

| Key            | Default         | Description                                                                                                                       |
| -------------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `recursive`    | `false`         | Scan `.env` files in subdirectories recursively, skipping hidden directories, `node_modules` and `vendor`.                        |
| `show-secrets` | `false`         | Display secret values in cleartext. When false, each secret value is replaced by a fixed `••••••••` mask. Toggleable at runtime with `Ctrl+S`. |
| `no-backup`    | `false`         | Skip creating `.bak` backup files before the first save of each file in a session.                                                |
| `no-git-check` | `false`         | Skip checking if files are covered by `.gitignore`. Auto-disabled if `git` is not available.                                      |
| `no-theme-bg`  | `false`         | Discard the theme's background color, keeping the terminal's native background. Useful for transparent terminals.                 |
| `no-mouse`     | `false`         | Disable mouse support. Useful in terminals or multiplexers with mouse conflicts.                                                  |
| `read-only`    | `false`         | Disable all write operations. Editing keybindings are suppressed and a `[READ-ONLY]` badge is shown in the status bar.            |
| `session-summary` | `true`       | Print a recap of disk-level changes (created, duplicated, templated, renamed, deleted, edited or reordered files) when you exit. `--read-only` forces off.        |
| `sort`         | `"position"`    | Variable list sort order: `"position"` preserves file order, `"alphabetical"` sorts by key name. Toggleable at runtime with `o`.  |
| `group`        | `false`         | Start with prefix grouping enabled (variables sharing a prefix collapse under headers). Toggleable at runtime with `g`.           |
| `theme`        | See description | Built-in color theme. When empty, `default-dark` or `default-light` is used automatically based on terminal background detection. |

Run `lazyenv --list-themes` to see all available themes, or `lazyenv --themes` to browse them interactively with a live preview. [Theme credits](#theme-credits) names the palette each one comes from.

## [files]

Control which files are detected by `lazyenv`. Exclude patterns are checked first and take priority over include.

```toml
[files]
include = [".env", ".env.*", "*.env"]
exclude = ["*.bak", "*.example"]
```

| Key       | Default                       | Description                                                                          |
| --------- | ----------------------------- | ------------------------------------------------------------------------------------ |
| `include` | `[".env", ".env.*", "*.env"]` | Glob patterns to match env files. A file is included if it matches any pattern.      |
| `exclude` | `["*.bak"]`                   | Glob patterns to reject. Checked before include — a matching file is always skipped. |

## [layout]

Fine-tune column widths and padding. All values are in characters, except `mouse-scroll-lines`. Each must be greater than zero, except `file-list-width` and `mouse-scroll-lines`, which also take `0`. Where the table gives a minimum, a lower value is raised to it: below it, rows or panels no longer fit.

```toml
[layout]
var-list-max-key-width = 30
diff-max-key-width = 25
matrix-key-width = 20
matrix-col-width = 14
var-list-min-value-width = 10
var-list-padding = 12
diff-min-value-width = 8
diff-padding = 10
file-list-width = 0
mouse-scroll-lines = 1
```

| Key                        | Default | Description                                                                                 |
| -------------------------- | ------- | ------------------------------------------------------------------------------------------- |
| `var-list-max-key-width`   | `30`    | Maximum width for variable keys in the main panel. Longer keys are cut, ending in `..`.     |
| `var-list-min-value-width` | `10`    | Minimum width for variable values, preventing them from becoming unreadably narrow.         |
| `var-list-padding`         | `12`    | Horizontal space reserved for borders, markers, and spacing in variable list rows. Minimum `8`. |
| `diff-max-key-width`       | `25`    | Maximum width for keys in the compare/diff view.                                            |
| `diff-min-value-width`     | `8`     | Minimum width for values in the diff view.                                                  |
| `diff-padding`             | `10`    | Horizontal space reserved for borders, markers, and spacing in diff rows. Minimum `9`.      |
| `matrix-key-width`         | `20`    | Width of the key column in the completeness matrix.                                         |
| `matrix-col-width`         | `14`    | Width of each file column in the matrix. Controls how many files fit on screen.             |
| `file-list-width`          | `0`     | Width of the file list panel. `0` = auto (1/4 of screen). Minimum `20`. At most the terminal width minus `40`, which the variables panel keeps. File names truncate to fit. |
| `mouse-scroll-lines`       | `1`     | Number of lines scrolled per mouse wheel event. `0` disables wheel scrolling.               |

## [secrets]

Customize how secret detection works. By default, keys are detected as secrets by name patterns and values by entropy-based heuristic.

```toml
[secrets]
safe_patterns = ["_HOST", "_URL", "_ENDPOINT"]
extra_patterns = ["_CREDENTIAL"]
value_heuristic = false
```

| Key               | Default | Description                                                                                     |
| ----------------- | ------- | ----------------------------------------------------------------------------------------------- |
| `safe_patterns`   | `[]`    | Key patterns to never treat as secret, overriding all built-in detection.                       |
| `extra_patterns`  | `[]`    | Key patterns to always treat as secret, in addition to built-ins.                               |
| `value_heuristic` | `true`  | Enable entropy-based value heuristic. When false, only key name and value prefix matching apply. |

**Pattern convention:** `_HOST` (starts with `_`) matches as suffix, `PUBLIC_` (ends with `_`) matches as prefix, `DATABASE_HOST` matches as exact key name. Case insensitive.

### Built-in detection rules

A variable is detected as a secret if it matches any of the following. Key name rules are case insensitive; value prefixes are case sensitive.

| Rule           | Patterns                                                                             |
| -------------- | ------------------------------------------------------------------------------------ |
| Exact match    | `PASSWORD`, `SECRET`, `TOKEN`, `API_KEY`, `ACCESS_KEY`, `PRIVATE_KEY`                |
| Suffix match   | `_KEY`, `_SECRET`, `_TOKEN`, `_PASSWORD`, `_PASS`, `_API_KEY`, `_AUTH`, `_CREDENTIAL`, `_PRIVATE` |
| Prefix match   | `SECRET_`, `TOKEN_`, `AUTH_`, `PRIVATE_`                                             |
| Value prefix   | `sk-`, `pk-`, `ghp_`, `gho_`, `Bearer ` (always active, even with `value_heuristic = false`) |
| Value entropy  | Values ≥16 chars with at least one digit and one letter, and Shannon entropy ≥4.0 bits/char (gated by `value_heuristic`) |

`safe_patterns` is checked first and overrides everything, including built-in rules and `extra_patterns`.

## [colors]

Color overrides, each a hex color (`#RGB`, `#RGBA`, `#RRGGBB` or `#RRGGBBAA`) or an ANSI 256 color number written as a string (`"0"` to `"255"`). These override individual theme colors. Leave empty or omit to use the theme or auto-detected defaults.

```toml
theme = "nord"

[colors]
primary = "#FF6600"    # override just primary, rest from nord
```

**Resolution order:** theme defaults → explicit overrides → `no-theme-bg` clears `bg`.

| Key         | Dark default | Light default | Used for                                                             |
| ----------- | ------------ | ------------- | -------------------------------------------------------------------- |
| `primary`   | `#BD93F9`    | `#7B2FBE`     | Panel titles, focused panel border, selected items, status bar keys and informational messages, matrix column headers. |
| `warning`   | `#FFB86C`    | `#D97706`     | Empty value warnings, placeholder warnings, `[READ-ONLY]` badge, the compare file picker's prompt, status bar notices for refused actions, changed lines in diff and their count in the status bar. |
| `error`     | `#FF5555`    | `#DC2626`     | Config error screen, status bar errors and delete prompts, duplicate key warnings, git warnings, removed lines in diff and their count in the status bar, missing cells in the matrix. |
| `success`   | `#50FA7B`    | `#059669`     | Added lines in diff and their count in the status bar.               |
| `muted`     | `#6272A4`    | `#6B7280`     | Status bar, masked secrets, deleted variables, peek lines, equal lines in diff, present cells in the matrix, secondary text. |
| `fg`        | `#F8F8F2`    | `#1F2937`     | Default text: keys, values, normal list items, help screen.          |
| `bg`        | (none)       | (none)        | Terminal background. Empty keeps the terminal's native background.   |
| `border`    | `#44475A`    | `#D1D5DB`     | Panel borders (unfocused panels and the diff view).                  |
| `cursor-bg` | `#44475A`    | `#E5E7EB`     | Background highlight on the cursor line.                             |
| `modified`  | `#FFB86C`    | `#D97706`     | Marker color for modified variables and files (`*`).                 |
| `added`     | `#50FA7B`    | `#059669`     | Marker color for added variables (`+`).                              |
| `deleted`   | `#FF5555`    | `#DC2626`     | Marker color for deleted variables (`-`).                            |

## Config file location

lazyenv reads a single config file: the first one it finds, in this order.

1. The file passed with `-c` / `--config`.
2. `.lazyenvrc` in the current directory. This is the directory you start lazyenv from, not the path it scans: `lazyenv ./services` does not read `./services/.lazyenvrc`.
3. `lazyenv/config.toml` in the user config directory: `$XDG_CONFIG_HOME` or `~/.config` on Linux and FreeBSD, `~/Library/Application Support` on macOS, `%AppData%` on Windows.
4. `~/.lazyenvrc`.

Files are not merged: the file found is used whole, and a key it leaves out takes its built-in default, even when a file further down the list sets it. Run `lazyenv --check-config` to list the search paths and see which file is in use.

## Invalid configuration

The file is validated as a whole. A TOML syntax error, an unknown key or an invalid value rejects the entire file: lazyenv opens on an error screen naming the file and listing the problems and quits on the next key press; on a terminal too small for the whole list, the screen cuts it and points to `--check-config`. `--show-config` lists the errors on stderr, prints the built-in defaults instead, with any command-line flags applied, and exits with status 1. Run `lazyenv --check-config` to validate the file without starting the TUI: it lists the errors and exits with status 1.

Two inputs are adjusted instead of rejected. An unknown `--theme` on the command line shows a notice in the status bar for 3 seconds (`--show-config` prints it on stderr), and the TUI runs with the default colors plus any `[colors]` overrides. A layout value below its minimum is raised to it (see [`[layout]`](#layout)).

## Theme credits

The built-in themes map these palettes onto lazyenv's colors. Thanks to their authors.

| Theme | Palette source |
|---|---|
| `andromeda` | <https://github.com/EliverLara/Andromeda> |
| `ayu-dark` | <https://github.com/ayu-theme/ayu-colors> |
| `ayu-light` | <https://github.com/ayu-theme/ayu-colors> |
| `ayu-mirage` | <https://github.com/ayu-theme/ayu-colors> |
| `base16-eighties` | <https://github.com/chriskempson/base16> |
| `catppuccin-frappe` | <https://catppuccin.com/palette> |
| `catppuccin-latte` | <https://catppuccin.com/palette> |
| `catppuccin-macchiato` | <https://catppuccin.com/palette> |
| `catppuccin-mocha` | <https://catppuccin.com/palette> |
| `cobalt2` | <https://github.com/wesbos/cobalt2-vscode> |
| `cyberpunk` | lazyenv original |
| `dawnfox` | <https://github.com/EdenEast/nightfox.nvim> |
| `default-dark` | lazyenv original |
| `default-light` | lazyenv original |
| `dracula` | <https://draculatheme.com/contribute#color-palette> |
| `everforest` | <https://github.com/sainnhe/everforest> |
| `github-dark` | <https://github.com/primer/github-vscode-theme> |
| `github-light` | <https://github.com/primer/github-vscode-theme> |
| `gruvbox-dark` | <https://github.com/morhetz/gruvbox> |
| `gruvbox-light` | <https://github.com/morhetz/gruvbox> |
| `horizon` | <https://horizontheme.com> |
| `iceberg` | <https://cocopon.github.io/iceberg.vim/> |
| `kanagawa` | <https://github.com/rebelot/kanagawa.nvim> |
| `material-darker` | <https://github.com/material-theme/vsc-material-theme> |
| `material-ocean` | <https://github.com/material-theme/vsc-material-theme> |
| `melange` | <https://github.com/savq/melange-nvim> |
| `monokai` | <https://monokai.pro/classic> |
| `monokai-pro` | <https://monokai.pro> |
| `moonlight` | <https://github.com/atomiks/moonlight-vscode-theme> |
| `night-owl` | <https://github.com/sdras/night-owl-vscode-theme> |
| `nightfox` | <https://github.com/EdenEast/nightfox.nvim> |
| `nord` | <https://www.nordtheme.com> |
| `one-dark` | <https://github.com/Binaryify/OneDark-Pro> |
| `one-light` | <https://github.com/atom/atom/tree/master/packages/one-light-syntax> |
| `oxocarbon` | <https://github.com/nyoom-engineering/oxocarbon.nvim> |
| `palenight` | <https://github.com/material-theme/vsc-material-theme> |
| `poimandres` | <https://github.com/drcmda/poimandres-theme> |
| `rose-pine` | <https://rosepinetheme.com> |
| `rose-pine-dawn` | <https://rosepinetheme.com/palette> |
| `rose-pine-moon` | <https://rosepinetheme.com/palette> |
| `shades-of-purple` | <https://github.com/ahmadawais/shades-of-purple-vscode> |
| `snazzy` | <https://github.com/sindresorhus/hyper-snazzy> |
| `solarized-dark` | <https://ethanschoonover.com/solarized> |
| `solarized-light` | <https://ethanschoonover.com/solarized> |
| `sonokai` | <https://github.com/sainnhe/sonokai> |
| `synthwave-84` | <https://github.com/robb0wen/synthwave-vscode> |
| `tokyo-night` | <https://github.com/enkia/tokyo-night-vscode-theme> |
| `tokyo-night-light` | <https://github.com/enkia/tokyo-night-vscode-theme> |
| `tokyo-night-moon` | <https://github.com/enkia/tokyo-night-vscode-theme> |
| `tokyo-night-storm` | <https://github.com/enkia/tokyo-night-vscode-theme> |
| `tomorrow` | <https://github.com/chriskempson/tomorrow-theme> |
| `tomorrow-night` | <https://github.com/chriskempson/tomorrow-theme> |
| `vesper` | <https://github.com/raunofreiberg/vesper> |
| `vitesse-dark` | <https://github.com/antfu/vscode-theme-vitesse> |
| `vitesse-light` | <https://github.com/antfu/vscode-theme-vitesse> |
| `zenburn` | <https://kippura.org/zenburnpage/> |
