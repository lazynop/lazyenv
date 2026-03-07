package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
)

func (a App) handleComparingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.compareFirstFile = nil
		// Restore cursor to the selected file so file list and var panel stay in sync.
		a.fileList.Cursor = a.fileList.Selected
	case key.Matches(msg, a.keys.Up):
		a.diffView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.diffView.MoveDown()
	case key.Matches(msg, a.keys.Right):
		if k := a.diffView.CopyToRight(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileB.Name)
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
	case key.Matches(msg, a.keys.Left):
		if k := a.diffView.CopyToLeft(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileA.Name)
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
	case key.Matches(msg, a.keys.Filter):
		a.diffView.ToggleFilter()
		if a.diffView.HideEqual {
			a.statusBar.SetMessage("Showing differences only")
		} else {
			a.statusBar.SetMessage("Showing all entries")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	case key.Matches(msg, a.keys.Save):
		return a.handleCompareSave()
	case key.Matches(msg, a.keys.Edit): // e = edit left
		return a.startCompareEdit(a.diffView.FileA)
	case msg.String() == "E": // E = edit right
		return a.startCompareEdit(a.diffView.FileB)
	case msg.String() == "r":
		if errMsg := a.diffView.Reset(); errMsg != "" {
			a.statusBar.SetMessage(errMsg)
		} else {
			// Update file references in the main list too
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
			a.statusBar.SetMessage("Reset to saved state")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	return a, nil
}

func (a App) handleCompareSave() (App, tea.Cmd) {
	saved := []string{}
	var warn strings.Builder
	for _, f := range []*model.EnvFile{a.diffView.FileA, a.diffView.FileB} {
		if f != nil && f.Modified {
			warn.WriteString(a.backupIfNeeded(f.Path))
			if err := parser.WriteFile(f); err != nil {
				a.statusBar.SetMessage("Error saving " + f.Name + ": " + err.Error())
				return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
			}
			// Re-parse to refresh RawLines
			refreshed, err := parser.ParseFile(f.Path)
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
	if len(saved) == 0 {
		a.statusBar.SetMessage("No changes to save")
	} else {
		a.statusBar.SetMessage(warn.String() + "Saved " + strings.Join(saved, ", "))
	}
	// Recompute diff after save
	a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
	a.diffView.recompute()
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
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
		a.statusBar.SetMessage(e.Key + " not in " + file.Name)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
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
		a.statusBar.SetMessage("Modified " + a.compareEditFile.Vars[a.compareEditVarIdx].Key)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
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
			a.diffView.Width = a.width - 2
			a.diffView.Height = a.height - 4
			a.mode = ModeComparing
			a.statusBar.ClearMessage()
		}
	}
	return a, nil
}
