package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// helpText is the help screen's text, one row per line. Its helpWheelSlot
// becomes helpWheelNote while the wheel scrolls, and goes away otherwise.
const helpText = `  lazyenv — TUI for managing .env files

  Files panel
    Enter          Select file and switch to the variables panel
    N              Create new .env file
    C              Duplicate file with new name
    D              Delete file from disk (with confirmation)
    R              Rename file
    T              Create template (keys only, no values)

  Variables panel
    e              Edit variable value
    E              Rename variable key
    a              Add new variable
    d              Delete variable (with confirmation)
    y              Copy value to clipboard
    Y              Copy KEY=value to clipboard
    p              Peek original value (toggle)
    Enter/Space    Collapse/expand group (on header)

  Both panels
    ↑/↓  j/k       Navigate items
    ←/→  h/l       Switch panels (files / variables)
    w              Save changes
    r              Reset file (discard changes)
    c              Compare two files (diff view)
    m              Completeness matrix (multi-file)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    O              Reorder variables on disk (alphabetical / grouped)
    g              Toggle variable grouping by prefix
    Ctrl+S         Toggle secret masking
    ?              Show this help
    q / Ctrl+C     Quit

  Compare: pick the second file
    ↑/↓  j/k       Choose the file
    Enter          Compare with it
    Esc            Cancel
    q / Ctrl+C     Cancel

  Compare view
    ↑/↓  j/k       Navigate entries
    n/N            Jump to next/prev difference
    ←/→  h/l       Copy value to the left/right file; if the source file lacks the key, delete it there
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
    d              Delete variable from the file (with confirmation)
    o              Toggle sort (alphabetical / completeness)
    Esc            Back to main view
    q / Ctrl+C     Back to main view

  Prompts
    Enter          Confirm a text input or a search
    y              Confirm a yes/no prompt
    n              Decline a yes/no prompt
    a / g          Reorder alphabetical / grouped (after O)
    Esc            Cancel (in a search, also clear the filter)

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
    ▸              Collapsed group
    ▾              Expanded group

  Value Indicators
    ↵              Line break
    ⇥              Tab
    ↩              Carriage return
    ␀ … ␟          Other control character (␛ for Escape)
    ␡              Delete character
    �              Invalid byte or C1 control character

  Compare Indicators
    =              Same value in both files
    ≠              Different values
    +              Only in the left file
    -              Only in the right file
    *              File modified (after its name)

  Matrix Indicators
    ✓              The file has the variable
    ✗              The file lacks the variable

  This help
    Esc / ?        Close
    q / Ctrl+C     Close
    ↑/↓  j/k       Scroll{wheel}
    PgUp/PgDn      Scroll a page
    Home/End       Jump to the top/bottom`

// helpWheelSlot marks where helpWheelNote goes in helpText.
const helpWheelSlot = "{wheel}"

// helpWheelNote names the mouse wheel on the scroll row of helpText.
const helpWheelNote = " (or the mouse wheel)"

// HelpViewModel manages the help screen (?): helpText, scrolled when it is
// taller than the terminal.
type HelpViewModel struct {
	text   string // helpText as this session shows it
	offset int
	width  int
	height int
	lines  []string // text wrapped to width, one row per line
	rows   []int    // for each of lines, the row of text it wraps
}

// NewHelpViewModel creates the help screen; wheel says whether the mouse
// wheel scrolls it, so the help names the wheel only then.
func NewHelpViewModel(wheel bool) HelpViewModel {
	note := ""
	if wheel {
		note = helpWheelNote
	}
	return HelpViewModel{text: strings.Replace(helpText, helpWheelSlot, note, 1)}
}

// SetSize lays the text out for a screen of the given size, pulling the
// offset back if the resize leaves blank rows below the text. App.Update
// calls it after every message, so the text is wrapped again only when the
// width changes, keeping on top the row of text that was there.
func (m *HelpViewModel) SetSize(width, height int) {
	if width != m.width {
		top := 0
		if m.offset < len(m.rows) {
			top = m.rows[m.offset]
		}
		m.width = width
		textWidth := max(width-2*helpPaddingWidth, 1)
		m.lines, m.rows = nil, nil
		for i, row := range strings.Split(m.text, "\n") {
			if i == top {
				m.offset = len(m.lines)
			}
			for _, line := range wrapHelpRow(row, textWidth) {
				m.lines = append(m.lines, line)
				m.rows = append(m.rows, i)
			}
		}
	}
	m.height = height
	m.scrollTo(m.offset)
}

// wrapHelpRow word-wraps one row of the help text to width with a hanging indent:
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

	hint := "  Press Esc ? q to close"
	if len(m.lines) > visible {
		hint = "  Press ↑/↓ j/k PgUp/PgDn to scroll, Esc ? q to close"
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
