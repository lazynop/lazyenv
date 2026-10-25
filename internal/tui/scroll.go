package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// scrollOffset returns the first row a viewport of visible rows shows over
// total rows: offset moved as little as needed to keep cursor in view, then
// lowered so no blank rows show below while rows above are scrolled off.
func scrollOffset(cursor, offset, visible, total int) int {
	offset = max(min(offset, cursor), cursor-visible+1)
	return max(min(offset, total-visible), 0)
}

// visibleIndex returns the index of the entry a viewport of visible rows
// starting at offset draws on row, counted from its first row, or -1 for a row
// that draws no entry: above or below the viewport, or past the last of total
// entries. Mouse clicks map through it, so a click on chrome never selects an
// entry scrolled off screen.
func visibleIndex(row, offset, visible, total int) int {
	if row < 0 || row >= visible || offset+row >= total {
		return -1
	}
	return offset + row
}

// pageMove returns the cursor and offset of a viewport of visible rows over
// total positions after a page move, down for pages 1 and up for -1. Both
// move by the visible rows, so the cursor keeps its row on screen and the
// view skips no row; the cursor stops at the first or last position, and the
// view brings the offset back within its rows (scrollOffset) once it passes
// an end.
func pageMove(cursor, offset, visible, total, pages int) (newCursor, newOffset int) {
	step := pages * visible
	return max(min(cursor+step, total-1), 0), offset + step
}

// navigable is a view the navigation keys move through, a row at a time, a
// page at a time, or to either end: a cursor over a list's positions, which
// the view scrolls to follow.
type navigable interface {
	MoveUp()
	MoveDown()
	PageUp()
	PageDown()
	MoveToFirst()
	MoveToLast()
}

// navigate moves view as the navigation key msg asks (Up, Down, PageUp,
// PageDown, Home or End) and reports whether msg is one of them.
func (k KeyMap) navigate(msg tea.KeyPressMsg, view navigable) bool {
	switch {
	case key.Matches(msg, k.Up):
		view.MoveUp()
	case key.Matches(msg, k.Down):
		view.MoveDown()
	case key.Matches(msg, k.PageUp):
		view.PageUp()
	case key.Matches(msg, k.PageDown):
		view.PageDown()
	case key.Matches(msg, k.Home):
		view.MoveToFirst()
	case key.Matches(msg, k.End):
		view.MoveToLast()
	default:
		return false
	}
	return true
}
