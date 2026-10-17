package tui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// --- FileListModel.SetCursor ---

func TestFileListSetCursor(t *testing.T) {
	files := make([]*model.EnvFile, 5)
	for i := range files {
		files[i] = &model.EnvFile{Name: fmt.Sprintf("file%d", i)}
	}

	var fl FileListModel
	fl.Files = files
	fl.Height = 20

	fl.SetCursor(3)
	assert.Equal(t, 3, fl.Cursor, "Cursor should be set to 3")
	assert.Equal(t, 3, fl.Selected, "Selected should mirror Cursor after SetCursor")

	fl.SetCursor(-1)
	assert.Equal(t, 0, fl.Cursor, "SetCursor(-1) should clamp to 0")
	assert.Equal(t, 0, fl.Selected)

	fl.SetCursor(99)
	assert.Equal(t, 4, fl.Cursor, "SetCursor(99) should clamp to len-1 = 4")
	assert.Equal(t, 4, fl.Selected)
}

func TestFileListSetCursorEmpty(t *testing.T) {
	var fl FileListModel
	// Files is nil/empty — should not panic
	assert.NotPanics(t, func() {
		fl.SetCursor(0)
		fl.SetCursor(-1)
		fl.SetCursor(5)
	})
	assert.Equal(t, 0, fl.Cursor)
	assert.Equal(t, 0, fl.Selected)
}

// --- VarListModel.SetCursor ---

func TestVarListSetCursor(t *testing.T) {
	m := NewVarListModel(config.DefaultConfig().Layout)
	f := &model.EnvFile{Vars: []model.EnvVar{
		{Key: "A"}, {Key: "B"}, {Key: "C"}, {Key: "D"}, {Key: "E"},
	}}
	m.SetFile(f)
	m.Height = 20

	m.SetCursor(3)
	assert.Equal(t, 3, m.Cursor, "Cursor should be set to 3")

	m.SetCursor(-1)
	assert.Equal(t, 0, m.Cursor, "SetCursor(-1) should clamp to 0")

	m.SetCursor(99)
	assert.Equal(t, 4, m.Cursor, "SetCursor(99) should clamp to len-1 = 4")
}

func TestVarListSetCursorEmpty(t *testing.T) {
	m := NewVarListModel(config.DefaultConfig().Layout)
	// No file set — displayItems is empty; should not panic
	assert.NotPanics(t, func() {
		m.SetCursor(0)
		m.SetCursor(-1)
		m.SetCursor(5)
	})
	assert.Equal(t, 0, m.Cursor)
}

// --- Click on group header toggles collapse (E009) ---

func TestMouseClickOnHeader_TogglesCollapse(t *testing.T) {
	f := makeTestFile(".env",
		"DB_HOST", "DB_PORT", "DB_USER",
		"REDIS_URL", "REDIS_PORT",
		"PORT",
	)
	app := newTestApp([]*model.EnvFile{f})
	// Trigger layout so fileWidth is computed for the click X dispatch.
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)
	app.varList.Grouping = true
	app.varList.Refresh()

	// Layout: row 0 = DB header, then DB vars, etc. Y=2 maps to row 0
	// (Y=0 border, Y=1 title, Y=2 first item) and X must land in the var
	// panel (X >= a.fileWidth).
	clickX := app.fileWidth + 5
	updated, _ = app.Update(tea.MouseClickMsg{X: clickX, Y: 2, Button: tea.MouseLeft})
	app = updated.(App)

	require.Equal(t, FocusVars, app.focus)
	assert.True(t, app.varList.collapsed["DB"],
		"clicking the DB header must toggle collapse on")

	// Click again to expand.
	updated, _ = app.Update(tea.MouseClickMsg{X: clickX, Y: 2, Button: tea.MouseLeft})
	app = updated.(App)
	assert.False(t, app.varList.collapsed["DB"],
		"second click must toggle collapse off")
}

