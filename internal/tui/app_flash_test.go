package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

// TestLongMessageKeepsTheScreenInTheTerminal shows a message that wraps the
// status bar onto many rows, in a view without an input bar and in one that
// opens it while the message is still up: the screen stays within the
// terminal.
func TestLongMessageKeepsTheScreenInTheTerminal(t *testing.T) {
	for _, keys := range [][]string{{"right"}, {"right", "e"}} {
		updated, _ := newTestApp(layoutFiles()).Update(tea.WindowSizeMsg{Width: 60, Height: 16})
		app := updated.(App)
		app.flashError("Error saving: " + strings.Repeat("permission denied ", 12))

		app = press(app, keys...)

		assert.LessOrEqual(t, lipgloss.Height(app.View().Content), 16, "after %v: taller than the terminal", keys)
	}
}

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
	app.fileList.Files[0].UpdateVar(0, "it's a trap", false)
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
	app.fileList.Files[0].UpdateVar(0, "plain update", false)
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

func TestFlashWarning_SetsMessageAndClearTimer(t *testing.T) {
	app := newTestApp(nil)

	cmd := app.flashWarning("not allowed")

	assert.Equal(t, "not allowed", app.statusBar.Message)
	assert.NotNil(t, cmd, "flashWarning must return the auto-clear timer cmd")
	assertStatusColor(t, statusView(app), statusWarningSGR)
}

func TestFlashError_RendersInErrorColor(t *testing.T) {
	app := newTestApp(nil)

	app.flashError("something broke")

	view := statusView(app)
	assertStatusColor(t, view, statusErrorSGR)
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
	require.NoError(t, os.Remove(pathB))

	app = pressKeys(app, "r")

	view := statusView(app)
	assert.Contains(t, view, "Failed to reload .env.prod")
	assertStatusColor(t, view, statusErrorSGR)
}

func TestFlashMessage_AfterErrorRendersInPrimaryColor(t *testing.T) {
	app := newTestApp(nil)

	app.flashError("something broke")
	app.flashMessage("Saved .env")

	view := statusView(app)
	assertStatusColor(t, view, statusPrimarySGR)
}

// TestClearMessageAfterOnlyInFlashHelpers keeps transient status messages
// behind flashMessage/flashWarning/flashError: only app_flash.go may schedule
// the auto-clear, so every message gets the timeout matching its kind.
func TestClearMessageAfterOnlyInFlashHelpers(t *testing.T) {
	assertRefsAllowed(t, sourceRefs(t, "", "clearMessageAfter"),
		map[string]string{"app_flash.go": "the flash helpers"},
		"call a.flashMessage, a.flashWarning or a.flashError instead of clearMessageAfter")
}

// pressKeys sends each key to app in turn and returns the updated app.
func pressKeys(app App, keys ...string) App {
	for _, k := range keys {
		app = sendKey(app, tea.KeyPressMsg{Text: k})
	}
	return app
}

