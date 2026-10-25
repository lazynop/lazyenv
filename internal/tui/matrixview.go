package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// The matrix's cell marks, which the help lists under Matrix Indicators.
const (
	presentGlyph = "✓"
	missingGlyph = "✗"
)

// MatrixModel manages the completeness matrix view.
type MatrixModel struct {
	files     []*model.EnvFile
	entries   []model.MatrixEntry
	fileNames []string

	rows scrollCursor // over the keys (rows)
	cols scrollCursor // over the files (columns)

	sortMode model.SortMode

	layout config.LayoutConfig

	Width  int
	Height int

	editing  bool
	editKey  string // key being added
	editFile int    // file index target
	editor   textinput.Model
}

// NewMatrixModel creates a matrix model from the loaded files.
func NewMatrixModel(files []*model.EnvFile, layout config.LayoutConfig) MatrixModel {
	entries, names := model.ComputeMatrix(files)
	ti := textinput.New() // no CharLimit, like the value editor
	return MatrixModel{
		files:     files,
		entries:   entries,
		fileNames: names,
		layout:    layout,
		editor:    ti,
	}
}

func (m *MatrixModel) recompute() {
	m.entries, m.fileNames = model.ComputeMatrix(m.files)
	model.SortEntries(m.entries, m.sortMode)
	m.rows.clamp(len(m.entries))
}

// MoveUp moves the cursor up.
func (m *MatrixModel) MoveUp() {
	m.rows.prev()
	m.ensureVisible()
}

// MoveDown moves the cursor down.
func (m *MatrixModel) MoveDown() {
	m.rows.next(len(m.entries))
	m.ensureVisible()
}

// PageUp scrolls up a page, the row cursor moving with the view (pageMove).
func (m *MatrixModel) PageUp() {
	m.rows.pageMove(m.visibleLines(), len(m.entries), -1)
	m.ensureVisible()
}

// PageDown scrolls down a page, the row cursor moving with the view
// (pageMove).
func (m *MatrixModel) PageDown() {
	m.rows.pageMove(m.visibleLines(), len(m.entries), 1)
	m.ensureVisible()
}

// MoveToFirst moves the cursor to the first row.
func (m *MatrixModel) MoveToFirst() {
	m.SetCursor(0, m.cols.Cursor)
}

// MoveToLast moves the cursor to the last row.
func (m *MatrixModel) MoveToLast() {
	m.SetCursor(len(m.entries)-1, m.cols.Cursor)
}

// MoveLeft moves the column cursor left.
func (m *MatrixModel) MoveLeft() {
	m.cols.prev()
	m.ensureHorizontalVisible()
}

// MoveRight moves the column cursor right.
func (m *MatrixModel) MoveRight() {
	m.cols.next(len(m.fileNames))
	m.ensureHorizontalVisible()
}

// SetCursor positions the cursor at the given row and column with bounds checking.
func (m *MatrixModel) SetCursor(row, col int) {
	m.rows.moveTo(row, len(m.entries))
	m.cols.moveTo(col, len(m.fileNames))
	m.ensureVisible()
	m.ensureHorizontalVisible()
}

// visibleLines returns how many body rows the matrix shows.
func (m *MatrixModel) visibleLines() int {
	return max(1, m.Height-matrixChromeHeight)
}

// CellAt returns the row and column of the cell drawn at screen position x, y,
// the matrix being drawn from the top-left corner, or a row of -1 for a line
// that draws no key.
// A click on the key column or the row counter right of the last column keeps
// the cursor's column.
func (m *MatrixModel) CellAt(x, y int) (row, col int) {
	row = visibleIndex(y-matrixContentOffsetY, m.rows.Offset, m.visibleLines(), len(m.entries))
	col = m.cols.Cursor
	// The matrix is drawn after its left padding.
	if x -= matrixPaddingWidth/2 + m.layout.MatrixKeyWidth; x >= 0 {
		if c := visibleIndex(x/m.layout.MatrixColWidth, m.cols.Offset, m.visibleCols(), len(m.fileNames)); c >= 0 {
			col = c
		}
	}
	return row, col
}

