package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/config/themes"
)

func TestNewThemePreview(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())

	assert.NotEmpty(t, m.themes, "themes list should be populated")
	assert.NotEmpty(t, m.searchPaths, "searchPaths should be populated")
	assert.Equal(t, 0, m.cursor, "cursor should start at 0")
	assert.Equal(t, "", m.selected, "selected should start empty")

	// Themes list should match config.ThemeNames()
	expected := config.ThemeNames()
	assert.Equal(t, expected, m.themes)
}

func TestThemePreviewNavigateDown(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	require.Greater(t, len(m.themes), 1, "need at least 2 themes to test navigation")

	// "j" key moves cursor down
	updated, _ := m.Update(keyMsg("j"))
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 1, m.cursor)

	// "down" arrow also moves cursor down
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 2, m.cursor)
}

func TestThemePreviewNavigateUp(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	require.Greater(t, len(m.themes), 1, "need at least 2 themes to test navigation")

	// Move down first
	updated, _ := m.Update(keyMsg("j"))
	m = updated.(ThemePreviewModel)
	updated, _ = m.Update(keyMsg("j"))
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 2, m.cursor)

	// "k" key moves cursor up
	updated, _ = m.Update(keyMsg("k"))
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 1, m.cursor)

	// "up" arrow also moves cursor up
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor)
}

func TestThemePreviewCursorDoesNotGoBelowZero(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	assert.Equal(t, 0, m.cursor)

	// Pressing up at top should not go below 0
	updated, _ := m.Update(keyMsg("k"))
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")
}

func TestThemePreviewCursorDoesNotExceedMax(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	maxIdx := len(m.themes) - 1

	// Move to the end
	for range len(m.themes) + 5 {
		updated, _ := m.Update(keyMsg("j"))
		m = updated.(ThemePreviewModel)
	}
	assert.Equal(t, maxIdx, m.cursor, "cursor should not exceed last index")
}

func TestThemePreviewSelectWithEnter(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	expectedTheme := m.themes[0]

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, expectedTheme, m.selected, "selected should be set to current theme")
	require.NotNil(t, cmd, "cmd should not be nil after Enter")

	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "Enter should return tea.Quit")
}

func TestThemePreviewSelectWithEnterAfterNavigation(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	require.Greater(t, len(m.themes), 1)

	// Navigate to index 1
	updated, _ := m.Update(keyMsg("j"))
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
	m := NewThemePreview(config.DefaultConfig())

	updated, cmd := m.Update(keyMsg("q"))
	m = updated.(ThemePreviewModel)

	assert.Equal(t, "", m.selected, "selected should remain empty on quit")
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "q should return tea.Quit")
}

func TestThemePreviewQuitWithEsc(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, "", m.selected, "selected should remain empty on Esc")
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "Esc should return tea.Quit")
}

func TestThemePreviewWindowSizeMsg(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
	assert.Nil(t, cmd)
}

func TestThemePreviewViewEmptyWhenWidthZero(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
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
	m := NewThemePreview(config.DefaultConfig())
	assert.Equal(t, "", m.Selected())

	// After Enter, Selected() should return the theme name
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(ThemePreviewModel)
	assert.Equal(t, m.themes[0], m.Selected())
}

func TestThemePreviewMouseClick(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	require.Greater(t, len(m.themes), 5, "need at least 5 themes")

	// Send WindowSizeMsg first so width/height are set
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(ThemePreviewModel)

	// Click in left panel (X=5 < listWidth=26) at Y=6
	// index = Y - 2 + scrollOffset() = 6 - 2 + 0 = 4
	updated, _ = m.Update(tea.MouseClickMsg{X: 5, Y: 6, Button: tea.MouseLeft})
	m = updated.(ThemePreviewModel)

	assert.Equal(t, 4, m.cursor, "click at Y=6 with offset=0 should set cursor to 4")
}

// TestThemePreviewMouseClickOutsideTheDrawnRowsKeepsTheCursor scrolls the
// theme list to its middle, then clicks its title row, its bottom border and
// the status bar: they draw no theme, though themes sit just off screen on
// either side, so the cursor stays put.
func TestThemePreviewMouseClickOutsideTheDrawnRowsKeepsTheCursor(t *testing.T) {
	updated, _ := NewThemePreview(config.DefaultConfig()).Update(tea.WindowSizeMsg{Width: 80, Height: MinTerminalHeight})
	m := updated.(ThemePreviewModel)
	require.Greater(t, len(m.themes), 12, "need themes below the scrolled list")
	for range 10 {
		updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = updated.(ThemePreviewModel)
	}
	require.Positive(t, m.scrollOffset())

	for _, y := range []int{panelContentOffsetY - 1, panelContentOffsetY + m.visibleLines(), MinTerminalHeight - 1} {
		updated, _ = m.Update(tea.MouseClickMsg{X: 5, Y: y, Button: tea.MouseLeft})
		assert.Equal(t, 10, updated.(ThemePreviewModel).cursor, "a click at Y=%d draws no theme", y)
	}
}

