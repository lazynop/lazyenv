package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	updated, _ := app.Update(keyMsg("c"))
	app = updated.(App)
	assert.Equal(t, ModeCompareSelect, app.mode)

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor, "cursor should have moved to second file")

	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)
	assert.Equal(t, ModeComparing, app.mode)

	// Exit compare with 'q'.
	updated, _ = app.Update(keyMsg("q"))
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
	updated, _ := app.Update(keyMsg("c"))
	app = updated.(App)
	assert.Equal(t, ModeCompareSelect, app.mode)

	// Move cursor to second file.
	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor)

	// Press Escape to cancel.
	updated, _ = app.Update(keyMsg("esc"))
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

	// Size the views the way a real terminal does
	updated, _ := app.Update(tea.WindowSizeMsg{Width: app.width, Height: app.height})
	app = updated.(App)

	// Press 'c' to start compare select
	updated, _ = app.Update(keyMsg("c"))
	app = updated.(App)

	// Move to second file and confirm
	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)
	assert.Equal(t, ModeComparing, app.mode)
	return app
}

func TestCompareToggleSecretsRefiltersKeptSearch(t *testing.T) {
	f1 := makeTestFile(".env", "API_TOKEN", "HUNTER_NAME")
	f1.Vars[0].Value, f1.Vars[0].IsSecret = "hunter2", true
	f2 := makeTestFile(".env.prod", "API_TOKEN")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f1, f2)
	app.fileList.Selected = 0
	app.varList.SetFile(f1)
	app.focus = FocusVars
	app.varList.SetSearch("hunter")
	require.Equal(t, 1, len(app.varList.displayItems), "the masked secret's value must not match")

	// Compare the files, reveal secrets there, exit.
	for _, k := range []string{"c", "j", "enter", "ctrl+s", "q"} {
		updated, _ := app.Update(keyMsg(k))
		app = updated.(App)
	}

	require.Equal(t, ModeNormal, app.mode)
	assert.True(t, app.varList.ShowSecrets, "the compare toggle carries over to the var list")
	assert.Equal(t, 2, len(app.varList.displayItems), "revealing secrets in compare must re-filter the kept search")
	require.NotNil(t, app.varList.SelectedVar())
	assert.Equal(t, "HUNTER_NAME", app.varList.SelectedVar().Key, "cursor stays on the same var")
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
	updated, _ := app.Update(keyMsg("l"))
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
	updated, _ := app.Update(keyMsg("h"))
	app = updated.(App)
	assert.Contains(t, app.statusBar.Message, "SHARED")
}

// TestCompareCopyMasksTheCopiedValueWhenItIsASecret copies a value in compare
// (#52), then leaves compare and shows the target file: its row is masked
// exactly when the key and the copied value are detected as a secret, as they
// are once the file is saved and read again.
func TestCompareCopyMasksTheCopiedValueWhenItIsASecret(t *testing.T) {
	tests := []struct {
		name, left, right, copyKey string
		showTarget                 []string // keys that leave compare and show the target file
		copied                     string
		masked                     bool
	}{
		{"token copied left", "hello", "ghp_x7Kq9", "h", []string{"q"}, "ghp_x7Kq9", true},
		{"token copied right", "ghp_x7Kq9", "hello", "l", []string{"q", "j"}, "ghp_x7Kq9", true},
		{"plain value copied left", "ghp_x7Kq9", "hello", "h", []string{"q"}, "hello", false},
		{"plain value copied right", "hello", "ghp_x7Kq9", "l", []string{"q", "j"}, "hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := parser.ParseBytes(".env", []byte("GREETING="+tt.left+"\n"), config.SecretsConfig{})
			right := parser.ParseBytes(".env.prod", []byte("GREETING="+tt.right+"\n"), config.SecretsConfig{})
			app := press(enterCompareMode(t, left, right), tt.copyKey)
			app = press(app, tt.showTarget...)
			require.Equal(t, ModeNormal, app.mode)

			view := stripAnsi(app.View().Content)

			assertMaskedInView(t, view, tt.copied, tt.masked)
		})
	}
}

