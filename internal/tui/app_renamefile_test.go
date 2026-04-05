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

func TestRenameFile_EntersModeOnR(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "R"})
	app = updated.(App)

	assert.Equal(t, ModeRenameFile, app.mode)
	assert.Equal(t, ".env", app.renameFileInput.Value())
}

func TestRenameFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "R"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestRenameFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "R"})
	app = updated.(App)

	assert.NotEqual(t, ModeRenameFile, app.mode)
}

func TestRenameFile_BlocksIfModified(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	f.Modified = true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "R"})
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
	assert.Nil(t, app.renameSource)
}

func TestRenameFile_SameNameNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.renameSource = f
	app.renameFileInput.SetValue(".env")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}

func TestRenameFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.renameSource = f
	app.renameFileInput.SetValue(".env.local")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Renamed")
	assert.Contains(t, app.statusBar.Message, "→")

	// Old file gone, new file exists
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, ".env.local"))
	require.NoError(t, err)

	// In-memory file updated
	assert.Equal(t, ".env.local", f.Name)
	assert.Equal(t, filepath.Join(dir, ".env.local"), f.Path)
}

func TestRenameFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.local"), []byte("B=2\n"), 0644))

	f, err := parser.ParseFile(filepath.Join(dir, ".env"), config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.renameSource = f
	app.renameFileInput.SetValue(".env.local")

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
	app.renameSource = f
	app.renameFileInput.SetValue("config.yaml")

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
	app.renameSource = f
	app.renameFileInput.SetValue("sub/.env")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "path separators")
}

func TestRenameFile_UpdatesBackupTracking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.backedUpPaths[path] = true
	app.mode = ModeRenameFile
	app.renameSource = f
	app.renameFileInput.SetValue(".env.local")

	result, _ := app.confirmRenameFile()
	app = result.(App)

	newPath := filepath.Join(dir, ".env.local")
	assert.False(t, app.backedUpPaths[path])
	assert.True(t, app.backedUpPaths[newPath])
}
