package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestFileListMoveUp(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")

	var fl FileListModel
	fl.SetFiles(nil)
	fl.Files = append(fl.Files, f1, f2, f3)
	fl.Height = 20

	fl.Cursor = 2
	fl.MoveUp()
	assert.Equal(t, 1, fl.Cursor)

	fl.MoveUp()
	assert.Equal(t, 0, fl.Cursor)

	// Already at top — should stay
	fl.MoveUp()
	assert.Equal(t, 0, fl.Cursor)
}

func TestFileListMoveDown(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")

	var fl FileListModel
	fl.Files = append(fl.Files, f1, f2)
	fl.Height = 20

	fl.MoveDown()
	assert.Equal(t, 1, fl.Cursor)

	// At bottom — should stay
	fl.MoveDown()
	assert.Equal(t, 1, fl.Cursor)
}

func TestFileListSelect(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")

	var fl FileListModel
	fl.Files = append(fl.Files, f1, f2)
	assert.Equal(t, 0, fl.Selected)

	fl.Cursor = 1
	fl.Select()
	assert.Equal(t, 1, fl.Selected)
}

func TestFileListSelectedFile(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	var fl FileListModel
	fl.Files = append(fl.Files, f1)
	fl.Selected = 0

	assert.Equal(t, f1, fl.SelectedFile())

	// Out of range
	fl.Selected = 5
	assert.Nil(t, fl.SelectedFile())

	// Negative
	fl.Selected = -1
	assert.Nil(t, fl.SelectedFile())
}

func TestFileListCursorFile(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	var fl FileListModel
	fl.Files = append(fl.Files, f1)
	fl.Cursor = 0

	assert.Equal(t, f1, fl.CursorFile())

	// Out of range
	fl.Cursor = 5
	assert.Nil(t, fl.CursorFile())
}

func TestFileListScrolling(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	f4 := makeTestFile(".env.test", "D")

	var fl FileListModel
	fl.Files = append(fl.Files, f1, f2, f3, f4)
	fl.Height = 4 // visible = 4 - 2 = 2

	fl.MoveDown() // cursor=1
	fl.MoveDown() // cursor=2, should scroll
	assert.Equal(t, 2, fl.Cursor)
	assert.Greater(t, fl.Offset, 0, "should scroll when cursor exceeds visible area")

	// Move up past offset
	fl.MoveUp()
	fl.MoveUp()
	assert.Equal(t, 0, fl.Cursor)
	assert.Equal(t, 0, fl.Offset)
}

func TestFileListSetFilesResetsCursor(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")

	var fl FileListModel
	fl.Files = append(fl.Files, f1, f2)
	fl.Cursor = 1
	fl.Selected = 1

	// Setting to a smaller list should reset
	fl.SetFiles(nil)
	assert.Equal(t, 0, fl.Cursor)
	assert.Equal(t, 0, fl.Selected)
}

func TestFileListViewEmpty(t *testing.T) {
	var fl FileListModel
	fl.Width = 30
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)
	assert.Contains(t, view, "No .env files found")
}

func TestFileListViewTruncatesLongNames(t *testing.T) {
	f := makeTestFile(".env.development.local.backup.extra", "FOO")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 25
	fl.Height = 10
	fl.Focused = true

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)

	// Name should be truncated — full name should NOT appear
	assert.NotContains(t, view, ".env.development.local.backup.extra")
	// Truncation marker should appear
	assert.Contains(t, view, "..")
}

func TestFileListViewShortNameNotTruncated(t *testing.T) {
	f := makeTestFile(".env", "FOO")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 30
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)
	assert.Contains(t, view, ".env")
	assert.NotContains(t, view, "..")
}

func TestFileListViewWithFiles(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f1.Modified = true
	f1.GitWarning = true

	var fl FileListModel
	fl.Files = append(fl.Files, f1)
	fl.Width = 40
	fl.Height = 10
	fl.Focused = true

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)
	assert.Contains(t, view, ".env")
}

func TestFileListViewShowsVarCount(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 40
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)
	// Right-aligned count: number appears after gap
	assert.Contains(t, view, "3")
}

func TestFileListViewShowsZeroVarCount(t *testing.T) {
	f := makeTestFile(".env")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 40
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := fl.View(theme)
	assert.Contains(t, view, "0")
}

func TestFileListViewVarCountUpdatesAfterAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 40
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})

	view := fl.View(theme)
	assert.Contains(t, view, "1")

	f.AddVar("BAR", "val", false)
	view = fl.View(theme)
	assert.Contains(t, view, "2")
}

func TestFileListViewVarCountUpdatesAfterDelete(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")

	var fl FileListModel
	fl.Files = append(fl.Files, f)
	fl.Width = 40
	fl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})

	view := fl.View(theme)
	assert.Contains(t, view, "3")

	f.DeleteVar(0)
	view = fl.View(theme)
	assert.Contains(t, view, "2")
}
