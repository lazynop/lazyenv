package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/model"
)

func TestRenameKey_EntersModeOnE(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)

	assert.Equal(t, ModeEditing, app.mode)
	assert.Equal(t, "FOO", app.editor.input.Value())
	assert.True(t, app.editor.isRenameKey)
}

func TestRenameKey_OnlyFromFocusVars(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)

	assert.NotEqual(t, ModeEditing, app.mode)
}

func TestRenameKey_Success(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("NEW_FOO")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "NEW_FOO", f.Vars[0].Key)
	assert.True(t, f.Modified)
	assert.Contains(t, app.statusBar.Message, "Renamed")
	assert.Contains(t, app.statusBar.Message, "→")
}

func TestRenameKey_SameNameNoOp(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("FOO")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}

func TestRenameKey_EmptyNameNoOp(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "FOO", f.Vars[0].Key)
}

func TestRenameKey_InvalidKey(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("has space")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Invalid key")
	assert.Equal(t, "FOO", f.Vars[0].Key)
}

func TestRenameKey_DuplicateKey(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("BAR")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "already exists")
	assert.Equal(t, "FOO", f.Vars[0].Key)
}

func TestRenameKey_EscapeCancels(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("NEW_NAME")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "FOO", f.Vars[0].Key)
}

func TestRenameKey_UpdatesSecretDetection(t *testing.T) {
	f := makeTestFile(".env", "BORING_VAR")
	f.Vars[0].IsSecret = false
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("DB_PASSWORD")

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, "DB_PASSWORD", f.Vars[0].Key)
	assert.True(t, f.Vars[0].IsSecret)
}

func TestRenameKey_ThenAddDoesNotConfuseFlows(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Start rename, then cancel
	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)
	assert.True(t, app.editor.isRenameKey)

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)

	// Now add a variable — should NOT be treated as rename
	updated, _ = app.Update(tea.KeyPressMsg{Text: "a"})
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)
	assert.False(t, app.editor.isRenameKey)
	assert.True(t, app.editor.isAdd)

	// Type key name and confirm
	app.editor.input.SetValue("NEW_VAR")
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	// Should be in add-value step, not confirmRenameKey
	assert.Equal(t, ModeEditing, app.mode)
	assert.Contains(t, app.editor.label, "NEW_VAR")
}
