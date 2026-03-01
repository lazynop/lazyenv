package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestCompareExitRestoresCursorToSelected(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "BAR")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f1, f2)
	app.fileList.Selected = 0
	app.varList.SetFile(f1)
	app.focus = FocusVars

	// Enter compare mode: press 'c' to start, move down, Enter to confirm.
	updated, _ := app.Update(tea.KeyPressMsg{Text: "c"})
	app = updated.(App)
	assert.Equal(t, ModeCompareSelect, app.mode)

	updated, _ = app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor, "cursor should have moved to second file")

	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)
	assert.Equal(t, ModeComparing, app.mode)

	// Exit compare with 'q'.
	updated, _ = app.Update(tea.KeyPressMsg{Text: "q"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, app.fileList.Selected, app.fileList.Cursor,
		"cursor must match selected file after exiting compare")
}

func TestCompareSelectEscapeRestoresCursor(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "BAR")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f1, f2)
	app.fileList.Selected = 0
	app.varList.SetFile(f1)
	app.focus = FocusVars

	// Enter compare select mode.
	updated, _ := app.Update(tea.KeyPressMsg{Text: "c"})
	app = updated.(App)
	assert.Equal(t, ModeCompareSelect, app.mode)

	// Move cursor to second file.
	updated, _ = app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor)

	// Press Escape to cancel.
	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, app.fileList.Selected, app.fileList.Cursor,
		"cursor must match selected file after cancelling compare select")
}
