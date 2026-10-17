package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

func (a App) handleEditingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		result := a.editor.Finish()
		a.mode = ModeNormal
		if result.Cancelled {
			return a, nil
		}
		if result.IsRenameKey {
			return a.confirmRenameKey(result)
		}
		if result.IsAdd {
			if result.AddStep == addStepKey {
				a.editor.StartAddValue(result.Value)
				a.mode = ModeEditing
				return a, a.editor.input.Focus()
			}
			a.varList.File.AddVar(a.editor.addKey, result.Value, util.IsSecret(a.editor.addKey, result.Value, a.config.Secrets))
			a.varList.Refresh()
			return a, a.flashMessage("Added " + a.editor.addKey)
		}
		cmd, _ := a.commitValueEdit(a.varList.File, result)
		return a, cmd
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

// commitValueEdit writes the value confirmed in the edit bar to its var in f,
// unless the edit was untouched, and reports the change. It returns whether it
// wrote.
func (a *App) commitValueEdit(f *model.EnvFile, result EditorResult) (tea.Cmd, bool) {
	if result.Untouched {
		return nil, false
	}
	varKey := f.Vars[result.VarIndex].Key
	f.UpdateVar(result.VarIndex, result.Value, util.IsSecret(varKey, result.Value, a.config.Secrets))
	a.refreshIfShown(f)
	return a.flashMessage("Modified " + varKey), true
}
