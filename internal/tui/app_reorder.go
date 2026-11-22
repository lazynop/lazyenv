package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

func (a App) startReorder() (tea.Model, tea.Cmd) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd
	}
	f := a.varList.File
	if f == nil || len(f.Vars) < 2 {
		cmd := a.flashWarning("Need at least 2 variables to reorder")
		return a, cmd
	}
	a.mode = ModeReorderMenu
	a.statusBar.SetMessage(fmt.Sprintf("Reorder on disk: [%s]alphabetical  [%s]grouped  [%s]cancel",
		shortKeys(a.keys.ReorderAlpha), shortKeys(a.keys.ReorderGrouped), shortKeys(a.keys.Escape)))
	return a, nil
}

func (a App) handleReorderKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.mode == ModeReorderMenu {
		return a.handleReorderMenuKey(msg)
	}
	return a.handleReorderConfirmKey(msg)
}

func (a App) handleReorderMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		cmd := a.flashMessage("Reorder cancelled")
		return a, cmd
	case key.Matches(msg, a.keys.ReorderAlpha):
		a.reorderMode = model.ReorderAlphabetical
		return a.promptReorderConfirm()
	case key.Matches(msg, a.keys.ReorderGrouped):
		a.reorderMode = model.ReorderGrouped
		return a.promptReorderConfirm()
	}
	return a, nil
}

func (a App) promptReorderConfirm() (tea.Model, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.mode = ModeNormal
		cmd := a.flashWarning("No file selected")
		return a, cmd
	}
	a.mode = ModeReorderConfirm
	a.statusBar.SetMessage(fmt.Sprintf(
		"Reorder %d variables in %s (%s)? This rewrites the file. %s",
		len(f.Vars), f.Name, a.reorderModeLabel(), yesNo(a.keys),
	))
	return a, nil
}

func (a App) handleReorderConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		return a.executeReorder()
	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		cmd := a.flashMessage("Reorder cancelled")
		return a, cmd
	}
	return a, nil
}

func (a App) executeReorder() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	f := a.varList.File
	if f == nil {
		cmd := a.flashWarning("No file selected")
		return a, cmd
	}

	f.Reorder(a.reorderMode)

	cmd := a.saveAndRefresh(f, "Reordered", fmt.Sprintf("Reordered %s (%s)", f.Name, a.reorderModeLabel()))
	return a, cmd
}

func (a App) reorderModeLabel() string {
	if a.reorderMode == model.ReorderGrouped {
		return "grouped"
	}
	return "alphabetical"
}