func TestCompareFilterToggle(t *testing.T) {
	f1 := makeTestFile(".env", "SAME", "DIFF")
	f2 := makeTestFile(".env.prod", "SAME", "DIFF")
	f2.Vars[1].Value = "other"

	app := enterCompareMode(t, f1, f2)

	totalBefore := len(app.diffView.Entries)
	assert.False(t, app.diffView.HideEqual)

	// Press 'f' to filter
	updated, _ := app.Update(keyMsg("f"))
	app = updated.(App)
	assert.True(t, app.diffView.HideEqual)
	assert.Less(t, len(app.diffView.Entries), totalBefore)
	assert.Contains(t, app.statusBar.Message, "differences only")

	// Press 'f' again to unfilter
	updated, _ = app.Update(keyMsg("f"))
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
	updated, _ := app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.diffView.Cursor)

	// Move up
	updated, _ = app.Update(keyMsg("k"))
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
	updated, _ := app.Update(keyMsg("e"))
	app = updated.(App)
	assert.Equal(t, ModeEditingCompare, app.mode)
	assert.Equal(t, f1, app.compareEditFile)

	// Cancel with escape
	updated, _ = app.Update(keyMsg("esc"))
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
	updated, _ := app.Update(keyMsg("E"))
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
	updated, _ := app.Update(keyMsg("e"))
	app = updated.(App)

	// Set new value and confirm
	app.editor.input.SetValue("edited_value")
	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)

	assert.Equal(t, ModeComparing, app.mode)
	assert.Equal(t, "edited_value", f1.Vars[0].Value)
	assert.Contains(t, app.statusBar.Message, "Modified")
}

// TestCompareEditMasksTheNewValueWhenItIsASecret edits a value in compare: the
// row is masked exactly when the key and the new value are detected as a
// secret, as they are once the file is saved and read again.
func TestCompareEditMasksTheNewValueWhenItIsASecret(t *testing.T) {
	tests := []struct {
		name, left, right, editKey, newValue string
		masked                               bool
	}{
		{"plain value edited to a token", "localhost", "example.com", "e", "ghp_x7Kq9", true},
		{"token edited to a plain value", "localhost", "ghp_x7Kq9", "E", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := parser.ParseBytes(".env", []byte("HOST="+tt.left+"\n"), config.SecretsConfig{})
			right := parser.ParseBytes(".env.prod", []byte("HOST="+tt.right+"\n"), config.SecretsConfig{})
			app := press(enterCompareMode(t, left, right), tt.editKey)
			require.Equal(t, ModeEditingCompare, app.mode)
			app.editor.input.SetValue(tt.newValue)

			app = press(app, "enter")

			require.Equal(t, ModeComparing, app.mode)
			view := stripAnsi(app.View().Content)
			assertMaskedInView(t, view, tt.newValue, tt.masked)
		})
	}
}

// TestCompareEditRefiltersKeptSearch keeps a search that matches a var by its
// value, edits that value in compare so it no longer matches, and goes back:
// the var list drops the var.
func TestCompareEditRefiltersKeptSearch(t *testing.T) {
	left := parser.ParseBytes(".env", []byte("HOST=localhost\nPORT=80\n"), config.SecretsConfig{})
	right := parser.ParseBytes(".env.prod", []byte("HOST=example.com\n"), config.SecretsConfig{})
	app := enterCompareMode(t, left, right)
	app = press(app, "q", "/", "l", "o", "c", "a", "l", "enter")
	require.Equal(t, 1, len(app.varList.displayItems), "the search should match HOST by its value")

	app = press(app, "c", "j", "enter", "e")
	require.Equal(t, ModeEditingCompare, app.mode)
	app.editor.input.SetValue("remote")
	app = press(app, "enter", "q")

	assert.Empty(t, app.varList.displayItems, "HOST no longer matches the kept search")
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

	updated, _ := app.Update(keyMsg("x"))
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
	app.diffView.FileA.UpdateVar(0, "changed", false)
	require.True(t, app.diffView.FileA.Modified)

	app.keys.Save = key.NewBinding(key.WithKeys("Z"))

	updated, _ := app.Update(keyMsg("Z"))
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

	updated, _ := app.Update(keyMsg("Y"))
	app = updated.(App)

	assert.Equal(t, ModeEditingCompare, app.mode,
		"rebound EditRight key must open the editor on the right file")
	require.NotNil(t, app.compareEditFile)
	assert.Equal(t, ".env.prod", app.compareEditFile.Name,
		"editor target should be the right-hand file (FileB)")
}

