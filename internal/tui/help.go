package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// helpText is the help screen's text, one row per line.
const helpText = `  lazyenv — TUI for managing .env files

  Navigation
    ↑/↓  j/k      Navigate items
    ←/→  h/l      Switch panels (files / variables)
    Enter          Select file

  Actions
    e              Edit variable value
    E              Rename variable key
    a              Add new variable
    d              Delete variable (with confirmation)
    y              Copy value to clipboard
    Y              Copy KEY=value to clipboard
    p              Peek original value (toggle)
    N              Create new .env file
    C              Duplicate file with new name
    D              Delete file from disk
    R              Rename file
    T              Create template (keys only, no values)
    w              Save changes
    r              Reset file (discard changes)
    c              Compare two files (diff view)
    n/N            Jump to next/prev diff (compare mode)
    m              Completeness matrix (multi-file)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    O              Reorder variables on disk (alphabetical / grouped)
    g              Toggle variable grouping by prefix
    Space/Enter    Collapse/expand group (on header)
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
    q / Ctrl+C     Quit (in compare and matrix: back to main view)
    Esc            Back / cancel`

// HelpViewModel manages the help screen (?): helpText, scrolled when it is
// taller than the terminal.
type HelpViewModel struct {
	Offset int
	Width  int
	Height int
}

// lines returns helpText wrapped to the screen's width, one row per line.
func (m *HelpViewModel) lines() []string {
	wrapped := lipgloss.NewStyle().Width(max(m.Width-2*helpPaddingWidth, 1)).Render(helpText)
	return strings.Split(wrapped, "\n")
}

// visibleLines returns how many rows of helpText fit on the screen.
func (m *HelpViewModel) visibleLines() int {
	return max(m.Height-helpChromeHeight, 1)
}

// maxOffset returns the offset that shows the last row of helpText at the
// bottom of the screen.
func (m *HelpViewModel) maxOffset() int {
	return max(len(m.lines())-m.visibleLines(), 0)
}

// ScrollToTop shows the text from its first row.
func (m *HelpViewModel) ScrollToTop() {
	m.Offset = 0
}

// ScrollUp scrolls the text up one row.
func (m *HelpViewModel) ScrollUp() {
	m.Offset = max(m.Offset-1, 0)
}

// ScrollDown scrolls the text down one row.
func (m *HelpViewModel) ScrollDown() {
	m.Offset = min(m.Offset+1, m.maxOffset())
}

// ensureVisible pulls the offset back after a resize leaves blank rows
// below the text.
func (m *HelpViewModel) ensureVisible() {
	m.Offset = min(m.Offset, m.maxOffset())
}

// View renders the rows of helpText that fit, then the footer, which names
// the scroll keys only when the text does not fit.
func (m *HelpViewModel) View(theme Theme) string {
	lines := m.lines()
	visible := m.visibleLines()
	end := min(m.Offset+visible, len(lines))
	text := lipgloss.NewStyle().Foreground(theme.ColorFg).Render(strings.Join(lines[m.Offset:end], "\n"))

	hint := "  Press Esc or ? to close"
	if len(lines) > visible {
		hint = "  Press ↑/↓ to scroll, Esc or ? to close"
	}
	footer := theme.MutedItem.Render(hint)

	return lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Padding(helpPaddingHeight, helpPaddingWidth).
		Render(text + "\n\n" + footer)
}
