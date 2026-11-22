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

func TestFileListMoveCursorTo(t *testing.T) {
	files := make([]*model.EnvFile, 5)
	for i := range files {
		files[i] = &model.EnvFile{Name: fmt.Sprintf("file%d", i)}
	}

	var fl FileListModel
	fl.Files = files
	fl.Height = 20
	fl.Selected = 2

	fl.MoveCursorTo(3)
	assert.Equal(t, 3, fl.Cursor, "Cursor should be set to 3")
	assert.Equal(t, 2, fl.Selected, "MoveCursorTo should keep the selected file")

	fl.MoveCursorTo(-1)
	assert.Equal(t, 0, fl.Cursor, "MoveCursorTo(-1) should clamp to 0")

	fl.MoveCursorTo(99)
	assert.Equal(t, 4, fl.Cursor, "MoveCursorTo(99) should clamp to len-1 = 4")
	assert.Equal(t, 2, fl.Selected)
}

func TestFileListMoveCursorToEmpty(t *testing.T) {
	var fl FileListModel
	assert.NotPanics(t, func() {
		fl.MoveCursorTo(0)
		fl.MoveCursorTo(-1)
		fl.MoveCursorTo(5)
	})
	assert.Equal(t, 0, fl.Cursor)
	assert.Equal(t, 0, fl.Selected)
}

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
	assert.NotPanics(t, func() {
		m.SetCursor(0)
		m.SetCursor(-1)
		m.SetCursor(5)
	})
	assert.Equal(t, 0, m.Cursor)
}

func TestMouseClickOnHeader_TogglesCollapse(t *testing.T) {
	f := makeTestFile(".env",
		"DB_HOST", "DB_PORT", "DB_USER",
		"REDIS_URL", "REDIS_PORT",
		"PORT",
	)
	app := newTestApp([]*model.EnvFile{f})
	app = resize(app, 120, 40)
	app.varList.Grouping = true
	app.varList.Refresh()

	clickX := app.fileWidth + 5
	headerY := panelContentOffsetY
	updated, _ := app.Update(tea.MouseClickMsg{X: clickX, Y: headerY, Button: tea.MouseLeft})
	app = updated.(App)

	require.Equal(t, FocusVars, app.focus)
	assert.True(t, app.varList.collapsed["DB"],
		"clicking the DB header must toggle collapse on")

	updated, _ = app.Update(tea.MouseClickMsg{X: clickX, Y: headerY, Button: tea.MouseLeft})
	app = updated.(App)
	assert.False(t, app.varList.collapsed["DB"],
		"second click must toggle collapse off")
}

func TestMouseClickOnVar_DoesNotToggleCollapse(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "DB_PORT", "DB_USER")
	app := newTestApp([]*model.EnvFile{f})
	app = resize(app, 120, 40)
	app.varList.Grouping = true
	app.varList.Refresh()

	clickX := app.fileWidth + 5
	firstVarY := panelContentOffsetY + 1
	updated, _ := app.Update(tea.MouseClickMsg{X: clickX, Y: firstVarY, Button: tea.MouseLeft})
	app = updated.(App)

	assert.False(t, app.varList.collapsed["DB"],
		"clicking a var row must not collapse the group")
	assert.Equal(t, "DB_HOST", app.varList.SelectedVar().Key)
}

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
	assert.NotPanics(t, func() {
		m.SetCursor(0)
		m.SetCursor(5)
	})
	assert.Equal(t, 0, m.Cursor)
}

func TestMatrixSetCursor(t *testing.T) {
	m := NewMatrixModel(fiveKeyFiles(), config.DefaultConfig().Layout)
	m.Width = 80
	m.Height = 20

	m.SetCursor(3, 1)
	assert.Equal(t, 3, m.rows.Cursor, "rows.Cursor should be set to 3")
	assert.Equal(t, 1, m.cols.Cursor, "cols.Cursor should be set to 1")

	m.SetCursor(-1, -1)
	assert.Equal(t, 0, m.rows.Cursor, "rows.Cursor should clamp to 0")
	assert.Equal(t, 0, m.cols.Cursor, "cols.Cursor should clamp to 0")

	m.SetCursor(99, 99)
	assert.Equal(t, 4, m.rows.Cursor, "rows.Cursor should clamp to len(entries)-1 = 4")
	assert.Equal(t, 2, m.cols.Cursor, "cols.Cursor should clamp to len(files)-1 = 2")
}

