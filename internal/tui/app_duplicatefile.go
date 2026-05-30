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

	// Place beside the source so it appears in the same watched directory
	destPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		return a, a.flashMessage(errMsg)
	}

	// Copy raw bytes to preserve comments, formatting, and round-trip fidelity
	data, err := os.ReadFile(src.Path)
	if err != nil {
		return a, a.flashError("Error reading source: " + err.Error())
	}

	if err := os.WriteFile(destPath, data, envFilePerm); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	a.sessionStats.RecordCreateDuplicate(destPath, src.Path, src.Vars)
	return a.finaliseNewFile(destPath, "Duplicated "+src.Name+" → "+name)
}