func TestMouseClickOnVar_DoesNotToggleCollapse(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "DB_PORT", "DB_USER")
	app := newTestApp([]*model.EnvFile{f})
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)
	app.varList.Grouping = true
	app.varList.Refresh()

	// Y=3 → row 1 = first DB var (DB_HOST). Click on a var must just
	// move the cursor without toggling.
	clickX := app.fileWidth + 5
	updated, _ = app.Update(tea.MouseClickMsg{X: clickX, Y: 3, Button: tea.MouseLeft})
	app = updated.(App)

	assert.False(t, app.varList.collapsed["DB"],
		"clicking a var row must not collapse the group")
	assert.Equal(t, "DB_HOST", app.varList.SelectedVar().Key)
}

// --- DiffViewModel.SetCursor ---

func TestDiffViewSetCursor(t *testing.T) {
	m := DiffViewModel{
		Entries: make([]model.DiffEntry, 10),
		Height:  20,
	}

	m.SetCursor(5)
	assert.Equal(t, 5, m.Cursor, "Cursor should be set to 5")

	m.SetCursor(-1)
	assert.Equal(t, 0, m.Cursor, "SetCursor(-1) should clamp to 0")

	m.SetCursor(99)
	assert.Equal(t, 9, m.Cursor, "SetCursor(99) should clamp to len-1 = 9")
}

func TestDiffViewSetCursorEmpty(t *testing.T) {
	m := DiffViewModel{Height: 20}
	// Entries is nil/empty — should not panic
	assert.NotPanics(t, func() {
		m.SetCursor(0)
		m.SetCursor(5)
	})
	assert.Equal(t, 0, m.Cursor)
}

// --- MatrixModel.SetCursor ---

func TestMatrixSetCursor(t *testing.T) {
	m := MatrixModel{
		entries:   make([]model.MatrixEntry, 5),
		fileNames: []string{"a", "b", "c"},
		Height:    20,
		Width:     80,
		layout:    config.DefaultConfig().Layout,
	}
	// Populate Present slices to match fileNames length
	for i := range m.entries {
		m.entries[i].Present = make([]bool, 3)
	}

	m.SetCursor(3, 1)
	assert.Equal(t, 3, m.cursorRow, "cursorRow should be set to 3")
	assert.Equal(t, 1, m.cursorCol, "cursorCol should be set to 1")

	// Bounds clamping
	m.SetCursor(-1, -1)
	assert.Equal(t, 0, m.cursorRow, "cursorRow should clamp to 0")
	assert.Equal(t, 0, m.cursorCol, "cursorCol should clamp to 0")

	m.SetCursor(99, 99)
	assert.Equal(t, 4, m.cursorRow, "cursorRow should clamp to len(entries)-1 = 4")
	assert.Equal(t, 2, m.cursorCol, "cursorCol should clamp to len(fileNames)-1 = 2")
}

func TestMatrixSetCursorEmpty(t *testing.T) {
	m := MatrixModel{
		Height: 20,
		Width:  80,
		layout: config.DefaultConfig().Layout,
	}
	// No entries, no fileNames — should not panic
	assert.NotPanics(t, func() {
		m.SetCursor(0, 0)
		m.SetCursor(5, 5)
	})
	assert.Equal(t, 0, m.cursorRow)
	assert.Equal(t, 0, m.cursorCol)
}

// --- Mouse wheel: Normal mode scrolls the file and var lists ---

func TestMouseWheelNormal_FileList(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)

	// Wheel down over the file panel (X < fileWidth)
	updated, _ = app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	require.Greater(t, app.fileList.Cursor, 0, "wheel down should advance the file cursor")
	assert.Equal(t, app.fileList.SelectedFile(), app.varList.File,
		"var panel should follow the scrolled file selection")

	// Wheel up returns to the top
	updated, _ = app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelUp})
	app = updated.(App)
	assert.Equal(t, 0, app.fileList.Cursor)
}

func TestMouseWheelNormal_VarList(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E")
	app := newTestApp([]*model.EnvFile{f})
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)

	updated, _ = app.Update(tea.MouseWheelMsg{X: app.fileWidth + 5, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.varList.Cursor, 0,
		"wheel down over the var panel should advance the var cursor")
}

