package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	configErrorHint    = "Press any key to exit"
	configErrorCutHint = "Press any key to exit · see lazyenv --check-config"
)

func (a App) viewConfigError() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ColorError).
		Padding(1, 3).
		Align(lipgloss.Center)

	// The path and the errors wrap to the columns the box's frame leaves, so
	// a long one never pushes the box past the terminal.
	room := max(a.width-box.GetHorizontalFrameSize(), 1)
	var path []string
	if a.configPath != "" {
		path = strings.Split(ansi.Wrap(a.configPath, room, ""), "\n")
	}
	msg := strings.Split(ansi.Wrap(a.configError, room, ""), "\n")
	hint := configErrorHint

	// When the box is taller than the terminal it drops its vertical padding,
	// then the blank rows, then cuts the errors and points to --check-config,
	// which lists them all.
	blankRows := 2
	rowsBesideMsg := func() int { return box.GetVerticalFrameSize() + 1 + len(path) + blankRows + 1 }
	if rowsBesideMsg()+len(msg) > a.height {
		box = box.Padding(0, 3)
	}
	if rowsBesideMsg()+len(msg) > a.height {
		blankRows = 0
	}
	if fit := max(a.height-rowsBesideMsg(), 1); len(msg) > fit {
		msg = append(msg[:fit-1], ansi.Truncate(msg[fit-1], room-1, "")+"…")
		hint = configErrorCutHint
	}

	lines := []string{a.theme.GitWarning.Render("Configuration Error")}
	for _, p := range path {
		lines = append(lines, a.theme.MutedItem.Render(p))
	}
	if blankRows > 0 {
		lines = append(lines, "")
	}
	for _, m := range msg {
		lines = append(lines, a.theme.NormalItem.Render(m))
	}
	if blankRows > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, a.theme.MutedItem.Render(hint))

	alert := box.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, alert)
}
