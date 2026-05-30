package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (a App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeNormal, ModeSearching:
		return a.handleNormalMouseClick(msg), nil
	case ModeComparing, ModeEditingCompare:
		return a.handleCompareMouseClick(msg), nil
	case ModeCompareSelect:
		return a.handleCompareSelectMouseClick(msg), nil
	case ModeMatrix:
		return a.handleMatrixMouseClick(msg), nil
	case ModeHelp:
		a.mode = ModeNormal
		return a, nil
	}
	return a, nil
}

func (a App) handleNormalMouseClick(msg tea.MouseClickMsg) App {
	if msg.X < a.fileWidth {
		// Click on file list panel
		a.focus = FocusFiles
		a.fileList.Focused = true
		a.varList.Focused = false
		index := msg.Y - 2 + a.fileList.Offset // Y=0 border, Y=1 title, Y=2 first item
		if index >= 0 {
			prev := a.fileList.SelectedFile()
			a.fileList.SetCursor(index)
			if f := a.fileList.SelectedFile(); f != nil && f != prev {
				a.varList.SetFile(f)
			}
		}
	} else {
		// Click on var list panel
		a.focus = FocusVars
		a.fileList.Focused = false
		a.varList.Focused = true
		index := msg.Y - 2 + a.varList.Offset
		if index >= 0 && index < a.varList.DisplayCount() {
			a.varList.SetCursor(index)
			if a.varList.IsHeaderAtCursor() {
				a.varList.ToggleCollapseAtCursor()
			}
		}
	}
	return a
}

func (a App) handleCompareMouseClick(msg tea.MouseClickMsg) App {
	index := msg.Y - 3 + a.diffView.Offset // Y=0 border, Y=1 title, Y=2 header, Y=3 first entry
	if index >= 0 {
		a.diffView.SetCursor(index)
	}
	return a
}

func (a App) handleCompareSelectMouseClick(msg tea.MouseClickMsg) App {
	index := msg.Y - 2 + a.fileList.Offset
	if index >= 0 {
		// Only move cursor, don't update Selected (compare target, not active file)
		a.fileList.Cursor = max(0, min(index, len(a.fileList.Files)-1))
		visible := a.fileList.Height - 2
		if visible > 0 && a.fileList.Cursor < a.fileList.Offset {
			a.fileList.Offset = a.fileList.Cursor
		}
		if visible > 0 && a.fileList.Cursor >= a.fileList.Offset+visible {
			a.fileList.Offset = a.fileList.Cursor - visible + 1
		}
	}
	return a
}

func (a App) handleMatrixMouseClick(msg tea.MouseClickMsg) App {
	row := msg.Y - 2 + a.matrixView.offsetRow // Y=0 header, Y=1 separator, Y=2 first row
	if row < 0 {
		return a
	}
	keyWidth := a.matrixView.layout.MatrixKeyWidth
	colWidth := a.matrixView.layout.MatrixColWidth
	if msg.X >= keyWidth {
		col := (msg.X-keyWidth)/colWidth + a.matrixView.offsetCol
		a.matrixView.SetCursor(row, col)
	} else {
		a.matrixView.SetCursor(row, a.matrixView.cursorCol)
	}
	return a
}

func (a App) handleMouseWheel(msg tea.MouseWheelMsg) App {
	up := msg.Button == tea.MouseWheelUp
	switch a.mode {
	case ModeNormal, ModeSearching:
		if msg.X < a.fileWidth {
			a = a.scrollFileList(up)
		} else {
			a = a.scrollVarList(up)
		}
	case ModeComparing, ModeEditingCompare:
		a = a.scrollDiffView(up)
	case ModeCompareSelect:
		a = a.scrollFileList(up)
	case ModeMatrix, ModeMatrixEditing, ModeConfirmMatrixDelete:
		a = a.scrollMatrix(up)
	}
	return a
}

func (a App) scrollFileList(up bool) App {
	prev := a.fileList.SelectedFile()
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.fileList.MoveUp()
		} else {
			a.fileList.MoveDown()
		}
	}
	a.fileList.Select()
	if f := a.fileList.SelectedFile(); f != nil && f != prev {
		a.varList.SetFile(f)
	}
	return a
}

func (a App) scrollVarList(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.varList.MoveUp()
		} else {
			a.varList.MoveDown()
		}
	}
	return a
}

func (a App) scrollDiffView(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.diffView.MoveUp()
		} else {
			a.diffView.MoveDown()
		}
	}
	return a
}

func (a App) scrollMatrix(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.matrixView.MoveUp()
		} else {
			a.matrixView.MoveDown()
		}
	}
	return a
}
