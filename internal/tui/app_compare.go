package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func (a App) handleComparingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Mutating actions (copy, edit, save) — blocked in read-only mode
	if key.Matches(msg, a.keys.Right, a.keys.Left, a.keys.Save, a.keys.Edit) || msg.String() == "E" {
		return a.handleComparingEdit(msg)
	}

	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.compareFirstFile = nil
		a.varList.ShowSecrets = a.diffView.ShowSecrets // sync back
		// Restore cursor to the selected file so file list and var panel stay in sync.
		a.fileList.Cursor = a.fileList.Selected
	case key.Matches(msg, a.keys.Up):
		a.diffView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.diffView.MoveDown()
	case key.Matches(msg, a.keys.NextDiff):
		a.diffView.NextDiff()
	case key.Matches(msg, a.keys.PrevDiff):
		a.diffView.PrevDiff()
	case key.Matches(msg, a.keys.Filter):
		return a.handleCompareFilter()
	case msg.String() == "r":
		return a.handleCompareReset()
	case key.Matches(msg, a.keys.ToggleSecret):
		a.diffView.ShowSecrets = !a.diffView.ShowSecrets
		a.varList.ShowSecrets = a.diffView.ShowSecrets // keep in sync
		if a.diffView.ShowSecrets {
			return a, a.flashMessage("Secrets revealed")
		}
		return a, a.flashMessage("Secrets hidden")
	}
	return a, nil
}

func (a App) handleComparingEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd
	}

	switch {
	case key.Matches(msg, a.keys.Right):
		return a.handleCompareCopy(true)
	case key.Matches(msg, a.keys.Left):
		return a.handleCompareCopy(false)
	case key.Matches(msg, a.keys.Save):
		return a.handleCompareSave()
	case key.Matches(msg, a.keys.Edit):
		return a.startCompareEdit(a.diffView.FileA)
	case msg.String() == "E":
		return a.startCompareEdit(a.diffView.FileB)
	default:
		return a, nil
	}
}

func (a App) handleCompareCopy(toRight bool) (tea.Model, tea.Cmd) {
	var k, target string
	if toRight {
		k = a.diffView.CopyToRight()
		target = a.diffView.FileB.Name
	} else {
		k = a.diffView.CopyToLeft()
		target = a.diffView.FileA.Name
	}
	if k != "" {
		return a, a.flashMessage(k + " → " + target)
	}
	return a, nil
}

func (a App) handleCompareFilter() (tea.Model, tea.Cmd) {
	a.diffView.ToggleFilter()
	if a.diffView.HideEqual {
		return a, a.flashMessage("Showing differences only")
	}
	return a, a.flashMessage("Showing all entries")
}

func (a App) handleCompareReset() (App, tea.Cmd) {
	if errMsg := a.diffView.Reset(); errMsg != "" {
		return a, a.flashMessage(errMsg)
	}
	for i, f := range a.fileList.Files {
		if f.Path == a.diffView.FileA.Path {
			a.fileList.Files[i] = a.diffView.FileA
			if a.fileList.Selected == i {
				a.varList.SetFile(a.diffView.FileA)
			}
		}
		if f.Path == a.diffView.FileB.Path {
			a.fileList.Files[i] = a.diffView.FileB
			if a.fileList.Selected == i {
				a.varList.SetFile(a.diffView.FileB)
			}
		}
	}
	return a, a.flashMessage("Reset to saved state")
}

func (a App) handleCompareSave() (App, tea.Cmd) {
	saved := []string{}
	var warn strings.Builder
	for _, f := range []*model.EnvFile{a.diffView.FileA, a.diffView.FileB} {
		if f != nil && f.Modified {
			warn.WriteString(a.backupIfNeeded(f.Path))
			if err := parser.WriteFile(f); err != nil {
				return a, a.flashError("Error saving " + f.Name + ": " + err.Error())
			}
			// Re-parse to refresh RawLines
			refreshed, err := parser.ParseFile(f.Path, a.config.Secrets)
			if err == nil {
				refreshed.GitWarning = f.GitWarning
				for i, existing := range a.fileList.Files {
					if existing.Path == f.Path {
						a.fileList.Files[i] = refreshed
						if a.fileList.Selected == i {
							a.varList.SetFile(refreshed)
						}
						break
					}
				}
				if f == a.diffView.FileA {
					a.diffView.FileA = refreshed
				} else {
					a.diffView.FileB = refreshed
				}
			}
			saved = append(saved, f.Name)
		}
	}
	// Recompute diff after save
	a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
	a.diffView.recompute()
	if len(saved) == 0 {
		return a, a.flashMessage("No changes to save")
	}
	return a, a.flashMessage(warn.String() + "Saved " + strings.Join(saved, ", "))
}

func (a App) startCompareEdit(file *model.EnvFile) (tea.Model, tea.Cmd) {
	if a.diffView.Cursor < 0 || a.diffView.Cursor >= len(a.diffView.Entries) {
		return a, nil
	}
	e := a.diffView.Entries[a.diffView.Cursor]

	// Find the var index in the target file
	varIdx := -1
	for i := len(file.Vars) - 1; i >= 0; i-- {
		if file.Vars[i].Key == e.Key {
			varIdx = i
			break
		}
	}

	if varIdx < 0 {
		// Key doesn't exist in this file
		return a, a.flashMessage(e.Key + " not in " + file.Name)
	}

	a.compareEditFile = file
	a.compareEditVarIdx = varIdx
	a.editor.StartEdit(&file.Vars[varIdx], varIdx)
	a.editor.label = fmt.Sprintf("Edit %s in %s: ", e.Key, file.Name)
	a.mode = ModeEditingCompare
	return a, a.editor.input.Focus()
}

func (a App) handleEditingCompareKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeComparing
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		result := a.editor.Finish()
		a.compareEditFile.UpdateVar(a.compareEditVarIdx, result.Value)
		// Recompute diff
		a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
		a.diffView.recompute()
		a.mode = ModeComparing
		return a, a.flashMessage("Modified " + a.compareEditFile.Vars[a.compareEditVarIdx].Key)
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

func (a App) handleCompareSelectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.compareFirstFile = nil
		a.statusBar.ClearMessage()
		// Restore cursor to the selected file so file list and var panel stay in sync.
		a.fileList.Cursor = a.fileList.Selected
	case key.Matches(msg, a.keys.Up):
		a.fileList.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.fileList.MoveDown()
	case key.Matches(msg, a.keys.Enter):
		second := a.fileList.CursorFile()
		if second != nil && a.compareFirstFile != nil && second != a.compareFirstFile {
			a.diffView.SetFiles(a.compareFirstFile, second)
			a.diffView.ShowSecrets = a.varList.ShowSecrets
			a.diffView.Width = a.width - 2
			a.diffView.Height = a.height - 4
			a.mode = ModeComparing
			a.statusBar.ClearMessage()
		}
	}
	return a, nil
}
