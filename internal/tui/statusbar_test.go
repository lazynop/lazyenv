package tui

import (
	"testing"

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
	view := sb.View(theme, ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "compare")
	assert.Contains(t, view, "help")
	assert.Contains(t, view, ".env")
	assert.Contains(t, view, "5 vars")
}

func TestStatusBarViewNormalVarFocus(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeNormal, FocusVars, ".env", 3)
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
	view := sb.View(theme, ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "copy")
	assert.Contains(t, view, "filter")
}

func TestStatusBarViewComparingAllEqual(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	stats := DiffStats{Equal: 5}
	view := sb.View(theme, ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "all equal")
}

func TestStatusBarViewEditing(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeEditing, FocusVars, "", 0)
	assert.Contains(t, view, "Enter to confirm")
}

func TestStatusBarViewConfirmDelete(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeConfirmDelete, FocusVars, "", 0)
	assert.Contains(t, view, "Delete variable")
}

func TestStatusBarViewHelp(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeHelp, FocusVars, "", 0)
	assert.Contains(t, view, "close help")
}

func TestStatusBarViewSearching(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeSearching, FocusVars, "", 0)
	assert.Contains(t, view, "Type to search")
}

func TestStatusBarViewMatrix(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeMatrix, FocusVars, "", 0)
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "add missing")
}

func TestStatusBarViewCompareSelect(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeCompareSelect, FocusFiles, "", 0)
	assert.Contains(t, view, "Select second file")
}

func TestStatusBarReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.ReadOnly = true
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "READ-ONLY")
	assert.Contains(t, view, ".env")
}

func TestStatusBarNoReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeNormal, FocusFiles, ".env", 5)
	assert.NotContains(t, view, "READ-ONLY")
}

func TestStatusBarMessageOverridesHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetMessage("File saved successfully")
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, ModeNormal, FocusVars, ".env", 5)
	assert.Contains(t, view, "File saved successfully")
}
