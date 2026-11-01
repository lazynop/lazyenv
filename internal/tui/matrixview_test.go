package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)

	// Union of keys: DB, HOST, PORT, SECRET = 4 (sorted alpha)
	assert.Equal(t, 4, len(m.entries), "should have 4 unique keys")
	assert.Equal(t, []*model.EnvFile{f1, f2}, m.files, "the columns should be the files, in order")
	assert.Equal(t, 0, m.rows.Cursor, "initial rows.Cursor should be 0")
	assert.Equal(t, 0, m.cols.Cursor, "initial cols.Cursor should be 0")
}

func TestMatrixMoveDown(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	// entries: A, B, C (3 keys total)
	assert.Equal(t, 0, m.rows.Cursor)

	m.MoveDown()
	assert.Equal(t, 1, m.rows.Cursor)

	m.MoveDown()
	assert.Equal(t, 2, m.rows.Cursor)

	// At the bottom: should not go beyond
	m.MoveDown()
	assert.Equal(t, 2, m.rows.Cursor, "should stop at last row")
}

func TestMatrixMoveUp(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	// Start at 0, move up should stay at 0
	m.MoveUp()
	assert.Equal(t, 0, m.rows.Cursor, "should stop at 0")

	// Move to row 2, then up
	m.MoveDown()
	m.MoveDown()
	assert.Equal(t, 2, m.rows.Cursor)

	m.MoveUp()
	assert.Equal(t, 1, m.rows.Cursor)

	m.MoveUp()
	assert.Equal(t, 0, m.rows.Cursor)
}

func TestMatrixMoveRight(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	f3 := makeTestFile(".env.prod", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2, f3}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	assert.Equal(t, 0, m.cols.Cursor)

	m.MoveRight()
	assert.Equal(t, 1, m.cols.Cursor)

	m.MoveRight()
	assert.Equal(t, 2, m.cols.Cursor)

	// At the rightmost column: should not go beyond
	m.MoveRight()
	assert.Equal(t, 2, m.cols.Cursor, "should stop at last column")
}

func TestMatrixMoveLeft(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local", "A")
	f3 := makeTestFile(".env.prod", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2, f3}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	// At col 0, move left should stay at 0
	m.MoveLeft()
	assert.Equal(t, 0, m.cols.Cursor, "should stop at 0")

	// Move to col 2, then left
	m.MoveRight()
	m.MoveRight()
	assert.Equal(t, 2, m.cols.Cursor)

	m.MoveLeft()
	assert.Equal(t, 1, m.cols.Cursor)

	m.MoveLeft()
	assert.Equal(t, 0, m.cols.Cursor)
}

