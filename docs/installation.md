# Installation

## Requirements

lazyenv needs a terminal of at least 60x10 (columns x rows). In a smaller one it shows a notice in place of its screens, and ignores every key and click but Ctrl+C, which quits, until the terminal grows back.

## From releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/lazynop/lazyenv/releases). Builds are available for Linux (amd64, arm64, armv7), macOS (amd64, arm64), Windows (amd64), and FreeBSD (amd64, arm64).

## Homebrew (macOS & Linux)

```bash
brew install lazynop/tap/lazyenv
```

lazyenv ships as a cask, which needs Homebrew 6.0.13 or later.

## AUR (Arch Linux)

Install with your preferred AUR helper:

```bash
yay -S lazyenv-bin
```

## Scoop (Windows)

```powershell
scoop bucket add lazynop https://github.com/lazynop/scoop-bucket
scoop install lazyenv
```

## mise (macOS, Linux & Windows)

```bash
mise use -g github:lazynop/lazyenv
```

This pins lazyenv to `latest` in mise's global config, so `mise upgrade` updates it.

## From source

Requires Go 1.26+.

```bash
go install github.com/lazynop/lazyenv@latest
```

## Build locally

```bash
just build        # build to bin/lazyenv
just run          # run
```

## Next steps

See the [Usage](usage.md) page for all CLI flags and options.
