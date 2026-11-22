package tui

import (
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

		if err := a.deleteFile(f.Path); err != nil {
			cmd := a.flashError("Error deleting file: " + err.Error())
			return a, cmd
		}

		a.fileList.removeFile(f.Path)

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

		cmd := a.flashMessage("Deleted " + f.Name)
		return a, cmd

	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.statusBar.ClearMessage()
	}
	return a, nil
}
