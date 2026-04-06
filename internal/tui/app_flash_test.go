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

// These tests verify the flash message contract: after an action that shows
// a transient status message, statusBar.Message is set and a non-nil cmd
// (the clear timer) is returned. They must pass both before and after
// extracting a flashMessage/flashError helper.

func TestFlash_CreateFileInvalidPattern(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue("config.yaml")

	result, cmd := app.confirmCreateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "must match")
	assert.NotNil(t, cmd, "should return a clear timer cmd")
}

func TestFlash_CreateFileSuccess(t *testing.T) {
	dir := t.TempDir()
	app := newTestApp(nil)
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeCreateFile
	app.createFileInput.SetValue(".env.test")

	result, cmd := app.confirmCreateFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "Created")
	assert.NotNil(t, cmd)
}

func TestFlash_DuplicateFileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.copy"), []byte(""), 0644))

	src, err := parser.ParseFile(filepath.Join(dir, ".env"), config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = src
	app.duplicateFileInput.SetValue(".env.copy")

	result, cmd := app.confirmDuplicateFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "already exists")
	assert.NotNil(t, cmd)
}

func TestFlash_RenameFileSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("A=1\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeRenameFile
	app.renameSource = f
	app.renameFileInput.SetValue(".env.local")

	result, cmd := app.confirmRenameFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "Renamed")
	assert.NotNil(t, cmd)
}

func TestFlash_RenameKeyInvalid(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	result, cmd := app.confirmRenameKey(EditorResult{
		Value:       "has space",
		VarIndex:    0,
		IsRenameKey: true,
	})
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "Invalid key")
	assert.NotNil(t, cmd)
}

func TestFlash_RenameKeySuccess(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	result, cmd := app.confirmRenameKey(EditorResult{
		Value:       "BAR",
		VarIndex:    0,
		IsRenameKey: true,
	})
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "Renamed")
	assert.NotNil(t, cmd)
}

func TestFlash_DeleteFileConfirm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("A=1\n"), 0644))

	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.focus = FocusFiles
	app.mode = ModeConfirmDeleteFile

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Deleted")
	assert.NotNil(t, cmd)
}

func TestFlash_ToggleSort(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "o"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Sorted")
	assert.NotNil(t, cmd)
}

func TestFlash_ToggleSecrets(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Secrets")
	assert.NotNil(t, cmd)
}
