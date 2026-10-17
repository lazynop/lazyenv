package tui

// scrollOffset returns the first row a viewport of visible rows shows over
// total rows: offset moved as little as needed to keep cursor in view, then
// lowered so no blank rows show below while rows above are scrolled off.
func scrollOffset(cursor, offset, visible, total int) int {
	offset = max(min(offset, cursor), cursor-visible+1)
	return max(min(offset, total-visible), 0)
}
