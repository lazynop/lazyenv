package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestPeekToggle(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars

	assert.False(t, app.varList.Peeking)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "p"})
	app = updated.(App)
	assert.True(t, app.varList.Peeking, "first press should enable peek")

	updated, _ = app.Update(tea.KeyPressMsg{Text: "p"})
	app = updated.(App)
	assert.False(t, app.varList.Peeking, "second press should disable peek")
}

func TestPeekNoOpOnFilePanel(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "p"})
	app = updated.(App)
	assert.False(t, app.varList.Peeking, "peek should not activate on file panel")
}

func TestPeekShowsOriginalForModifiedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Edit the variable.
	f.UpdateVar(0, "new_value")
	assert.Equal(t, "val_FOO", f.Vars[0].OriginalValue)

	// Enable peek.
	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show original value")
}

func TestPeekShowsNewVariableHint(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Add a new variable and move cursor to it.
	f.AddVar("NEW_VAR", "some_val", false)
	app.varList.Refresh()
	app.varList.MoveDown() // cursor on NEW_VAR

	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.Contains(t, view, "new variable", "peek should show 'new variable' for added vars")
}

func TestPeekShowsNothingForUnmodifiedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Enable peek without editing anything.
	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.NotContains(t, view, "was:", "peek should not show anything for unmodified vars")
	assert.NotContains(t, view, "new variable")
}

func TestPeekHiddenWhenNotFocused(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.varList.Width = 80
	app.varList.Height = 20

	f.UpdateVar(0, "new_value")

	app.varList.Peeking = true
	app.varList.Focused = false // not focused
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.NotContains(t, view, "was:", "peek line should only show when panel is focused")
}

func TestDeletedVarsShownInVarList(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(1) // delete BAR
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "BAR", "deleted var should still appear in the list")
	assert.Contains(t, view, "-", "deleted var should show - marker")
}

func TestDeletedVarsDisappearAfterReAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(1) // delete BAR
	vlm.Refresh()
	f.AddVar("BAR", "new_val", false) // re-add BAR
	vlm.Refresh()

	assert.Empty(t, f.DeletedVars, "re-added var should be removed from DeletedVars")
}

func TestPeekAfterDeleteAndReAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(0)                      // delete FOO (original value: val_FOO)
	f.AddVar("FOO", "new_value", false) // re-add with different value
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show original value after delete+re-add")
}
