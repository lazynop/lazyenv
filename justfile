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

# Generate API documentation
docs:
    @mkdir -p docs
    gomarkdoc ./... > docs/API.md
    @echo "Documentation generated in docs/API.md"

# Clean build artifacts
clean:
    rm -rf bin coverage.out dist

# Build a snapshot release (no publish)
release-snapshot:
    goreleaser release --snapshot --clean