// TestRefusedActionsFlashWarning walks every action refused because of the
// input or the current state: each must report the reason in the warning
// color, not in the primary color of a completed action.
func TestRefusedActionsFlashWarning(t *testing.T) {
	confirm := func(app App, run func(App) (tea.Model, tea.Cmd)) App {
		result, _ := run(app)
		return result.(App)
	}
	withFileOp := func(name string) App {
		src := makeTestFile(".env", "FOO")
		app := newTestApp([]*model.EnvFile{src})
		app.fileOpSource = src
		app.fileInput.SetValue(name)
		return app
	}
	modifiedFileFocused := func() App {
		f := makeTestFile(".env", "FOO")
		f.Modified = true
		app := newTestApp([]*model.EnvFile{f})
		app.focus = FocusFiles
		return app
	}

	tests := []struct {
		name string
		act  func(t *testing.T) App
		want string
	}{
		{"edit in read-only mode", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO")})
			app.config.ReadOnly = true
			app.focus = FocusVars
			return pressKeys(app, "e")
		}, "Read-only mode"},
		{"create a file outside the patterns", func(t *testing.T) App {
			app := newTestApp(nil)
			app.config.Dir = t.TempDir()
			app.fileInput.SetValue("config.yaml")
			return confirm(app, App.confirmCreateFile)
		}, "Invalid name"},
		{"duplicate onto an existing file", func(t *testing.T) App {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.copy"), nil, 0644))
			app := withFileOp(".env.copy")
			app.fileOpSource.Path = filepath.Join(dir, ".env")
			return confirm(app, App.confirmDuplicateFile)
		}, "File already exists"},
		{"rename a file to a path", func(t *testing.T) App {
			return confirm(withFileOp("sub/.env.local"), App.confirmRenameFile)
		}, "Invalid name"},
		{"template a file to a path", func(t *testing.T) App {
			return confirm(withFileOp("sub/.env.example"), App.confirmTemplateFile)
		}, "Invalid name"},
		{"rename a key to an invalid name", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO")})
			app.focus = FocusVars
			result, _ := app.confirmRenameKey(EditorResult{Value: "has space", VarIndex: 0, IsRenameKey: true})
			return result.(App)
		}, "Invalid key name"},
		{"rename a key onto an existing key", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO", "BAR")})
			app.focus = FocusVars
			result, _ := app.confirmRenameKey(EditorResult{Value: "BAR", VarIndex: 0, IsRenameKey: true})
			return result.(App)
		}, "Key already exists"},
		{"delete a modified file", func(t *testing.T) App {
			return pressKeys(modifiedFileFocused(), "D")
		}, "before deleting"},
		{"rename a modified file", func(t *testing.T) App {
			return pressKeys(modifiedFileFocused(), "R")
		}, "before renaming"},
		{"open the matrix with one file", func(t *testing.T) App {
			return pressKeys(newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO")}), "m")
		}, "at least 2 files"},
		{"edit a group header", func(t *testing.T) App {
			app, _ := newGroupingApp(t)
			app.varList.Grouping = true
			app.varList.Refresh()
			app.varList.SetCursor(0)
			return pressKeys(app, "e")
		}, "No variable selected"},
		{"edit a key missing on the compare side", func(t *testing.T) App {
			app := enterCompareMode(t, makeTestFile(".env", "FOO"), makeTestFile(".env.prod"))
			return pressKeys(app, "E")
		}, "FOO not in .env.prod"},
		{"reorder a single variable", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO")})
			app.focus = FocusVars
			return pressKeys(app, "O")
		}, "at least 2 variables"},
		{"save with no file", func(t *testing.T) App {
			return pressKeys(newTestApp(nil), "w")
		}, "No file selected"},
		{"reset with no file", func(t *testing.T) App {
			return pressKeys(newTestApp(nil), "r")
		}, "No file selected"},
		{"pick a reorder with no file", func(t *testing.T) App {
			app := newTestApp(nil)
			app.mode = ModeReorderMenu
			return pressKeys(app, "a")
		}, "No file selected"},
		{"confirm a reorder with no file", func(t *testing.T) App {
			app := newTestApp(nil)
			app.mode = ModeReorderConfirm
			return pressKeys(app, "y")
		}, "No file selected"},
		{"add a variable present in the matrix cell", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.local", "A")})
			return pressKeys(app, "m", "a")
		}, "Variable already exists"},
		{"delete a variable missing from the matrix cell", func(t *testing.T) App {
			app := newTestApp([]*model.EnvFile{makeTestFile(".env", "A", "B"), makeTestFile(".env.local", "A")})
			return pressKeys(app, "m", "j", "l", "d")
		}, "not present in this file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.act(t)

			assert.Contains(t, app.statusBar.Message, tt.want)
			assertStatusColor(t, statusView(app), statusWarningSGR)
		})
	}
}

func TestDeletePromptsRenderInErrorColor(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		app := newTestApp([]*model.EnvFile{makeTestFile(".env", "FOO")})
		app.focus = FocusFiles

		app = pressKeys(app, "D")

		require.Equal(t, ModeConfirmDeleteFile, app.mode)
		assert.Equal(t, "Delete .env? (y/n)", app.statusBar.Message)
		assertStatusColor(t, statusView(app), statusErrorSGR)
	})

	t.Run("matrix variable", func(t *testing.T) {
		app := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.local", "A")})

		app = pressKeys(app, "m", "d")

		require.Equal(t, ModeConfirmMatrixDelete, app.mode)
		assert.Equal(t, "Delete A from .env? (y/n)", app.statusBar.Message)
		assertStatusColor(t, statusView(app), statusErrorSGR)
	})
}
