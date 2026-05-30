package tui

import (
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (a App) handleConfirmDeleteFileKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		f := a.fileList.SelectedFile()
		if f == nil {
			f = a.fileList.CursorFile()
		}
		a.mode = ModeNormal
		if f == nil {
			return a, nil
		}

		if err := os.Remove(f.Path); err != nil {
			return a, a.flashError("Error deleting file: " + err.Error())
		}

		a.sessionStats.RecordDelete(f.Path)

		// Remove from file list
		idx := -1
		for i, ef := range a.fileList.Files {
			if ef.Path == f.Path {
				idx = i
				break
			}
		}
		if idx >= 0 {
			a.fileList.Files = append(a.fileList.Files[:idx], a.fileList.Files[idx+1:]...)
		}

		if len(a.fileList.Files) == 0 {
			a.fileList.Cursor = 0
			a.fileList.Selected = 0
			a.varList.SetFile(nil)
		} else {
			if a.fileList.Cursor >= len(a.fileList.Files) {
				a.fileList.Cursor = len(a.fileList.Files) - 1
			}
			a.fileList.Selected = a.fileList.Cursor
			a.varList.SetFile(a.fileList.Files[a.fileList.Cursor])
		}

		return a, a.flashMessage("Deleted " + f.Name)

	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.statusBar.ClearMessage()
	}
	return a, nil
}
