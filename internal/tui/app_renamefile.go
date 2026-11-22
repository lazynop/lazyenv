package tui

import (
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

	newPath, cmd := a.validateNewPath(name, filepath.Dir(src.Path))
	if cmd != nil {
		return a, cmd
	}

	if err := a.renameFile(src.Path, newPath); err != nil {
		cmd = a.flashError("Error renaming file: " + err.Error())
		return a, cmd
	}

	if a.backupAttempted[src.Path] {
		delete(a.backupAttempted, src.Path)
		a.backupAttempted[newPath] = true
	}

	oldName := src.Name
	src.Path = newPath
	src.Name = name

	if !a.config.NoGitCheck {
		src.GitWarning = false
		CheckGitIgnore([]*model.EnvFile{src})
	}

	a.varList.SetFile(src)

	cmd = a.flashMessage("Renamed " + oldName + " → " + name)
	return a, cmd
}