// TestThemePreviewMouseWheel scrolls the theme list with the wheel: each tick
// moves the cursor by mouse-scroll-lines themes, and 0 turns the wheel off.
func TestThemePreviewMouseWheel(t *testing.T) {
	for _, lines := range []int{0, 1, 3} {
		cfg := config.DefaultConfig()
		cfg.Layout.MouseScrollLines = lines
		updated, _ := NewThemePreview(cfg).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m := updated.(ThemePreviewModel)
		require.Greater(t, len(m.themes), 3, "need at least 4 themes")

		updated, _ = m.Update(tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelDown})
		m = updated.(ThemePreviewModel)
		assert.Equal(t, lines, m.cursor, "a wheel tick down with mouse-scroll-lines = %d", lines)

		updated, _ = m.Update(tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelUp})
		assert.Equal(t, 0, updated.(ThemePreviewModel).cursor, "a wheel tick up with mouse-scroll-lines = %d", lines)
	}
}

func TestThemePreviewScrollOffset(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())

	// Height 24 leaves visibleLines = 24 - statusBarHeight - panelChromeHeight = 20
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(ThemePreviewModel)

	// cursor=0 < visibleLines=20 → offset should be 0
	m.cursor = 0
	assert.Equal(t, 0, m.scrollOffset(), "scrollOffset should be 0 when cursor < visibleLines")

	// cursor=5 < visibleLines=20 → offset should still be 0
	m.cursor = 5
	assert.Equal(t, 0, m.scrollOffset(), "scrollOffset should be 0 when cursor < visibleLines")

	// cursor=20 >= visibleLines=20 → offset = 20 - 20 + 1 = 1
	require.Greater(t, len(m.themes), 20, "need more than 20 themes for this test")
	m.cursor = 20
	assert.Equal(t, 1, m.scrollOffset(), "scrollOffset should be positive when cursor >= visibleLines")

	// cursor=25 >= visibleLines=20 → offset = 25 - 20 + 1 = 6
	require.Greater(t, len(m.themes), 25, "need more than 25 themes for this test")
	m.cursor = 25
	assert.Equal(t, 6, m.scrollOffset(), "scrollOffset should increase with cursor")
}

func TestThemePreviewFitsSmallTerminals(t *testing.T) {
	// From the minimum terminal size: below it the preview draws tooSmallView.
	// The panels fill the terminal in both directions, with no columns left
	// over on the right.
	for _, width := range []int{MinTerminalWidth, 80, 120} {
		for height := MinTerminalHeight; height <= 40; height++ {
			m := NewThemePreview(config.DefaultConfig())
			m.searchPaths = []string{".lazyenvrc", "/home/user/.config/lazyenv/config.toml", "/home/user/.lazyenvrc"}
			updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
			rows := strings.Split(stripAnsi(updated.(ThemePreviewModel).View().Content), "\n")

			if assert.Len(t, rows, height, "%dx%d: the preview should fill the terminal exactly", width, height) {
				assert.Contains(t, rows[height-2], "╯╰", "%dx%d: both panels should end above the status bar", width, height)
				for i, row := range rows[:height-1] {
					assert.Equal(t, width, lipgloss.Width(strings.TrimRight(row, " ")), "%dx%d: panel row %d should span the terminal width", width, height, i)
				}
			}
		}
	}
}

func TestThemePreviewInitReturnsNil(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	assert.Nil(t, m.Init(), "theme preview has no startup command")
}

func TestThemePreviewCurrentColors(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	assert.NotEqual(t, themes.Colors{}, m.currentColors(),
		"current theme must resolve to a non-zero palette")
}

// TestThemePreviewViewRendersAllPanels exercises the full View() body and every
// render* helper (list, showcase, config box, status bar), which the width==0
// early-return test never reaches.
func TestThemePreviewViewRendersAllPanels(t *testing.T) {
	m := NewThemePreview(config.DefaultConfig())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(ThemePreviewModel)

	content := m.View().Content
	require.NotEmpty(t, content)

	assert.Contains(t, content, "Themes (", "theme list panel title")
	assert.Contains(t, content, "primary", "showcase color rows")
	assert.Contains(t, content, "deleted", "showcase color rows")
	assert.Contains(t, content, "Config files", "config box")
	assert.Contains(t, content, "theme =", "config snippet")
	assert.Contains(t, content, "navigate", "status bar")
}

func TestThemePreviewViewRendersWithMouseDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.NoMouse = true
	m := NewThemePreview(cfg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(ThemePreviewModel)

	assert.NotEmpty(t, m.View().Content, "view should still render with mouse disabled")
}

// TestThemePreviewTooSmallTerminal sizes the preview below the minimum: it
// draws the notice that fits the terminal, ignores navigation, and quits on
// Ctrl+C.
func TestThemePreviewTooSmallTerminal(t *testing.T) {
	updated, _ := NewThemePreview(config.DefaultConfig()).Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m := updated.(ThemePreviewModel)

	view := m.View().Content
	assert.Contains(t, stripAnsi(view), "too small")
	assert.LessOrEqual(t, lipgloss.Width(view), 40, "the notice should fit the terminal")
	assert.LessOrEqual(t, lipgloss.Height(view), 8, "the notice should fit the terminal")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter should not select a theme")
	updated, _ = updated.Update(keyMsg("j"))
	assert.Equal(t, 0, updated.(ThemePreviewModel).cursor, "j should not move the cursor")

	_, cmd = updated.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}
