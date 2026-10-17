package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

// openHelp sizes a new app to the terminal and opens the help screen.
func openHelp(width, height int) App {
	app := newTestApp(nil)
	updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return press(updated.(App), "?")
}

// scrollPastTheEnd is more down presses than the help has rows to scroll
// at 80x24.
const scrollPastTheEnd = 60

// helpScreen is the help screen as plain text.
func helpScreen(app App) string {
	return stripAnsi(app.View().Content)
}

func TestViewHelp(t *testing.T) {
	help := helpScreen(openHelp(100, 40))
	assert.Contains(t, help, "lazyenv")
	assert.Contains(t, help, "Navigation")
	assert.Contains(t, help, "Actions")
	assert.Contains(t, help, "Esc or ? to close")
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
	assert.Contains(t, screen, "Esc or ? to close", "the footer stays on screen")
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
	assert.Contains(t, helpScreen(openHelp(80, 24)), "↑/↓ to scroll")

	tall := helpScreen(openHelp(120, 80))
	assert.NotContains(t, tall, "↑/↓ to scroll")
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