func TestCompareCopyDeleteThenExitDoesNotPanic(t *testing.T) {
	// Copying a left-only entry to the left deletes it from the active file
	// (FileA). The var list must be refreshed before returning to normal
	// mode, or its display rows still reference the removed var index and
	// the next render indexes out of range.
	f1 := makeTestFile(".env", "SHARED", "ONLY_LEFT")
	f2 := makeTestFile(".env.prod", "SHARED")

	app := enterCompareMode(t, f1, f2)

	// Move the cursor to the entry that exists only in the left file.
	for i, e := range app.diffView.Entries {
		if e.Status == model.DiffAdded {
			app.diffView.Cursor = i
			break
		}
	}

	// 'h' copies to the left: for a left-only entry this deletes it from f1.
	updated, _ := app.Update(keyMsg("h"))
	app = updated.(App)
	require.Len(t, f1.Vars, 1, "ONLY_LEFT should have been deleted from the active file")

	// Exit compare mode back to normal.
	updated, _ = app.Update(keyMsg("esc"))
	app = updated.(App)
	require.Equal(t, ModeNormal, app.mode)

	assert.Equal(t, 1, len(app.varList.displayItems),
		"var list rows must be recomputed after compare mutated the active file")
	assert.NotPanics(t, func() { app.View() },
		"rendering after compare exit must not index removed vars")
}

// panelBottomRow returns the row of the last panel bottom border in a
// rendered view, or -1 when there is none.
func panelBottomRow(view string) int {
	row := -1
	for i, line := range strings.Split(stripAnsi(view), "\n") {
		if strings.Contains(line, "╰") {
			row = i
		}
	}
	return row
}

func TestCompareViewFillsTheNormalPanelHeight(t *testing.T) {
	app := enterCompareMode(t, makeTestFile(".env", "FOO", "BAR"), makeTestFile(".env.prod", "FOO", "BAZ"))
	normal, _ := newTestApp(nil).Update(tea.WindowSizeMsg{Width: app.width, Height: app.height})
	normalBottom := panelBottomRow(normal.(App).View().Content)
	require.Positive(t, normalBottom)

	assert.Equal(t, normalBottom, panelBottomRow(app.View().Content),
		"the diff panels should end on the same row as the normal panels")

	app = sendKey(app, keyMsg("e"))
	require.Equal(t, ModeEditingCompare, app.mode)
	assert.LessOrEqual(t, lipgloss.Height(app.View().Content), app.height,
		"the editor bar must still fit in the terminal")
}

// TestCompareSelectQuitKeysCancel presses q and Ctrl+C while picking the
// second file to compare: like Esc, they cancel the pick and go back to the
// main view, as they go back from compare and matrix, instead of quitting.
func TestCompareSelectQuitKeysCancel(t *testing.T) {
	for name, msg := range map[string]tea.KeyPressMsg{
		"q":      {Code: 'q', Text: "q"},
		"ctrl+c": {Code: 'c', Mod: tea.ModCtrl},
	} {
		t.Run(name, func(t *testing.T) {
			f1, f2 := makeTestFile(".env", "FOO"), makeTestFile(".env.prod", "BAR")
			app := press(newTestApp([]*model.EnvFile{f1, f2}), "c", "j")
			require.Equal(t, ModeCompareSelect, app.mode)

			updated, cmd := app.Update(msg)
			app = updated.(App)

			assert.Nil(t, cmd, "%s must not quit", name)
			assert.Equal(t, ModeNormal, app.mode)
			assert.Equal(t, app.fileList.Selected, app.fileList.Cursor, "the cursor goes back to the active file")
			assert.Nil(t, app.compareFirstFile)
		})
	}
}

// TestCompareKeyNeedsAtLeast2Files presses c with a single file: like m, it
// stays in the main view and says why.
func TestCompareKeyNeedsAtLeast2Files(t *testing.T) {
	app := press(newTestApp([]*model.EnvFile{makeTestFile(".env", "A")}), "c")

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "at least 2 files")
}

// TestCompareSelectEnterOnTheFirstFileWarns presses Enter while picking the
// second file with the cursor still on the first: the pick stays open and
// says why nothing happened.
func TestCompareSelectEnterOnTheFirstFileWarns(t *testing.T) {
	app := press(newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B")}), "c", "enter")

	assert.Equal(t, ModeCompareSelect, app.mode)
	assert.Contains(t, app.statusBar.Message, "different file")
}

// TestCompareSelectPromptReturnsAfterAWarning lets the warning from Enter on
// the first file expire: the picker's prompt shows again.
func TestCompareSelectPromptReturnsAfterAWarning(t *testing.T) {
	app := press(newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B")}), "c")
	require.Contains(t, stripAnsi(app.View().Content), "Select second file to compare")

	app = press(app, "enter")
	updated, _ := app.Update(ClearMessageMsg{gen: app.statusBar.msgGen})

	assert.Contains(t, stripAnsi(updated.(App).View().Content), "Select second file to compare")
}
