package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func (a App) backupIfNeeded(path string) string {
	if a.config.NoBackup || a.backupAttempted[path] {
		return ""
	}
	a.backupAttempted[path] = true
	if err := parser.CreateBackup(path); err != nil {
		return "backup failed: " + err.Error() + " - "
	}
	return ""
}

func (a App) backupAndSave(f *model.EnvFile) (warn string, err error) {
	warn = a.backupIfNeeded(f.Path)
	return warn, a.saveFile(f)
}

func (a App) handleSave() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		return a, a.flashWarning("No file selected")
	}
	if !f.Modified {
		return a, a.flashMessage("No changes to save")
	}

	return a, a.saveAndRefresh(f, "Saved", "Saved "+f.Name)
}

func (a *App) saveAndRefresh(f *model.EnvFile, verb, done string) tea.Cmd {
	warn, err := a.backupAndSave(f)
	if err != nil {
		return a.flashError("Error saving: " + err.Error())
	}

	if _, err := a.refreshFile(f); err != nil {
		return a.flashError(warn + verb + " but refresh failed: " + err.Error())
	}

	return a.flashMessage(warn + done)
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
