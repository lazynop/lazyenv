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

func makeTestFile(name string, keys ...string) *model.EnvFile {
	vars := make([]model.EnvVar, len(keys))
	lines := make([]model.RawLine, len(keys))
	for i, k := range keys {
		vars[i] = model.EnvVar{Key: k, Value: "val_" + k, LineNum: i + 1}
		lines[i] = model.RawLine{Type: model.LineVariable, Content: k + "=val_" + k, VarIdx: i}
	}
	return &model.EnvFile{Name: name, Vars: vars, Lines: lines}
}

func matrixKeys(entries []model.MatrixEntry) []string {
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	return keys
}

func TestNewMatrixModel(t *testing.T) {
	f1 := makeTestFile(".env", "HOST", "PORT", "DB")
	f2 := makeTestFile(".env.local", "HOST", "SECRET")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)

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

	assert.Equal(t, 0, m.rows.Cursor)

	m.MoveDown()
	assert.Equal(t, 1, m.rows.Cursor)

	m.MoveDown()
	assert.Equal(t, 2, m.rows.Cursor)

	m.MoveDown()
	assert.Equal(t, 2, m.rows.Cursor, "should stop at last row")
}

func TestMatrixMoveUp(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	m.MoveUp()
	assert.Equal(t, 0, m.rows.Cursor, "should stop at 0")

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

	m.MoveLeft()
	assert.Equal(t, 0, m.cols.Cursor, "should stop at 0")

	m.MoveRight()
	m.MoveRight()
	assert.Equal(t, 2, m.cols.Cursor)

	m.MoveLeft()
	assert.Equal(t, 1, m.cols.Cursor)

	m.MoveLeft()
	assert.Equal(t, 0, m.cols.Cursor)
}

func TestMatrixToggleSort(t *testing.T) {
	f1 := makeTestFile(".env", "A", "B", "C")
	f2 := makeTestFile(".env.local", "A", "C")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	assert.Equal(t, model.SortAlpha, m.sortMode)
	assert.Equal(t, []string{"A", "B", "C"}, matrixKeys(m.entries))

	m.ToggleSort()
	assert.Equal(t, model.SortCompleteness, m.sortMode)
	assert.Equal(t, []string{"B", "A", "C"}, matrixKeys(m.entries), "B has fewest present, should be first")
	assert.Equal(t, 1, m.rows.Cursor, "the cursor should follow A, its key, to row 1")

	m.ToggleSort()
	assert.Equal(t, model.SortAlpha, m.sortMode)
	assert.Equal(t, []string{"A", "B", "C"}, matrixKeys(m.entries))
	assert.Equal(t, 0, m.rows.Cursor, "the cursor should follow A back to row 0")
}

func newMissingCellMatrix() (MatrixModel, *model.EnvFile) {
	const cRowAboveTheLast, missingCol = 2, 1
	firstFileAlsoLackingC := makeTestFile(".env", "A", "B", "D")
	missing := makeTestFile(".env.local", "A", "B", "D")
	lastFileAtCsRowIndex := makeTestFile(".env.prod", "A", "B", "C", "D")
	files := []*model.EnvFile{firstFileAlsoLackingC, missing, lastFileAtCsRowIndex}
	m := NewMatrixModel(files, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40
	m.SetCursor(cRowAboveTheLast, missingCol)
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
	for _, tc := range []struct {
		name     string
		row, col int
		key      string
	}{
		{"complete row, A in .env.local", 0, 1, "A"},
		{"row with a gap, B in .env while .env.local at B's index lacks it", 1, 0, "B"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMatrixModel([]*model.EnvFile{makeTestFile(".env", "A", "B"), makeTestFile(".env.local", "A")}, config.DefaultConfig().Layout)
			m.Width = 120
			m.Height = 40
			m.SetCursor(tc.row, tc.col)

			cell, ok := m.CursorCell()
			require.True(t, ok)
			require.Equal(t, tc.key, cell.Key)
			require.True(t, cell.Present)

			cmd := m.StartEdit()

			assert.Nil(t, m.edit.File, "no edit should open on a present cell")
			assert.Nil(t, cmd, "StartEdit should return no focus command")
		})
	}
}

func TestMatrixConfirmEdit(t *testing.T) {
	m, missing := newMissingCellMatrix()
	m.StartEdit()
	require.NotNil(t, m.edit.File)

	m.editor.SetValue("new_value")

	assert.Same(t, missing, m.ConfirmEdit(), "ConfirmEdit should return the file it added to")

	assert.Nil(t, m.edit.File, "the edit should close after confirm")

	found := missing.VarByKey("C")
	require.NotNil(t, found, "C should exist in .env.local after confirm")
	assert.Equal(t, "new_value", found.Value)
	assert.Equal(t, makeTestFile(".env", "A", "B", "D"), m.files[0], ".env should stay as it was")
	assert.Equal(t, makeTestFile(".env.prod", "A", "B", "C", "D"), m.files[2], ".env.prod should stay as it was")

	require.Len(t, m.entries, 4)
	require.Equal(t, "C", m.entries[2].Key)
	assert.Equal(t, []bool{false, true, true}, m.entries[2].Present, "C should be present in .env.local after confirm")
}

