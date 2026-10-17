package tui

import "charm.land/lipgloss/v2"

func (a App) viewConfigError() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ColorError).
		Padding(1, 3).
		Align(lipgloss.Center)

	title := a.theme.GitWarning.Render("Configuration Error")

	msg := a.theme.NormalItem.Render(a.configError)

	hint := a.theme.MutedItem.Render("Press any key to exit")

	alert := box.Render(
		lipgloss.JoinVertical(lipgloss.Center, title, "", msg, "", hint),
	)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, alert)
}
