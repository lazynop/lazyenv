package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

// startReorder opens the reorder mode-selection menu for the active file.
func (a App) startReorder() (tea.Model, tea.Cmd) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd
	}
	f := a.varList.File
	if f == nil || len(f.Vars) < 2 {
		return a, a.flashWarning("Need at least 2 variables to reorder")
	}
	a.mode = ModeReorderMenu
	a.statusBar.SetMessage(fmt.Sprintf("Reorder on disk: [%s]alphabetical  [%s]grouped  [%s]cancel",
		shortKeys(a.keys.ReorderAlpha), shortKeys(a.keys.ReorderGrouped), shortKeys(a.keys.Escape)))
	return a, nil
}

// handleReorderKey routes key presses for both reorder sub-modes.
func (a App) handleReorderKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.mode == ModeReorderMenu {
		return a.handleReorderMenuKey(msg)
	}
	return a.handleReorderConfirmKey(msg)
}

// handleReorderMenuKey handles the mode-selection menu shown after pressing O.
func (a App) handleReorderMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		return a, a.flashMessage("Reorder cancelled")
	case key.Matches(msg, a.keys.ReorderAlpha):
		a.reorderMode = model.ReorderAlphabetical
		return a.promptReorderConfirm()
	case key.Matches(msg, a.keys.ReorderGrouped):
		a.reorderMode = model.ReorderGrouped
		return a.promptReorderConfirm()
	}
	return a, nil
}

// promptReorderConfirm switches to the confirmation prompt for the chosen mode.
func (a App) promptReorderConfirm() (tea.Model, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.mode = ModeNormal
		return a, a.flashWarning("No file selected")
	}
	a.mode = ModeReorderConfirm
	a.statusBar.SetMessage(fmt.Sprintf(
		"Reorder %d variables in %s (%s)? This rewrites the file. %s",
		len(f.Vars), f.Name, a.reorderModeLabel(), yesNo(a.keys),
	))
	return a, nil
}

// handleReorderConfirmKey handles the y/n confirmation for an on-disk reorder.
func (a App) handleReorderConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		return a.executeReorder()
	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		return a, a.flashMessage("Reorder cancelled")
	}
	return a, nil
}

// executeReorder reorders the active file and persists it to disk, mirroring
// the save flow (backup, atomic write, session stats, re-parse).
func (a App) executeReorder() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	f := a.varList.File
	if f == nil {
		return a, a.flashWarning("No file selected")
	}

	f.Reorder(a.reorderMode)

	warn := a.backupIfNeeded(f.Path)
	if err := a.saveFile(f); err != nil {
		return a, a.flashError("Error saving: " + err.Error())
	}

	if _, err := a.refreshFile(f); err != nil {
		return a, a.flashError(warn + "Reordered but refresh failed: " + err.Error())
	}

	return a, a.flashMessage(fmt.Sprintf("%sReordered %s (%s)", warn, f.Name, a.reorderModeLabel()))
}

func (a App) reorderModeLabel() string {
	if a.reorderMode == model.ReorderGrouped {
		return "grouped"
	}
	return "alphabetical"
}
