package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearMessageMsg{}
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
	return clearMessageAfter(a.config.Layout.MessageTimeout)
}

// flashError sets a transient error message with a longer timeout.
func (a *App) flashError(msg string) tea.Cmd {
	a.statusBar.SetMessage(msg)
	return clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
}
