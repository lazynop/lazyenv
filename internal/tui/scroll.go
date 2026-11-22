package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func scrollOffset(cursor, offset, visible, total int) int {
	cursorInView := max(min(offset, cursor), cursor-visible+1)
	lastPageOffset := total - visible
	return max(min(cursorInView, lastPageOffset), 0)
}

func visibleIndex(row, offset, visible, total int) int {
	if row < 0 || row >= visible || offset+row >= total {
		return -1
	}
	return offset + row
}

type scrollCursor struct {
	Cursor int
	Offset int
}

func (s *scrollCursor) prev() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

func (s *scrollCursor) next(total int) {
	if s.Cursor < total-1 {
		s.Cursor++
	}
}

func (s *scrollCursor) pageMove(visible, total, pages int) {
	step := pages * visible
	s.Cursor += step
	s.Offset += step
	s.clamp(total)
}

func (s *scrollCursor) moveTo(index, total int) {
	if total > 0 {
		s.Cursor = index
		s.clamp(total)
	}
}

func (s *scrollCursor) clamp(total int) {
	s.Cursor = max(min(s.Cursor, total-1), 0)
}

func (s *scrollCursor) follow(visible, total int) {
	s.Offset = scrollOffset(s.Cursor, s.Offset, visible, total)
}

type navigable interface {
	MoveUp()
	MoveDown()
	PageUp()
	PageDown()
	MoveToFirst()
	MoveToLast()
}

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
