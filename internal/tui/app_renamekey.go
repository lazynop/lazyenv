package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) confirmRenameKey(result EditorResult) (tea.Model, tea.Cmd) {
	newKey := strings.TrimSpace(result.Value)
	idx := result.VarIndex
	f := a.varList.File

	if f == nil || idx < 0 || idx >= len(f.Vars) {
		return a, nil
	}

	oldKey := f.Vars[idx].Key
	if newKey == "" || newKey == oldKey {
		return a, nil
	}

	if problem := newKeyProblem(f, newKey, idx); problem != "" {
		cmd := a.flashWarning(problem)
		return a, cmd
	}

	f.RenameVar(idx, newKey)
	a.varList.Refresh()
	cmd := a.flashMessage("Renamed " + oldKey + " → " + newKey)
	return a, cmd
}
