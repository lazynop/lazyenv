package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/parser"
)

func (a App) backupIfNeeded(path string) string {
	if a.config.NoBackup || a.backedUpPaths[path] {
		return ""
	}
	if err := parser.CreateBackup(path); err != nil {
		a.backedUpPaths[path] = true // don't retry on every save
		return "backup failed: " + err.Error() + " - "
	}
	a.backedUpPaths[path] = true
	return ""
}

func (a App) handleSave() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		return a, a.flashWarning("No file selected")
	}
	if !f.Modified {
		return a, a.flashMessage("No changes to save")
	}

	warn := a.backupIfNeeded(f.Path)

	if err := a.saveFile(f); err != nil {
		return a, a.flashError("Error saving: " + err.Error())
	}

	if _, err := a.refreshFile(f); err != nil {
		return a, a.flashError(warn + "Saved but refresh failed: " + err.Error())
	}

	return a, a.flashMessage(warn + "Saved " + f.Name)
}

func (a App) handleReset() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		return a, a.flashWarning("No file selected")
	}
	if !f.Modified {
		return a, a.flashMessage("No changes to reset")
	}

	if _, err := a.refreshFile(f); err != nil {
		return a, a.flashError("Error reloading: " + err.Error())
	}

	return a, a.flashMessage("Reset to saved state")
}
