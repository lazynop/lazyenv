package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

func (a App) handleMatrixKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
	case key.Matches(msg, a.keys.Up):
		a.matrixView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.matrixView.MoveDown()
	case key.Matches(msg, a.keys.Left):
		a.matrixView.MoveLeft()
	case key.Matches(msg, a.keys.Right):
		a.matrixView.MoveRight()
	case key.Matches(msg, a.keys.ToggleSort):
		a.matrixView.ToggleSort()
		if a.matrixView.sortMode == model.SortCompleteness {
			return a, a.flashMessage("Sorted by completeness")
		}
		return a, a.flashMessage("Sorted alphabetically")
	case key.Matches(msg, a.keys.Add), key.Matches(msg, a.keys.Delete):
		return a.handleMatrixEdit(msg)
	}
	return a, nil
}

func (a App) handleMatrixEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd
	}

	switch {
	case key.Matches(msg, a.keys.Add):
		cmd := a.matrixView.StartEdit()
		if a.matrixView.editing {
			a.mode = ModeMatrixEditing
			return a, cmd
		}
		if a.matrixView.message != "" {
			flashMsg := a.matrixView.message
			a.matrixView.message = ""
			return a, a.flashMessage(flashMsg)
		}
	case key.Matches(msg, a.keys.Delete):
		if len(a.matrixView.entries) == 0 {
			return a, nil
		}
		entry := a.matrixView.entries[a.matrixView.cursorRow]
		if !entry.Present[a.matrixView.cursorCol] {
			return a, a.flashMessage("Variable not present in this file")
		}
		a.mode = ModeConfirmMatrixDelete
		a.statusBar.SetMessage(fmt.Sprintf("Delete %s from %s? (y/n)", entry.Key, a.matrixView.fileNames[a.matrixView.cursorCol]))
	}
	return a, nil
}

func (a App) handleConfirmMatrixDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		a.mode = ModeMatrix
		deletedFile := a.matrixView.files[a.matrixView.cursorCol]
		key, fileName := a.matrixView.DeleteAtCursor()
		if key != "" {
			if a.varList.File == deletedFile {
				a.varList.Refresh()
			}
			return a, a.flashMessage("Deleted " + key + " from " + fileName)
		}
		return a, nil
	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeMatrix
		a.statusBar.ClearMessage()
	}
	return a, nil
}

func (a App) handleMatrixEditingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.matrixView.CancelEdit()
		a.mode = ModeMatrix
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		a.matrixView.ConfirmEdit()
		a.mode = ModeMatrix
		return a, a.flashMessage("Variable added")
	default:
		var cmd tea.Cmd
		a.matrixView.editor, cmd = a.matrixView.editor.Update(msg)
		return a, cmd
	}
}
