package tui

import (
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// reparse re-reads f from disk and returns the EnvFile that replaces it in
// memory. parser.ParseFile knows only what is on disk, so reparse hands it the
// [secrets] config f was parsed with and carries over the state set at
// runtime (GitWarning, from CheckGitIgnore). Unsaved edit state (Modified,
// DeletedVars, per-variable edit flags) is dropped: the result matches the
// disk.
//
// Every re-parse of a file already held in memory goes through here;
// TestParseFileOnlyThroughReparse fails on a direct parser.ParseFile call
// elsewhere in this package.
func reparse(f *model.EnvFile) (*model.EnvFile, error) {
	refreshed, err := parser.ParseFile(f.Path, f.Secrets)
	if err != nil {
		return nil, err
	}
	refreshed.GitWarning = f.GitWarning
	return refreshed, nil
}

// refreshFile re-parses f from disk and swaps the result in for f wherever the
// App holds it (see replaceFile).
func (a *App) refreshFile(f *model.EnvFile) (*model.EnvFile, error) {
	refreshed, err := reparse(f)
	if err != nil {
		return nil, err
	}
	a.replaceFile(refreshed)
	return refreshed, nil
}

// refreshIfShown recomputes the var list's rows when it shows f, which a
// compare or matrix action changed: added or deleted vars would leave its
// rows indexing the old Vars, and a changed value may match the kept search
// or stop matching it.
func (a *App) refreshIfShown(f *model.EnvFile) {
	if a.varList.File == f {
		a.varList.Refresh()
	}
}

// replaceFile puts f in place of the in-memory file with the same path, in the
// file list and, when that file is selected, in the vars panel.
func (a *App) replaceFile(f *model.EnvFile) {
	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = f
			if a.fileList.Selected == i {
				a.varList.SetFile(f)
			}
			return
		}
	}
}