func TestMatrixSetCursorEmpty(t *testing.T) {
	m := MatrixModel{
		Height: 20,
		Width:  80,
		layout: config.DefaultConfig().Layout,
	}
	assert.NotPanics(t, func() {
		m.SetCursor(0, 0)
		m.SetCursor(5, 5)
	})
	assert.Equal(t, 0, m.rows.Cursor)
	assert.Equal(t, 0, m.cols.Cursor)
}

func TestMouseWheelNormal_FileList(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	app = resize(app, 120, 40)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	require.Greater(t, app.fileList.Cursor, 0, "wheel down should advance the file cursor")
	assert.Equal(t, app.fileList.SelectedFile(), app.varList.File,
		"var panel should follow the scrolled file selection")

	updated, _ = app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelUp})
	app = updated.(App)
	assert.Equal(t, 0, app.fileList.Cursor)
}

func TestMouseWheelNormal_VarList(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E")
	app := newTestApp([]*model.EnvFile{f})
	app = resize(app, 120, 40)

	updated, _ := app.Update(tea.MouseWheelMsg{X: app.fileWidth + 5, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.varList.Cursor, 0,
		"wheel down over the var panel should advance the var cursor")
}

func rowDrawing(t *testing.T, view, text string) int {
	t.Helper()
	row := slices.IndexFunc(strings.Split(stripAnsi(view), "\n"), func(line string) bool {
		return strings.Contains(line, text)
	})
	require.GreaterOrEqual(t, row, 0, "%q should be drawn", text)
	return row
}

func TestMouseClickVarList_SelectsTheVarDrawnAtTheRow(t *testing.T) {
	for _, tt := range []struct{ name, click, selects string }{
		{"a var below the peek line", "BRAVO", "BRAVO"},
		{"a var further down", "DELTA", "DELTA"},
		{"the peek line selects its var", "↳ was:", "ALPHA"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := makeTestFile(".env", "ALPHA", "BRAVO", "CHARLIE", "DELTA")
			f.UpdateVar(0, "edited")
			app := press(resize(newTestApp([]*model.EnvFile{f}), 80, 24), "right", "p")
			row := rowDrawing(t, app.View().Content, tt.click)

			updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: row, Button: tea.MouseLeft})

			app = updated.(App)
			require.NotNil(t, app.varList.SelectedVar())
			assert.Equal(t, tt.selects, app.varList.SelectedVar().Key)
		})
	}
}

var fileNameWiderThanAnyPanel = ".env." + strings.Repeat("verylongname", 6)

func TestMouseClickVarList_UnderAFileNameCutToKeepTheTitleOnOneRow(t *testing.T) {
	f := makeTestFile(fileNameWiderThanAnyPanel, "ALPHA", "BRAVO", "CHARLIE")
	app := onVars(resize(newTestApp([]*model.EnvFile{f}), 80, 24))
	view := app.View().Content
	require.Contains(t, strings.Split(stripAnsi(view), "\n")[panelContentOffsetY-1], "(3 vars)", "the title row ends in the count")

	updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: rowDrawing(t, view, "BRAVO"), Button: tea.MouseLeft})

	clicked := updated.(App)
	assert.Equal(t, "BRAVO", clicked.varList.SelectedVar().Key)
}

