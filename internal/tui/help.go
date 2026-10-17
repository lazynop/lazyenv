package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// helpText is the help screen's text, one row per line.
const helpText = `  lazyenv — TUI for managing .env files

  Main view
    ↑/↓  j/k       Navigate items
    ←/→  h/l       Switch panels (files / variables)
    Enter          Select file, or collapse/expand a group header
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
    m              Completeness matrix (multi-file)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    O              Reorder variables on disk (alphabetical / grouped)
    g              Toggle variable grouping by prefix
    Space          Collapse/expand group (on header)
    Ctrl+S         Toggle secret masking
    ?              Show this help
    q / Ctrl+C     Quit

  Compare view
    ↑/↓  j/k       Navigate entries
    n/N            Jump to next/prev difference
    ←/→  h/l       Copy value to the left/right file
    e/E            Edit value in the left/right file
    f              Toggle filter: differences only
    w              Save both files
    r              Reset both files
    Ctrl+S         Toggle secret masking
    Esc            Back to main view
    q / Ctrl+C     Back to main view

  Matrix
    ↑/↓  j/k       Move between variables
    ←/→  h/l       Move between files
    a              Add missing variable to the file
    d              Delete variable from the file
    o              Toggle sort (alphabetical / completeness)
    Esc            Back to main view
    q / Ctrl+C     Back to main view

  Prompts
    Enter          Confirm a text input or a search
    y              Confirm a yes/no prompt
    Esc            Cancel

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

  This help
    Esc / ?        Close
    q / Ctrl+C     Close
    ↑/↓  j/k       Scroll (or the mouse wheel)
    PgUp/PgDn      Scroll a page
    Home/End       Jump to the top/bottom`

// HelpViewModel manages the help screen (?): helpText, scrolled when it is
// taller than the terminal.
type HelpViewModel struct {
	offset int
	width  int
	height int
	lines  []string // helpText wrapped to width, one row per line
}

// SetSize lays the text out for a screen of the given size, wrapping it to
// the width and pulling the offset back if the resize leaves blank rows
// below the text. App.Update calls it after every message, so it does
// nothing while the size stays the same.
func (m *HelpViewModel) SetSize(width, height int) {
	if width == m.width && height == m.height {
		return
	}
	m.width = width
	m.height = height
	textWidth := max(width-2*helpPaddingWidth, 1)
	m.lines = nil
	for _, row := range strings.Split(helpText, "\n") {
		m.lines = append(m.lines, wrapHelpRow(row, textWidth)...)
	}
	m.scrollTo(m.offset)
}

// wrapHelpRow word-wraps one row of helpText to width with a hanging indent:
// the continuation lines start under the row's description, the text after
// its last run of two or more spaces (a heading has none and continues under
// its first word). A row that fits is returned unchanged, and one whose
// indent leaves no room is wrapped without it.
func wrapHelpRow(row string, width int) []string {
	if lipgloss.Width(row) <= width {
		return []string{row}
	}
	start := len(row) - len(strings.TrimLeft(row, " "))
	if gap := strings.LastIndex(row, "  "); gap+2 > start {
		start = gap + 2
	}
	indent := lipgloss.Width(row[:start])
	if indent >= width {
		return strings.Split(ansi.Wrap(row, width, ""), "\n")
	}
	lines := strings.Split(ansi.Wrap(row[start:], width-indent, ""), "\n")
	lines[0] = row[:start] + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return lines
}

// visibleLines returns how many rows of helpText fit on the screen.
func (m *HelpViewModel) visibleLines() int {
	return max(m.height-helpChromeHeight, 1)
}

// scrollTo shows the text from row offset, held between the first row and
// the offset that shows the last row at the bottom of the screen.
func (m *HelpViewModel) scrollTo(offset int) {
	m.offset = scrollOffset(offset, offset, m.visibleLines(), len(m.lines))
}

// ScrollToTop shows the text from its first row.
func (m *HelpViewModel) ScrollToTop() {
	m.scrollTo(0)
}

// ScrollToBottom shows the last row of the text at the bottom of the screen.
func (m *HelpViewModel) ScrollToBottom() {
	m.scrollTo(len(m.lines))
}

// PageUp scrolls the text up one screen.
func (m *HelpViewModel) PageUp() {
	m.scrollTo(m.offset - m.visibleLines())
}

// PageDown scrolls the text down one screen.
func (m *HelpViewModel) PageDown() {
	m.scrollTo(m.offset + m.visibleLines())
}

// ScrollUp scrolls the text up one row.
func (m *HelpViewModel) ScrollUp() {
	m.scrollTo(m.offset - 1)
}

// ScrollDown scrolls the text down one row.
func (m *HelpViewModel) ScrollDown() {
	m.scrollTo(m.offset + 1)
}

// View renders the rows of helpText that fit, then the footer, which names
// the scroll keys only when the text does not fit.
func (m *HelpViewModel) View(theme Theme) string {
	visible := m.visibleLines()
	end := min(m.offset+visible, len(m.lines))
	text := lipgloss.NewStyle().Foreground(theme.ColorFg).Render(strings.Join(m.lines[m.offset:end], "\n"))

	hint := "  Press Esc, ? or q to close"
	if len(m.lines) > visible {
		hint = "  Press ↑/↓ or j/k to scroll, Esc, ? or q to close"
	}
	footer := theme.MutedItem.Render(hint)

	// The newline that ends the text, then the gap rows. The outer style only
	// pads and sizes text already styled, so it must set no colors of its own
	// (CODING_STANDARDS.md § Pitfalls, nested ANSI).
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(helpPaddingHeight, helpPaddingWidth).
		Render(text + strings.Repeat("\n", helpFooterGapHeight+1) + footer)
}
