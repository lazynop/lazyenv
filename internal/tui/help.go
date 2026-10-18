package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// helpTitle is the first row of the help screen.
const helpTitle = "lazyenv — TUI for managing .env files"

// helpWheelNote names the mouse wheel on the help's scroll row.
const helpWheelNote = " (or the mouse wheel)"

// helpSection is one titled group of rows on the help screen.
type helpSection struct {
	title string
	rows  []helpRow
}

// helpRow is one row of a helpSection: the keys it names and what they do.
type helpRow struct {
	keys string
	desc string
}

// alternatives is the row of bindings whose keys all do desc, joined as
// alternatives: "q / Ctrl+C".
func alternatives(desc string, bindings ...key.Binding) helpRow {
	var labels []string
	for _, b := range bindings {
		for _, k := range b.Keys() {
			labels = append(labels, KeyLabel(k))
		}
	}
	return helpRow{strings.Join(labels, " / "), desc}
}

// pair is the row of two bindings that do the two halves of desc (up and
// down, left and right file), their keys paired by position: "↑/↓  k/j".
func pair(desc string, first, second key.Binding) helpRow {
	return helpRow{strings.Join(pairColumns(first, second), "  "), desc}
}

// pairColumns are the keys of first and second paired by position, one
// column for each: "↑/↓", "k/j".
func pairColumns(first, second key.Binding) []string {
	var columns []string
	for i := range max(len(first.Keys()), len(second.Keys())) {
		var labels []string
		for _, b := range []key.Binding{first, second} {
			if i < len(b.Keys()) {
				labels = append(labels, KeyLabel(b.Keys()[i]))
			}
		}
		columns = append(columns, strings.Join(labels, "/"))
	}
	return columns
}

// shortKeys is the short form of bindings that the status-bar hints, the
// prompts and the help's footer name: the first key of each, joined by "/":
// "q", "←/→".
func shortKeys(bindings ...key.Binding) string {
	labels := make([]string, len(bindings))
	for i, b := range bindings {
		labels[i] = KeyLabel(b.Keys()[0])
	}
	return strings.Join(labels, "/")
}

// yesNo is the "(y/n)" that ends a yes/no prompt, naming the short keys of
// keys.Confirm and keys.Deny.
func yesNo(keys KeyMap) string {
	return "(" + shortKeys(keys.Confirm, keys.Deny) + ")"
}

// glyph is the row of an indicator, a glyph the panels draw rather than a key.
func glyph(label, desc string) helpRow {
	return helpRow{label, desc}
}

// helpContents returns the sections of the help screen for keys, in the
// order it shows them. Each screen's section sits next to the handler of its
// keys, except promptsHelp, whose keys several handlers match; wheel says
// whether the mouse wheel scrolls the help.
func helpContents(keys KeyMap, wheel bool) []helpSection {
	return []helpSection{
		filesPanelHelp(keys),
		varsPanelHelp(keys),
		bothPanelsHelp(keys),
		compareSelectHelp(keys),
		compareHelp(keys),
		matrixHelp(keys),
		promptsHelp(keys),
		{"File Indicators", []helpRow{
			glyph(selectedFileGlyph, "Selected file"),
			glyph(modifiedFileGlyph, "Modified (unsaved changes)"),
			glyph(gitWarningGlyph, "Not covered by .gitignore"),
		}},
		{"Variable Indicators", []helpRow{
			glyph(newVarGlyph, "Newly added variable"),
			glyph(modifiedVarGlyph, "Modified variable"),
			glyph(deletedVarGlyph, "Deleted variable (until save)"),
			glyph(duplicateKeyGlyph, "Duplicate key"),
			glyph(emptyValueGlyph, "Empty value"),
			glyph(placeholderGlyph, "Placeholder value"),
			glyph(collapsedGroupGlyph, "Collapsed group"),
			glyph(expandedGroupGlyph, "Expanded group"),
		}},
		// Each value glyph is what flattenValue draws for the character.
		{"Value Indicators", []helpRow{
			glyph(flattenValue("\n"), "Line break"),
			glyph(flattenValue("\t"), "Tab"),
			glyph(flattenValue("\r"), "Carriage return"),
			glyph(flattenValue("\x00")+" … "+flattenValue("\x1f"), "Other control character ("+flattenValue("\x1b")+" for Escape)"),
			glyph(flattenValue("\x7f"), "Delete character"),
			glyph(flattenValue("\x80"), "Invalid byte or C1 control character"),
		}},
		{"Compare Indicators", []helpRow{
			glyph(diffEqualGlyph, "Same value in both files"),
			glyph(diffChangedGlyph, "Different values"),
			glyph(diffAddedGlyph, "Only in the left file"),
			glyph(diffRemovedGlyph, "Only in the right file"),
			glyph(modifiedFileGlyph, "File modified (after its name)"),
		}},
		{"Matrix Indicators", []helpRow{
			glyph(presentGlyph, "The file has the variable"),
			glyph(missingGlyph, "The file lacks the variable"),
		}},
		helpScreenHelp(keys, wheel),
	}
}

