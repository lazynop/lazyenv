package tui

import (
	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// reparse re-reads f from disk and returns the EnvFile that replaces it in
// memory. parser.ParseFile knows only what is on disk, so reparse carries over
// the state set at runtime (GitWarning, from CheckGitIgnore). Unsaved edit
// state (Modified, DeletedVars, per-variable edit flags) is dropped: the
// result matches the disk.
//
// Every re-parse of a file already held in memory goes through here;
// TestParseFileOnlyThroughReparse fails on a direct parser.ParseFile call
// elsewhere in this package.
func reparse(f *model.EnvFile, secrets config.SecretsConfig) (*model.EnvFile, error) {
	refreshed, err := parser.ParseFile(f.Path, secrets)
	if err != nil {
		return nil, err
	}
	refreshed.GitWarning = f.GitWarning
	return refreshed, nil
}
