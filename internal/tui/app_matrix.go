package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

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
			cmd := a.flashMessage("Sorted by completeness")
			return a, cmd
		}
		cmd := a.flashMessage("Sorted alphabetically")
		return a, cmd
	case key.Matches(msg, a.keys.Add), key.Matches(msg, a.keys.Delete):
		return a.handleMatrixEdit(msg)
	}
	return a, nil
}

func (a App) handleMatrixEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd
	}

	cell, ok := a.matrixView.CursorCell()
	if !ok {
		return a, nil
	}

	switch {
	case key.Matches(msg, a.keys.Add):
		if cell.Present {
			cmd := a.flashWarning("Variable already exists")
			return a, cmd
		}
		cmd := a.matrixView.StartEdit()
		a.mode = ModeMatrixEditing
		return a, cmd
	case key.Matches(msg, a.keys.Delete):
		if !cell.Present {
			cmd := a.flashWarning("Variable not present in this file")
			return a, cmd
		}
		a.mode = ModeConfirmMatrixDelete
		a.statusBar.SetAlert(fmt.Sprintf("Delete %s from %s? %s", cell.Key, cell.File.Name, yesNo(a.keys)))
	}
	return a, nil
}

func (a App) handleConfirmMatrixDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		a.mode = ModeMatrix
		deleted, file := a.matrixView.DeleteAtCursor()
		if file != nil {
			a.refreshIfShown(file)
			cmd := a.flashMessage("Deleted " + deleted + " from " + file.Name)
			return a, cmd
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
		cmd := a.flashMessage("Variable added")
		return a, cmd
	default:
		var cmd tea.Cmd
		a.matrixView.editor, cmd = a.matrixView.editor.Update(msg)
		return a, cmd
	}
}
