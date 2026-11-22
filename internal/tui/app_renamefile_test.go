package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

func TestRenameFile_EntersModeOnR(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("R"))
	app = updated.(App)

	assert.Equal(t, ModeRenameFile, app.mode)
	assert.Equal(t, ".env", app.fileInput.Value())
}

func TestRenameFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("R"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestRenameFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("R"))
	app = updated.(App)

	assert.NotEqual(t, ModeRenameFile, app.mode)
}

func TestRenameFile_BlocksIfModified(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	f.Modified = true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("R"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Save or reset")
}

func TestRenameFile_EscapeCancels(t *testing.T) {
	app := newTestApp(nil)
	app.mode = ModeRenameFile

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Nil(t, app.fileOpSource)
}

func TestRenameFile_SameNameNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	f := writeAndParse(t, path, "FOO=bar\n")

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.fileOpSource = f
	app.fileInput.SetValue(".env")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}

func TestRenameFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	f := writeAndParse(t, path, "FOO=bar\n")

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles

	app = submitFileName(app, "R", ".env.local")

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Renamed")
	assert.Contains(t, app.statusBar.Message, "→")

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, ".env.local"))
	require.NoError(t, err)

	assert.Equal(t, ".env.local", f.Name)
	assert.Equal(t, filepath.Join(dir, ".env.local"), f.Path)
}

func TestRenameFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	f := writeAndParse(t, filepath.Join(dir, ".env"), "A=1\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.local"), []byte("B=2\n"), 0o644))

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.fileOpSource = f
	app.fileInput.SetValue(".env.local")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "already exists")
}

func TestRenameFile_InvalidPattern(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.fileOpSource = f
	app.fileInput.SetValue("config.yaml")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "must match")
}

func TestRenameFile_PathSeparator(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.fileOpSource = f
	app.fileInput.SetValue("sub/.env")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "path separators")
}

func TestRenameFile_UpdatesBackupTracking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	f := writeAndParse(t, path, "FOO=bar\n")

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.backupAttempted[path] = true
	app.focus = FocusFiles

	app = submitFileName(app, "R", ".env.local")

	newPath := filepath.Join(dir, ".env.local")
	assert.False(t, app.backupAttempted[path])
	assert.True(t, app.backupAttempted[newPath])
}