func TestMatrixToggleSort(t *testing.T) {
	// f1 has all keys, f2 is missing B => B has fewer present
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A", "C")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
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

// newMissingCellMatrix builds a matrix over .env, .env.local and .env.prod,
// which all hold A, B and D while only .env.prod holds C, and puts the cursor
// on the gap C leaves in .env.local: row 2, column 1. The cursor's file,
// .env.local, differs from every file a wrong pick would read: the first file,
// which is also the first one lacking C, and the last file, which is also the
// one at the row's index; the cursor's row is not the last.
// newMissingCellMatrix returns .env.local with the matrix.
func newMissingCellMatrix() (MatrixModel, *model.EnvFile) {
	missing := makeTestFile(".env.local", "A", "B", "D")
	files := []*model.EnvFile{makeTestFile(".env", "A", "B", "D"), missing, makeTestFile(".env.prod", "A", "B", "C", "D")}
	m := NewMatrixModel(files, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40
	m.SetCursor(2, 1)
	return m, missing
}

func TestMatrixStartEditMissing(t *testing.T) {
	m, missing := newMissingCellMatrix()

	cell, ok := m.CursorCell()
	require.True(t, ok)
	require.Equal(t, "C", cell.Key)
	require.False(t, cell.Present, "C should be missing in .env.local")

	cmd := m.StartEdit()

	assert.Equal(t, "C", m.edit.Key)
	assert.Same(t, missing, m.edit.File, "the edit should fill .env.local")
	assert.NotNil(t, cmd, "StartEdit should return a focus command")
}

func TestMatrixStartEditPresent(t *testing.T) {
	// f1 has A+B, f2 has A => A is present in both
	f1 := makeTestFile(".env", "A", "B")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	// cursor at (0, 0) = A in f1, which is present
	require.Equal(t, "A", m.entries[0].Key)
	require.True(t, m.entries[0].Present[0], "A should be present in f1")

	_ = m.StartEdit()

	assert.Nil(t, m.edit.File, "no edit should open on a present cell")
}

func TestMatrixConfirmEdit(t *testing.T) {
	m, missing := newMissingCellMatrix()
	m.StartEdit()
	require.NotNil(t, m.edit.File)

	// Simulate typing a value
	m.editor.SetValue("new_value")

	assert.Same(t, missing, m.ConfirmEdit(), "ConfirmEdit should return the file it added to")

	assert.Nil(t, m.edit.File, "the edit should close after confirm")

	// The variable should have been added to .env.local
	found := missing.VarByKey("C")
	require.NotNil(t, found, "C should exist in .env.local after confirm")
	assert.Equal(t, "new_value", found.Value)

	// Matrix should be recomputed: C should now be present in .env.local too
	require.Len(t, m.entries, 4)
	require.Equal(t, "C", m.entries[2].Key)
	assert.Equal(t, []bool{false, true, true}, m.entries[2].Present, "C should be present in .env.local after confirm")
}

func TestMatrixCancelEdit(t *testing.T) {
	m, missing := newMissingCellMatrix()
	m.StartEdit()
	require.NotNil(t, m.edit.File)

	origVarCount := len(missing.Vars)

	m.CancelEdit()

	assert.Nil(t, m.edit.File, "the edit should close after cancel")
	assert.Equal(t, origVarCount, len(missing.Vars), ".env.local should not have new vars after cancel")
}

func TestMatrixViewContainsCheckAndCross(t *testing.T) {
	// f1 has A, f2 does not => one check, one cross
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local")

	// f2 needs at least one var to have a key in the union,
	// but here A is only in f1, so A will be present in f1, absent in f2
	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	theme := BuildTheme(true, config.ColorConfig{}) // dark theme
	view := m.View(theme)

	assert.True(t, strings.Contains(view, "\u2713"), "view should contain check mark")
	assert.True(t, strings.Contains(view, "\u2717"), "view should contain cross mark")
}

func TestMatrixViewFooterStaysOnTheSameRow(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})
	render := func(n int) []string {
		keys := make([]string, n)
		for i := range keys {
			keys[i] = fmt.Sprintf("K%02d", i)
		}
		files := []*model.EnvFile{makeTestFile(".env", keys...), makeTestFile(".env.local", keys...)}
		m := NewMatrixModel(files, config.DefaultConfig().Layout)
		m.Width, m.Height = 70, 15
		return strings.Split(stripAnsi(m.View(theme)), "\n")
	}

	few, many := render(3), render(30) // fewer and more keys than body rows
	assert.Equal(t, len(many), len(few), "the stats footer should sit on the same row whatever the key count")
	assert.Contains(t, few[len(few)-1], "3 keys")
	assert.Contains(t, many[len(many)-1], "30 keys")
}

func TestMatrixViewNarrowWidthKeepsEachRowOnOneRow(t *testing.T) {
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	files := []*model.EnvFile{makeTestFile(".env", keys...), makeTestFile(".env.local", keys[1:]...)}
	theme := BuildTheme(true, config.ColorConfig{})
	const height = 18

	// Narrower than 40 columns, the key column, one file column and the
	// counter no longer fit on a row.
	assertNoRowWraps(t, 40, 100, height, []string{"keys"}, func(width int) string {
		m := NewMatrixModel(files, config.DefaultConfig().Layout)
		m.Width, m.Height = width, height
		return m.View(theme)
	})
}

// TestMatrixViewAlignsColumnsAfterANonASCIIFileName checks that a file name
// whose display width differs from its rune count (wide characters, a
// combining accent) fills exactly one column, so the next file's name starts
// where the column layout puts it, right above that file's check mark.
func TestMatrixViewAlignsColumnsAfterANonASCIIFileName(t *testing.T) {
	layout := config.DefaultConfig().Layout
	theme := BuildTheme(true, config.ColorConfig{})
	colOf := func(line, s string) int {
		i := strings.LastIndex(line, s)
		require.GreaterOrEqual(t, i, 0, "%q not in %q", s, line)
		return ansi.StringWidth(line[:i])
	}

	for _, name := range []string{".env.日本", ".env.cafe\u0301"} {
		// KEY_A is only in the second file: its row reads ✗ then ✓.
		files := []*model.EnvFile{makeTestFile(name), makeTestFile(".env", "KEY_A")}
		m := NewMatrixModel(files, layout)
		m.Width, m.Height = 80, 10
		lines := strings.Split(stripAnsi(m.View(theme)), "\n")

		header, row := lines[0], lines[2]
		secondCol := colOf(header, "KEY") + layout.MatrixKeyWidth + layout.MatrixColWidth
		assert.Equal(t, secondCol, colOf(header, ".env"), "second file name after %q", name)
		assert.Equal(t, colOf(header, ".env")+1, colOf(row, "✓"), "check mark under the second file after %q", name)
	}
}

// TestZeroMatrixModelSurvivesALayout sizes a zero MatrixModel, never built
// with a layout: it scrolls without dividing by its zero column width.
func TestZeroMatrixModelSurvivesALayout(t *testing.T) {
	var m MatrixModel

	assert.NotPanics(t, func() { m.SetSize(80, 20) })
}
