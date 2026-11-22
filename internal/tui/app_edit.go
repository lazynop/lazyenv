package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func (a App) handleEditingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		result := a.editor.Finish()
		a.mode = ModeNormal
		if result.IsRenameKey {
			return a.confirmRenameKey(result)
		}
		if result.IsAdd {
			if result.AddStep == addStepKey {
				newKey := strings.TrimSpace(result.Value)
				if newKey == "" {
					return a, nil
				}
				if problem := newKeyProblem(a.varList.File, newKey, noVarToSkip); problem != "" {
					return a, a.flashWarning(problem)
				}
				a.editor.StartAddValue(newKey)
				a.mode = ModeEditing
				return a, a.editor.input.Focus()
			}
			a.varList.File.AddVar(a.editor.addKey, result.Value)
			a.varList.Refresh()
			return a, a.flashMessage("Added " + a.editor.addKey)
		}
		return a, a.commitValueEdit(a.varList.File, result)
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

const uneditableValueWarning = "Value holds line breaks or control characters: can't edit it here"

func (a *App) commitValueEdit(f *model.EnvFile, result EditorResult) tea.Cmd {
	if result.Untouched {
		return nil
	}
	varKey := f.Vars[result.VarIndex].Key
	f.UpdateVar(result.VarIndex, result.Value)
	a.refreshIfShown(f)
	return a.flashMessage("Modified " + varKey)
}

const noVarToSkip = -1

func newKeyProblem(f *model.EnvFile, name string, skipIndex int) string {
	if !parser.IsValidKey(name) {
		return "Invalid key name"
	}
	for i, v := range f.Vars {
		if i != skipIndex && v.Key == name {
			return "Key already exists: " + name
		}
	}
	return ""
}
