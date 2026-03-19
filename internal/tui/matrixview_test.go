package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// makeTestFile creates a test EnvFile with the given name and variable keys.
func makeTestFile(name string, keys ...string) *model.EnvFile {
	vars := make([]model.EnvVar, len(keys))
	lines := make([]model.RawLine, len(keys))
	for i, k := range keys {
		vars[i] = model.EnvVar{Key: k, Value: "val_" + k, LineNum: i + 1}
		lines[i] = model.RawLine{Type: model.LineVariable, Content: k + "=val_" + k, VarIdx: i}
	}
	return &model.EnvFile{Name: name, Vars: vars, Lines: lines}
}

func TestNewMatrixModel(t *testing.T) {
	f1 := makeTestFile(".env", "HOST", "PORT", "DB")
	f2 := makeTestFile(".env.local", "HOST", "SECRET")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})

	// Union of keys: DB, HOST, PORT, SECRET = 4 (sorted alpha)
	assert.Equal(t, 4, len(m.entries), "should have 4 unique keys")
	assert.Equal(t, []string{".env", ".env.local"}, m.fileNames)
	assert.Equal(t, 0, m.cursorRow, "initial cursorRow should be 0")
	assert.Equal(t, 0, m.cursorCol, "initial cursorCol should be 0")
}

func TestMatrixMoveDown(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// entries: A, B, C (3 keys total)
	assert.Equal(t, 0, m.cursorRow)

	m.MoveDown()
	assert.Equal(t, 1, m.cursorRow)

	m.MoveDown()
	assert.Equal(t, 2, m.cursorRow)

	// At the bottom: should not go beyond
	m.MoveDown()
	assert.Equal(t, 2, m.cursorRow, "should stop at last row")
}

func TestMatrixMoveUp(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// Start at 0, move up should stay at 0
	m.MoveUp()
	assert.Equal(t, 0, m.cursorRow, "should stop at 0")

	// Move to row 2, then up
	m.MoveDown()
	m.MoveDown()
	assert.Equal(t, 2, m.cursorRow)

	m.MoveUp()
	assert.Equal(t, 1, m.cursorRow)

	m.MoveUp()
	assert.Equal(t, 0, m.cursorRow)
}

func TestMatrixMoveRight(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	f3 := makeTestFile(".env.prod", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2, f3}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	assert.Equal(t, 0, m.cursorCol)

	m.MoveRight()
	assert.Equal(t, 1, m.cursorCol)

	m.MoveRight()
	assert.Equal(t, 2, m.cursorCol)

	// At the rightmost column: should not go beyond
	m.MoveRight()
	assert.Equal(t, 2, m.cursorCol, "should stop at last column")
}

func TestMatrixMoveLeft(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	f3 := makeTestFile(".env.prod", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2, f3}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// At col 0, move left should stay at 0
	m.MoveLeft()
	assert.Equal(t, 0, m.cursorCol, "should stop at 0")

	// Move to col 2, then left
	m.MoveRight()
	m.MoveRight()
	assert.Equal(t, 2, m.cursorCol)

	m.MoveLeft()
	assert.Equal(t, 1, m.cursorCol)

	m.MoveLeft()
	assert.Equal(t, 0, m.cursorCol)
}

func TestMatrixToggleSort(t *testing.T) {
	// f1 has all keys, f2 is missing B => B has fewer present
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A", "C")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// Default: SortAlpha => A, B, C
	assert.Equal(t, model.SortAlpha, m.sortMode)
	assert.Equal(t, "A", m.entries[0].Key)
	assert.Equal(t, "B", m.entries[1].Key)
	assert.Equal(t, "C", m.entries[2].Key)

	// Toggle to SortCompleteness: B (1 present) should come first, then A (2), C (2)
	m.ToggleSort()
	assert.Equal(t, model.SortCompleteness, m.sortMode)
	assert.Equal(t, "B", m.entries[0].Key, "B has fewest present, should be first")

	// Toggle back to SortAlpha
	m.ToggleSort()
	assert.Equal(t, model.SortAlpha, m.sortMode)
	assert.Equal(t, "A", m.entries[0].Key)
}

func TestMatrixStartEditMissing(t *testing.T) {
	// f1 has A+B, f2 has only A => B is missing in f2
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// entries sorted alpha: A, B
	// Navigate to row=1 (B), col=1 (f2) where B is missing
	m.MoveDown()  // cursorRow = 1 (B)
	m.MoveRight() // cursorCol = 1 (f2)

	require.Equal(t, "B", m.entries[m.cursorRow].Key)
	require.False(t, m.entries[m.cursorRow].Present[m.cursorCol], "B should be missing in f2")

	cmd := m.StartEdit()

	assert.True(t, m.editing, "editing should be true")
	assert.Equal(t, "B", m.editKey)
	assert.Equal(t, 1, m.editFile)
	assert.NotNil(t, cmd, "StartEdit should return a focus command")
}

func TestMatrixStartEditPresent(t *testing.T) {
	// f1 has A+B, f2 has A => A is present in both
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// cursor at (0, 0) = A in f1, which is present
	require.Equal(t, "A", m.entries[0].Key)
	require.True(t, m.entries[0].Present[0], "A should be present in f1")

	_ = m.StartEdit()

	assert.False(t, m.editing, "editing should NOT be set for present cell")
	assert.Equal(t, "Variable already exists", m.message)
}

func TestMatrixConfirmEdit(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// Navigate to missing cell: row=1 (B), col=1 (f2)
	m.MoveDown()
	m.MoveRight()
	m.StartEdit()
	require.True(t, m.editing)

	// Simulate typing a value
	m.editor.SetValue("new_value")

	m.ConfirmEdit()

	assert.False(t, m.editing, "editing should be false after confirm")

	// The variable should have been added to f2
	found := f2.VarByKey("B")
	require.NotNil(t, found, "B should exist in f2 after confirm")
	assert.Equal(t, "new_value", found.Value)

	// Matrix should be recomputed: B should now be present everywhere
	for _, e := range m.entries {
		if e.Key == "B" {
			for fi, p := range e.Present {
				assert.True(t, p, "B should be present in file %d after confirm", fi)
			}
		}
	}
}

func TestMatrixCancelEdit(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	// Navigate to missing cell and start editing
	m.MoveDown()
	m.MoveRight()
	m.StartEdit()
	require.True(t, m.editing)

	origVarCount := len(f2.Vars)

	m.CancelEdit()

	assert.False(t, m.editing, "editing should be false after cancel")
	assert.Equal(t, origVarCount, len(f2.Vars), "f2 should not have new vars after cancel")
}

func TestMatrixViewContainsCheckAndCross(t *testing.T) {
	// f1 has A, f2 does not => one check, one cross
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local")

	// f2 needs at least one var to have a key in the union,
	// but here A is only in f1, so A will be present in f1, absent in f2
	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout, config.SecretsConfig{})
	m.Width = 120
	m.Height = 40

	theme := BuildTheme(true, config.ColorConfig{}) // dark theme
	view := m.View(theme)

	assert.True(t, strings.Contains(view, "\u2713"), "view should contain check mark")
	assert.True(t, strings.Contains(view, "\u2717"), "view should contain cross mark")
}