func TestMouseClickVarList_UnderAGroupHeaderCutToKeepItOnOneRow(t *testing.T) {
	prefix := strings.Repeat("LONGPREFIX", 8)
	f := makeTestFile(".env", prefix+"_HOST", prefix+"_PORT")
	valueTellingTheCutKeysApart := "host"
	f.Vars[0].Value = valueTellingTheCutKeysApart
	onTheSecondVar := press(onVars(resize(newTestApp([]*model.EnvFile{f}), 80, 24)), "g", "j")
	rows := strings.Split(stripAnsi(onTheSecondVar.View().Content), "\n")
	require.Contains(t, rows[panelContentOffsetY], "(2)", "the header row ends in the count")
	firstVar := panelContentOffsetY + 1
	require.Contains(t, rows[firstVar], valueTellingTheCutKeysApart, "the first var follows the header")

	updated, _ := onTheSecondVar.Update(tea.MouseClickMsg{X: 40, Y: firstVar, Button: tea.MouseLeft})

	clicked := updated.(App)
	assert.Equal(t, prefix+"_HOST", clicked.varList.SelectedVar().Key)
}

func TestMouseClickCompareMode_UnderAFileNameCutToKeepEachKeyOnOneRow(t *testing.T) {
	keys := []string{"ALPHA", "BRAVO", "CHARLIE"}
	long := makeTestFile(fileNameWiderThanAnyPanel, keys...)
	long.Modified = true
	app := enterCompareMode(t, makeTestFile(".env", keys...), long)
	view := app.View().Content
	require.Contains(t, strings.Split(stripAnsi(view), "\n")[panelContentOffsetY-1], ".."+modifiedFileGlyph+" │",
		"the cut name keeps its mark, and the title its padding")
	row := rowDrawing(t, view, "BRAVO")
	require.Equal(t, 2, strings.Count(strings.Split(stripAnsi(view), "\n")[row], "  BRAVO "), "both panels draw BRAVO on one row")

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: row, Button: tea.MouseLeft})

	clicked := updated.(App).diffView
	assert.Equal(t, "BRAVO", clicked.Entries[clicked.Cursor].Key)
}

func TestMouseClickVarList_OnADeletedVarKeepsTheCursorAsNoCursorReachesIt(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "GONE")
	f.DeleteVar(2)
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, 24), "right", "j")
	row := rowDrawing(t, app.View().Content, "GONE")

	updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: row, Button: tea.MouseLeft})

	clicked := updated.(App)
	assert.Equal(t, "B", clicked.varList.SelectedVar().Key)
}

func TestMouseClickVarList_OutsideTheDrawnRowsKeepsTheCursor(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J")
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "j", "j", "j", "j", "j")
	require.Equal(t, 2, app.varList.Offset)

	titleRow, bottomBorderRow := panelContentOffsetY-1, panelContentOffsetY+4
	for _, y := range []int{titleRow, bottomBorderRow} {
		updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: y, Button: tea.MouseLeft})
		clicked := updated.(App)
		assert.Equal(t, "F", clicked.varList.SelectedVar().Key, "a click at Y=%d draws no var", y)
	}
}

func TestMouseClickFileList_OutsideTheDrawnRowsKeepsTheCursor(t *testing.T) {
	files := make([]*model.EnvFile, 10)
	for i := range files {
		files[i] = makeTestFile(fmt.Sprintf(".env.%d", i), "K")
	}
	app := press(resize(newTestApp(files), 80, fourRowPanelsHeight), "j", "j", "j", "j", "j")
	require.Equal(t, 2, app.fileList.Offset)

	titleRow, bottomBorderRow := panelContentOffsetY-1, panelContentOffsetY+4
	for _, mode := range []AppMode{ModeNormal, ModeCompareSelect} {
		app.mode = mode
		for _, y := range []int{titleRow, bottomBorderRow} {
			updated, _ := app.Update(tea.MouseClickMsg{X: 5, Y: y, Button: tea.MouseLeft})
			clicked := updated.(App).fileList
			assert.Equal(t, 5, clicked.Cursor, "in mode %v a click at Y=%d draws no file", mode, y)
			assert.Equal(t, 2, clicked.Offset, "in mode %v a click at Y=%d scrolls nothing", mode, y)
		}
	}
}

