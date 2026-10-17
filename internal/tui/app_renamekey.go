package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/util"
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
		return a, a.flashWarning(problem)
	}

	f.RenameVar(idx, newKey, util.IsSecret(newKey, f.Vars[idx].Value, a.config.Secrets))
	a.varList.Refresh()
	return a, a.flashMessage("Renamed " + oldKey + " → " + newKey)
}
