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

	fullPath, errMsg := a.validateNewPath(name, a.config.Dir)
	if errMsg != "" {
		return a, a.flashMessage(errMsg)
	}

	// Write a single newline so the new file is POSIX-conventional. With an
	// empty file ParseBytes would set TrailingNewline=false and the first
	// save would emit a file without a trailing newline.
	if err := a.createFile(fullPath, []byte("\n")); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	return a.finaliseNewFile(fullPath, "Created "+name)
}
