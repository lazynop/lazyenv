package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
	"github.com/lazynop/lazyenv/internal/util"
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

// TestEditMasksTheNewValueWhenItIsASecret edits a value in the var list: the
// row is masked exactly when the key and the new value are detected as a
// secret, as they are once the file is saved and read again.
func TestEditMasksTheNewValueWhenItIsASecret(t *testing.T) {
	tests := []struct {
		name, line, newValue string
		masked               bool
	}{
		{"plain value edited to a token", "GREETING=hello", "ghp_x7Kq9", true},
		{"token edited to a plain value", "GREETING=ghp_x7Kq9", "hello", false},
		{"secret key edited to a plain value", "API_TOKEN=hunter2", "hello", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parser.ParseBytes(".env", []byte(tt.line+"\n"), config.SecretsConfig{})
			updated, _ := newTestApp([]*model.EnvFile{f}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			app := press(updated.(App), "right", "e")
			app.editor.input.SetValue(tt.newValue)

			view := stripAnsi(press(app, "enter").View().Content)

			assertMaskedInView(t, view, tt.newValue, tt.masked)
		})
	}
}

// assertMaskedInView checks view masks value when masked is set, and shows it
// otherwise.
func assertMaskedInView(t *testing.T, view, value string, masked bool) {
	t.Helper()
	if masked {
		assert.NotContains(t, view, value, "a secret value must be masked")
		assert.Contains(t, view, util.MaskValue(value))
	} else {
		assert.Contains(t, view, value, "a plain value must be shown")
	}
}

// TestConfirmingAnUntouchedEditKeepsTheValue opens the edit bar on a value the
// bar can't show as is (a line break, a control character) and confirms it
// without typing: the value and the file stay unchanged.
func TestConfirmingAnUntouchedEditKeepsTheValue(t *testing.T) {
	for name, open := range map[string][]string{
		"var list": {"right", "e"},
		"compare":  {"c", "j", "enter", "e"},
	} {
		t.Run(name, func(t *testing.T) {
			f := parser.ParseBytes(".env", []byte("CERT=\"line1\\nline2\"\nTAB=\"a\\tb\"\n"), config.SecretsConfig{})
			other := parser.ParseBytes(".env.prod", []byte("CERT=x\n"), config.SecretsConfig{})
			updated, _ := newTestApp([]*model.EnvFile{f, other}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})

			press(updated.(App), append(open, "enter")...)

			assert.Equal(t, "line1\nline2", f.Vars[0].Value)
			assert.False(t, f.Modified, "an untouched edit must not modify the file")
		})
	}
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

func TestToggleSecretsRefiltersKeptSearch(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN", "HUNTER_NAME")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	for _, k := range []string{"/", "h", "u", "n", "t", "e", "r", "enter"} {
		updated, _ := app.Update(tea.KeyPressMsg{Text: k})
		app = updated.(App)
	}
	require.Equal(t, "hunter", app.varList.SearchQuery)
	assert.Equal(t, 1, len(app.varList.displayItems), "the masked secret's value must not match")

	updated, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)
	assert.Equal(t, 2, len(app.varList.displayItems), "revealing secrets must re-filter the kept search")

	updated, _ = app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)
	assert.Equal(t, 1, len(app.varList.displayItems), "hiding secrets must re-filter the kept search")
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