func TestMouseClickFileList_BelowTheLastFileKeepsTheCursor(t *testing.T) {
	files := []*model.EnvFile{makeTestFile(".env", "K"), makeTestFile(".env.a", "K"), makeTestFile(".env.b", "K")}
	app := resize(newTestApp(files), 80, 24)

	for _, mode := range []AppMode{ModeNormal, ModeCompareSelect} {
		app.mode = mode
		updated, _ := app.Update(tea.MouseClickMsg{X: 5, Y: panelContentOffsetY + len(files), Button: tea.MouseLeft})
		clicked := updated.(App).fileList
		assert.Equal(t, 0, clicked.Cursor, "in mode %v a blank row draws no file", mode)
	}
}

func TestMouseClickVarList_FromTheFileListSelectsTheVarDrawnBeforeTheFocusScrolls(t *testing.T) {
	app := peekedBottomRowApp()
	row := rowDrawing(t, app.View().Content, "val_D")

	updated, _ := app.Update(tea.MouseClickMsg{X: 40, Y: row, Button: tea.MouseLeft})

	app = updated.(App)
	require.NotNil(t, app.varList.SelectedVar())
	assert.Equal(t, "D", app.varList.SelectedVar().Key)
}

func TestMouseClickCompareMode_SelectsTheClickedEntry(t *testing.T) {
	keys := []string{"ALPHA", "BRAVO", "CHARLIE", "DELTA"}
	app := enterCompareMode(t, makeTestFile(".env", keys...), makeTestFile(".env.prod", keys...))

	row := rowDrawing(t, app.View().Content, "CHARLIE")

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: row, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, "CHARLIE", app.diffView.Entries[app.diffView.Cursor].Key)
}

func TestMouseClickCompareMode_OutsideTheDrawnRowsKeepsTheCursor(t *testing.T) {
	keys := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	app := enterCompareMode(t, makeTestFile(".env", keys...), makeTestFile(".env.prod", keys...))
	app = press(resize(app, 80, fourRowPanelsHeight), "j", "j", "j", "j", "j")
	require.Equal(t, 2, app.diffView.Offset)

	titleRow, bottomBorderRow := panelContentOffsetY-1, panelContentOffsetY+4
	for _, y := range []int{titleRow, bottomBorderRow} {
		updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: y, Button: tea.MouseLeft})
		clicked := updated.(App).diffView
		assert.Equal(t, 5, clicked.Cursor, "a click at Y=%d draws no entry", y)
		assert.Equal(t, 2, clicked.Offset, "a click at Y=%d scrolls nothing", y)
	}
}

func TestMouseWheelCompareMode_ScrollsDiff(t *testing.T) {
	app := newTestApp(nil)
	app = resize(app, 120, 40)
	app.mode = ModeComparing
	app.diffView = DiffViewModel{Entries: make([]model.DiffEntry, 10), Height: 20}

	updated, _ := app.Update(tea.MouseWheelMsg{X: 10, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.diffView.Cursor, 0, "wheel down should scroll the diff view")
}

func TestMouseClickCompareSelectMode_MovesCursorOnly(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	app = resize(app, 120, 40)
	app.mode = ModeCompareSelect

	thirdFileRow := panelContentOffsetY + 2
	updated, _ := app.Update(tea.MouseClickMsg{X: 5, Y: thirdFileRow, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 2, app.fileList.Cursor, "compare-select click moves the cursor")
	assert.Equal(t, 0, app.fileList.Selected,
		"compare-select must not change the active file (Selected)")
}

func TestMouseWheelCompareSelectMode_MovesCursorOnly(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := press(resize(newTestApp([]*model.EnvFile{f1, f2, f3}), 120, 40), "c")
	require.Equal(t, ModeCompareSelect, app.mode)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 5, Y: 4, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor, "the wheel moves the cursor")
	assert.Equal(t, 0, app.fileList.Selected, "the wheel must not change the active file")
	assert.Same(t, f1, app.varList.File, "the var list keeps the active file")
}

func TestMouseWheelCompareSelectMode_IgnoresTheVarPanel(t *testing.T) {
	f1, f2 := makeTestFile(".env", "A"), makeTestFile(".env.prod", "B")
	app := press(resize(newTestApp([]*model.EnvFile{f1, f2}), 120, 40), "c")

	updated, _ := app.Update(tea.MouseWheelMsg{X: app.fileWidth + 5, Y: 4, Button: tea.MouseWheelDown})

	assert.Equal(t, 0, updated.(App).fileList.Cursor, "the wheel over the var panel moves nothing")
}

func TestMouseClickEditingCompareMode_KeepsTheEntryTheEditBarEdits(t *testing.T) {
	keys := []string{"ALPHA", "BRAVO", "CHARLIE"}
	a, b := makeTestFile(".env", keys...), makeTestFile(".env.prod", keys...)
	b.Vars[0].Value = "different"
	app := press(enterCompareMode(t, a, b), "e")
	require.Equal(t, ModeEditingCompare, app.mode)
	row := rowDrawing(t, app.View().Content, "CHARLIE")

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: row, Button: tea.MouseLeft})

	assert.Equal(t, 0, updated.(App).diffView.Cursor, "the click should not move off the edited entry")
}

