package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func terminalTooSmall(width, height int) bool {
	return width < MinTerminalWidth || height < MinTerminalHeight
}

func tooSmallView(width, height int, title, muted lipgloss.Style) string {
	notice := lipgloss.JoinVertical(lipgloss.Center,
		title.Render("Terminal too small"),
		muted.Render(fmt.Sprintf("%dx%d, needs %dx%d", width, height, MinTerminalWidth, MinTerminalHeight)),
		muted.Render("Ctrl+C to quit"),
	)
	placed := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, notice)
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(placed)
}

func tooSmallInput(width, height int, msg tea.Msg) (handled bool, cmd tea.Cmd) {
	if awaitingWindowSize(width) || !terminalTooSmall(width, height) {
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

func awaitingWindowSize(width int) bool { return width == 0 }