// SetSize sizes the matrix and scrolls the cursor cell back into it.
func (m *MatrixModel) SetSize(width, height int) {
	m.Width, m.Height = width, height
	m.ensureVisible()
	m.ensureHorizontalVisible()
}

func (m *MatrixModel) ensureVisible() {
	m.rows.follow(m.visibleLines(), len(m.entries))
}

func (m *MatrixModel) ensureHorizontalVisible() {
	visibleCols := m.visibleCols()
	if visibleCols < 1 {
		return
	}
	m.cols.follow(visibleCols, len(m.fileNames))
}

func (m *MatrixModel) counterWidth() int {
	// " X/Y" where X and Y have the same digit count as len(fileNames)
	digits := len(fmt.Sprintf("%d", len(m.fileNames)))
	return 1 + digits + 1 + digits // space + digits + slash + digits
}

func (m *MatrixModel) visibleCols() int {
	available := m.Width - matrixPaddingWidth - m.layout.MatrixKeyWidth - m.counterWidth()
	if m.layout.MatrixColWidth <= 0 || available < m.layout.MatrixColWidth { // no layout: a zero MatrixModel
		return 1
	}
	return min(available/m.layout.MatrixColWidth, len(m.fileNames))
}

// ToggleSort toggles between alphabetical and completeness sorting.
func (m *MatrixModel) ToggleSort() {
	if m.sortMode == model.SortAlpha {
		m.sortMode = model.SortCompleteness
	} else {
		m.sortMode = model.SortAlpha
	}
	model.SortEntries(m.entries, m.sortMode)
}

// StartEdit begins inline editing for the current cell (if missing).
func (m *MatrixModel) StartEdit() tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	entry := m.entries[m.rows.Cursor]
	if entry.Present[m.cols.Cursor] {
		return nil
	}
	m.editing = true
	m.editKey = entry.Key
	m.editFile = m.cols.Cursor
	m.editor.SetValue("")
	m.editor.Placeholder = "value"
	return m.editor.Focus()
}

// EditBar returns the label and the text input of the bar that adds a value.
func (m *MatrixModel) EditBar() (string, *textinput.Model) {
	return fmt.Sprintf("  Add %s to %s: ", m.editKey, m.fileNames[m.editFile]), &m.editor
}

// ConfirmEdit adds the variable to the target file.
func (m *MatrixModel) ConfirmEdit() {
	if !m.editing {
		return
	}
	f := m.files[m.editFile]
	f.AddVar(m.editKey, m.editor.Value())
	m.editing = false
	m.recompute()
}

// DeleteAtCursor deletes the variable at the current cursor position.
// Returns the deleted key name and file name, or empty strings if nothing was deleted.
func (m *MatrixModel) DeleteAtCursor() (string, string) {
	if len(m.entries) == 0 {
		return "", ""
	}
	entry := m.entries[m.rows.Cursor]
	if !entry.Present[m.cols.Cursor] {
		return "", ""
	}
	f := m.files[m.cols.Cursor]
	if i := f.LastIndexOfKey(entry.Key); i >= 0 {
		f.DeleteVar(i)
	}
	fileName := m.fileNames[m.cols.Cursor]
	m.recompute()
	return entry.Key, fileName
}

// CancelEdit cancels inline editing.
func (m *MatrixModel) CancelEdit() {
	m.editing = false
}

func (m *MatrixModel) renderHeaderRow(endCol int, theme Theme) string {
	var hdr strings.Builder
	hdr.WriteString(padRight("KEY", m.layout.MatrixKeyWidth))
	for ci := m.cols.Offset; ci < endCol; ci++ {
		name := fitName(m.fileNames[ci], m.layout.MatrixColWidth-2)
		if ci == m.cols.Cursor {
			hdr.WriteString(theme.SelectedItem.Render(padRight(name, m.layout.MatrixColWidth)))
		} else {
			hdr.WriteString(theme.HelpKey.Render(padRight(name, m.layout.MatrixColWidth)))
		}
	}
	return hdr.String()
}

