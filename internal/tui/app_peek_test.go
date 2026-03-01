package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
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
	theme := BuildTheme(true)
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
	f.AddVar("NEW_VAR", "some_val")
	app.varList.Refresh()
	app.varList.MoveDown() // cursor on NEW_VAR

	app.varList.Peeking = true
	theme := BuildTheme(true)
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
	theme := BuildTheme(true)
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
	theme := BuildTheme(true)
	view := app.varList.View(theme)

	assert.NotContains(t, view, "was:", "peek line should only show when panel is focused")
}
