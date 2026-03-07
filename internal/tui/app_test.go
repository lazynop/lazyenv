package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
)

func TestUpdateWindowSizeMsg(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Equal(t, 200, app.width)
	assert.Equal(t, 50, app.height)
	assert.True(t, app.ready)
}

func TestUpdateBackgroundColorMsg(t *testing.T) {
	app := newTestApp(nil)
	assert.True(t, app.hasDarkBg) // default

	// Simulate light background
	updated, cmd := app.Update(tea.BackgroundColorMsg{})
	app = updated.(App)
	assert.Nil(t, cmd)
	// The theme should have been rebuilt (we can't easily check IsDark() on a zero msg,
	// but we verify it doesn't crash and returns)
}

func TestUpdateFilesLoadedMsg(t *testing.T) {
	app := newTestApp(nil)
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "BAR")

	updated, cmd := app.Update(FilesLoadedMsg{
		Files: []*model.EnvFile{f1, f2},
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Len(t, app.fileList.Files, 2)
	assert.Equal(t, f1, app.varList.File)
}

func TestUpdateFilesLoadedMsgError(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(FilesLoadedMsg{
		Err: assert.AnError,
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Contains(t, app.statusBar.Message, "Error")
}

func TestUpdateFilesLoadedMsgEmpty(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(FilesLoadedMsg{
		Files: []*model.EnvFile{},
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Empty(t, app.fileList.Files)
}

func TestUpdateClearMessageMsg(t *testing.T) {
	app := newTestApp(nil)
	app.statusBar.SetMessage("temporary")
	assert.Equal(t, "temporary", app.statusBar.Message)

	updated, cmd := app.Update(ClearMessageMsg{})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Equal(t, "", app.statusBar.Message)
}

func TestViewHelp(t *testing.T) {
	app := newTestApp(nil)
	app.width = 100
	app.height = 40

	help := app.viewHelp()
	assert.Contains(t, help, "lazyenv")
	assert.Contains(t, help, "Navigation")
	assert.Contains(t, help, "Actions")
	assert.Contains(t, help, "Press Esc or ? to close")
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hello", truncate("hello", 5))
	assert.Equal(t, "hel..", truncate("hello world", 5))
	assert.Equal(t, "he", truncate("hello", 2))
	assert.Equal(t, "", truncate("", 5))
	assert.Equal(t, "h", truncate("hello", 1))
}

func TestNavigationUpDownInFilePanel(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	app.focus = FocusFiles

	// Move down
	updated, _ := app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor)
	assert.Equal(t, f2, app.varList.File, "var panel should sync with file cursor")

	// Move up
	updated, _ = app.Update(tea.KeyPressMsg{Text: "k"})
	app = updated.(App)
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.Equal(t, f1, app.varList.File)
}

func TestNavigationUpDownInVarPanel(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	assert.Equal(t, 1, app.varList.Cursor)

	updated, _ = app.Update(tea.KeyPressMsg{Text: "k"})
	app = updated.(App)
	assert.Equal(t, 0, app.varList.Cursor)
}

func TestEnterSelectsFileAndSwitchesToVars(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	app := newTestApp([]*model.EnvFile{f1, f2})
	app.focus = FocusFiles

	// Move to second file and press enter
	updated, _ := app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	assert.Equal(t, FocusVars, app.focus)
	assert.Equal(t, f2, app.varList.File)
}

func TestQuitReturnsQuitCmd(t *testing.T) {
	app := newTestApp(nil)

	_, cmd := app.Update(tea.KeyPressMsg{Text: "q"})
	// tea.Quit is a function, we can check cmd is not nil
	assert.NotNil(t, cmd)
}

func TestViewNotReady(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	// ready is false by default
	view := app.View()
	assert.Contains(t, view.Content, "Loading")
}