// TestMatrixConfirmEditResorts confirms an edit in each sort mode: M fills its
// gap in .env while B and Z keep theirs, so the rows keep their alphabetical
// order, or by completeness keep B and Z ahead of the rest, in an order that
// is alphabetical neither way. The cursor stays on M; only the completeness
// case moves the rows, so only it sees the cursor follow M from row 1 to row
// 3, which is neither the last row (Y's), M's alphabetical index (2), nor the
// row of Z, now at row 1, or of B, at row 1 of the rows rebuilt
// alphabetically. The view, two rows tall, scrolls just enough to show M: by
// completeness M goes below it and lands on its last row, not its first.
func TestMatrixConfirmEditResorts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mode          model.SortMode
		before, after []string
		mRowBefore    int
		offsetAfter   int
	}{
		{"alpha", model.SortAlpha, []string{"A", "B", "M", "Y", "Z"}, []string{"A", "B", "M", "Y", "Z"}, 2, 1},
		{"completeness", model.SortCompleteness, []string{"B", "M", "Z", "A", "Y"}, []string{"B", "Z", "A", "M", "Y"}, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := makeTestFile(".env", "A", "Y")
			m := NewMatrixModel([]*model.EnvFile{env, makeTestFile(".env.local", "A", "B", "M", "Y", "Z")}, config.DefaultConfig().Layout)
			m.Width = 120
			m.Height = matrixChromeHeight + 2
			if tc.mode != m.sortMode {
				m.ToggleSort()
			}
			require.Equal(t, tc.before, matrixKeys(m.entries))
			m.SetCursor(tc.mRowBefore, 0)
			m.StartEdit()
			m.editor.SetValue("new_value")

			require.Same(t, env, m.ConfirmEdit())

			assert.Equal(t, tc.after, matrixKeys(m.entries), "the rows should follow the sort mode after confirm")
			cell, _ := m.CursorCell()
			assert.Equal(t, "M", cell.Key, "the cursor should stay on M, the key it filled")
			assert.Equal(t, tc.offsetAfter, m.rows.Offset, "the view should scroll just enough to show M")
		})
	}
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

func TestMatrixDeleteAtCursorKeepsTheCursorOnItsKey(t *testing.T) {
	files := []*model.EnvFile{makeTestFile(".env", "A", "B", "C", "D", "E", "F"), makeTestFile(".env.local", "A", "C", "E")}
	m := NewMatrixModel(files, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = matrixChromeHeight + 2
	m.ToggleSort()
	require.Equal(t, []string{"B", "D", "F", "A", "C", "E"}, matrixKeys(m.entries))
	m.SetCursor(4, 1)
	require.Equal(t, 3, m.rows.Offset, "the view should show C on its last row")

	_, from := m.DeleteAtCursor()
	require.Same(t, files[1], from)

	require.Equal(t, []string{"B", "C", "D", "F", "A", "E"}, matrixKeys(m.entries), "C, now with a gap, should sort among B, D and F")
	assert.Equal(t, 1, m.rows.Cursor, "the cursor should follow C to row 1, where neither the key now at C's old row nor the one there alphabetically is C")
	assert.Equal(t, 1, m.rows.Offset, "the view should scroll up just as far as C")
}

func TestMatrixDeleteAtCursorKeepsTheCursorIndexWhenTheRowGoes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		row    int
		after  []string
		cursor int
	}{
		{"a middle row", 1, []string{"A", "C", "D"}, 1},
		{"the last row", 3, []string{"A", "B", "C"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := []*model.EnvFile{makeTestFile(".env", "A", "B", "C", "D"), makeTestFile(".env.local", "A")}
			m := NewMatrixModel(files, config.DefaultConfig().Layout)
			m.Width = 120
			m.Height = 40
			m.SetCursor(tc.row, 0)

			_, from := m.DeleteAtCursor()
			require.Same(t, files[0], from)

			require.Equal(t, tc.after, matrixKeys(m.entries))
			assert.Equal(t, tc.cursor, m.rows.Cursor, "the cursor should keep its index, within the rows")
		})
	}
}

func TestMatrixViewContainsCheckAndCross(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.local")

	m := NewMatrixModel([]*model.EnvFile{f1, f2}, config.DefaultConfig().Layout)
	m.Width = 120
	m.Height = 40

	theme := BuildTheme(true, config.ColorConfig{})
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

	fewerThanBodyRows, moreThanBodyRows := render(3), render(30)
	assert.Equal(t, len(moreThanBodyRows), len(fewerThanBodyRows), "the stats footer should sit on the same row whatever the key count")
	assert.Contains(t, fewerThanBodyRows[len(fewerThanBodyRows)-1], "3 keys")
	assert.Contains(t, moreThanBodyRows[len(moreThanBodyRows)-1], "30 keys")
}

func TestMatrixViewNarrowWidthKeepsEachRowOnOneRow(t *testing.T) {
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	files := []*model.EnvFile{makeTestFile(".env", keys...), makeTestFile(".env.local", keys[1:]...)}
	theme := BuildTheme(true, config.ColorConfig{})
	const height = 18

	const minWidthForKeyFileAndCounter = 40
	assertNoRowWraps(t, minWidthForKeyFileAndCounter, 100, height, []string{"keys"}, func(width int) string {
		m := NewMatrixModel(files, config.DefaultConfig().Layout)
		m.Width, m.Height = width, height
		return m.View(theme)
	})
}

func TestMatrixViewAlignsColumnsAfterANonASCIIFileName(t *testing.T) {
	layout := config.DefaultConfig().Layout
	theme := BuildTheme(true, config.ColorConfig{})
	colOf := func(line, s string) int {
		i := strings.LastIndex(line, s)
		require.GreaterOrEqual(t, i, 0, "%q not in %q", s, line)
		return ansi.StringWidth(line[:i])
	}

	for _, name := range []string{".env.日本", ".env.cafe\u0301"} {
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

func TestZeroMatrixModelSizesWithoutDividingByItsZeroColWidth(t *testing.T) {
	var m MatrixModel

	assert.NotPanics(t, func() { m.SetSize(80, 20) })
}
