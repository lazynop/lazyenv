package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFile_EntersModeOnN(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "N"})
	app = updated.(App)

	assert.Equal(t, ModeCreateFile, app.mode)
}

func TestCreateFile_OnlyFromFocusFiles(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "N"})
	app = updated.(App)

	assert.NotEqual(t, ModeCreateFile, app.mode)
}

func TestCreateFile_EscapeCancels(t *testing.T) {
	app := newTestApp(nil)
	app.mode = ModeCreateFile

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestCreateFile_EmptyNameCancels(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue("")

	result, _ := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}

func TestCreateFile_PathSeparatorRejected(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue("subdir/.env")

	result, _ := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "path separators")
}

func TestCreateFile_InvalidPatternRejected(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue("config.yaml")

	result, _ := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "must match")
}

func TestCreateFile_AlreadyExistsRejected(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(""), 0644))

	app := newTestApp(nil)
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue(".env")

	result, _ := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "already exists")
}

func TestCreateFile_Success(t *testing.T) {
	dir := t.TempDir()

	app := newTestApp(nil)
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue(".env.staging")

	result, _ := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Created")

	// File exists on disk
	_, err := os.Stat(filepath.Join(dir, ".env.staging"))
	require.NoError(t, err)

	// File is in the list and selected
	assert.Len(t, app.fileList.Files, 1)
	assert.Equal(t, 0, app.fileList.Selected)
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.NotNil(t, app.varList.File)
}
