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

// newTestAppWithDiskFile creates an App with a real file on disk for save/reset tests.
func newTestAppWithDiskFile(t *testing.T, content string) (App, string) {
	t.Helper()
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte(content), 0644))

	f, err := parser.ParseFile(envPath, config.SecretsConfig{})
	require.NoError(t, err)
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

	// Simulate an edit so Modified=true (reset requires it).
	app.fileList.Files[0].UpdateVar(0, "changed")
	require.True(t, app.fileList.Files[0].Modified)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "r"})
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

	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	savedFile := app.fileList.Files[0]
	assert.False(t, savedFile.Modified, "Modified should be cleared after save")
	assert.True(t, savedFile.GitWarning, "GitWarning must survive save")
	assert.Equal(t, "new_value", savedFile.Vars[0].Value)
}

func TestCompareSavePreservesGitWarning(t *testing.T) {
	// Create two real files on disk.
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fA.GitWarning = true

	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)
	fB.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, fA, fB)
	app.fileList.Selected = 0
	app.varList.SetFile(fA)

	// Enter compare mode and modify file A.
	app.mode = ModeComparing
	app.diffView.FileA = fA
	app.diffView.FileB = fB
	app.diffView.SetFiles(fA, fB)
	fA.UpdateVar(0, "changed")
	require.True(t, fA.Modified)

	// Press 'w' to save in compare mode.
	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning on file A must survive compare save")
	assert.True(t, app.fileList.Files[1].GitWarning, "GitWarning on file B must survive compare save")
}

func TestCompareResetPreservesGitWarning(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fA.GitWarning = true

	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)
	fB.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, fA, fB)
	app.fileList.Selected = 0
	app.varList.SetFile(fA)

	// Enter compare mode and modify file A.
	app.mode = ModeComparing
	app.diffView.SetFiles(fA, fB)
	fA.UpdateVar(0, "changed")
	require.True(t, fA.Modified)

	// Press 'r' to reset in compare mode.
	updated, _ := app.Update(tea.KeyPressMsg{Text: "r"})
	app = updated.(App)

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning on file A must survive compare reset")
	assert.True(t, app.fileList.Files[1].GitWarning, "GitWarning on file B must survive compare reset")
}

func TestSaveFromMatrixPreservesGitWarning(t *testing.T) {
	// Create two real files, enter matrix, add a var, then save.
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("BAR=b\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fA.GitWarning = true

	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)
	fB.GitWarning = true

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, fA, fB)
	app.fileList.Selected = 0
	app.varList.SetFile(fA)

	// Add a var to file B via AddVar (simulating matrix add).
	fB.AddVar("FOO", "from_matrix", false)
	require.True(t, fB.Modified)

	// Save file B.
	app.varList.SetFile(fB)
	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	for _, f := range app.fileList.Files {
		assert.True(t, f.GitWarning, "%s: GitWarning must survive save", f.Name)
	}
	// Verify the added var was saved.
	var prodFile *model.EnvFile
	for _, f := range app.fileList.Files {
		if f.Name == ".env.prod" {
			prodFile = f
		}
	}
	require.NotNil(t, prodFile)
	found := false
	for _, v := range prodFile.Vars {
		if v.Key == "FOO" {
			found = true
		}
	}
	assert.True(t, found, "FOO should exist in .env.prod after save")
}
