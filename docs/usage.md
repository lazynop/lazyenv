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
|      | `--no-mouse`     | Disable mouse support. Useful in terminals or multiplexers with mouse conflicts.                            |
|      | `--read-only`    | Disable all write operations. Useful for safely inspecting production files.                                 |
|      | `--session-summary` / `--no-session-summary` | Force-enable or disable the on-exit summary of disk changes (default: enabled). `--read-only` forces it off. |
|      | `--file-list-width` | Width of the file list panel in characters. `0` = auto (1/4 screen, min 20).                             |
| `-c` | `--config`       | Path to configuration file. Takes highest priority over default search paths.                               |
|      | `--check-config` | Validate configuration file and exit. Shows search paths and any errors found.                              |
|      | `--show-config`  | Print the effective configuration (TOML) and exit.                                                          |
|      | `--list-themes`  | List available built-in themes and exit.                                                                    |
|      | `--themes`       | Interactive theme preview. Browse all built-in themes and see their colors live.                             |
| `-v` | `--version`      | Show version, commit, and build date.                                                                       |
| `-h` | `--help`         | Show help.                                                                                                  |

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
| `--no-mouse`    | `no-mouse`     |
| `--read-only`   | `read-only`    |
| `--session-summary` / `--no-session-summary` | `session-summary` |

Flags without a config equivalent (`--check-config`, `--show-config`, `--list-themes`, `--themes`, `--version`, `--help`) are one-shot actions that print output and exit.

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