// promptsHelp is the help section of the input bars and yes/no prompts,
// whose keys the editing, search, confirm and reorder handlers match.
func promptsHelp(keys KeyMap) helpSection {
	return helpSection{"Prompts", []helpRow{
		alternatives("Confirm a text input or a search", keys.Enter),
		alternatives("Confirm a yes/no prompt", keys.Confirm),
		alternatives("Decline a yes/no prompt", keys.Deny),
		pair("Reorder alphabetical / grouped (after O)", keys.ReorderAlpha, keys.ReorderGrouped),
		alternatives("Cancel (in a search, also clear the filter)", keys.Escape),
	}}
}

// helpText lays sections out as the help screen's text, one row per line:
// the title, then each section's title and its rows, the descriptions
// aligned three columns past the widest key column.
func helpText(sections []helpSection) string {
	width := 0
	for _, section := range sections {
		for _, row := range section.rows {
			width = max(width, lipgloss.Width(row.keys))
		}
	}
	lines := []string{"  " + helpTitle}
	for _, section := range sections {
		lines = append(lines, "", "  "+section.title)
		for _, row := range section.rows {
			lines = append(lines, "    "+padRight(row.keys, width+3)+row.desc)
		}
	}
	return strings.Join(lines, "\n")
}

// HelpViewModel manages the help screen (?): the help text, scrolled when it
// is taller than the terminal.
type HelpViewModel struct {
	text   string // the help text, one row per line
	scroll string // the scroll keys the footer names: "↑/↓ k/j PgUp/PgDn"
	close  string // the close keys the footer names: "Esc ? q"
	offset int
	width  int
	height int
	lines  []string // text wrapped to width, one row per line
	rows   []int    // for each of lines, the row of text it wraps
}

// NewHelpViewModel creates the help screen for keys; wheel says whether the
// mouse wheel scrolls it, so the help names the wheel only then.
func NewHelpViewModel(keys KeyMap, wheel bool) HelpViewModel {
	return HelpViewModel{
		text:   helpText(helpContents(keys, wheel)),
		scroll: strings.Join(slices.Concat(pairColumns(keys.Up, keys.Down), pairColumns(keys.PageUp, keys.PageDown)), " "),
		close:  strings.Join([]string{shortKeys(keys.Escape), shortKeys(keys.Help), shortKeys(keys.Quit)}, " "),
	}
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
		return wrapRows(row, width)
	}
	lines := wrapRows(row[start:], width-indent)
	lines[0] = row[:start] + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return lines
}

// visibleLines returns how many rows of the help text fit on the screen.
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

// View renders the rows of the help text that fit, then the footer, which names
// the scroll keys only when the text does not fit.
func (m *HelpViewModel) View(theme Theme) string {
	visible := m.visibleLines()
	end := min(m.offset+visible, len(m.lines))
	text := theme.NormalItem.Render(strings.Join(m.lines[m.offset:end], "\n"))

	hint := "  Press " + m.close + " to close"
	if len(m.lines) > visible {
		hint = "  Press " + m.scroll + " to scroll, " + m.close + " to close"
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
