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

func (a App) viewHelp() string {
	helpText := `
  lazyenv — TUI for managing .env files

  Navigation
    ↑/↓  j/k      Navigate items
    ←/→  h/l      Switch panels (files / variables)
    Enter          Select file

  Actions
    e              Edit variable value
    a              Add new variable
    d              Delete variable (with confirmation)
    y              Copy value to clipboard
    Y              Copy KEY=value to clipboard
    p              Peek original value (toggle)
    w              Save changes
    r              Reset file (discard changes)
    c              Compare two files (diff view)
    m              Completeness matrix (multi-file)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    Ctrl+S         Toggle secret masking

  File Indicators
    ●              Selected file
    *              Modified (unsaved changes)
    !              Not covered by .gitignore

  Variable Indicators
    +              Newly added variable
    *              Modified variable
    -              Deleted variable (until save)
    D              Duplicate key
    ○              Empty value
    …              Placeholder value

  General
    ?              Show/hide this help
    q / Ctrl+C     Quit
    Esc            Back / cancel
`

	style := lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Padding(1, 2).
		Foreground(a.theme.ColorFg)

	footer := a.theme.MutedItem.Render("\n  Press Esc or ? to close")

	return style.Render(helpText + footer)
}
