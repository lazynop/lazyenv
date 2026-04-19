package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/parser"
)

// backupIfNeeded creates a .bak backup of the file before the first save of
// the session. It is a no-op if --no-backup was set or the file was already
// backed up. Returns a warning message (empty on success or skip).
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
		return a, a.flashMessage("No file selected")
	}
	if !f.Modified {
		return a, a.flashMessage("No changes to save")
	}

	warn := a.backupIfNeeded(f.Path)

	if err := parser.WriteFile(f); err != nil {
		return a, a.flashError("Error saving: " + err.Error())
	}

	// Record the stat before re-parse: the disk mutation already happened and
	// must be reflected even if the refresh fails.
	a.sessionStats.RecordSave(f.Path, f.Vars)

	// Re-parse to refresh RawLines
	refreshed, err := parser.ParseFile(f.Path, a.config.Secrets)
	if err != nil {
		return a, a.flashError(warn + "Saved but refresh failed: " + err.Error())
	}
	refreshed.GitWarning = f.GitWarning

	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = refreshed
			if a.fileList.Selected == i {
				a.varList.SetFile(refreshed)
			}
			break
		}
	}

	return a, a.flashMessage(warn + "Saved " + f.Name)
}

func (a App) handleReset() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		return a, a.flashMessage("No file selected")
	}
	if !f.Modified {
		return a, a.flashMessage("No changes to reset")
	}

	refreshed, err := parser.ParseFile(f.Path, a.config.Secrets)
	if err != nil {
		return a, a.flashError("Error reloading: " + err.Error())
	}
	refreshed.GitWarning = f.GitWarning

	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = refreshed
			if a.fileList.Selected == i {
				a.varList.SetFile(refreshed)
			}
			break
		}
	}

	return a, a.flashMessage("Reset to saved state")
}
