# LazyEnv — TUI for managing .env files

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
    go test ./... -v

# Run tests with race detection and coverage
test-cover:
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run static analysis
vet:
    go vet ./...

# Format code
fmt:
    gofmt -w .

# Check formatting (CI-friendly, fails if unformatted)
fmt-check:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Run all checks (format + vet + tests)
check: fmt-check vet test

# Clean build artifacts
clean:
    rm -rf bin coverage.out dist

# Build a snapshot release (no publish)
release-snapshot:
    goreleaser release --snapshot --clean

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

# Shorthand recipes for each example config
try-minimal: (try-config "minimal")
try-dracula: (try-config "dracula-theme")
try-catppuccin: (try-config "catppuccin-mocha")
try-wide: (try-config "wide-layout")
try-monorepo: (try-config "monorepo")
try-full: (try-config "full")
