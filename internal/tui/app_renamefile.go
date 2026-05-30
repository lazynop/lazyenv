package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

func (a App) confirmRenameFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.fileOpSource
	a.fileOpSource = nil

	name := strings.TrimSpace(a.fileInput.Value())
	if name == "" || src == nil || name == src.Name {
		return a, nil
	}

	newPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		return a, a.flashMessage(errMsg)
	}

	if err := os.Rename(src.Path, newPath); err != nil {
		return a, a.flashError("Error renaming file: " + err.Error())
	}

	a.sessionStats.RecordRename(src.Path, newPath)

	if a.backedUpPaths[src.Path] {
		delete(a.backedUpPaths, src.Path)
		a.backedUpPaths[newPath] = true
	}

	oldName := src.Name
	src.Path = newPath
	src.Name = name

	// Re-check gitignore — the new name may have different coverage
	if !a.config.NoGitCheck {
		src.GitWarning = false
		CheckGitIgnore([]*model.EnvFile{src})
	}

	a.varList.SetFile(src)

	return a, a.flashMessage("Renamed " + oldName + " → " + name)
}
