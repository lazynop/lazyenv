package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (a App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	//exhaustive:enforce
	switch a.mode {
	case ModeNormal, ModeSearching:
		return a.handleNormalMouseClick(msg), nil
	case ModeComparing:
		return a.handleCompareMouseClick(msg), nil
	case ModeCompareSelect:
		return a.handleCompareSelectMouseClick(msg), nil
	case ModeMatrix:
		return a.handleMatrixMouseClick(msg), nil
	case ModeEditing, ModeEditingCompare, ModeMatrixEditing, ModeConfirmDelete, ModeConfirmDeleteFile,
		ModeConfirmMatrixDelete, ModeCreateFile, ModeDuplicateFile, ModeRenameFile,
		ModeTemplateFile, ModeReorderMenu, ModeReorderConfirm, ModeHelp, ModeConfigError:
	}
	return a, nil
}

func (a App) handleNormalMouseClick(msg tea.MouseClickMsg) App {
	if msg.X < a.fileWidth {
		a.setFocus(FocusFiles)
		index := a.fileList.IndexAt(msg.Y)
		if index >= 0 {
			a.fileList.MoveCursorTo(index)
			a.selectCursorFile()
		}
	} else {
		index := a.varList.IndexAt(msg.Y)
		a.setFocus(FocusVars)
		if index >= 0 {
			a.varList.SetCursor(index)
			if a.varList.IsHeaderAtCursor() {
				a.varList.ToggleCollapseAtCursor()
			}
		}
	}
	return a
}

func (a App) handleCompareMouseClick(msg tea.MouseClickMsg) App {
	index := a.diffView.IndexAt(msg.Y)
	if index >= 0 {
		a.diffView.SetCursor(index)
	}
	return a
}

func (a App) handleCompareSelectMouseClick(msg tea.MouseClickMsg) App {
	if msg.X >= a.fileWidth {
		return a
	}
	index := a.fileList.IndexAt(msg.Y)
	if index >= 0 {
		a.fileList.MoveCursorTo(index)
	}
	return a
}

func (a App) handleMatrixMouseClick(msg tea.MouseClickMsg) App {
	if row, col := a.matrixView.CellAt(msg.X, msg.Y); row >= 0 {
		a.matrixView.SetCursor(row, col)
	}
	return a
}

func (a App) handleMouseWheel(msg tea.MouseWheelMsg) App {
	up := msg.Button == tea.MouseWheelUp
	lines := a.config.Layout.MouseScrollLines
	//exhaustive:enforce
	switch a.mode {
	case ModeNormal, ModeSearching:
		if msg.X < a.fileWidth {
			a = a.scrollFileList(up)
		} else {
			wheelScroll(lines, up, a.varList.MoveUp, a.varList.MoveDown)
		}
	case ModeComparing:
		wheelScroll(lines, up, a.diffView.MoveUp, a.diffView.MoveDown)
	case ModeCompareSelect:
		if msg.X < a.fileWidth {
			wheelScroll(lines, up, a.fileList.MoveUp, a.fileList.MoveDown)
		}
	case ModeMatrix:
		wheelScroll(lines, up, a.matrixView.MoveUp, a.matrixView.MoveDown)
	case ModeHelp:
		wheelScroll(lines, up, a.helpView.MoveUp, a.helpView.MoveDown)
	case ModeEditing, ModeEditingCompare, ModeMatrixEditing, ModeConfirmDelete, ModeConfirmDeleteFile,
		ModeConfirmMatrixDelete, ModeCreateFile, ModeDuplicateFile, ModeRenameFile, ModeTemplateFile,
		ModeReorderMenu, ModeReorderConfirm, ModeConfigError:
	}
	return a
}

func (a App) scrollFileList(up bool) App {
	wheelScroll(a.config.Layout.MouseScrollLines, up, a.fileList.MoveUp, a.fileList.MoveDown)
	a.selectCursorFile()
	return a
}

func wheelScroll(lines int, up bool, moveUp, moveDown func()) {
	step := moveDown
	if up {
		step = moveUp
	}
	for range lines {
		step()
	}
}
