package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

// pageStep is a press of a page or jump key and where it leaves the cursor:
// on item, drawn on screen row row.
type pageStep struct {
	key  string
	item string
	row  int
}

// pressPageStep presses step's key in app and checks that the cursor lands on
// step's item, as cursorItem names the item at the cursor, and that the
// screen draws the item on step's row.
func pressPageStep(t *testing.T, app App, step pageStep, cursorItem func(App) string) App {
	t.Helper()
	app = press(app, step.key)
	assert.Equal(t, step.item, cursorItem(app), "after %s, the cursor should be on %s", step.key, step.item)
	assert.Contains(t, screenRow(t, app, step.row), step.item, "after %s, screen row %d should draw %s", step.key, step.row, step.item)
	return app
}

// pageSteps returns the steps of the page and jump keys through the 40 items
// named in names, in a view that shows 18 rows from screen row first, the
// cursor starting on the second item. A page moves the cursor 18 positions and
// scrolls the view with it, so the cursor keeps its row on screen; where the
// view cannot scroll a whole page, the cursor still moves a page, stopping at
// the first or last item.
func pageSteps(names []string, first int) []pageStep {
	last := first + 17
	return []pageStep{
		{"pgdown", names[19], first + 1},
		{"pgup", names[1], first + 1},
		{"pgdown", names[19], first + 1},
		{"pgdown", names[37], first + 15},
		{"pgdown", names[39], last},
		{"pgdown", names[39], last},
		{"pgup", names[21], last},
		{"pgup", names[3], first + 3},
		{"pgup", names[0], first},
		{"pgup", names[0], first},
		{"end", names[39], last},
		{"home", names[0], first},
	}
}

// numbered returns n names: prefix followed by 00, 01 and so on.
func numbered(prefix string, n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s%02d", prefix, i)
	}
	return names
}

// pageKeys are the 40 variables pageKeyFiles holds.
var pageKeys = numbered("KEY_", 40)

// pageKeyFiles returns two files that both hold pageKeys, for the compare
// view and the matrix to page through.
func pageKeyFiles() []*model.EnvFile {
	return []*model.EnvFile{makeTestFile(".env", pageKeys...), makeTestFile(".env.local", pageKeys...)}
}

// screenRow returns row of app's screen, without colors.
func screenRow(t *testing.T, app App, row int) string {
	t.Helper()
	rows := strings.Split(stripAnsi(app.View().Content), "\n")
	require.Greater(t, len(rows), row)
	return rows[row]
}

// varCursorItem names the var list's item at the cursor as its row draws it:
// a variable by its key, a group header by its arrow and prefix.
func varCursorItem(a App) string {
	if v := a.varList.SelectedVar(); v != nil {
		return v.Key
	}
	return expandedGroupGlyph + " " + a.varList.captureCursor().headerPrefix
}

// TestPageKeysMoveTheVarListCursor pages through a grouped var list taller
// than its panel, which shows 18 rows at 80x24: the same keys as pageSteps,
// over 42 positions that include the group headers.
func TestPageKeysMoveTheVarListCursor(t *testing.T) {
	keys := append(numbered("DB_", 20), numbered("API_", 20)...)
	// Grouped, the positions are the DB header (0), DB_00 to DB_19 (1 to 20),
	// the API header (21) and API_00 to API_19 (22 to 41); the cursor starts
	// on DB_00, on the second row.
	app := press(onVars(resize(newTestApp([]*model.EnvFile{makeTestFile(".env", keys...)}), 80, 24)), "g")
	require.Equal(t, "DB_00", varCursorItem(app))
	first, last := panelContentOffsetY, panelContentOffsetY+17
	for _, step := range []pageStep{
		{"pgdown", "DB_18", first + 1},
		{"pgup", "DB_00", first + 1},
		{"pgdown", "DB_18", first + 1},
		{"pgdown", "API_15", first + 13},
		{"pgdown", "API_19", last},
		{"pgdown", "API_19", last},
		{"pgup", "API_01", last},
		{"pgup", "DB_04", first + 5},
		{"pgup", "▾ DB", first},
		{"pgup", "▾ DB", first},
		{"end", "API_19", last},
		{"home", "▾ DB", first},
	} {
		app = pressPageStep(t, app, step, varCursorItem)
	}
}

