package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

// matrixHelp is the help section of the keys handleMatrixKey and
// handleMatrixEdit match.
func matrixHelp(keys KeyMap) helpSection {
	return helpSection{"Matrix", []helpRow{
		pair("Move between variables", keys.Up, keys.Down),
		pair("Scroll a page of variables", keys.PageUp, keys.PageDown),
		pair("Jump to the first/last variable", keys.Home, keys.End),
		pair("Move between files", keys.Left, keys.Right),
		alternatives("Add missing variable to the file", keys.Add),
		alternatives("Delete variable from the file (with confirmation)", keys.Delete),
		alternatives("Toggle sort (alphabetical / completeness)", keys.ToggleSort),
		alternatives("Back to main view", keys.Escape),
		alternatives("Back to main view", keys.Quit),
	}}
}

func (a App) handleMatrixKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.keys.navigate(msg, &a.matrixView) {
		return a, nil
	}

	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
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

	cell, ok := a.matrixView.cursorCell()
	if !ok {
		return a, nil
	}

	switch {
	case key.Matches(msg, a.keys.Add):
		if cell.present {
			return a, a.flashWarning("Variable already exists")
		}
		cmd := a.matrixView.StartEdit()
		a.mode = ModeMatrixEditing
		return a, cmd
	case key.Matches(msg, a.keys.Delete):
		if !cell.present {
			return a, a.flashWarning("Variable not present in this file")
		}
		a.mode = ModeConfirmMatrixDelete
		a.statusBar.SetAlert(fmt.Sprintf("Delete %s from %s? %s", cell.key, cell.file.Name, yesNo(a.keys)))
	}
	return a, nil
}

func (a App) handleConfirmMatrixDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		a.mode = ModeMatrix
		key, file := a.matrixView.DeleteAtCursor()
		if file != nil {
			a.refreshIfShown(file)
			return a, a.flashMessage("Deleted " + key + " from " + file.Name)
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
		addedTo := a.matrixView.ConfirmEdit()
		a.refreshIfShown(addedTo)
		a.mode = ModeMatrix
		return a, a.flashMessage("Variable added")
	default:
		var cmd tea.Cmd
		a.matrixView.editor, cmd = a.matrixView.editor.Update(msg)
		return a, cmd
	}
}
