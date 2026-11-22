package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func writeAndParse(t *testing.T, path, content string) *model.EnvFile {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)
	return f
}

func newTestAppWithDiskFile(t *testing.T, content string) (App, string) {
	t.Helper()
	envPath := filepath.Join(t.TempDir(), ".env")
	f := writeAndParse(t, envPath, content)
	f.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.fileList.Selected = 0
	app.varList.SetFile(f)
	app.focus = FocusVars
	return app, envPath
}

func TestResetPreservesGitWarning(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "FOO=bar\n")

	app.fileList.Files[0].UpdateVar(0, "changed")
	require.True(t, app.fileList.Files[0].Modified, "reset acts only on a modified file")

	updated, _ := app.Update(keyMsg("r"))
	app = updated.(App)

	resetFile := app.fileList.Files[0]
	assert.False(t, resetFile.Modified, "Modified should be cleared after reset")
	assert.True(t, resetFile.GitWarning, "GitWarning must survive reset")
	assert.Equal(t, "bar", resetFile.Vars[0].Value, "Value should be restored from disk")
}

func TestSavePreservesGitWarning(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "FOO=bar\n")

	app.fileList.Files[0].UpdateVar(0, "new_value")
	require.True(t, app.fileList.Files[0].Modified)

	updated, _ := app.Update(keyMsg("w"))
	app = updated.(App)

	savedFile := app.fileList.Files[0]
	assert.False(t, savedFile.Modified, "Modified should be cleared after save")
	assert.True(t, savedFile.GitWarning, "GitWarning must survive save")
	assert.Equal(t, "new_value", savedFile.Vars[0].Value)
}

func TestCompareSavePreservesGitWarning(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	fA := writeAndParse(t, pathA, "FOO=a\n")
	fA.GitWarning = true
	fB := writeAndParse(t, pathB, "FOO=b\n")
	fB.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, fA, fB)
	app.fileList.Selected = 0
	app.varList.SetFile(fA)

	app.mode = ModeComparing
	app.diffView.FileA = fA
	app.diffView.FileB = fB
	app.diffView.SetFiles(fA, fB)
	fA.UpdateVar(0, "changed")
	require.True(t, fA.Modified)

	updated, _ := app.Update(keyMsg("w"))
	app = updated.(App)

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning on file A must survive compare save")
	assert.True(t, app.fileList.Files[1].GitWarning, "GitWarning on file B must survive compare save")
}

func TestCompareResetPreservesGitWarning(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	fA := writeAndParse(t, pathA, "FOO=a\n")
	fA.GitWarning = true
	fB := writeAndParse(t, pathB, "FOO=b\n")
	fB.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, fA, fB)
	app.fileList.Selected = 0
	app.varList.SetFile(fA)

	app.mode = ModeComparing
	app.diffView.SetFiles(fA, fB)
	fA.UpdateVar(0, "changed")
	require.True(t, fA.Modified)

	updated, _ := app.Update(keyMsg("r"))
	app = updated.(App)

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning on file A must survive compare reset")
	assert.True(t, app.fileList.Files[1].GitWarning, "GitWarning on file B must survive compare reset")
}

func TestSavePreservesNoTrailingNewline(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "FOO=bar")
	require.False(t, app.fileList.Files[0].TrailingNewline,
		"ParseFile must detect no trailing newline on source")

	app.fileList.Files[0].UpdateVar(0, "first")
	require.True(t, app.fileList.Files[0].Modified)

	updated, _ := app.Update(keyMsg("w"))
	app = updated.(App)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "FOO=first", string(data),
		"first save must not inject a trailing newline")
	assert.False(t, app.fileList.Files[0].TrailingNewline,
		"re-parsed file must still have TrailingNewline=false")

	app.fileList.Files[0].UpdateVar(0, "second")
	updated, _ = app.Update(keyMsg("w"))
	app = updated.(App)

	data, err = os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "FOO=second", string(data),
		"second save must still not inject a trailing newline")
}

func TestSavePreservesTrailingNewline(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "FOO=bar\n")
	require.True(t, app.fileList.Files[0].TrailingNewline)

	app.fileList.Files[0].UpdateVar(0, "first")
	updated, _ := app.Update(keyMsg("w"))
	app = updated.(App)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "FOO=first\n", string(data),
		"save must preserve the trailing newline that was there on read")
}

func TestSaveFromMatrixPreservesGitWarning(t *testing.T) {
	app := loadFixtureApp(t, config.DefaultConfig(), map[string]string{
		".env":      "FOO=a\n",
		".env.prod": "BAR=b\n",
	}, 80, 24)
	for _, f := range app.fileList.Files {
		f.GitWarning = true
	}
	env, prod := app.fileList.Files[0], app.fileList.Files[1]
	require.Equal(t, ".env.prod", prod.Name)
	require.Same(t, env, app.varList.File, "the var list starts on .env, so the save must follow the file selected after the matrix")

	app = press(app, "m", "j", "right")
	cell, ok := app.matrixView.CursorCell()
	require.True(t, ok)
	require.Equal(t, "FOO", cell.Key)
	require.Same(t, prod, cell.File)
	require.False(t, cell.Present)

	app = press(app, "a")
	require.Equal(t, ModeMatrixEditing, app.mode)
	app = press(app, typing("from_matrix")...)
	app = press(app, "enter", "esc")
	require.Equal(t, ModeNormal, app.mode)
	require.True(t, prod.Modified, "the matrix add stays in memory until a save")

	require.Equal(t, FocusFiles, app.focus, "leaving the matrix keeps the file list focused")
	app = press(app, "j")
	require.Same(t, prod, app.varList.File)
	app = press(app, "w")

	for _, f := range app.fileList.Files {
		assert.True(t, f.GitWarning, "%s: GitWarning must survive save", f.Name)
	}
	data, err := os.ReadFile(prod.Path)
	require.NoError(t, err)
	assert.Equal(t, "BAR=b\nFOO=from_matrix\n", string(data))
}
