package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/config/themes"
)

func TestNewThemePreview(t *testing.T) {
	m := NewThemePreview()

	assert.NotEmpty(t, m.themes, "themes list should be populated")
	assert.NotEmpty(t, m.searchPaths, "searchPaths should be populated")
	assert.Equal(t, 0, m.cursor, "cursor should start at 0")
	assert.Equal(t, "", m.selected, "selected should start empty")

	// Themes list should match config.ThemeNames()
	expected := config.ThemeNames()
	assert.Equal(t, expected, m.themes)
}

func TestThemePreviewNavigateDown(t *testing.T) {
	m := NewThemePreview()
	require.Greater(t, len(m.themes), 1, "need at least 2 themes to test navigation")

	// "j" key moves cursor down
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 1, m.cursor)

	// "down" arrow also moves cursor down
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 2, m.cursor)
}

func TestThemePreviewNavigateUp(t *testing.T) {
	m := NewThemePreview()
	require.Greater(t, len(m.themes), 1, "need at least 2 themes to test navigation")

	// Move down first
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(ThemePreviewModel)
	updated, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 2, m.cursor)

	// "k" key moves cursor up
	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 1, m.cursor)

	// "up" arrow also moves cursor up
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor)
}

func TestThemePreviewCursorDoesNotGoBelowZero(t *testing.T) {
	m := NewThemePreview()
	assert.Equal(t, 0, m.cursor)

	// Pressing up at top should not go below 0
	updated, _ := m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")
}

func TestThemePreviewCursorDoesNotExceedMax(t *testing.T) {
	m := NewThemePreview()
	maxIdx := len(m.themes) - 1

	// Move to the end
	for range len(m.themes) + 5 {
		updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
		m = updated.(ThemePreviewModel)
	}
	assert.Equal(t, maxIdx, m.cursor, "cursor should not exceed last index")
}

func TestThemePreviewSelectWithEnter(t *testing.T) {
	m := NewThemePreview()
	expectedTheme := m.themes[0]

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, expectedTheme, m.selected, "selected should be set to current theme")
	require.NotNil(t, cmd, "cmd should not be nil after Enter")

	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "Enter should return tea.Quit")
}

func TestThemePreviewSelectWithEnterAfterNavigation(t *testing.T) {
	m := NewThemePreview()
	require.Greater(t, len(m.themes), 1)

	// Navigate to index 1
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(ThemePreviewModel)

	expectedTheme := m.themes[1]
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, expectedTheme, m.selected)
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestThemePreviewQuitWithQ(t *testing.T) {
	m := NewThemePreview()

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, "", m.selected, "selected should remain empty on quit")
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "q should return tea.Quit")
}

func TestThemePreviewQuitWithEsc(t *testing.T) {
	m := NewThemePreview()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, "", m.selected, "selected should remain empty on Esc")
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "Esc should return tea.Quit")
}

func TestThemePreviewWindowSizeMsg(t *testing.T) {
	m := NewThemePreview()
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
	assert.Nil(t, cmd)
}

func TestThemePreviewViewEmptyWhenWidthZero(t *testing.T) {
	m := NewThemePreview()
	// No WindowSizeMsg sent — width stays 0

	view := m.View()
	assert.Equal(t, "", view.Content, "View should return empty string when width is 0")
}

func TestResolveThemeColors(t *testing.T) {
	tc := themes.Colors{
		Primary:  "#ff0000",
		Warning:  "#ffff00",
		Error:    "#ff00ff",
		Success:  "#00ff00",
		Muted:    "#888888",
		Fg:       "#ffffff",
		Bg:       "#000000",
		Border:   "#444444",
		CursorBg: "#333333",
		Modified: "#0000ff",
		Added:    "#00ffff",
		Deleted:  "#ff8800",
	}

	rc := resolveThemeColors(tc)

	assert.NotNil(t, rc.primary)
	assert.NotNil(t, rc.warning)
	assert.NotNil(t, rc.err)
	assert.NotNil(t, rc.success)
	assert.NotNil(t, rc.muted)
	assert.NotNil(t, rc.fg)
	assert.NotNil(t, rc.bg)
	assert.NotNil(t, rc.border)
	assert.NotNil(t, rc.cursorBg)
	assert.NotNil(t, rc.modified)
	assert.NotNil(t, rc.added)
	assert.NotNil(t, rc.deleted)
}

func TestResolveThemeColorsFromRealTheme(t *testing.T) {
	names := config.ThemeNames()
	require.NotEmpty(t, names)

	tc, ok := themes.Lookup(names[0])
	require.True(t, ok)

	rc := resolveThemeColors(tc)

	assert.NotNil(t, rc.primary)
	assert.NotNil(t, rc.fg)
	assert.NotNil(t, rc.bg)
}

func TestThemePreviewSelectedMethod(t *testing.T) {
	m := NewThemePreview()
	assert.Equal(t, "", m.Selected())

	// After Enter, Selected() should return the theme name
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, m.themes[0], m.Selected())
}
