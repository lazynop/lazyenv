# Configuration

Create a `.lazyenvrc` file (TOML) in your project root, or `~/.config/lazyenv/config.toml` for global settings. CLI flags override config file values.

```toml
recursive = true
sort = "alphabetical"
theme = "catppuccin-mocha"
```

Run `lazyenv --show-config` to see the effective configuration.

## Themes

Activate a built-in color theme with a single line:

```toml
theme = "dracula"
```

| Theme | Type |
| ----- | ---- |
| `catppuccin-latte` | Light |
| `catppuccin-mocha` | Dark |
| `cyberpunk` | Dark |
| `default-dark` | Dark |
| `default-light` | Light |
| `dracula` | Dark |
| `everforest` | Dark |
| `gruvbox-dark` | Dark |
| `gruvbox-light` | Light |
| `kanagawa` | Dark |
| `monokai-pro` | Dark |
| `nord` | Dark |
| `one-dark` | Dark |
| `rose-pine` | Dark |
| `solarized-dark` | Dark |
| `solarized-light` | Light |
| `tokyo-night` | Dark |

Run `lazyenv --list-themes` to see all available themes.

## Color overrides

Individual colors can override the theme:

```toml
theme = "nord"

[colors]
primary = "#FF6600"    # override just primary, rest from nord
```

Available color keys:

| Key | Used for |
| --- | -------- |
| `primary` | Titles, selected items, keybinding highlights |
| `warning` | Empty values, placeholders |
| `error` | Errors, duplicate warnings, git warnings |
| `success` | Success messages |
| `muted` | Comments, help text, secondary info |
| `fg` | Default text foreground |
| `bg` | Background (empty = terminal native) |
| `border` | Panel borders |
| `cursor-bg` | Cursor line background |
| `modified` | Modified variable marker |
| `added` | Added variable marker |
| `deleted` | Deleted variable marker |

## File patterns

Control which files are detected:

```toml
[files]
include = [".env", ".env.*", "*.env"]
exclude = ["*.bak", "*.example"]
```

## Layout tuning

Fine-tune column widths and padding (values in characters):

```toml
[layout]
var-list-max-key-width = 30
diff-max-key-width = 25
matrix-key-width = 20
matrix-col-width = 14
```

## Full example

See the [full configuration reference](https://gitlab.com/traveltoaiur/lazyenv/-/blob/main/examples/config/full.toml) for all available options.
