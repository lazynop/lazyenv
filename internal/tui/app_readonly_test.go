package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

func newReadOnlyApp(files []*model.EnvFile) App {
	cfg := config.DefaultConfig()
	cfg.ReadOnly = true
	app := NewApp(cfg, nil)
	app.width = 120
	app.height = 40
	app.ready = true
	app.fileList.SetFiles(files)
	if len(files) > 0 {
		app.varList.SetFile(files[0])
	}
	return app
}

// --- Config integration ---

func TestReadOnlyDefaultFalse(t *testing.T) {
	cfg := config.DefaultConfig()
	assert.False(t, cfg.ReadOnly)
}

func TestReadOnlyFromConfig(t *testing.T) {
	app := newReadOnlyApp(nil)
	assert.True(t, app.config.ReadOnly)
	assert.True(t, app.statusBar.ReadOnly)
}

// --- Var actions blocked ---

func TestReadOnlyBlocksEdit(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "edit should be blocked in read-only mode")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksEditKey(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "E"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "a"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksDelete(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode, "should not enter ModeConfirmDelete")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

// --- Var read-only actions still work ---

func TestReadOnlyAllowsPeek(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "p"})
	app = updated.(App)

	assert.True(t, app.varList.Peeking, "peek should work in read-only mode")
}

func TestReadOnlyAllowsSearch(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "/"})
	app = updated.(App)

	assert.Equal(t, ModeSearching, app.mode, "search should work in read-only mode")
}

func TestReadOnlyAllowsToggleSort(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "o"})
	app = updated.(App)

	assert.True(t, app.varList.SortAlpha, "sort toggle should work in read-only mode")
}

func TestReadOnlyAllowsToggleSecrets(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	app = updated.(App)

	assert.True(t, app.varList.ShowSecrets, "secrets toggle should work in read-only mode")
}

// --- File actions blocked ---

func TestReadOnlyBlocksCreateFile(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "N"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksDeleteFile(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "D"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

// --- Global actions ---

func TestReadOnlyBlocksSave(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyAllowsCompare(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "c"})
	app = updated.(App)

	assert.Equal(t, ModeCompareSelect, app.mode, "compare should work in read-only mode")
}

func TestReadOnlyAllowsMatrix(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "m"})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "matrix should work in read-only mode")
}

func TestReadOnlyAllowsHelp(t *testing.T) {
	app := newReadOnlyApp(nil)

	updated, _ := app.Update(tea.KeyPressMsg{Text: "?"})
	app = updated.(App)

	assert.Equal(t, ModeHelp, app.mode, "help should work in read-only mode")
}

// --- Compare mode ---

func TestReadOnlyBlocksCompareCopy(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})
	app.diffView.SetFiles(f1, f2)
	app.diffView.Width = 100
	app.diffView.Height = 30
	app.mode = ModeComparing

	// Right arrow = copy to right
	updated, _ := app.Update(tea.KeyPressMsg{Text: "right"})
	app = updated.(App)

	assert.Equal(t, ModeComparing, app.mode, "should stay in compare mode")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksCompareEdit(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})
	app.diffView.SetFiles(f1, f2)
	app.diffView.Width = 100
	app.diffView.Height = 30
	app.mode = ModeComparing

	updated, _ := app.Update(tea.KeyPressMsg{Text: "e"})
	app = updated.(App)

	assert.Equal(t, ModeComparing, app.mode, "should not enter edit mode")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksCompareSave(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})
	app.diffView.SetFiles(f1, f2)
	app.diffView.Width = 100
	app.diffView.Height = 30
	app.mode = ModeComparing

	updated, _ := app.Update(tea.KeyPressMsg{Text: "w"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Read-only")
}

// --- Matrix mode ---

func TestReadOnlyBlocksMatrixDelete(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "FOO")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})

	// Enter matrix mode
	updated, _ := app.Update(tea.KeyPressMsg{Text: "m"})
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)

	// Try to delete — should be blocked
	updated, _ = app.Update(tea.KeyPressMsg{Text: "d"})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "should not enter confirm delete")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}

func TestReadOnlyBlocksMatrixAdd(t *testing.T) {
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.local", "BAR")
	app := newReadOnlyApp([]*model.EnvFile{f1, f2})

	// Enter matrix mode via 'm' key
	updated, _ := app.Update(tea.KeyPressMsg{Text: "m"})
	app = updated.(App)
	assert.Equal(t, ModeMatrix, app.mode)

	// Try to add — should be blocked
	updated, _ = app.Update(tea.KeyPressMsg{Text: "a"})
	app = updated.(App)

	assert.Equal(t, ModeMatrix, app.mode, "should not enter matrix editing")
	assert.Contains(t, app.statusBar.Message, "Read-only")
}
