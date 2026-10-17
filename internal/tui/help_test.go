package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openHelp sizes a new app to the terminal and opens the help screen.
func openHelp(width, height int) App {
	app := newTestApp(nil)
	updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return press(updated.(App), "?")
}

// scrollPastTheEnd is more down presses than the help has rows to scroll at
// any size the layout tests render, the 60x10 minimum included.
const scrollPastTheEnd = 80

// helpScreen is the help screen as plain text.
func helpScreen(app App) string {
	return stripAnsi(app.View().Content)
}

func TestViewHelp(t *testing.T) {
	help := helpScreen(openHelp(100, 40))
	assert.Contains(t, help, "lazyenv")
	assert.Contains(t, help, "Navigation")
	assert.Contains(t, help, "Actions")
	assert.Contains(t, help, "Esc, ? or q to close")
}

func TestHelpScrollsDownToItsLastRow(t *testing.T) {
	app := openHelp(80, 24)
	assert.Contains(t, helpScreen(app), "lazyenv — TUI for managing .env files")
	assert.NotContains(t, helpScreen(app), "Back / cancel")

	for range scrollPastTheEnd {
		app = press(app, "down")
	}

	screen := helpScreen(app)
	assert.Equal(t, 24, lipgloss.Height(screen))
	assert.Contains(t, screen, "Back / cancel", "the last row is reachable")
	assert.NotContains(t, screen, "lazyenv — TUI for managing .env files")
	assert.Contains(t, screen, "Esc, ? or q to close", "the footer stays on screen")
}

func TestHelpScrollStopsAtBothEnds(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)
	assert.Equal(t, top, helpScreen(press(app, "up")), "up at the top does nothing")

	for range scrollPastTheEnd {
		app = press(app, "j")
	}
	bottom := helpScreen(app)
	assert.Equal(t, bottom, helpScreen(press(app, "j")), "down at the bottom does nothing")
	assert.NotEqual(t, bottom, helpScreen(press(app, "k")), "up from the bottom scrolls back")
}

func TestHelpReopensAtTheTop(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)

	app = press(app, "down", "down", "down", "esc", "?")

	assert.Equal(t, top, helpScreen(app))
}

func TestHelpScrollsWithTheMouseWheel(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.NotEqual(t, top, helpScreen(app), "the wheel scrolls down")

	updated, _ = app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelUp})
	app = updated.(App)
	assert.Equal(t, top, helpScreen(app), "and back up")
}

func TestHelpHintsAtScrollingOnlyWhenItOverflows(t *testing.T) {
	assert.Contains(t, helpScreen(openHelp(80, 24)), "↑/↓ or j/k to scroll")

	tall := helpScreen(openHelp(120, 80))
	assert.NotContains(t, tall, "to scroll")
	assert.Contains(t, tall, "lazyenv — TUI", "a terminal tall enough shows the whole help")
	assert.Contains(t, tall, "Back / cancel", "a terminal tall enough shows the whole help")
}

func TestHelpGrowingTheTerminalScrollsBackToFit(t *testing.T) {
	app := openHelp(80, 24)
	for range scrollPastTheEnd {
		app = press(app, "down")
	}

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 80})

	assert.Contains(t, helpScreen(updated.(App)), "lazyenv — TUI for managing .env files",
		"a terminal tall enough for the whole help shows it from the top")
}

func TestHelpIgnoresClicks(t *testing.T) {
	app := press(openHelp(80, 24), "down", "down")
	before := helpScreen(app)

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: 5, Button: tea.MouseLeft})
	app = updated.(App)

	assert.Equal(t, ModeHelp, app.mode, "a click keeps the help open")
	assert.Equal(t, before, helpScreen(app), "and where it was scrolled to")
}

func TestHelpFooterNamesEveryCloseAndScrollKeyOnOneRowAtTheMinimumWidth(t *testing.T) {
	overflowing := helpScreen(openHelp(MinTerminalWidth, MinTerminalHeight))
	assert.Contains(t, overflowing, "Press ↑/↓ or j/k to scroll, Esc, ? or q to close")

	fitting := helpScreen(openHelp(MinTerminalWidth, 80))
	assert.Contains(t, fitting, "Back / cancel", "a terminal tall enough shows the whole help")
	assert.Contains(t, fitting, "Press Esc, ? or q to close")
}

// continuationIndent finds the screen row holding description and returns the
// column the description starts at and the column the text of the row below
// starts at.
func continuationIndent(t *testing.T, screen, description string) (descCol, nextCol int) {
	t.Helper()
	rows := strings.Split(screen, "\n")
	for i, row := range rows {
		col := strings.Index(row, description)
		if col < 0 {
			continue
		}
		require.Less(t, i+1, len(rows), "a row follows %q", description)
		next := rows[i+1]
		return lipgloss.Width(row[:col]), lipgloss.Width(next) - lipgloss.Width(strings.TrimLeft(next, " "))
	}
	require.Failf(t, "row not on screen", "%q", description)
	return 0, 0
}

func TestHelpIndentsWrappedRowsUnderTheirDescription(t *testing.T) {
	tall := helpScreen(openHelp(MinTerminalWidth, 80))
	for _, description := range []string{"Quit (in compare", "Reorder variables on disk"} {
		descCol, nextCol := continuationIndent(t, tall, description)
		assert.Equal(t, descCol, nextCol, "the continuation of %q starts under its description", description)
	}

	app := openHelp(MinTerminalWidth, MinTerminalHeight)
	for range scrollPastTheEnd {
		app = press(app, "down")
	}
	screen := helpScreen(app)
	assert.Contains(t, screen, "Back / cancel", "scrolling counts the wrapped rows")
	descCol, nextCol := continuationIndent(t, screen, "Quit (in compare")
	assert.Equal(t, descCol, nextCol, "the wrapped row scrolls as a whole")
}
