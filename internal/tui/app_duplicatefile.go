package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) confirmDuplicateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.fileOpSource
	a.fileOpSource = nil

	name := strings.TrimSpace(a.fileInput.Value())
	if name == "" || src == nil {
		return a, nil
	}

	destPath, cmd := a.validateNewPath(name, filepath.Dir(src.Path))
	if cmd != nil {
		return a, cmd
	}

	data, err := os.ReadFile(src.Path)
	if err != nil {
		return a, a.flashError("Error reading source: " + err.Error())
	}

	if err := a.duplicateFile(destPath, data, src.Path); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	return a.finaliseNewFile(destPath, "Duplicated "+src.Name+" → "+name)
}
