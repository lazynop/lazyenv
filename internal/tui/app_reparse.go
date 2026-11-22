package tui

import (
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func reparse(f *model.EnvFile) (*model.EnvFile, error) {
	refreshed, err := parser.ParseFile(f.Path, f.Secrets)
	if err != nil {
		return nil, err
	}
	refreshed.GitWarning = f.GitWarning
	return refreshed, nil
}

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
