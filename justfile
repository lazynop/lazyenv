# LazyEnv — TUI for managing .env files

# Import grouping: stdlib, third-party, then this module
gci := "go run github.com/daixiang0/gci@v0.14.0"
gci_flags := "-s standard -s default -s 'prefix(github.com/lazynop/lazyenv)' --skip-generated --skip-vendor"

# Default recipe: show available commands
default:
    @just --list

# Build the binary
build:
    @mkdir -p bin
    go build -o bin/lazyenv .

# Run the application
run *args:
    go run . {{args}}

# Run all tests
test:
    go test -race ./... -v

# Run tests with coverage report
test-cover:
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run static analysis
vet:
    @go vet ./...

# Apply Go modernization fixes
fix:
    @go fix ./...

# Format code and group imports
fmt:
    @gofmt -w .
    @{{gci}} write {{gci_flags}} .

# Run linters (cyclomatic complexity, dead assignments, dead code, enforced exhaustive switches)
lint:
    #!/usr/bin/env bash
    set -o pipefail
    rc=0
    gocyclo -over 15 . || rc=1
    ineffassign ./... || rc=1
    dead=$(go run golang.org/x/tools/cmd/deadcode@v0.51.0 -test ./...) || rc=1
    [ -z "$dead" ] || { echo "$dead"; rc=1; }
    go run github.com/nishanths/exhaustive/cmd/exhaustive@v0.15.0 -e ./... || rc=1
    exit $rc

# Check formatting and import grouping (CI-friendly, fails if unformatted)
fmt-check:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
    @ungrouped=$({{gci}} list {{gci_flags}} .) || exit 1; test -z "$ungrouped" || { echo "Imports not grouped as stdlib, third-party, github.com/lazynop/lazyenv (run just fmt):"; echo "$ungrouped"; exit 1; }

# Run all checks (format + vet + lint + tests)
check: fmt-check vet lint test

# Clean build artifacts
clean:
    @rm -rf bin coverage.out dist

# Build a snapshot release (no publish)
release-snapshot:
    goreleaser release --snapshot --clean

# Regenerate and compress the README demo GIF (needs vhs, ttyd, ffmpeg, gifsicle)
demo:
    vhs docs/assets/demo.tape
    gifsicle -O3 --lossy=80 docs/assets/demo.gif -o docs/assets/demo.gif

# Run with a built-in theme by name (e.g. just try-theme tokyo-night)
try-theme name:
    @echo 'theme = "{{name}}"' > .lazyenvrc
    @echo "Using theme: {{name}}"
    -go run . env
    @rm -f .lazyenvrc

# Run with an example config (copies it as .lazyenvrc, runs, then removes it)
try-config name:
    @cp examples/config/{{name}}.toml .lazyenvrc
    @echo "Using examples/config/{{name}}.toml"
    -go run . env
    @rm -f .lazyenvrc

# Show effective config for an example config
show-config name:
    @cp examples/config/{{name}}.toml .lazyenvrc
    -go run . --show-config
    @rm -f .lazyenvrc

# Shorthand recipes for example configs
try-minimal: (try-config "minimal")
try-dracula: (try-config "dracula-theme")
try-catppuccin: (try-config "catppuccin-mocha")
try-nord: (try-config "nord-theme")
try-override: (try-config "theme-with-override")
try-wide: (try-config "wide-layout")
try-monorepo: (try-config "monorepo")
try-full: (try-config "full")
