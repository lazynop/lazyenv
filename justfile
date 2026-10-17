# LazyEnv — TUI for managing .env files

# Lint and format tools, pinned in tools/go.mod
tool := "go tool -modfile=tools/go.mod"

# Import grouping: stdlib, third-party, then this module
gci_flags := "-s standard -s default -s 'prefix(github.com/lazynop/lazyenv)' --skip-generated --skip-vendor"

# Go sources to format: explicit paths, so hidden dirs (e.g. .claude/worktrees) are never walked
go_src := "*.go internal/"

# VHS on the fresh bin/lazyenv, without XDG config; the tapes also point HOME
# at a missing /home/demo, so recordings show the defaults whoever runs them
vhs := 'env -u XDG_CONFIG_HOME PATH="$PWD/bin:$PATH" vhs'

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
    go test -race ./...

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
    @gofmt -w {{go_src}}
    @{{tool}} gci write {{gci_flags}} {{go_src}}

# Run linters (cyclomatic complexity, dead assignments, dead code, enforced exhaustive switches)
lint:
    #!/usr/bin/env bash
    set -o pipefail
    rc=0
    {{tool}} gocyclo -over 15 . || rc=1
    {{tool}} ineffassign ./... || rc=1
    dead=$({{tool}} deadcode -test ./...) || rc=1
    [ -z "$dead" ] || { echo "$dead"; rc=1; }
    {{tool}} exhaustive -e ./... || rc=1
    exit $rc

# Check formatting and import grouping (CI-friendly, fails if unformatted)
fmt-check:
    @test -z "$(gofmt -l {{go_src}})" || (echo "Files need formatting:" && gofmt -l {{go_src}} && exit 1)
    @ungrouped=$({{tool}} gci list {{gci_flags}} {{go_src}}) || exit 1; test -z "$ungrouped" || { echo "Imports not grouped as stdlib, third-party, github.com/lazynop/lazyenv (run just fmt):"; echo "$ungrouped"; exit 1; }

# Run all checks (format + vet + lint + tests)
check: fmt-check vet lint test

# Use the repo's git hooks: format and vet before each commit, and a Screens: line when views.golden changes
install-hooks:
    git config core.hooksPath .githooks

# Clean build artifacts
clean:
    @rm -rf bin coverage.out dist

# Build a snapshot release (no publish)
release-snapshot:
    goreleaser release --snapshot --clean

# Regenerate and compress the README demo GIF (needs vhs, ttyd, ffmpeg, gifsicle)
demo: build
    @test ! -e .lazyenvrc || { echo "Move .lazyenvrc out of the repo root first: the demo must show the defaults" >&2; exit 1; }
    {{vhs}} docs/assets/demo.tape
    gifsicle -O3 --lossy=80 docs/assets/demo.gif -o docs/assets/demo.gif

# Render the lazynop.dev project page media into dist/page-media (needs vhs, ttyd, ffmpeg with libx264 and libwebp)
page-media: build
    #!/usr/bin/env bash
    set -euo pipefail
    out=dist/page-media
    mkdir -p "$out"
    # The tapes run on a scratch copy of the demo project: no git repo is
    # found and the repo's files stay untouched
    PAGE_DEMO_DIR=$(mktemp -d)
    export PAGE_DEMO_DIR
    trap 'rm -rf "$PAGE_DEMO_DIR"' EXIT
    cp -r docs/assets/page/project/. "$PAGE_DEMO_DIR"
    # Each still is the last frame of its clip
    for tape in demo card; do
        {{vhs}} "docs/assets/page/$tape.tape"
        ffmpeg -v error -y -sseof -0.5 -i "$out/$tape.gif" -update 1 -f image2 -c:v libwebp -preset text -quality 80 "$out/$tape.webp"
    done
    # H.264 High, BT.709-tagged so the video's colors match the poster's
    ffmpeg -v error -y -i "$out/demo.gif" \
        -vf "fps=30,scale=out_color_matrix=bt709:out_range=tv,format=yuv420p" \
        -an -c:v libx264 -profile:v high -preset veryslow -tune animation -crf 28 \
        -colorspace bt709 -color_primaries bt709 -color_trc bt709 -color_range tv \
        -movflags +faststart "$out/demo.mp4"
    rm "$out"/*.gif
    ls -l "$out"

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
