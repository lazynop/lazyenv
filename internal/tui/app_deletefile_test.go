package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func TestDeleteFile_EntersModeOnD(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "D"})
	app = updated.(App)

	assert.Equal(t, ModeConfirmDeleteFile, app.mode)
	assert.Contains(t, app.statusBar.Message, "Delete")
	assert.Contains(t, app.statusBar.Message, "(y/n)")
}

func TestDeleteFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "D"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestDeleteFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "D"})
	app = updated.(App)

	assert.NotEqual(t, ModeConfirmDeleteFile, app.mode)
}

func TestDeleteFile_BlocksIfModified(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	f.Modified = true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "D"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Save or reset")
}

func TestDeleteFile_ConfirmDeletes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile

	updated, _ := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Deleted")

	// File removed from disk
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))

	// File removed from list
	assert.Empty(t, app.fileList.Files)
	assert.Nil(t, app.varList.File)
}

func TestDeleteFile_ConfirmWithMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, ".env")
	path2 := filepath.Join(dir, ".env.local")
	require.NoError(t, os.WriteFile(path1, []byte("A=1\n"), 0644))
	require.NoError(t, os.WriteFile(path2, []byte("B=2\n"), 0644))

	f1, err := parser.ParseFile(path1, config.SecretsConfig{})
	require.NoError(t, err)
	f2, err := parser.ParseFile(path2, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f1, f2})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile
	// First file is selected (index 0)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "y"})
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
	require.NoError(t, os.WriteFile(path1, []byte("A=1\n"), 0644))
	require.NoError(t, os.WriteFile(path2, []byte("B=2\n"), 0644))

	f1, err := parser.ParseFile(path1, config.SecretsConfig{})
	require.NoError(t, err)
	f2, err := parser.ParseFile(path2, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f1, f2})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile
	// Select the last file (index 1)
	app.fileList.Cursor = 1
	app.fileList.Selected = 1
	app.varList.SetFile(f2)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, app.fileList.Files, 1)
	assert.Equal(t, ".env", app.fileList.Files[0].Name)
	// Cursor clamped back to 0
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.Equal(t, 0, app.fileList.Selected)
}

func TestDeleteFile_DenyCancels(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile
	app.statusBar.SetMessage("Delete .env? (y/n)")

	updated, _ := app.Update(tea.KeyPressMsg{Text: "n"})
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
