package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestStatusBarSetClearMessage(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, "", sb.Message)

	sb.SetMessage("Saved!")
	assert.Equal(t, "Saved!", sb.Message)

	sb.ClearMessage()
	assert.Equal(t, "", sb.Message)
}

func TestStatusBarViewNormal(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	// Normal mode, file panel
	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "compare")
	assert.Contains(t, view, "help")
	assert.Contains(t, view, ".env")
	assert.Contains(t, view, "5 vars")
}

func TestStatusBarViewNormalVarFocus(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusVars, ".env", 3)
	assert.Contains(t, view, "edit")
	assert.Contains(t, view, "add")
	assert.Contains(t, view, "del")
	assert.Contains(t, view, "yank")
}

func TestStatusBarViewComparing(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	stats := DiffStats{Changed: 2, Added: 1, Removed: 0, Equal: 3}
	view := sb.View(theme, DefaultKeyMap(), ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "copy")
	assert.Contains(t, view, "filter")
}

func TestStatusBarViewComparingAllEqual(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	stats := DiffStats{Equal: 5}
	view := sb.View(theme, DefaultKeyMap(), ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "all equal")
}

func TestStatusBarViewEditing(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeEditing, FocusVars, "", 0)
	assert.Contains(t, view, "Enter to confirm")
}

func TestStatusBarViewConfirmDelete(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeConfirmDelete, FocusVars, "", 0)
	assert.Contains(t, view, "Delete variable")
}

func TestStatusBarViewSearching(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeSearching, FocusVars, "", 0)
	assert.Contains(t, view, "Type to search")
}

func TestStatusBarViewMatrix(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeMatrix, FocusVars, "", 0)
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "add missing")
}

func TestStatusBarViewCompareSelect(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeCompareSelect, FocusFiles, "", 0)
	assert.Contains(t, view, "Select second file")
}

func TestStatusBarReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.ReadOnly = true
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "READ-ONLY")
	assert.Contains(t, view, ".env")
}

func TestStatusBarNoReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.NotContains(t, view, "READ-ONLY")
}

func TestStatusBarMessageOverridesHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetMessage("File saved successfully")
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusVars, ".env", 5)
	assert.Contains(t, view, "File saved successfully")
}

// SGR fragments of the primary, warning and error colors in statusColorTheme,
// used to tell which of them a status-bar message was rendered in.
const (
	statusPrimarySGR = "38;2;0;255;0"
	statusWarningSGR = "38;2;0;0;255"
	statusErrorSGR   = "38;2;255;0;0"
)

func statusColorTheme() Theme {
	return BuildTheme(true, config.ColorConfig{Primary: "#00FF00", Warning: "#0000FF", Error: "#FF0000"})
}

// assertStatusColor checks that view renders its message in want, one of the
// status*SGR fragments, and in neither of the other two colors.
func assertStatusColor(t *testing.T, view, want string) {
	t.Helper()
	for _, sgr := range []string{statusPrimarySGR, statusWarningSGR, statusErrorSGR} {
		if sgr == want {
			assert.Contains(t, view, sgr)
		} else {
			assert.NotContains(t, view, sgr)
		}
	}
}

// statusView renders app's status bar with statusColorTheme.
func statusView(app App) string {
	return app.statusBar.View(statusColorTheme(), app.keys, app.mode, app.focus, "", 0)
}

func TestStatusBarWarningRendersInWarningColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetWarning("File already exists: .env")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "File already exists: .env")
	assertStatusColor(t, view, statusWarningSGR)
}

func TestStatusBarAlertRendersInErrorColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetAlert("Error saving: disk full")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "Error saving: disk full")
	assertStatusColor(t, view, statusErrorSGR)
}

func TestStatusBarMessageAfterAlertRendersInPrimaryColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetAlert("Error saving: disk full")
	sb.SetMessage("Saved .env")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "Saved .env")
	assertStatusColor(t, view, statusPrimarySGR)
}

func TestStatusBarHintsNameTheKeysTheKeyMapBinds(t *testing.T) {
	keys := DefaultKeyMap()
	keys.Save = key.NewBinding(key.WithKeys("s"))
	keys.Enter = key.NewBinding(key.WithKeys("ctrl+j"))
	keys.Quit = key.NewBinding(key.WithKeys("x", "ctrl+c"))
	keys.Deny = key.NewBinding(key.WithKeys("z"))
	sb := NewStatusBarModel()
	sb.Width = 200
	theme := BuildTheme(true, config.ColorConfig{})
	hint := func(mode AppMode, focus Focus) string {
		return stripAnsi(sb.View(theme, keys, mode, focus, ".env", 5, DiffStats{}))
	}

	for _, focus := range []Focus{FocusFiles, FocusVars} {
		assert.Contains(t, hint(ModeNormal, focus), "[s]save")
		assert.NotContains(t, hint(ModeNormal, focus), "[w]", "the old key is gone")
	}
	assert.Contains(t, hint(ModeComparing, FocusVars), "[s]save")
	assert.Contains(t, hint(ModeMatrix, FocusVars), "[x]back")
	assert.Contains(t, hint(ModeCompareSelect, FocusFiles), "Ctrl+J to pick, Esc/x to cancel")
	assert.Contains(t, hint(ModeEditing, FocusVars), "Ctrl+J to confirm")
	assert.Contains(t, hint(ModeConfirmDelete, FocusVars), "(y/z)")
}
