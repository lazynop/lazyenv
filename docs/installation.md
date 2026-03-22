# Installation

## From releases

Download the latest binary for your platform from [GitHub Releases](https://github.com/lazynop/lazyenv/releases). Builds are available for Linux, macOS, Windows, and FreeBSD (amd64 and arm64).

## Homebrew (macOS & Linux)

```bash
brew install lazynop/tap/lazyenv
```

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
