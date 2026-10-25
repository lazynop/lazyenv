package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

// compareHelp is the help section of the keys handleComparingKey and
// handleComparingEdit match.
func compareHelp(keys KeyMap) helpSection {
	return helpSection{"Compare view", []helpRow{
		pair("Navigate entries", keys.Up, keys.Down),
		pair("Scroll a page", keys.PageUp, keys.PageDown),
		pair("Jump to the first/last entry", keys.Home, keys.End),
		pair("Jump to next/prev difference", keys.NextDiff, keys.PrevDiff),
		pair("Copy value to the left/right file; if the source file lacks the key, delete it there", keys.Left, keys.Right),
		pair("Edit value in the left/right file", keys.Edit, keys.EditRight),
		alternatives("Toggle filter: differences only", keys.Filter),
		alternatives("Save both files", keys.Save),
		alternatives("Reset both files", keys.Reset),
		alternatives("Toggle secret masking", keys.ToggleSecret),
		alternatives("Back to main view", keys.Escape),
		alternatives("Back to main view", keys.Quit),
	}}
}

func (a App) handleComparingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Mutating actions (copy, edit, save) — blocked in read-only mode
	if key.Matches(msg, a.keys.Right, a.keys.Left, a.keys.Save, a.keys.Edit, a.keys.EditRight) {
		return a.handleComparingEdit(msg)
	}

	if a.keys.navigate(msg, &a.diffView) {
		return a, nil
	}

	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.exitCompare()
	case key.Matches(msg, a.keys.NextDiff):
		a.diffView.NextDiff()
	case key.Matches(msg, a.keys.PrevDiff):
		a.diffView.PrevDiff()
	case key.Matches(msg, a.keys.Filter):
		return a.handleCompareFilter()
	case key.Matches(msg, a.keys.Reset):
		return a.handleCompareReset()
	case key.Matches(msg, a.keys.ToggleSecret):
		a.diffView.ShowSecrets = !a.diffView.ShowSecrets
		a.varList.SetShowSecrets(a.diffView.ShowSecrets) // keep in sync
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
	case key.Matches(msg, a.keys.EditRight):
		return a.startCompareEdit(a.diffView.FileB)
	default:
		return a, nil
	}
}

func (a App) handleCompareCopy(toRight bool) (tea.Model, tea.Cmd) {
	var k, target string
	var mutated *model.EnvFile
	if toRight {
		k = a.diffView.CopyToRight()
		mutated = a.diffView.FileB
		target = a.diffView.FileB.Name
	} else {
		k = a.diffView.CopyToLeft()
		mutated = a.diffView.FileA
		target = a.diffView.FileA.Name
	}
	if k != "" {
		a.refreshIfShown(mutated)
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
		return a, a.flashError(errMsg)
	}
	a.replaceFile(a.diffView.FileA)
	a.replaceFile(a.diffView.FileB)
	return a, a.flashMessage("Reset to saved state")
}

func (a App) handleCompareSave() (App, tea.Cmd) {
	saved := []string{}
	var warn strings.Builder
	for _, f := range []*model.EnvFile{a.diffView.FileA, a.diffView.FileB} {
		if f != nil && f.Modified {
			warn.WriteString(a.backupIfNeeded(f.Path))
			if err := a.saveFile(f); err != nil {
				return a, a.flashError("Error saving " + f.Name + ": " + err.Error())
			}
			// Re-parse to refresh RawLines
			refreshed, err := a.refreshFile(f)
			if err == nil {
				if f == a.diffView.FileA {
					a.diffView.FileA = refreshed
				} else {
					a.diffView.FileB = refreshed
				}
			}
			saved = append(saved, f.Name)
		}
	}
	a.diffView.Rediff()
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

	varIdx := file.LastIndexOfKey(e.Key)

	if varIdx < 0 {
		// Key doesn't exist in this file
		return a, a.flashWarning(e.Key + " not in " + file.Name)
	}

	if !a.editor.StartEdit(&file.Vars[varIdx], varIdx) {
		return a, a.flashWarning(uneditableValueWarning)
	}
	a.compareEditFile = file
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
		a.mode = ModeComparing
		cmd := a.commitValueEdit(a.compareEditFile, a.editor.Finish())
		a.diffView.Rediff()
		return a, cmd
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

// compareSelectHelp is the help section of the keys handleCompareSelectKey
// matches.
func compareSelectHelp(keys KeyMap) helpSection {
	return helpSection{"Compare: pick the second file", []helpRow{
		pair("Choose the file", keys.Up, keys.Down),
		alternatives("Compare with it", keys.Enter),
		alternatives("Cancel", keys.Escape),
		alternatives("Cancel", keys.Quit),
	}}
}

// exitCompare goes back to the main view from compare or from picking its
// second file, the file list's cursor back on the active file so the list and
// the vars panel stay in sync.
func (a *App) exitCompare() {
	a.mode = ModeNormal
	a.compareFirstFile = nil
	a.fileList.MoveCursorTo(a.fileList.Selected)
}

func (a App) handleCompareSelectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape, a.keys.Quit):
		// q and Ctrl+C go back here, as they do from compare and matrix.
		a.exitCompare()
		a.statusBar.ClearMessage()
	case key.Matches(msg, a.keys.Up):
		a.fileList.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.fileList.MoveDown()
	case key.Matches(msg, a.keys.Enter):
		second := a.fileList.CursorFile()
		if second == nil {
			break
		}
		if second == a.compareFirstFile {
			return a, a.flashWarning("Pick a different file to compare with")
		}
		a.diffView.SetFiles(a.compareFirstFile, second)
		a.diffView.ShowSecrets = a.varList.ShowSecrets
		a.mode = ModeComparing
		a.statusBar.ClearMessage()
	}
	return a, nil
}
