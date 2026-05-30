package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/lazynop/lazyenv/internal/parser"
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
	if err := parser.WriteFileAtomic(fullPath, []byte("\n"), envFilePerm); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	a.sessionStats.RecordCreateScratch(fullPath, nil)
	return a.finaliseNewFile(fullPath, "Created "+name)
}
