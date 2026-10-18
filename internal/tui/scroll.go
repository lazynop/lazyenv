package tui

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
