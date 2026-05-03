package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

// newGroupingApp returns an App focused on the var panel with a file that
// produces two named groups (DB, REDIS) plus an Ungrouped bucket.
func newGroupingApp(t *testing.T) (App, *model.EnvFile) {
	t.Helper()
	f := makeTestFile(".env",
		"DB_HOST", "DB_PORT", "DB_USER",
		"REDIS_URL", "REDIS_PORT",
		"PORT", "DEBUG",
	)
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.varList.Focused = true
	app.fileList.Focused = false
	return app, f
}

func TestApp_ToggleGroupingFlash(t *testing.T) {
	app, _ := newGroupingApp(t)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "g"})
	app = updated.(App)

	assert.True(t, app.varList.Grouping, "Grouping should be enabled")
	assert.Contains(t, app.statusBar.Message, "Grouping enabled")
	assert.Contains(t, app.statusBar.Message, "2 groups")
}

func TestApp_ToggleGroupingDisableFlash(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()

	updated, _ := app.Update(tea.KeyPressMsg{Text: "g"})
	app = updated.(App)

	assert.False(t, app.varList.Grouping)
	assert.Contains(t, app.statusBar.Message, "Grouping disabled")
}

func TestApp_EnterOnHeaderTogglesCollapse(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.Update(tea.KeyPressMsg{Text: "g"}) // ignore: just to keep state local
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0) // DB header
	require.True(t, app.varList.IsHeaderAtCursor())

	// Enter is a special key — Code is meaningful, Text is empty.
	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.True(t, app.varList.IsCollapsed("DB"), "DB should be collapsed after Enter on header")

	// Pressing Enter again expands.
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)
	assert.False(t, app.varList.IsCollapsed("DB"), "DB should be expanded after second Enter")
}

func TestApp_SpaceOnHeaderTogglesCollapse(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0) // DB header
	require.True(t, app.varList.IsHeaderAtCursor())

	// Space: Text=" ", Code=' ' (space rune)
	updated, _ := app.Update(tea.KeyPressMsg{Text: " ", Code: ' '})
	app = updated.(App)

	assert.True(t, app.varList.IsCollapsed("DB"), "DB should be collapsed after Space on header")
}

func TestApp_EnterOnVarStillWorksOnFileFocus(t *testing.T) {
	// Regression: enter on file focus must still load the file as before.
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newTestApp([]*model.EnvFile{f1, f2})
	app.focus = FocusFiles
	app.fileList.Focused = true
	app.varList.Focused = false
	app.fileList.Cursor = 1 // .env.local

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, FocusVars, app.focus, "focus should switch to vars")
	assert.Equal(t, ".env.local", app.varList.File.Name)
}

func TestApp_EditOnHeaderFlashes(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0) // DB header
	require.True(t, app.varList.IsHeaderAtCursor())

	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "must not enter edit mode on header")
	assert.Contains(t, app.statusBar.Message, "No variable selected")
}

func TestApp_RenameKeyOnHeaderFlashes(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "No variable selected")
}

func TestApp_AddOnHeaderFlashes(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "a"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "Add must not open the editor on header")
	assert.Contains(t, app.statusBar.Message, "No variable selected")
}

func TestApp_DeleteOnHeaderFlashes(t *testing.T) {
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(0)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "must not enter ModeConfirmDelete on header")
	assert.Contains(t, app.statusBar.Message, "No variable selected")
}

func TestApp_EditOnVarStillWorksWithGrouping(t *testing.T) {
	// Regression: when grouping is on and the cursor is on a var (not a
	// header), editing must still open the editor.
	app, _ := newGroupingApp(t)
	app.varList.Grouping = true
	app.varList.Refresh()
	app.varList.SetCursor(1) // first DB var (DB_HOST)
	require.False(t, app.varList.IsHeaderAtCursor())

	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)

	assert.Equal(t, ModeEditing, app.mode, "edit on a var must still work")
}
