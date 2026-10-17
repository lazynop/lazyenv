package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (a App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	// `just lint` (exhaustive) requires a case for every AppMode.
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
		// Keyboard-only modes (text input, prompts, menus) and static screens:
		// clicks are ignored. The help screen has nothing to click, and a
		// stray click while scrolling it with the wheel must not close it.
	}
	return a, nil
}

func (a App) handleNormalMouseClick(msg tea.MouseClickMsg) App {
	if msg.X < a.fileWidth {
		// Click on file list panel
		a.setFocus(FocusFiles)
		index := msg.Y - panelContentOffsetY + a.fileList.Offset
		if index >= 0 {
			prev := a.fileList.SelectedFile()
			a.fileList.SetCursor(index)
			if f := a.fileList.SelectedFile(); f != nil && f != prev {
				a.varList.SetFile(f)
			}
		}
	} else {
		// Click on var list panel. The row maps to an item as drawn when
		// clicked, before focusing the list draws a peek line and scrolls.
		index := a.varList.IndexAtRow(msg.Y - panelContentOffsetY)
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
	index := msg.Y - panelContentOffsetY + a.diffView.Offset
	if index >= 0 {
		a.diffView.SetCursor(index)
	}
	return a
}

func (a App) handleCompareSelectMouseClick(msg tea.MouseClickMsg) App {
	if msg.X >= a.fileWidth {
		return a // only the file list picks the file to compare with
	}
	index := msg.Y - panelContentOffsetY + a.fileList.Offset
	if index >= 0 {
		a.fileList.MoveCursorTo(index) // the compare target, not the active file
	}
	return a
}

func (a App) handleMatrixMouseClick(msg tea.MouseClickMsg) App {
	row := msg.Y - matrixContentOffsetY + a.matrixView.offsetRow
	if row < 0 {
		return a
	}
	keyWidth := a.matrixView.layout.MatrixKeyWidth
	colWidth := a.matrixView.layout.MatrixColWidth
	x := msg.X - matrixPaddingWidth/2 // the matrix is drawn after its left padding
	if x >= keyWidth {
		col := (x-keyWidth)/colWidth + a.matrixView.offsetCol
		a.matrixView.SetCursor(row, col)
	} else {
		a.matrixView.SetCursor(row, a.matrixView.cursorCol)
	}
	return a
}

func (a App) handleMouseWheel(msg tea.MouseWheelMsg) App {
	up := msg.Button == tea.MouseWheelUp
	lines := a.config.Layout.MouseScrollLines
	// `just lint` (exhaustive) requires a case for every AppMode.
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
		// Only the file list picks the compare target, and its cursor picks
		// it, not the active file.
		if msg.X < a.fileWidth {
			wheelScroll(lines, up, a.fileList.MoveUp, a.fileList.MoveDown)
		}
	case ModeMatrix:
		wheelScroll(lines, up, a.matrixView.MoveUp, a.matrixView.MoveDown)
	case ModeHelp:
		wheelScroll(lines, up, a.helpView.ScrollUp, a.helpView.ScrollDown)
	case ModeEditing, ModeEditingCompare, ModeMatrixEditing, ModeConfirmDelete, ModeConfirmDeleteFile,
		ModeConfirmMatrixDelete, ModeCreateFile, ModeDuplicateFile, ModeRenameFile, ModeTemplateFile,
		ModeReorderMenu, ModeReorderConfirm, ModeConfigError:
		// Keyboard-only modes and static screens: the wheel is ignored. Edit
		// bars and confirmations act on the entry they were opened on, so
		// moving the cursor under them would only mislead or retarget them.
	}
	return a
}

func (a App) scrollFileList(up bool) App {
	prev := a.fileList.SelectedFile()
	wheelScroll(a.config.Layout.MouseScrollLines, up, a.fileList.MoveUp, a.fileList.MoveDown)
	a.fileList.Select()
	if f := a.fileList.SelectedFile(); f != nil && f != prev {
		a.varList.SetFile(f)
	}
	return a
}

// wheelScroll moves a view by one wheel tick: lines steps of moveUp or
// moveDown.
func wheelScroll(lines int, up bool, moveUp, moveDown func()) {
	step := moveDown
	if up {
		step = moveUp
	}
	for range lines {
		step()
	}
}