// rowDrawing returns the screen row of view that holds text, failing when none
// does.
func rowDrawing(t *testing.T, view, text string) int {
	t.Helper()
	row := slices.IndexFunc(strings.Split(stripAnsi(view), "\n"), func(line string) bool {
		return strings.Contains(line, text)
	})
	require.GreaterOrEqual(t, row, 0, "%q should be drawn", text)
	return row
}

// TestMouseClickVarList_SelectsTheVarDrawnAtTheRow turns peek on over a
// modified var and clicks rows around its peek line: a click selects the var
// drawn on that row, and one on the peek line keeps the var it belongs to.
func TestMouseClickVarList_SelectsTheVarDrawnAtTheRow(t *testing.T) {
	for _, tt := range []struct{ click, selects string }{
		{"BRAVO", "BRAVO"},
		{"DELTA", "DELTA"},
		{"↳ was:", "ALPHA"},
	} {
		t.Run(tt.click, func(t *testing.T) {
			f := makeTestFile(".env", "ALPHA", "BRAVO", "CHARLIE", "DELTA")
			f.UpdateVar(0, "edited", false)
			updated, _ := newTestApp([]*model.EnvFile{f}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			app := press(updated.(App), "right", "p")
			row := rowDrawing(t, app.View().Content, tt.click)

			updated, _ = app.Update(tea.MouseClickMsg{X: 40, Y: row, Button: tea.MouseLeft})

			app = updated.(App)
			require.NotNil(t, app.varList.SelectedVar())
			assert.Equal(t, tt.selects, app.varList.SelectedVar().Key)
		})
	}
}

// TestMouseClickVarList_OutsideTheDrawnRowsKeepsTheCursor scrolls a var list
// four rows tall to its middle, then clicks its title row and its bottom
// border: they draw no var, though vars sit just off screen on either side,
// so the cursor stays put.
func TestMouseClickVarList_OutsideTheDrawnRowsKeepsTheCursor(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J")
	updated, _ := newTestApp([]*model.EnvFile{f}).Update(tea.WindowSizeMsg{Width: 80, Height: bottomChromeHeight + panelChromeHeight + 4})
	app := press(updated.(App), "right", "j", "j", "j", "j", "j") // F, with C to F drawn
	require.Equal(t, 2, app.varList.Offset)

	for _, y := range []int{panelContentOffsetY - 1, panelContentOffsetY + 4} {
		updated, _ = app.Update(tea.MouseClickMsg{X: 40, Y: y, Button: tea.MouseLeft})
		clicked := updated.(App)
		assert.Equal(t, "F", clicked.varList.SelectedVar().Key, "a click at Y=%d draws no var", y)
	}
}

// TestMouseClickVarList_FromTheFileListSelectsTheVarDrawnAtTheRow clicks a var
// while the file list has the focus and the var cursor sits on the bottom row,
// on a modified var: focusing the var list draws a peek line under it and
// scrolls, but the click selects the var that was drawn where it landed.
func TestMouseClickVarList_FromTheFileListSelectsTheVarDrawnAtTheRow(t *testing.T) {
	app := peekedBottomRowApp()
	row := rowDrawing(t, app.View().Content, "val_D")

	updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: row, Button: tea.MouseLeft})

	app = updated.(App)
	require.NotNil(t, app.varList.SelectedVar())
	assert.Equal(t, "D", app.varList.SelectedVar().Key)
}

// --- Compare mode: click and wheel drive the diff view ---

func TestMouseClickCompareMode_SelectsTheClickedEntry(t *testing.T) {
	keys := []string{"ALPHA", "BRAVO", "CHARLIE", "DELTA"}
	app := enterCompareMode(t, makeTestFile(".env", keys...), makeTestFile(".env.prod", keys...))

	row := slices.IndexFunc(strings.Split(stripAnsi(app.View().Content), "\n"), func(line string) bool {
		return strings.Contains(line, "CHARLIE")
	})
	require.GreaterOrEqual(t, row, 0, "CHARLIE should be drawn in the diff view")

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: row, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, "CHARLIE", app.diffView.Entries[app.diffView.Cursor].Key)
}

