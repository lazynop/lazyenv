package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

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

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	assert.Equal(t, 0, app.matrixView.rows.Cursor)
	assert.Equal(t, 0, app.matrixView.cols.Cursor)

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.rows.Cursor, "j should move cursor down")

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 2, app.matrixView.rows.Cursor)

	updated, _ = app.Update(keyMsg("k"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.rows.Cursor, "k should move cursor up")

	updated, _ = app.Update(keyMsg("l"))
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.cols.Cursor, "l should move cursor right")

	updated, _ = app.Update(keyMsg("h"))
	app = updated.(App)
	assert.Equal(t, 0, app.matrixView.cols.Cursor, "h should move cursor left")
}

func TestMatrixEscapeReturnsToNormal(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "Escape should return to ModeNormal")
}

func TestMatrixAddVariableFlow(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("l"))
	app = updated.(App)

	require.Equal(t, 1, app.matrixView.rows.Cursor)
	require.Equal(t, 1, app.matrixView.cols.Cursor)
	require.Equal(t, "B", app.matrixView.entries[1].Key)
	require.False(t, app.matrixView.entries[1].Present[1], "B should be missing in f2")

	updated, focusCmd := app.Update(keyMsg("a"))
	app = updated.(App)
	assert.Equal(t, ModeMatrixEditing, app.mode, "pressing 'a' on missing cell should enter editing mode")

	if focusCmd != nil {
		msg := focusCmd()
		if msg != nil {
			updated, _ = app.Update(msg)
			app = updated.(App)
		}
	}

	app.matrixView.editor.SetValue("added_value")

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "after Enter, should return to ModeMatrix")

	found := f2.VarByKey("B")
	require.NotNil(t, found, "B should exist in f2 after adding")
	assert.Equal(t, "added_value", found.Value)
}

func TestMatrixDeleteVariableConfirm(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A", "B")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	require.True(t, app.matrixView.entries[0].Present[0], "A should be present in f1, the cell under the cursor")

	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmMatrixDelete, app.mode)
	assert.Contains(t, app.statusBar.Message, "Delete")
	assert.Contains(t, app.statusBar.Message, "A")

	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)

	assert.Nil(t, f1.VarByKey("A"), "A should be deleted from f1")
	assert.True(t, f1.Modified)
}

func TestMatrixDeleteFlashNamesTheFile(t *testing.T) {
	local := makeTestFile(".env.local", "A", "B")
	app := press(newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), local}), "m", "j", "l", "d", "y")
	require.Equal(t, ModeMatrix, app.mode)
	assert.Nil(t, local.VarByKey("B"), "B should be deleted from .env.local")
	assert.Equal(t, "Deleted B from .env.local", app.statusBar.Message)
}

func TestMatrixDeleteRemovesEveryOccurrenceOfTheKey(t *testing.T) {
	env := parser.ParseBytes(".env", []byte("A=0\nD=0\n"), config.SecretsConfig{})
	local := parser.ParseBytes(".env.local", []byte("D=1\nA=2\nD=3\n"), config.SecretsConfig{})

	app := press(newTestApp([]*model.EnvFile{env, local}), "m", "j", "l", "d", "y")

	require.Equal(t, ModeMatrix, app.mode)
	assert.Equal(t, "A=2\n", string(parser.Marshal(local)), "every D should leave .env.local")
	assert.Equal(t, "A=0\nD=0\n", string(parser.Marshal(env)), ".env, the file the var list shows, should keep its D")
	cell, ok := app.matrixView.CursorCell()
	require.True(t, ok)
	assert.Equal(t, MatrixCell{Key: "D", File: local, Present: false}, cell, "the cell should show ✗")
	assert.Equal(t, "Deleted D from .env.local", app.statusBar.Message)
}

func TestMatrixEditKeysWithoutVariables(t *testing.T) {
	app := press(newTestApp([]*model.EnvFile{makeTestFile(".env"), makeTestFile(".env.local")}), "m")
	require.Equal(t, ModeMatrix, app.mode)
	for _, k := range []string{"a", "d"} {
		pressed := press(app, k)
		assert.Equal(t, ModeMatrix, pressed.mode, "%s should leave the matrix as it is", k)
		assert.Empty(t, pressed.statusBar.Message, "%s should show no message", k)
	}
}

func TestMatrixDeleteVariableDeny(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)
	require.Equal(t, ModeMatrix, app.mode)

	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmMatrixDelete, app.mode)

	updated, _ = app.Update(keyMsg("n"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)
	assert.NotNil(t, f1.VarByKey("A"), "A should still exist after deny")
}

func TestMatrixDeleteVariableEscape(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

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
	app.varList.SetFile(f1)

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode)
	require.Len(t, f1.Vars, 1, "deleting A, then B on the next row (A's cell, still listed for f2, now shows ✗), should leave f1 one var")

	assert.NotPanics(t, func() {
		updated, _ = app.Update(keyMsg("q"))
		app = updated.(App)
		app.View()
	}, "back in the main view, rendering the var list (calcColumnWidths) after matrix deletes should not panic")
	assert.Equal(t, ModeNormal, app.mode)
}

func TestMatrixDeleteNotPresent(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")
	app := newTestApp([]*model.EnvFile{f1, f2})

	updated, _ := app.Update(keyMsg("m"))
	app = updated.(App)

	updated, _ = app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("l"))
	app = updated.(App)
	require.False(t, app.matrixView.entries[1].Present[1], "B should be missing in f2")

	updated, _ = app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode, "should stay in matrix mode")
	assert.Contains(t, app.statusBar.Message, "not present")
}

func TestMatrixAddShowsInTheVarList(t *testing.T) {
	env := makeTestFile(".env", "A")
	app := press(resize(newTestApp([]*model.EnvFile{env, makeTestFile(".env.prod", "A", "B")}), 80, 24), "m", "j", "a")
	require.Equal(t, ModeMatrixEditing, app.mode)
	app.matrixView.editor.SetValue("added")

	app = press(app, "enter", "esc")

	require.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.varList.displayItems, 2, "the var list should list B")
}
