package tui

import (
	"os"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

func TestRenameKey_EntersModeOnE(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("E"))
	app = updated.(App)

	assert.Equal(t, ModeEditing, app.mode)
	assert.Equal(t, "FOO", app.editor.input.Value())
	assert.True(t, app.editor.isRenameKey)
}

func TestRenameKey_OnlyFromFocusVars(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("E"))
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
	updated, _ := app.Update(keyMsg("E"))
	app = updated.(App)
	assert.True(t, app.editor.isRenameKey)

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)

	// Now add a variable — should NOT be treated as rename
	updated, _ = app.Update(keyMsg("a"))
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

// renameKey renames the var under the cursor of app's var list to name, by
// keys.
func renameKey(app App, name string) App {
	return press(app, renameKeys(name)...)
}

// renameKeys returns the keys that rename the var under the cursor of the var
// list to name.
func renameKeys(name string) []string {
	return slices.Concat([]string{"E", "ctrl+u"}, typing(name), []string{"enter"})
}

// renameOntoDeleted deletes the var under the cursor of app's var list, the
// one named deleted, and renames the var the cursor moves onto to that name,
// by keys.
func renameOntoDeleted(app App, deleted string) App {
	return renameKey(press(app, "d", "y"), deleted)
}

// TestRenameKey_OntoADeletedKeyRevivesIt deletes FOO and renames BAR to FOO:
// the var list shows one FOO row, with BAR's value, and no deleted row, and
// peek shows FOO's value as the original.
func TestRenameKey_OntoADeletedKeyRevivesIt(t *testing.T) {
	app := onVars(resize(newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO", "BAR")}), 80, 24))

	app = press(renameOntoDeleted(app, "FOO"), "p")

	require.Equal(t, ModeNormal, app.mode)
	view := stripAnsi(app.View().Content)
	assert.Equal(t, 1, strings.Count(view, "FOO  val_"), "the var list should show one FOO row and no deleted one")
	assert.Contains(t, view, "FOO  val_BAR")
	assert.Contains(t, view, "was: val_FOO", "peek should show the deleted var's value as the original")
}

// TestRenameKey_BackOffARevivedKeyListsItAsDeletedAgain deletes FOO, renames
// BAR to FOO, then back to BAR, as when undoing a typo: the var list shows FOO
// deleted again, and peek shows BAR's own value as its original.
func TestRenameKey_BackOffARevivedKeyListsItAsDeletedAgain(t *testing.T) {
	app := onVars(resize(newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO", "BAR")}), 80, 24))

	app = press(renameKey(renameOntoDeleted(app, "FOO"), "BAR"), "p")

	require.Equal(t, ModeNormal, app.mode)
	view := stripAnsi(app.View().Content)
	assert.Contains(t, view, "BAR  val_BAR")
	assert.Contains(t, view, "was: val_BAR", "peek should show BAR's own value as the original")
	assert.Contains(t, view, "FOO  val_FOO", "the var list should show FOO deleted again")
}

// TestRenameKey_OntoADeletedSecretKeepsItsOriginalMasked deletes GREETING, a
// secret by its value only, and renames BAR to GREETING: BAR's plain value
// stays in clear, and peek shows the original masked.
func TestRenameKey_OntoADeletedSecretKeepsItsOriginalMasked(t *testing.T) {
	f := makeTestFile(".env", "GREETING", "BAR")
	f.Vars[0].Value, f.Vars[0].IsSecret = "ghp_abc123", true
	app := onVars(resize(newTestApp([]*model.EnvFile{f}), 80, 24))

	app = press(renameOntoDeleted(app, "GREETING"), "p")

	require.Equal(t, ModeNormal, app.mode)
	view := stripAnsi(app.View().Content)
	assert.Contains(t, view, "GREETING  val_BAR", "the renamed plain value should stay in clear")
	assert.Contains(t, view, "was: ••••••••", "peek should mask the original of a deleted secret")
	assert.NotContains(t, view, "ghp_abc123", "peek must not reveal the original of a deleted secret")
}

// TestRenameKey_OntoADeletedKeySavesTheRenamedValue deletes FOO, renames BAR
// to FOO and saves: the file holds FOO with BAR's value, and no BAR.
func TestRenameKey_OntoADeletedKeySavesTheRenamedValue(t *testing.T) {
	app, path := newTestAppWithDiskFile(t, "FOO=1\nBAR=2\n")

	press(renameOntoDeleted(app, "FOO"), "w")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "FOO=2\n", string(data))
}
