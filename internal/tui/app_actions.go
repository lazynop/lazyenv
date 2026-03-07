package tui

import (
	tea "charm.land/bubbletea/v2"

	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
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
		a.statusBar.SetMessage("No file selected")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	if !f.Modified {
		a.statusBar.SetMessage("No changes to save")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	warn := a.backupIfNeeded(f.Path)

	if err := parser.WriteFile(f); err != nil {
		a.statusBar.SetMessage("Error saving: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
	}

	// Re-parse to refresh RawLines
	refreshed, err := parser.ParseFile(f.Path)
	if err != nil {
		a.statusBar.SetMessage(warn + "Saved but refresh failed: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
	}
	refreshed.GitWarning = f.GitWarning

	// Replace file in the list
	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = refreshed
			if a.fileList.Selected == i {
				a.varList.SetFile(refreshed)
			}
			break
		}
	}

	a.statusBar.SetMessage(warn + "Saved " + f.Name)
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) handleReset() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.statusBar.SetMessage("No file selected")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	if !f.Modified {
		a.statusBar.SetMessage("No changes to reset")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	refreshed, err := parser.ParseFile(f.Path)
	if err != nil {
		a.statusBar.SetMessage("Error reloading: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
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

	a.statusBar.SetMessage("Reset to saved state")
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}
