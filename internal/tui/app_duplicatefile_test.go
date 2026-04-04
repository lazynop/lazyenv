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

func TestDuplicateFile_EntersModeOnC(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "C"})
	app = updated.(App)

	assert.Equal(t, ModeDuplicateFile, app.mode)
	assert.Equal(t, ".env.copy", app.duplicateFileInput.Value())
}

func TestDuplicateFile_EntersModeOnC_DotEnvSuffix(t *testing.T) {
	f := makeTestFile("demo.env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "C"})
	app = updated.(App)

	assert.Equal(t, ModeDuplicateFile, app.mode)
	assert.Equal(t, "demo.copy.env", app.duplicateFileInput.Value())
}

func TestDuplicateFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "C"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestDuplicateFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "C"})
	app = updated.(App)

	assert.NotEqual(t, ModeDuplicateFile, app.mode)
}

func TestDuplicateFile_EscapeCancels(t *testing.T) {
	app := newTestApp(nil)
	app.mode = ModeDuplicateFile

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Nil(t, app.duplicateSource)
}

func TestDuplicateFile_Success(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("FOO=bar\nBAZ=qux\n"), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = src
	app.duplicateFileInput.SetValue(".env.staging")

	result, _ := app.confirmDuplicateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Duplicated")
	assert.Contains(t, app.statusBar.Message, "→")

	// File exists on disk with same content
	data, err := os.ReadFile(filepath.Join(dir, ".env.staging"))
	require.NoError(t, err)
	assert.Equal(t, "FOO=bar\nBAZ=qux\n", string(data))

	// File is in the list and selected
	assert.Len(t, app.fileList.Files, 2)
	assert.Equal(t, 1, app.fileList.Selected)
}

func TestDuplicateFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("FOO=bar\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.local"), []byte(""), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = src
	app.duplicateFileInput.SetValue(".env.local")

	result, _ := app.confirmDuplicateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "already exists")
}

func TestDuplicateFile_InvalidPattern(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("FOO=bar\n"), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = src
	app.duplicateFileInput.SetValue("config.yaml")

	result, _ := app.confirmDuplicateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "must match")
}

func TestDuplicateFile_PathSeparator(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("FOO=bar\n"), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = src
	app.duplicateFileInput.SetValue("sub/.env")

	result, _ := app.confirmDuplicateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "path separators")
}

func TestDuplicateFile_EmptyName(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeDuplicateFile
	app.duplicateSource = makeTestFile(".env", "FOO")
	app.duplicateFileInput.SetValue("")

	result, _ := app.confirmDuplicateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}