func TestMouseWheelConfirmMatrixDelete_KeepsTheKeyToDelete(t *testing.T) {
	a := makeTestFile(".env", "ALPHA", "BRAVO", "CHARLIE")
	app := press(resize(newTestApp([]*model.EnvFile{a, makeTestFile(".env.prod", "ALPHA")}), 120, 24), "m", "d")
	require.Equal(t, ModeConfirmMatrixDelete, app.mode)
	require.Contains(t, app.statusBar.Message, "ALPHA")

	updated, _ := app.Update(tea.MouseWheelMsg{X: 10, Y: 5, Button: tea.MouseWheelDown})
	press(updated.(App), "y")

	assert.Nil(t, a.VarByKey("ALPHA"), "the key the prompt named should be deleted")
	assert.NotNil(t, a.VarByKey("BRAVO"), "the wheel should not retarget the delete")
}

func TestMouseClickCompareSelectMode_IgnoresTheVarPanel(t *testing.T) {
	app := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B"), makeTestFile(".env.local", "C")})
	app = press(resize(app, 120, 40), "c")
	require.Equal(t, ModeCompareSelect, app.mode)

	updated, _ := app.Update(tea.MouseClickMsg{X: app.fileWidth + 10, Y: 4, Button: tea.MouseLeft})

	assert.Equal(t, 0, updated.(App).fileList.Cursor, "a click on the var panel should not move the file cursor")
}

func fiveKeyFiles() []*model.EnvFile {
	return []*model.EnvFile{makeTestFile("a", "K1", "K2", "K3", "K4", "K5"), makeTestFile("b"), makeTestFile("c")}
}

func newMatrixTestApp(t *testing.T) App {
	t.Helper()
	app := press(resize(newTestApp(fiveKeyFiles()), 120, 40), "m")
	require.Equal(t, ModeMatrix, app.mode)
	return app
}

