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

## Next steps

See the [Usage](usage.md) page for all CLI flags and options.
