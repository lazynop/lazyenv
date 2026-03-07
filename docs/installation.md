# Installation

## From releases

Download the latest binary from [GitLab Releases](https://gitlab.com/traveltoaiur/lazyenv/-/releases).

Builds are available for Linux, macOS, Windows, and FreeBSD (amd64/arm64).

## From source

Requires Go 1.26+.

```bash
go install gitlab.com/traveltoaiur/lazyenv@latest
```

## Build locally

```bash
just build        # build to bin/lazyenv
just run          # build + run
just test         # run tests
just check        # fmt + vet + tests
```

## Usage

```
lazyenv [path] [flags]
```

| Flag | Description |
| ---- | ----------- |
| `-r` | Scan subdirectories recursively |
| `-a` | Show secrets in cleartext at startup |
| `-s` | Sort order: `position` or `alphabetical` |
| `-B` | Disable `.bak` backup before first save |
| `-G` | Disable `.gitignore` check |
| `--show-config` | Show effective configuration and exit |
| `--list-themes` | List available built-in themes and exit |
| `-v` | Show version |
| `-h` | Show help |