// TestPageKeysMoveTheFileListCursor pages through more files than the files
// panel shows at 80x24: like ↑/↓, each key selects the file it lands on, so
// the var list shows it.
func TestPageKeysMoveTheFileListCursor(t *testing.T) {
	names := numbered(".env.", 40)
	files := make([]*model.EnvFile, len(names))
	for i, name := range names {
		files[i] = makeTestFile(name, "KEY")
	}
	app := press(resize(newTestApp(files), 80, 24), "j")
	fileCursorItem := func(a App) string { return a.fileList.CursorFile().Name }
	for _, step := range pageSteps(names, panelContentOffsetY) {
		app = pressPageStep(t, app, step, fileCursorItem)
		assert.Equal(t, step.item, app.fileList.SelectedFile().Name, "after %s, the file at the cursor should be selected", step.key)
		assert.Contains(t, screenRow(t, app, panelContentOffsetY-1), step.item+"  (1 vars)",
			"after %s, the var list's title should name the file at the cursor", step.key)
	}
}

// TestPageKeysMoveTheCompareCursor pages through more entries than the
// compare view shows at 80x24.
func TestPageKeysMoveTheCompareCursor(t *testing.T) {
	app := press(resize(newTestApp(pageKeyFiles()), 80, 24), "c", "j", "enter", "j")
	require.Equal(t, ModeComparing, app.mode)
	diffCursorItem := func(a App) string { return a.diffView.Entries[a.diffView.Cursor].Key }
	for _, step := range pageSteps(pageKeys, panelContentOffsetY) {
		app = pressPageStep(t, app, step, diffCursorItem)
	}
}

// TestPageKeysMoveTheMatrixRowCursor pages through more variables than the
// matrix shows at 80x24, with the column cursor on the second file: the keys
// move the row cursor only.
func TestPageKeysMoveTheMatrixRowCursor(t *testing.T) {
	app := press(resize(newTestApp(pageKeyFiles()), 80, 24), "m", "j", "l")
	require.Equal(t, ModeMatrix, app.mode)
	require.Equal(t, 1, app.matrixView.cols.Cursor)
	matrixCursorItem := func(a App) string { return a.matrixView.entries[a.matrixView.rows.Cursor].Key }
	for _, step := range pageSteps(pageKeys, matrixContentOffsetY) {
		app = pressPageStep(t, app, step, matrixCursorItem)
		assert.Equal(t, 1, app.matrixView.cols.Cursor, "after %s, the column cursor should stay on the second file", step.key)
	}
}

// varListAtTheEnd returns an app at 80x24 over two files of 30 variables,
// FIRST_00 to FIRST_29 and LAST_00 to LAST_29, with the file whose variables
// start with prefix active: its var list has the cursor on the last
// variable, scrolled to the end, and peek on, and the focus is back on the
// files panel.
func varListAtTheEnd(t *testing.T, prefix string) App {
	t.Helper()
	files := []*model.EnvFile{
		makeTestFile(".env.first", numbered("FIRST_", 30)...),
		makeTestFile(".env.last", numbered("LAST_", 30)...),
	}
	app := resize(newTestApp(files), 80, 24)
	if prefix == "LAST" {
		app = press(app, "j")
	}
	app = press(app, "l", "end", "p", "h")
	require.Equal(t, prefix+"_29", varCursorItem(app))
	require.True(t, app.varList.Peeking)
	return app
}

// assertVarListKeptAtTheEnd checks that the var list varListAtTheEnd set up
// for the file whose variables start with prefix kept its cursor, its scroll
// (prefix_12 on the first row) and its peek, rather than starting the file
// over from the top.
func assertVarListKeptAtTheEnd(t *testing.T, app App, prefix string) {
	t.Helper()
	assert.Equal(t, prefix+"_29", varCursorItem(app), "the var list's cursor should stay on the last variable")
	assert.Contains(t, screenRow(t, app, panelContentOffsetY), prefix+"_12", "the var list should stay scrolled to its end")
	assert.True(t, app.varList.Peeking, "the var list should keep peek on")
}

// TestFileKeysThatKeepTheActiveFileKeepTheVarList presses, at either end of
// the file list, the keys that leave the active file as it is, Enter
// included: the var list keeps its cursor, scroll and peek.
func TestFileKeysThatKeepTheActiveFileKeepTheVarList(t *testing.T) {
	for _, tt := range []struct {
		prefix string // of the active file's variables
		key    string
	}{
		{"FIRST", "up"},
		{"FIRST", "pgup"},
		{"FIRST", "home"},
		{"FIRST", "enter"},
		{"LAST", "down"},
		{"LAST", "pgdown"},
		{"LAST", "end"},
	} {
		t.Run(tt.prefix+" "+tt.key, func(t *testing.T) {
			app := press(varListAtTheEnd(t, tt.prefix), tt.key)
			assertVarListKeptAtTheEnd(t, app, tt.prefix)
		})
	}
}
