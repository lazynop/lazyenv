# Usage

```
lazyenv [path] [flags]
```

Run `lazyenv` in the current directory, or pass a path to scan a different folder.

```bash
# Current directory
lazyenv

# Specific directory
lazyenv ./services

# Recursive scan
lazyenv ./services -r
```

## Flags

| Flag | Long form        | Description                                                                                                 |
| ---- | ---------------- | ----------------------------------------------------------------------------------------------------------- |
| `-r` | `--recursive`    | Scan subdirectories recursively.                                                                            |
| `-a` | `--show-all`     | Show secrets in cleartext at startup instead of masking them.                                               |
| `-s` | `--sort`         | Sort order for variables: `position` (default, preserves file order) or `alphabetical`.                     |
| `-B` | `--no-backup`    | Disable `.bak` backup before first save.                                                                    |
| `-G` | `--no-git-check` | Disable `.gitignore` check. Auto-disabled if `git` is not found in `$PATH`.                                 |
|      | `--no-theme-bg`  | Disable theme background color, keeping the terminal's native background. Useful for transparent terminals. |
|      | `--show-config`  | Print the effective configuration (TOML) and exit.                                                          |
|      | `--list-themes`  | List available built-in themes and exit.                                                                    |
| `-v` | `--version`      | Show version, commit, and build date.                                                                       |
| `-h` | `--help`         | Show help.                                                                                                  |

## Flag details

### `-r` / `--recursive`

By default lazyenv only looks for `.env` files in the given directory. With `-r` it walks subdirectories too. Equivalent to `recursive = true` in the [config file](configuration.md).

### `-a` / `--show-all`

Sensitive keys (passwords, tokens, secrets) are masked with `****` by default. This flag reveals them at startup. You can still toggle visibility at runtime with `Ctrl+S`.

### `-s` / `--sort`

Controls the initial sort order of the variable list:

- `position` — preserves the order variables appear in the file (default).
- `alphabetical` — sorts variables by key name.

Toggleable at runtime with `o`.

### `-B` / `--no-backup`

Skips creating a `.bak` copy of a file before its first save in the session. Useful if your workflow already has backups or version control.

### `-G` / `--no-git-check`

Disables the warning shown when `.env` files are not covered by `.gitignore`. If `git` is not installed, this check is automatically disabled.

### `--no-theme-bg`

Themes include a background color by default. This flag strips it, so the terminal's native background (including transparency) is preserved.

### `--show-config`

Prints the effective configuration as TOML (after merging defaults, config file, and CLI flags) and exits. Useful for debugging configuration issues.

```bash
lazyenv --show-config
```

### `--list-themes`

Prints all available built-in theme names, one per line, and exits.

```bash
lazyenv --list-themes
```

## Precedence

Configuration is resolved in this order, from lowest to highest priority:

1. **Built-in defaults** — sensible out-of-the-box values.
2. **Config file** — `.lazyenvrc` in the project root, or `~/.config/lazyenv/config.toml` for global settings. Project file wins over global.
3. **CLI flags** — always win. A flag passed on the command line overrides both the config file and the defaults.

For example, if your config file has `recursive = false` but you run `lazyenv -r`, the recursive scan is enabled for that session.

The following flags have a corresponding config file key:

| Flag            | Config key     |
| --------------- | -------------- |
| `-r`            | `recursive`    |
| `-a`            | `show-secrets` |
| `-s`            | `sort`         |
| `-B`            | `no-backup`    |
| `-G`            | `no-git-check` |
| `--no-theme-bg` | `no-theme-bg`  |

Flags without a config equivalent (`--show-config`, `--list-themes`, `--version`, `--help`) are one-shot actions that print output and exit.

See [Configuration](configuration.md) for all available config file options.

## Development

The project uses [just](https://github.com/casey/just) as a task runner. Run `just` with no arguments to see all available recipes:

```bash
just
```

### Common recipes for devs

| Recipe            | Description                                                                |
| ----------------- | -------------------------------------------------------------------------- |
| `just build`      | Build the binary to `bin/lazyenv`.                                         |
| `just run`        | Build and run with `go run`. Accepts arguments: `just run ./myproject -r`. |
| `just test`       | Run all tests with verbose output.                                         |
| `just test-cover` | Run tests with race detection and coverage report.                         |
| `just check`      | Run all checks: formatting, vet, and tests — same as CI. Read-only, does not modify files. |
| `just fmt`        | Format all Go source files.                                                                |
| `just fix`        | Apply Go modernization fixes (`go fix`). Run this before `just check`.                     |
| `just vet`        | Run static analysis.                                                                       |
| `just clean`      | Remove build artifacts (`bin/`, `coverage.out`, `dist/`).                  |

### Trying themes and configs for devs

| Recipe                    | Description                                                               |
| ------------------------- | ------------------------------------------------------------------------- |
| `just try-theme <name>`   | Launch lazyenv with a built-in theme (e.g. `just try-theme tokyo-night`). |
| `just try-config <name>`  | Launch with an example config from `examples/config/`.                    |
| `just show-config <name>` | Print the effective config for an example config file.                    |

Shorthand recipes are available for common configs: `just try-dracula`, `just try-nord`, `just try-catppuccin`, `just try-full`, etc.
