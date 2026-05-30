package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (a *App) toggleSortFlash() tea.Cmd {
	a.varList.ToggleSort()
	if a.varList.SortAlpha {
		return a.flashMessage("Sorted alphabetically")
	}
	return a.flashMessage("Sorted by position")
}

func (a *App) toggleGroupingFlash() tea.Cmd {
	n := a.varList.ToggleGrouping()
	if a.varList.Grouping {
		return a.flashMessage(fmt.Sprintf("Grouping enabled (%d groups)", n))
	}
	return a.flashMessage("Grouping disabled")
}
