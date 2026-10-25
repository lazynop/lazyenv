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

// scrollCursor is a cursor over a list's positions and the scroll offset of
// the viewport that follows it. Its moves take from the list the visible rows
// and the number of positions they need, since both can change between moves;
// the list scrolls after each move with follow, adding any scrolling of its
// own.
type scrollCursor struct {
	Cursor int // the position under the cursor
	Offset int // the first row the viewport shows
}

// prev moves the cursor to the previous position, if there is one.
func (s *scrollCursor) prev() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

// next moves the cursor to the next of total positions, if there is one.
func (s *scrollCursor) next(total int) {
	if s.Cursor < total-1 {
		s.Cursor++
	}
}

// pageMove moves the cursor and the offset a page of visible rows over total
// positions, down for pages 1 and up for -1. Both move by the visible rows,
// so the cursor keeps its row on screen and the view skips no row; the cursor
// stops at the first or last position, and follow brings the offset back
// within the rows once it passes an end.
func (s *scrollCursor) pageMove(visible, total, pages int) {
	step := pages * visible
	s.Cursor += step
	s.Offset += step
	s.clamp(total)
}

// moveTo moves the cursor to index, kept within total positions; with none,
// the cursor stays where it is.
func (s *scrollCursor) moveTo(index, total int) {
	if total > 0 {
		s.Cursor = index
		s.clamp(total)
	}
}

// clamp brings the cursor back within total positions, as after the list
// shrinks; with none, onto the first.
func (s *scrollCursor) clamp(total int) {
	s.Cursor = max(min(s.Cursor, total-1), 0)
}

// follow scrolls a viewport of visible rows over total rows just enough to
// keep the cursor in it (scrollOffset).
func (s *scrollCursor) follow(visible, total int) {
	s.Offset = scrollOffset(s.Cursor, s.Offset, visible, total)
}

// navigable is a view the navigation keys move through, a row at a time, a
// page at a time, or to either end: a cursor over a list's positions, which
// the view scrolls to follow, or the help's text, which has no cursor.
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
