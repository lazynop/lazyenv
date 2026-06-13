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

// readOnlyFlash returns a flash command if read-only mode is active, or nil otherwise.
func (a *App) readOnlyFlash() tea.Cmd {
	if !a.config.ReadOnly {
		return nil
	}
	return a.flashMessage("Read-only mode — editing disabled")
}

// flashMessage sets a transient status bar message and returns the auto-clear cmd.
func (a *App) flashMessage(msg string) tea.Cmd {
	a.statusBar.SetMessage(msg)
	return clearMessageAfter(a.config.Layout.MessageTimeout, a.statusBar.msgGen)
}

// flashError sets a transient error message with a longer timeout.
func (a *App) flashError(msg string) tea.Cmd {
	a.statusBar.SetMessage(msg)
	return clearMessageAfter(a.config.Layout.ErrorMessageTimeout, a.statusBar.msgGen)
}
