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
	app.fileInput.SetValue("config.yaml")

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
	app.fileInput.SetValue(".env.test")

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
	app.fileOpSource = src
	app.fileInput.SetValue(".env.copy")

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
	app.fileOpSource = f
	app.fileInput.SetValue(".env.local")

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

func TestFlash_SaveSilentlyDowngradesSingleQuoteWithApostrophe(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "FOO='original'\n")
	app.fileList.Files[0].UpdateVar(0, "it's a trap")
	require.True(t, app.fileList.Files[0].Modified)

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Saved")
	assert.NotContains(t, app.statusBar.Message, "double-quote",
		"downgrade must be silent — no mention in the flash")
	assert.NotContains(t, app.statusBar.Message, "→")
	assert.NotNil(t, cmd)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "FOO=\"it's a trap\"\n", string(data))
}

func TestFlash_SaveSingleQuoteSafeNoDowngradeMessage(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "FOO='original'\n")

	// A safe update (no apostrophe in value) must not trigger the downgrade
	// suffix and must preserve the user's single-quote style on disk.
	app.fileList.Files[0].UpdateVar(0, "plain update")
	require.True(t, app.fileList.Files[0].Modified)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Saved")
	assert.NotContains(t, app.statusBar.Message, "double-quote",
		"no quote style change happened; flash must not mention it")

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "FOO='plain update'\n", string(data))
}

func TestFlashError_SetsMessageAndClearTimer(t *testing.T) {
	app := newTestApp(nil)

	cmd := app.flashError("something broke")

	assert.Equal(t, "something broke", app.statusBar.Message)
	assert.NotNil(t, cmd, "flashError must return the auto-clear timer cmd")
}

func TestFlashError_RendersInErrorColor(t *testing.T) {
	app := newTestApp(nil)
	app.theme = statusColorTheme()

	app.flashError("something broke")

	view := app.statusBar.View(app.theme, app.mode, app.focus, "", 0)
	assert.Contains(t, view, statusErrorSGR)
	assert.NotContains(t, view, statusPrimarySGR)
}

func TestFlash_CompareResetReloadFailureRendersInErrorColor(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)

	app := enterCompareMode(t, fA, fB)
	app.theme = statusColorTheme()
	require.NoError(t, os.Remove(pathB))

	updated, _ := app.Update(tea.KeyPressMsg{Text: "r"})
	app = updated.(App)

	view := app.statusBar.View(app.theme, app.mode, app.focus, "", 0)
	assert.Contains(t, view, "Failed to reload .env.prod")
	assert.Contains(t, view, statusErrorSGR)
	assert.NotContains(t, view, statusPrimarySGR)
}

func TestFlashMessage_AfterErrorRendersInPrimaryColor(t *testing.T) {
	app := newTestApp(nil)
	app.theme = statusColorTheme()

	app.flashError("something broke")
	app.flashMessage("Saved .env")

	view := app.statusBar.View(app.theme, app.mode, app.focus, "", 0)
	assert.Contains(t, view, statusPrimarySGR)
	assert.NotContains(t, view, statusErrorSGR)
}

// TestClearMessageAfterOnlyInFlashHelpers keeps transient status messages
// behind flashMessage/flashError: only app_flash.go may schedule the
// auto-clear, so every message gets the timeout matching its kind.
func TestClearMessageAfterOnlyInFlashHelpers(t *testing.T) {
	assertRefsAllowed(t, sourceRefs(t, "", "clearMessageAfter"),
		map[string]string{"app_flash.go": "the flash helpers"},
		"call a.flashMessage or a.flashError instead of clearMessageAfter")
}
