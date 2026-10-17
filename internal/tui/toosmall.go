package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// terminalTooSmall reports whether a width x height terminal is below the
// minimum the TUI lays its screens out in.
func terminalTooSmall(width, height int) bool {
	return width < MinTerminalWidth || height < MinTerminalHeight
}

// tooSmallView is the screen drawn in place of any other while the terminal
// is below the minimum: it names the size needed and how to quit, centered
// and cut to the terminal.
func tooSmallView(width, height int, title, muted lipgloss.Style) string {
	notice := lipgloss.JoinVertical(lipgloss.Center,
		title.Render("Terminal too small"),
		muted.Render(fmt.Sprintf("%dx%d, needs %dx%d", width, height, MinTerminalWidth, MinTerminalHeight)),
		muted.Render("Ctrl+C to quit"),
	)
	placed := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, notice)
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(placed)
}

// tooSmallInput handles msg in place of the model while a sized terminal
// (width 0 means no tea.WindowSizeMsg yet) is below the minimum: it reports
// whether msg is input, which it drops since it would act on what the user
// cannot see, and the command it runs instead: Ctrl+C quits, from any mode,
// so the user is never stuck.
func tooSmallInput(width, height int, msg tea.Msg) (handled bool, cmd tea.Cmd) {
	if width == 0 || !terminalTooSmall(width, height) {
		return false, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return true, tea.Quit
		}
		return true, nil
	case tea.MouseMsg:
		return true, nil
	}
	return false, nil
}
