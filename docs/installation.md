# Installation

## From releases

Download the latest binary from [GitHub Releases](https://github.com/lazynop/lazyenv/releases). Builds are available for amd64 and arm64.

=== "Linux"

    Download the `.tar.gz` archive for your architecture, extract and move to your `$PATH`:

    ```bash
    tar xzf lazyenv_*_linux_amd64.tar.gz
    sudo mv lazyenv /usr/local/bin/
    ```

=== "macOS"

    Download the `.tar.gz` archive for your architecture:

    ```bash
    tar xzf lazyenv_*_darwin_arm64.tar.gz
    sudo mv lazyenv /usr/local/bin/
    ```

=== "Windows"

    Download the `.zip` archive, extract it, and add the folder to your `PATH`.

=== "FreeBSD"

    Download the `.tar.gz` archive for your architecture:

    ```bash
    tar xzf lazyenv_*_freebsd_amd64.tar.gz
    sudo mv lazyenv /usr/local/bin/
    ```

## Homebrew (macOS & Linux)

```bash
brew install lazynop/tap/lazyenv
```

## mise

```bash
mise use -g lazyenv
```

## AUR (Arch Linux)

Install with your preferred AUR helper:

```bash
yay -S lazyenv-bin
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
