package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func clearMessageAfter(d time.Duration, gen int) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearMessageMsg{gen: gen}
	})
}

func (a *App) readOnlyFlash() tea.Cmd {
	if !a.config.ReadOnly {
		return nil
	}
	return a.flashWarning("Read-only mode — editing disabled")
}

func (a *App) flashMessage(msg string) tea.Cmd {
	a.statusBar.SetMessage(msg)
	return clearMessageAfter(a.config.Layout.MessageTimeout, a.statusBar.msgGen)
}

func (a *App) flashWarning(msg string) tea.Cmd {
	a.statusBar.SetWarning(msg)
	return clearMessageAfter(a.config.Layout.MessageTimeout, a.statusBar.msgGen)
}

func (a *App) flashError(msg string) tea.Cmd {
	a.statusBar.SetAlert(msg)
	return clearMessageAfter(a.config.Layout.ErrorMessageTimeout, a.statusBar.msgGen)
}