func TestMouseClickMatrixMode_SetsCell(t *testing.T) {
	app := newMatrixTestApp(t)
	kw := app.matrixView.layout.MatrixKeyWidth
	cw := app.matrixView.layout.MatrixColWidth

	thirdRowY := matrixContentOffsetY + 2
	updated, _ := app.Update(tea.MouseClickMsg{X: matrixSidePaddingWidth + kw + cw, Y: thirdRowY, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 2, app.matrixView.rows.Cursor)
	assert.Equal(t, 1, app.matrixView.cols.Cursor)

	updated, _ = app.Update(tea.MouseClickMsg{X: kw - 1, Y: 3, Button: tea.MouseLeft})
	app = updated.(App)
	assert.Equal(t, 1, app.matrixView.rows.Cursor)
	assert.Equal(t, 1, app.matrixView.cols.Cursor, "key-column click keeps the current column")
}

func TestMouseClickMatrixMode_SelectsTheColumnDrawnAtX(t *testing.T) {
	app := press(resize(newTestApp([]*model.EnvFile{makeTestFile(".env.a", "K"), makeTestFile(".env.b", "K")}), 120, 24), "m")
	header, _, _ := strings.Cut(stripAnsi(app.View().Content), "\n")
	x := strings.Index(header, ".env.b")
	require.Positive(t, x, "the header should name .env.b")

	for _, tt := range []struct {
		cell   string
		x, col int
	}{
		{"the first cell of .env.b, under its name", x, 1},
		{"the last cell of .env.a, just before it", x - 1, 0},
	} {
		updated, _ := app.Update(tea.MouseClickMsg{X: tt.x, Y: matrixContentOffsetY, Button: tea.MouseLeft})
		assert.Equal(t, tt.col, updated.(App).matrixView.cols.Cursor, "click on %s at X=%d", tt.cell, tt.x)
	}
}

func TestMouseClickMatrixMode_OutsideTheDrawnCellsKeepsTheCursor(t *testing.T) {
	keys := make([]string, 40)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	app := press(resize(newTestApp([]*model.EnvFile{makeTestFile(".env.a", keys...), makeTestFile(".env.b", keys...)}), 80, 20), "m")
	app = press(app, down(20)...)
	require.Positive(t, app.matrixView.rows.Offset)
	lines := strings.Split(stripAnsi(app.View().Content), "\n")
	cursorRow := rowDrawing(t, app.View().Content, "K20")
	counterX := strings.Index(lines[cursorRow], "2/2")
	require.Positive(t, counterX, "the cursor row should draw its counter")

	for _, click := range []struct {
		name string
		x, y int
	}{
		{"header", matrixPaddingWidth, 0},
		{"separator", matrixPaddingWidth, matrixContentOffsetY - 1},
		{"stats footer", matrixPaddingWidth, rowDrawing(t, app.View().Content, "40 keys")},
		{"row counter", counterX, cursorRow},
	} {
		updated, _ := app.Update(tea.MouseClickMsg{X: click.x, Y: click.y, Button: tea.MouseLeft})
		clicked := updated.(App).matrixView
		assert.Equal(t, 20, clicked.rows.Cursor, "a click on the %s draws no key", click.name)
		assert.Equal(t, 0, clicked.cols.Cursor, "a click on the %s draws no file", click.name)
		assert.Equal(t, app.matrixView.rows.Offset, clicked.rows.Offset, "a click on the %s scrolls nothing", click.name)
	}
}

func TestMouseClickMatrixMode_BelowTheLastKeyKeepsTheCursor(t *testing.T) {
	app := newMatrixTestApp(t)
	require.Greater(t, app.matrixView.visibleLines(), len(app.matrixView.entries), "the matrix needs padding rows")

	updated, _ := app.Update(tea.MouseClickMsg{X: matrixPaddingWidth, Y: matrixContentOffsetY + len(app.matrixView.entries), Button: tea.MouseLeft})

	assert.Equal(t, 0, updated.(App).matrixView.rows.Cursor)
}

func TestMouseWheelMatrixMode_Scrolls(t *testing.T) {
	app := newMatrixTestApp(t)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.Greater(t, app.matrixView.rows.Cursor, 0, "wheel down should scroll the matrix rows")
}

func TestMouseOnTheActiveFileKeepsTheVarList(t *testing.T) {
	for _, tt := range []struct {
		name                string
		activeFileVarPrefix string
		msg                 tea.Msg
	}{
		{"click on the first file", "FIRST", tea.MouseClickMsg{X: 1, Y: panelContentOffsetY, Button: tea.MouseLeft}},
		{"wheel up on the first file", "FIRST", tea.MouseWheelMsg{X: 1, Y: panelContentOffsetY, Button: tea.MouseWheelUp}},
		{"click on the last file", "LAST", tea.MouseClickMsg{X: 1, Y: panelContentOffsetY + 1, Button: tea.MouseLeft}},
		{"wheel down on the last file", "LAST", tea.MouseWheelMsg{X: 1, Y: panelContentOffsetY + 1, Button: tea.MouseWheelDown}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := varListAtTheEnd(t, tt.activeFileVarPrefix).Update(tt.msg)
			assertVarListKeptAtTheEnd(t, updated.(App), tt.activeFileVarPrefix)
		})
	}
}
