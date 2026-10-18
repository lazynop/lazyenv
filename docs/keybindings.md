# Keybindings

Keybindings are context-sensitive: each screen only responds to the keys listed in its section. Navigation keys (`↑↓` / `jk`) are shared across most screens.

## Main view — File list (left panel)

| Key            | Action                                            |
| -------------- | ------------------------------------------------- |
| `↑` / `k`      | Move up and show that file's variables            |
| `↓` / `j`      | Move down and show that file's variables          |
| `→` / `l`      | Switch focus to variable panel                    |
| `Enter`        | Select file and switch focus to variable panel    |
| `N`            | Create new `.env` file in the scan directory      |
| `C`            | Duplicate selected file with a new name           |
| `D`            | Delete file from disk (with confirmation)         |
| `R`            | Rename selected file                              |
| `T`            | Create template from selected file (keys only)    |
| `c`            | Start compare: pick a second file (requires 2+ files) |
| `m`            | Open completeness matrix (requires 2+ files)      |
| `w`            | Save current file                                 |
| `r`            | Reset current file (discard unsaved changes)      |
| `Ctrl+S`       | Toggle secret masking                             |
| `o`            | Toggle sort order (position / alphabetical)       |
| `O`            | Reorder variables on disk (alphabetical / grouped) |
| `g`            | Toggle prefix grouping in the variable list       |
| `/`            | Search variables by name or value                 |
| `?`            | Show help screen                                  |
| `q` / `Ctrl+C` | Quit                                              |

## Main view — Variable list (right panel)

| Key            | Action                                       |
| -------------- | -------------------------------------------- |
| `↑` / `k`      | Move cursor up                               |
| `↓` / `j`      | Move cursor down                             |
| `←` / `h`      | Switch focus to file panel                   |
| `e`            | Edit selected variable value                 |
| `E`            | Rename selected variable key                 |
| `a`            | Add new variable                             |
| `d`            | Delete selected variable (with confirmation) |
| `y`            | Copy value to clipboard (OSC 52)             |
| `Y`            | Copy `KEY=value` to clipboard (OSC 52)       |
| `p`            | Toggle peek: show original value before edit |
| `/`            | Search variables by name or value            |
| `c`            | Start compare: pick a second file (requires 2+ files) |
| `m`            | Open completeness matrix (requires 2+ files) |
| `w`            | Save current file                            |
| `r`            | Reset current file (discard unsaved changes) |
| `Ctrl+S`       | Toggle secret masking                        |
| `o`            | Toggle sort order (position / alphabetical)  |
| `O`            | Reorder variables on disk (alphabetical / grouped) |
| `g`            | Toggle prefix grouping (`DB_*`, `REDIS_*`, …) |
| `Enter` / `Space` | Collapse/expand group (when cursor is on a group header) |
| `?`            | Show help screen                             |
| `q` / `Ctrl+C` | Quit                                         |

## Compare — Pick the second file

After `c`, pick the file to compare the active file with.

| Key            | Action                             |
| -------------- | ---------------------------------- |
| `↑` / `k`      | Move up                            |
| `↓` / `j`      | Move down                          |
| `Enter`        | Compare with the file under cursor |
| `Esc`          | Cancel                             |
| `q` / `Ctrl+C` | Cancel                             |

## Compare — Diff view

Side-by-side diff between two `.env` files.

| Key       | Action                                   |
| --------- | ---------------------------------------- |
| `↑` / `k` | Move cursor up                           |
| `↓` / `j` | Move cursor down                         |
| `n`       | Jump to next difference                  |
| `N`       | Jump to previous difference              |
| `→` / `l` | Copy selected value to the right file; if the left file lacks the key, delete it from the right file |
| `←` / `h` | Copy selected value to the left file; if the right file lacks the key, delete it from the left file  |
| `e`       | Edit selected variable in the left file  |
| `E`       | Edit selected variable in the right file |
| `f`       | Toggle filter: show differences only     |
| `w`       | Save both files (if modified)            |
| `r`       | Reset both files to saved state          |
| `Ctrl+S`  | Toggle secret masking                    |
| `q` / `Ctrl+C` | Return to main view                 |
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
| `d`       | Delete variable from the file under cursor (with confirmation) |
| `o`       | Toggle sort (alphabetical / completeness)     |
| `q` / `Ctrl+C` | Return to main view                      |
| `Esc`     | Return to main view                           |

## Help screen (`?`)

The help lists the keys of every screen. When it is taller than the terminal, it scrolls.

| Key                  | Action                         |
| -------------------- | ------------------------------ |
| `↑` / `k`            | Scroll up a row                |
| `↓` / `j`            | Scroll down a row              |
| `PgUp` / `PgDn`      | Scroll up / down a page        |
| `Home` / `End`       | Jump to the top / bottom       |
| `Esc` / `?`          | Close the help                 |
| `q` / `Ctrl+C`       | Close the help                 |

## Prompts

| Prompt                                                       | Confirm                       | Cancel            |
| ------------------------------------------------------------ | ----------------------------- | ----------------- |
| Text input: edit, rename or add a variable, name a file      | `Enter`                       | `Esc`             |
| Search                                                       | `Enter` (keeps the filter)    | `Esc` (clears it) |
| Reorder mode (`O`)                                           | `a` alphabetical, `g` grouped | `Esc`             |
| Delete a variable, a file or a matrix cell; confirm reorder  | `y`                           | `n` / `Esc`       |

Yes/no prompts confirm only with `y`: `Enter` and other keys are ignored. `Ctrl+C` does nothing while a prompt is open, unless the terminal is below the [minimum size](installation.md#requirements), where it always quits.

## Theme preview (`--themes`)

| Key            | Action                                         |
| -------------- | ---------------------------------------------- |
| `↑` / `k`      | Previous theme                                 |
| `↓` / `j`      | Next theme                                     |
| `Enter`        | Select theme and print config snippet          |
| `q` / `Esc`    | Quit without selection                         |
| `Ctrl+C`       | Quit without selection                         |

With the mouse on, a click on a theme in the list shows it, and the wheel over the list moves through the themes, `mouse-scroll-lines` themes per tick.

## Mouse

Mouse is enabled by default. Disable with `--no-mouse` or `no-mouse = true` in config.

| Action                | Effect                                   |
| --------------------- | ---------------------------------------- |
| Click on file         | Select the file                          |
| Click on variable     | Select the variable                      |
| Click on group header | Collapse / expand the group              |
| Click on panel        | Switch focus to that panel               |
| Click on diff entry   | Select the entry                         |
| Click on matrix cell  | Move cursor to that cell                 |
| Scroll wheel          | Scroll the panel under the mouse cursor; over the file list it also shows the file it lands on |

While an edit bar, a file-name bar, a prompt or a confirmation is open, clicks and the wheel are ignored: it acts on the entry it was opened on. The search bar is the exception: the mouse keeps working while you type a search. While you pick the second file to compare, only clicks and the wheel over the file list count, and they move the cursor without changing the active file. On the help screen the wheel scrolls it and clicks are ignored.
