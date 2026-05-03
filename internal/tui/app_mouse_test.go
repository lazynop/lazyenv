package tui

import (
	"fmt"
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

// --- VarListModel.DisplayCount ---

func TestVarListDisplayCount(t *testing.T) {
	m := NewVarListModel(config.DefaultConfig().Layout)

	assert.Equal(t, 0, m.DisplayCount(), "empty model should have 0 display items")

	f := &model.EnvFile{Vars: []model.EnvVar{
		{Key: "A"}, {Key: "B"}, {Key: "C"},
	}}
	m.SetFile(f)
	assert.Equal(t, 3, m.DisplayCount(), "DisplayCount should match number of vars")

	// With a search that matches 1
	m.SetSearch("A")
	assert.Equal(t, 1, m.DisplayCount(), "DisplayCount should reflect filtered results")

	// Reset search
	m.SetSearch("")
	assert.Equal(t, len(f.Vars), m.DisplayCount(), "DisplayCount should return full count after clearing search")
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
	assert.True(t, app.varList.isCollapsed("DB"),
		"clicking the DB header must toggle collapse on")

	// Click again to expand.
	updated, _ = app.Update(tea.MouseClickMsg{X: clickX, Y: 2, Button: tea.MouseLeft})
	app = updated.(App)
	assert.False(t, app.varList.isCollapsed("DB"),
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

	assert.False(t, app.varList.isCollapsed("DB"),
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
