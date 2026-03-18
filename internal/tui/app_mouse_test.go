package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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
	// No file set — displayIndices is empty; should not panic
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
