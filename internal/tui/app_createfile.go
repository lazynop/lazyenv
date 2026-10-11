package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) confirmCreateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal

	name := strings.TrimSpace(a.fileInput.Value())
	if name == "" {
		return a, nil
	}

	fullPath, cmd := a.validateNewPath(name, a.config.Dir)
	if cmd != nil {
		return a, cmd
	}

	if err := a.createFile(fullPath); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	return a.finaliseNewFile(fullPath, "Created "+name)
}