func (m *MatrixModel) renderBodyRow(ri, endCol int, theme Theme) string {
	entry := m.entries[ri]

	// Key name
	keyStr := truncate(entry.Key, m.layout.MatrixKeyWidth-2)
	keyFormatted := padRight(keyStr, m.layout.MatrixKeyWidth)
	if ri == m.rows.Cursor {
		keyFormatted = theme.SelectedItem.Render(keyFormatted)
	} else {
		keyFormatted = theme.NormalItem.Render(keyFormatted)
	}

	var row strings.Builder
	row.WriteString(keyFormatted)
	for ci := m.cols.Offset; ci < endCol; ci++ {
		var cell string
		if ri == m.rows.Cursor && ci == m.cols.Cursor {
			// Highlight active cell
			if entry.Present[ci] {
				cell = theme.SelectedItem.Render(padRight(" "+presentGlyph, m.layout.MatrixColWidth))
			} else {
				cell = theme.DiffRemoved.Bold(true).Render(padRight(" "+missingGlyph, m.layout.MatrixColWidth))
			}
		} else if entry.Present[ci] {
			cell = theme.DiffEqual.Render(padRight(" "+presentGlyph, m.layout.MatrixColWidth))
		} else {
			cell = theme.DiffRemoved.Render(padRight(" "+missingGlyph, m.layout.MatrixColWidth))
		}
		row.WriteString(cell)
	}

	// Row counter: present/total, right-aligned
	present := 0
	for _, p := range entry.Present {
		if p {
			present++
		}
	}
	total := len(entry.Present)
	totalWidth := len(fmt.Sprintf("%d", total))
	counter := fmt.Sprintf(" %*d/%d", totalWidth, present, total)
	if present == total {
		row.WriteString(theme.DiffEqual.Render(counter))
	} else {
		row.WriteString(theme.MutedItem.Render(counter))
	}

	return row.String()
}

// View renders the matrix.
func (m *MatrixModel) View(theme Theme) string {
	if len(m.entries) == 0 {
		style := lipgloss.NewStyle().
			Width(m.Width).
			Height(m.Height).
			Padding(1, 2).
			Foreground(theme.ColorFg)
		return style.Render("No variables to display")
	}

	var b strings.Builder

	visCols := m.visibleCols()
	endCol := min(m.cols.Offset+visCols, len(m.fileNames))

	// Header row
	b.WriteString(m.renderHeaderRow(endCol, theme))
	b.WriteString("\n")

	// Separator
	b.WriteString(theme.MutedItem.Render(strings.Repeat("─", max(m.Width-matrixPaddingWidth, 0))))
	b.WriteString("\n")

	// Body rows
	visible := m.visibleLines()
	endRow := min(m.rows.Offset+visible, len(m.entries))

	for ri := m.rows.Offset; ri < endRow; ri++ {
		b.WriteString(m.renderBodyRow(ri, endCol, theme))
		b.WriteString("\n")
	}

	// Stats footer
	missing := 0
	total := len(m.entries) * len(m.fileNames)
	for _, e := range m.entries {
		for _, p := range e.Present {
			if !p {
				missing++
			}
		}
	}
	stats := theme.MutedItem.Render(truncate(
		fmt.Sprintf("  %d keys  %d/%d cells missing  sort:%s",
			len(m.entries), missing, total, m.sortLabel()),
		m.Width-matrixPaddingWidth))

	content := b.String()

	// Pad the body to its full height, so the stats footer concatenated below
	// keeps its row however many keys there are.
	content += strings.Repeat("\n", max(0, matrixContentOffsetY+visible-strings.Count(content, "\n")))

	style := lipgloss.NewStyle().
		Width(m.Width).
		Padding(0, matrixPaddingWidth/2)

	return style.Render(content + stats)
}

func (m *MatrixModel) sortLabel() string {
	if m.sortMode == model.SortCompleteness {
		return "completeness"
	}
	return "alpha"
}
