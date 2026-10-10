// Lint and format tools used by the justfile, pinned in one place.
// Run one with: go tool -modfile=tools/go.mod <tool>
// Bump one with: go get -modfile=tools/go.mod -tool <module>@<version>
module github.com/lazynop/lazyenv/tools

go 1.26.0

tool (
	github.com/daixiang0/gci
	github.com/fzipp/gocyclo/cmd/gocyclo
	github.com/gordonklaus/ineffassign
	github.com/nishanths/exhaustive/cmd/exhaustive
	golang.org/x/tools/cmd/deadcode
)

require (
	github.com/daixiang0/gci v0.14.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/gordonklaus/ineffassign v0.2.0 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/nishanths/exhaustive v0.15.0 // indirect
	github.com/spf13/cobra v1.6.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/telemetry v0.0.0-20260924152758-ed294f943157 // indirect
	golang.org/x/tools v0.51.0 // indirect
)
