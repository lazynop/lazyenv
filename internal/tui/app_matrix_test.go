package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// newTestApp creates an App ready for testing with the given files loaded.
func newTestApp(files []*model.EnvFile) App {
	app := NewApp(config.DefaultConfig(), StartupMessages{})
	app.width = 120
	app.height = 40
	app.ready = true
	app.fileList.SetFiles(files)
	if len(files) > 0 {
		app.varList.SetFile(files[0])
	}
	return app
}

func TestMatrixKeyNeedsAtLeast2Files(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	app := newTestApp([]*model.EnvFile{f1})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "mode should stay ModeNormal with only 1 file")
	assert.Contains(t, app.statusBar.Message, "at least 2 files",
		"status bar should indicate 2 files required")
}

func TestMatrixKeyOpensMatrix(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A", "C")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "pressing 'm' with 2 files should open matrix")
	assert.Equal(t, 3, len(app.matrixView.entries), "matrix should have 3 unique keys (A, B, C)")
}

func TestMatrixNavigationInApp(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A", "B")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	// Initial cursor position
	assert.Equal(t, 0, app.matrixView.rows.Cursor)
	assert.Equal(t, 0, app.matrixView.cols.Cursor)

	// Move down with 'j'
	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.rows.Cursor, "j should move cursor down")

	// Move down again with 'j'
	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 2, app.matrixView.rows.Cursor)

	// Move up with 'k'
	updated, _ = app.Update(keyMsg("k"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.rows.Cursor, "k should move cursor up")

	// Move right with 'l'
	updated, _ = app.Update(keyMsg("l"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.cols.Cursor, "l should move cursor right")

	// Move left with 'h'
	updated, _ = app.Update(keyMsg("h"))
	app = updated.(App)
	assert.Equal(t, 0, app.matrixView.cols.Cursor, "h should move cursor left")
}

func TestMatrixEscapeReturnsToNormal(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	// Press Escape
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "Escape should return to ModeNormal")
}

func TestMatrixAddVariableFlow(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	// entries sorted alpha: A, B
	// Navigate to row=1 (B), col=1 (f2) where B is missing
	updated, _ = app.Update(keyMsg("j")) // down to B
	app = updated.(App)
	updated, _ = app.Update(keyMsg("l")) // right to f2
	app = updated.(App)

	require.Equal(t, 1, app.matrixView.rows.Cursor)
	require.Equal(t, 1, app.matrixView.cols.Cursor)
	require.Equal(t, "B", app.matrixView.entries[1].Key)
	require.False(t, app.matrixView.entries[1].Present[1], "B should be missing in f2")

	// Press 'a' to start adding
	updated, cmd := app.Update(keyMsg("a"))
	app = updated.(App)
	assert.Equal(t, ModeMatrixEditing, app.mode, "pressing 'a' on missing cell should enter editing mode")

	// Process the focus command (textinput.Focus)
	if cmd != nil {
		msg := cmd()
		if msg != nil {
			updated, _ = app.Update(msg)
			app = updated.(App)
		}
	}

	// Type a value into the editor
	app.matrixView.editor.SetValue("added_value")

	// Press Enter to confirm
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "after Enter, should return to ModeMatrix")

	// Verify variable was added to f2
	found := f2.VarByKey("B")
	require.NotNil(t, found, "B should exist in f2 after adding")
	assert.Equal(t, "added_value", found.Value)
}

func TestMatrixDeleteVariableConfirm(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A", "B")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	// Cursor at row=0 (A), col=0 (f1) — A is present
	require.True(t, app.matrixView.entries[0].Present[0])

	// Press 'd' to delete
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmMatrixDelete, app.mode)
	assert.Contains(t, app.statusBar.Message, "Delete")
	assert.Contains(t, app.statusBar.Message, "A")

	// Confirm with 'y'
	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)

	// Verify A was deleted from f1
	assert.Nil(t, f1.VarByKey("A"), "A should be deleted from f1")
	assert.True(t, f1.Modified)
}

func TestMatrixDeleteVariableDeny(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	// Press 'd'
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmMatrixDelete, app.mode)

	// Deny with 'n'
	updated, _ = app.Update(keyMsg("n"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)
	assert.NotNil(t, f1.VarByKey("A"), "A should still exist after deny")
}

func TestMatrixDeleteVariableEscape(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	// Press 'd' then Escape
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode)
	assert.NotNil(t, f1.VarByKey("A"), "A should still exist after escape")
}

func TestMatrixDeleteThenReturnToNormal(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A", "B", "C")
	app := newTestApp([]*model.EnvFile{f1, f2})
	// Select f1 so varList shows f1's vars
	app.varList.SetFile(f1)

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	// Delete A from f1 (row=0, col=0)
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)

	// After recompute: A still in entries (exists in f2), cursor at row=0 col=0 = A in f1 (now ✗).
	// Move to row=1 (B) which is still present in f1.
	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode)
	require.Len(t, f1.Vars, 1, "f1 should have 1 var left")

	// Return to normal — this should NOT panic
	assert.NotPanics(t, func() {
		updated, _ = app.Update(keyMsg("q"))
		app = updated.(App)
		// Force a View render to trigger calcColumnWidths
		app.View()
	})
	assert.Equal(t, ModeNormal, app.mode)
}

func TestMatrixDeleteNotPresent(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	// Open matrix
	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	// Navigate to B in f2 (not present)
	updated, _ = app.Update(keyMsg("j")) // down to B
	app = updated.(App)
	updated, _ = app.Update(keyMsg("l")) // right to f2
	app = updated.(App)
	require.False(t, app.matrixView.entries[1].Present[1], "B should be missing in f2")

	// Press 'd' — should flash message, not enter confirm
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode, "should stay in matrix mode")
	assert.Contains(t, app.statusBar.Message, "not present")
}

// TestMatrixAddShowsInTheVarList adds a var to the file the var list shows,
// from the matrix: back in the var list, the new var is listed.
func TestMatrixAddShowsInTheVarList(t *testing.T) {
	env := makeTestFile(".env", "A")
	app := press(resize(newTestApp([]*model.EnvFile{env, makeTestFile(".env.prod", "A", "B")}), 80, 24), "m", "j", "a") // B, missing from .env
	require.Equal(t, ModeMatrixEditing, app.mode)
	app.matrixView.editor.SetValue("added")

	app = press(app, "enter", "esc")

	require.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.varList.displayItems, 2, "the var list should list B")
}
