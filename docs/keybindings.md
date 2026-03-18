# Keybindings

Keybindings are context-sensitive: each screen only responds to the keys listed in its section. Navigation keys (`↑↓` / `jk`) are shared across most screens.

## Main view — File list (left panel)

| Key            | Action                                            |
| -------------- | ------------------------------------------------- |
| `↑` / `k`      | Move cursor up                                    |
| `↓` / `j`      | Move cursor down                                  |
| `→` / `l`      | Switch focus to variable panel                    |
| `Enter`        | Select file and switch focus to variable panel    |
| `c`            | Start compare: select a second file for diff view |
| `m`            | Open completeness matrix (requires 2+ files)      |
| `w`            | Save current file                                 |
| `r`            | Reset current file (discard unsaved changes)      |
| `Ctrl+S`       | Toggle secret masking                             |
| `o`            | Toggle sort order (position / alphabetical)       |
| `?`            | Show help screen                                  |
| `q` / `Ctrl+C` | Quit                                              |

## Main view — Variable list (right panel)

| Key            | Action                                       |
| -------------- | -------------------------------------------- |
| `↑` / `k`      | Move cursor up                               |
| `↓` / `j`      | Move cursor down                             |
| `←` / `h`      | Switch focus to file panel                   |
| `e`            | Edit selected variable value                 |
| `a`            | Add new variable                             |
| `d`            | Delete selected variable (with confirmation) |
| `y`            | Copy value to clipboard (OSC 52)             |
| `Y`            | Copy `KEY=value` to clipboard (OSC 52)       |
| `p`            | Toggle peek: show original value before edit |
| `/`            | Search variables by name or value            |
| `c`            | Start compare: select a second file for diff |
| `m`            | Open completeness matrix (requires 2+ files) |
| `w`            | Save current file                            |
| `r`            | Reset current file (discard unsaved changes) |
| `Ctrl+S`       | Toggle secret masking                        |
| `o`            | Toggle sort order (position / alphabetical)  |
| `?`            | Show help screen                             |
| `q` / `Ctrl+C` | Quit                                         |

## Compare — Diff view

Side-by-side diff between two `.env` files.

| Key       | Action                                   |
| --------- | ---------------------------------------- |
| `↑` / `k` | Move cursor up                           |
| `↓` / `j` | Move cursor down                         |
| `n`       | Jump to next difference                  |
| `N`       | Jump to previous difference              |
| `→` / `l` | Copy selected value to the right file    |
| `←` / `h` | Copy selected value to the left file     |
| `e`       | Edit selected variable in the left file  |
| `E`       | Edit selected variable in the right file |
| `f`       | Toggle filter: show differences only     |
| `w`       | Save both files (if modified)            |
| `r`       | Reset both files to saved state          |
| `q`       | Return to main view                      |
| `Esc`     | Return to main view                      |

## Completeness matrix

Multi-file grid showing which variables exist in which files.

| Key       | Action                                        |
| --------- | --------------------------------------------- |
| `↑` / `k` | Move cursor up                                |
| `↓` / `j` | Move cursor down                              |
| `←` / `h` | Move cursor left                              |
| `→` / `l` | Move cursor right                             |
| `a`       | Add missing variable to the file under cursor |
| `o`       | Toggle sort (alphabetical / completeness)     |
| `q`       | Return to main view                           |
| `Esc`     | Return to main view                           |

All modal prompts (editing, delete confirmation, search) follow standard conventions: `Enter` to confirm, `Esc` to cancel.

## Theme preview (`--themes`)

| Key            | Action                                         |
| -------------- | ---------------------------------------------- |
| `↑` / `k`      | Previous theme                                 |
| `↓` / `j`      | Next theme                                     |
| `Enter`        | Select theme and print config snippet          |
| `q` / `Esc`    | Quit without selection                         |
