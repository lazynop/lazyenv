package tui

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
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

// enterCompareMode is a helper to set up two files in compare mode.
func enterCompareMode(t *testing.T, f1, f2 *model.EnvFile) App {
	t.Helper()
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f1, f2)
	app.fileList.Selected = 0
	app.varList.SetFile(f1)
	app.focus = FocusVars

	// Press 'c' to start compare select
	updated, _ := app.Update(tea.KeyPressMsg{Text: "c"})
	app = updated.(App)

	// Move to second file and confirm
	updated, _ = app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)
	assert.Equal(t, ModeComparing, app.mode)
	return app
}

func TestCompareCopyRight(t *testing.T) {
	f1 := makeTestFile(".env", "SHARED", "ONLY_LEFT")
	f2 := makeTestFile(".env.prod", "SHARED")
	// Make SHARED differ
	f2.Vars[0].Value = "different"

	app := enterCompareMode(t, f1, f2)

	// Find the changed entry and copy to right
	for i, e := range app.diffView.Entries {
		if e.Status == model.DiffChanged {
			app.diffView.Cursor = i
			break
		}
	}
	updated, _ := app.Update(tea.KeyPressMsg{Text: "l"})
	app = updated.(App)
	assert.Contains(t, app.statusBar.Message, "SHARED")
}

func TestCompareCopyLeft(t *testing.T) {
	f1 := makeTestFile(".env", "SHARED")
	f2 := makeTestFile(".env.prod", "SHARED", "ONLY_RIGHT")
	f2.Vars[0].Value = "different"

	app := enterCompareMode(t, f1, f2)

	// Find the changed entry and copy to left
	for i, e := range app.diffView.Entries {
		if e.Status == model.DiffChanged {
			app.diffView.Cursor = i
			break
		}
	}
	updated, _ := app.Update(tea.KeyPressMsg{Text: "h"})
	app = updated.(App)
	assert.Contains(t, app.statusBar.Message, "SHARED")
}

func TestCompareFilterToggle(t *testing.T) {
	f1 := makeTestFile(".env", "SAME", "DIFF")
	f2 := makeTestFile(".env.prod", "SAME", "DIFF")
	f2.Vars[1].Value = "other"

	app := enterCompareMode(t, f1, f2)

	totalBefore := len(app.diffView.Entries)
	assert.False(t, app.diffView.HideEqual)

	// Press 'f' to filter
	updated, _ := app.Update(tea.KeyPressMsg{Text: "f"})
	app = updated.(App)
	assert.True(t, app.diffView.HideEqual)
	assert.Less(t, len(app.diffView.Entries), totalBefore)
	assert.Contains(t, app.statusBar.Message, "differences only")

	// Press 'f' again to unfilter
	updated, _ = app.Update(tea.KeyPressMsg{Text: "f"})
	app = updated.(App)
	assert.False(t, app.diffView.HideEqual)
	assert.Contains(t, app.statusBar.Message, "all entries")
}

func TestCompareNavigation(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.prod", "A", "B", "C")
	f2.Vars[1].Value = "different"

	app := enterCompareMode(t, f1, f2)

	assert.Equal(t, 0, app.diffView.Cursor)

	// Move down
	updated, _ := app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	assert.Equal(t, 1, app.diffView.Cursor)

	// Move up
	updated, _ = app.Update(tea.KeyPressMsg{Text: "k"})
	app = updated.(App)
	assert.Equal(t, 0, app.diffView.Cursor)
}

func TestCompareEditLeft(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "FOO")
	f2.Vars[0].Value = "different"

	app := enterCompareMode(t, f1, f2)

	// Find the entry with key FOO
	for i, e := range app.diffView.Entries {
		if e.Key == "FOO" {
			app.diffView.Cursor = i
			break
		}
	}

	// Press 'e' to edit left file
	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)
	assert.Equal(t, ModeEditingCompare, app.mode)
	assert.Equal(t, f1, app.compareEditFile)

	// Cancel with escape
	updated, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updated.(App)
	assert.Equal(t, ModeComparing, app.mode)
}

func TestCompareEditRight(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "FOO")
	f2.Vars[0].Value = "different"

	app := enterCompareMode(t, f1, f2)

	for i, e := range app.diffView.Entries {
		if e.Key == "FOO" {
			app.diffView.Cursor = i
			break
		}
	}

	// Press 'E' (uppercase) to edit right file
	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)
	assert.Equal(t, ModeEditingCompare, app.mode)
	assert.Equal(t, f2, app.compareEditFile)
}

func TestCompareEditConfirm(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "FOO")
	f2.Vars[0].Value = "different"

	app := enterCompareMode(t, f1, f2)

	for i, e := range app.diffView.Entries {
		if e.Key == "FOO" {
			app.diffView.Cursor = i
			break
		}
	}

	// Edit left
	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)

	// Set new value and confirm
	app.editor.input.SetValue("edited_value")
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	assert.Equal(t, ModeComparing, app.mode)
	assert.Equal(t, "edited_value", f1.Vars[0].Value)
	assert.Contains(t, app.statusBar.Message, "Modified")
}

func TestCompareNeedsAtLeast2Files(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f1})
	app.focus = FocusVars

	// With only 1 file, compare should not enter compare mode
	updated, _ := app.Update(tea.KeyPressMsg{Text: "c"})
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestCompareResetUsesKeyMap(t *testing.T) {
	// Compare-mode reset must route through keys.Reset, not a hardcoded 'r'.
	// Rebinding Reset to a different key must take effect.
	// Reset re-parses from disk, so the files have to live on disk.
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=val_FOO\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=val_FOO\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)

	app := enterCompareMode(t, fA, fB)
	app.keys.Reset = key.NewBinding(key.WithKeys("x"))

	updated, _ := app.Update(tea.KeyPressMsg{Text: "x"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Reset to saved state",
		"rebound Reset key must trigger handleCompareReset in compare mode")
}

func TestCompareSaveUsesKeyMap(t *testing.T) {
	// Compare-mode save must route through keys.Save. Sibling of
	// TestCompareResetUsesKeyMap and TestCompareEditRightUsesKeyMap: a
	// rebound Save key must persist the change end-to-end.
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)

	app := enterCompareMode(t, fA, fB)

	// Modify file A so handleCompareSave actually has work to do.
	app.diffView.FileA.UpdateVar(0, "changed")
	require.True(t, app.diffView.FileA.Modified)

	app.keys.Save = key.NewBinding(key.WithKeys("Z"))

	updated, _ := app.Update(tea.KeyPressMsg{Text: "Z"})
	app = updated.(App)

	data, err := os.ReadFile(pathA)
	require.NoError(t, err)
	assert.Equal(t, "FOO=changed\n", string(data),
		"rebound Save key must persist the change to disk in compare mode")
}

func TestCompareEditRightUsesKeyMap(t *testing.T) {
	// Compare-mode edit-right must route through keys.EditRight, not a
	// hardcoded 'E'. Rebinding EditRight to a different key must take effect.
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "FOO")
	app := enterCompareMode(t, f1, f2)
	app.keys.EditRight = key.NewBinding(key.WithKeys("Y"))

	updated, _ := app.Update(tea.KeyPressMsg{Text: "Y"})
	app = updated.(App)

	assert.Equal(t, ModeEditingCompare, app.mode,
		"rebound EditRight key must open the editor on the right file")
	require.NotNil(t, app.compareEditFile)
	assert.Equal(t, ".env.prod", app.compareEditFile.Name,
		"editor target should be the right-hand file (FileB)")
}
