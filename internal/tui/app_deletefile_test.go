package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/model"
)

func TestDeleteFile_EntersModeOnD(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("D"))
	app = updated.(App)

	assert.Equal(t, ModeConfirmDeleteFile, app.mode)
	assert.Contains(t, app.statusBar.Message, "Delete")
	assert.Contains(t, app.statusBar.Message, "(y/n)")
}

func TestDeleteFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("D"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestDeleteFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("D"))
	app = updated.(App)

	assert.NotEqual(t, ModeConfirmDeleteFile, app.mode)
}

func TestDeleteFile_BlocksIfModified(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	f.Modified = true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("D"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Save or reset")
}

func TestDeleteFile_ConfirmDeletes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	f := writeAndParse(t, path, "FOO=bar\n")

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile

	updated, _ := app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Deleted")

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))

	assert.Empty(t, app.fileList.Files)
	assert.Nil(t, app.varList.File)
}

func TestDeleteFile_ConfirmDeletesTheFirstOfTwoFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, ".env")
	path2 := filepath.Join(dir, ".env.local")
	f1 := writeAndParse(t, path1, "A=1\n")
	f2 := writeAndParse(t, path2, "B=2\n")

	app := newTestApp([]*model.EnvFile{f1, f2})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile

	updated, _ := app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.fileList.Files, 1)
	assert.Equal(t, ".env.local", app.fileList.Files[0].Name)
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.Equal(t, 0, app.fileList.Selected)
	assert.NotNil(t, app.varList.File)
}

func TestDeleteFile_ConfirmDeletesLastFile(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, ".env")
	path2 := filepath.Join(dir, ".env.local")
	f1 := writeAndParse(t, path1, "A=1\n")
	f2 := writeAndParse(t, path2, "B=2\n")

	app := newTestApp([]*model.EnvFile{f1, f2})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile
	app.fileList.Cursor = 1
	app.fileList.Selected = 1
	app.varList.SetFile(f2)

	updated, _ := app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.fileList.Files, 1)
	assert.Equal(t, ".env", app.fileList.Files[0].Name)
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.Equal(t, 0, app.fileList.Selected)
}

func TestDeleteFile_DenyCancels(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile
	app.statusBar.SetAlert("Delete .env? (y/n)")

	updated, _ := app.Update(keyMsg("n"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
	assert.Len(t, app.fileList.Files, 1)
}

func TestDeleteFile_EscapeCancels(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.fileList.Files, 1)
}