func TestMouseWheelCompareMode_ScrollsDiff(t *testing.T) {
	app := newTestApp(nil)
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)
	app.mode = ModeComparing
	app.diffView = DiffViewModel{Entries: make([]model.DiffEntry, 10), Height: 20}

	updated, _ = app.Update(tea.MouseWheelMsg{X: 10, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.diffView.Cursor, 0, "wheel down should scroll the diff view")
}

// --- Compare-select mode: click moves the cursor without changing the active file ---

func TestMouseClickCompareSelectMode_MovesCursorOnly(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)
	app.mode = ModeCompareSelect

	// index = Y - 2 + Offset = 4 - 2 + 0 = 2
	updated, _ = app.Update(tea.MouseClickMsg{X: 5, Y: 4, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 2, app.fileList.Cursor, "compare-select click moves the cursor")
	assert.Equal(t, 0, app.fileList.Selected,
		"compare-select must not change the active file (Selected)")
}

// TestMouseClickCompareSelectMode_IgnoresTheVarPanel clicks the var panel
// while picking the second file to compare: only the file list picks it, so
// the cursor stays on the first file.
func TestMouseClickCompareSelectMode_IgnoresTheVarPanel(t *testing.T) {
	app := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B"), makeTestFile(".env.local", "C")})
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = press(updated.(App), "c")
	require.Equal(t, ModeCompareSelect, app.mode)

	updated, _ = app.Update(tea.MouseClickMsg{X: app.fileWidth + 10, Y: 4, Button: tea.MouseLeft})

	assert.Equal(t, 0, updated.(App).fileList.Cursor, "a click on the var panel should not move the file cursor")
}

// --- Matrix mode: click sets a cell, wheel scrolls rows ---

func newMatrixTestApp(t *testing.T) App {
	t.Helper()
	app := newTestApp(nil)
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(App)
	app.mode = ModeMatrix
	mv := MatrixModel{
		entries:   make([]model.MatrixEntry, 5),
		fileNames: []string{"a", "b", "c"},
		Height:    20,
		Width:     120,
		layout:    config.DefaultConfig().Layout,
	}
	for i := range mv.entries {
		mv.entries[i].Present = make([]bool, 3)
	}
	app.matrixView = mv
	return app
}

func TestMouseClickMatrixMode_SetsCell(t *testing.T) {
	app := newMatrixTestApp(t)
	kw := app.matrixView.layout.MatrixKeyWidth
	cw := app.matrixView.layout.MatrixColWidth

	// Data cell: row 2 (Y=4), column 1 (X one column band past the key column)
	updated, _ := app.Update(tea.MouseClickMsg{X: matrixPaddingWidth/2 + kw + cw, Y: 4, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 2, app.matrixView.cursorRow)
	assert.Equal(t, 1, app.matrixView.cursorCol)

	// Key-column click (X < keyWidth) keeps the column, only moves the row
	updated, _ = app.Update(tea.MouseClickMsg{X: kw - 1, Y: 3, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.cursorRow)
	assert.Equal(t, 1, app.matrixView.cursorCol, "key-column click keeps the current column")
}

// TestMouseClickMatrixMode_SelectsTheColumnDrawnAtX clicks the first cell of a
// file's column, as drawn under its name in the header, and the cell just
// before it, the last of the previous column.
func TestMouseClickMatrixMode_SelectsTheColumnDrawnAtX(t *testing.T) {
	updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env.a", "K"), makeTestFile(".env.b", "K")}).Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	app := press(updated.(App), "m")
	header := strings.Split(stripAnsi(app.View().Content), "\n")[0]
	x := strings.Index(header, ".env.b")
	require.Positive(t, x, "the header should name .env.b")

	for _, tt := range []struct{ x, col int }{{x, 1}, {x - 1, 0}} {
		updated, _ = app.Update(tea.MouseClickMsg{X: tt.x, Y: matrixContentOffsetY, Button: tea.MouseLeft})
		assert.Equal(t, tt.col, updated.(App).matrixView.cursorCol, "click at X=%d", tt.x)
	}
}

func TestMouseWheelMatrixMode_Scrolls(t *testing.T) {
	app := newMatrixTestApp(t)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.matrixView.cursorRow, 0, "wheel down should scroll the matrix rows")
}
