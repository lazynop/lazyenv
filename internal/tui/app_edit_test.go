package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
)

// --- handleEditingKey tests ---

func TestEditExistingVarConfirm(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'e' to start editing FOO (cursor at 0)
	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	// Type new value by setting it directly on editor input
	app.editor.input.SetValue("new_value")

	// Press Enter to confirm
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "new_value", f.Vars[0].Value)
	assert.True(t, f.Modified)
}

func TestEditExistingVarCancel(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'e' to start editing
	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	// Change value
	app.editor.input.SetValue("changed")

	// Press Escape to cancel
	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "val_FOO", f.Vars[0].Value, "value should be unchanged after cancel")
}

func TestAddVariableTwoStepFlow(t *testing.T) {
	f := makeTestFile(".env", "EXISTING")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'a' to start add flow
	updated, _ := app.Update(tea.KeyPressMsg{Text: "a"})
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	// Enter key name
	app.editor.input.SetValue("NEW_KEY")
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	// Should still be in editing mode (value step)
	assert.Equal(t, ModeEditing, app.mode)
	assert.Equal(t, addStepValue, app.editor.addStep)

	// Enter value
	app.editor.input.SetValue("new_value")
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	require.Len(t, f.Vars, 2)
	assert.Equal(t, "NEW_KEY", f.Vars[1].Key)
	assert.Equal(t, "new_value", f.Vars[1].Value)
}

func TestEditWithNoVarSelected(t *testing.T) {
	// Empty file — no vars to edit
	f := makeTestFile(".env")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'e' — should stay in normal mode since no var is selected
	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

// --- handleConfirmDeleteKey tests ---

func TestDeleteConfirmYes(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'd' to trigger delete confirmation
	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)
	assert.Equal(t, ModeConfirmDelete, app.mode)

	// Press 'y' to confirm
	updated, _ = app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "one var should be deleted")
	assert.True(t, f.Modified)
}

func TestDeleteConfirmNo(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press 'd' to trigger delete
	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)
	assert.Equal(t, ModeConfirmDelete, app.mode)

	// Press 'n' to deny
	updated, _ = app.Update(tea.KeyPressMsg{Text: "n"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "no vars should be deleted")
}

func TestDeleteConfirmEscape(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)

	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "no vars should be deleted on escape")
}

// --- handleHelpKey tests ---

func TestHelpToggle(t *testing.T) {
	app := newTestApp(nil)

	// Press '?' to open help
	updated, _ := app.Update(tea.KeyPressMsg{Text: "?"})
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	// Press Escape to close help
	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestHelpToggleWithQuestionMark(t *testing.T) {
	app := newTestApp(nil)

	// Open help
	updated, _ := app.Update(tea.KeyPressMsg{Text: "?"})
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	// Close with '?' again
	updated, _ = app.Update(tea.KeyPressMsg{Text: "?"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestHelpExitWithQuit(t *testing.T) {
	app := newTestApp(nil)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "?"})
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	// 'q' also exits help
	updated, _ = app.Update(tea.KeyPressMsg{Text: "q"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

// --- handleSearchKey tests ---

func TestSearchFlowEscape(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press '/' to start search
	updated, _ := app.Update(tea.KeyPressMsg{Text: "/"})
	app = updated.(App)
	assert.Equal(t, ModeSearching, app.mode)

	// Escape clears search and exits
	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "", app.varList.SearchQuery, "search should be cleared on escape")
}

func TestSearchFlowEnter(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	// Press '/' to start search
	updated, _ := app.Update(tea.KeyPressMsg{Text: "/"})
	app = updated.(App)
	assert.Equal(t, ModeSearching, app.mode)

	// Set search text directly
	app.searchInput.SetValue("FOO")
	app.varList.SetSearch("FOO")

	// Press Enter to keep filter
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "FOO", app.varList.SearchQuery, "search should be preserved on enter")
}

// --- Normal mode additional tests ---

func TestFocusSwitching(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})

	// Default focus is files
	assert.Equal(t, FocusFiles, app.focus)

	// Press right to focus vars
	updated, _ := app.Update(tea.KeyPressMsg{Text: "l"})
	app = updated.(App)
	assert.Equal(t, FocusVars, app.focus)
	assert.True(t, app.varList.Focused)
	assert.False(t, app.fileList.Focused)

	// Press left to focus files
	updated, _ = app.Update(tea.KeyPressMsg{Text: "h"})
	app = updated.(App)
	assert.Equal(t, FocusFiles, app.focus)
	assert.True(t, app.fileList.Focused)
	assert.False(t, app.varList.Focused)
}

func TestToggleSecrets(t *testing.T) {
	f := makeTestFile(".env", "SECRET_KEY")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	assert.False(t, app.varList.ShowSecrets)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)
	assert.True(t, app.varList.ShowSecrets)

	updated, _ = app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)
	assert.False(t, app.varList.ShowSecrets)
}

func TestToggleSort(t *testing.T) {
	f := makeTestFile(".env", "ZZZ", "AAA", "MMM")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	assert.False(t, app.varList.SortAlpha)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "o"})
	app = updated.(App)
	assert.True(t, app.varList.SortAlpha)

	updated, _ = app.Update(tea.KeyPressMsg{Text: "o"})
	app = updated.(App)
	assert.False(t, app.varList.SortAlpha)
}

func TestDeleteWithNoFile(t *testing.T) {
	// No vars — 'd' should not enter confirm delete
	app := newTestApp(nil)
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}
