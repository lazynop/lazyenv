package tui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

const (
	presentGlyph = "✓"
	missingGlyph = "✗"
)

// MatrixModel manages the completeness matrix view.
type MatrixModel struct {
	files   []*model.EnvFile
	entries []model.MatrixEntry

	rows scrollCursor
	cols scrollCursor

	sortMode model.SortMode

	layout config.LayoutConfig

	Width  int
	Height int

	edit   MatrixCell
	editor textinput.Model
}

// NewMatrixModel creates a matrix model from the loaded files.
func NewMatrixModel(files []*model.EnvFile, layout config.LayoutConfig) MatrixModel {
	entries := model.ComputeMatrix(files)
	ti := newUncappedValueInput()
	return MatrixModel{
		files:   files,
		entries: entries,
		layout:  layout,
		editor:  ti,
	}
}

func (m *MatrixModel) recompute(key string) {
	m.entries = model.ComputeMatrix(m.files)
	m.sortRows(key)
}

func (m *MatrixModel) sortRows(key string) {
	model.SortEntries(m.entries, m.sortMode)
	if i := slices.IndexFunc(m.entries, func(e model.MatrixEntry) bool { return e.Key == key }); i >= 0 {
		m.rows.Cursor = i
	}
	m.rows.clamp(len(m.entries))
	m.ensureVisible()
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
	m.cols.next(len(m.files))
	m.ensureHorizontalVisible()
}

// SetCursor positions the cursor at the given row and column with bounds checking.
func (m *MatrixModel) SetCursor(row, col int) {
	m.rows.moveTo(row, len(m.entries))
	m.cols.moveTo(col, len(m.files))
	m.ensureVisible()
	m.ensureHorizontalVisible()
}

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
	if x -= matrixSidePaddingWidth + m.layout.MatrixKeyWidth; x >= 0 {
		if c := visibleIndex(x/m.layout.MatrixColWidth, m.cols.Offset, m.visibleCols(), len(m.files)); c >= 0 {
			col = c
		}
	}
	return row, col
}

// MatrixCell is a cell of the matrix: the key of its row, the file of its
// column, and whether the file holds the key.
type MatrixCell struct {
	Key     string
	File    *model.EnvFile
	Present bool
}

// CursorCell returns the cell under the cursor, or false when the matrix has
// no keys.
func (m *MatrixModel) CursorCell() (MatrixCell, bool) {
	if len(m.entries) == 0 {
		return MatrixCell{}, false
	}
	entry := m.entries[m.rows.Cursor]
	col := m.cols.Cursor
	return MatrixCell{Key: entry.Key, File: m.files[col], Present: entry.Present[col]}, true
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
	m.cols.follow(visibleCols, len(m.files))
}

func (m *MatrixModel) counterWidth() int {
	return len(rowCounter(len(m.files), len(m.files)))
}

func rowCounter(present, total int) string {
	return fmt.Sprintf(" %*d/%d", len(strconv.Itoa(total)), present, total)
}

func (m *MatrixModel) visibleCols() int {
	available := m.Width - matrixPaddingWidth - m.layout.MatrixKeyWidth - m.counterWidth()
	noLayout := m.layout.MatrixColWidth <= 0
	if noLayout || available < m.layout.MatrixColWidth {
		return 1
	}
	return min(available/m.layout.MatrixColWidth, len(m.files))
}

// ToggleSort toggles between alphabetical and completeness sorting, the cursor
// staying on its key (sortRows).
func (m *MatrixModel) ToggleSort() {
	cell, _ := m.CursorCell()
	if m.sortMode == model.SortAlpha {
		m.sortMode = model.SortCompleteness
	} else {
		m.sortMode = model.SortAlpha
	}
	m.sortRows(cell.Key)
}

// StartEdit begins inline editing for the current cell (if missing).
func (m *MatrixModel) StartEdit() tea.Cmd {
	cell, ok := m.CursorCell()
	if !ok || cell.Present {
		return nil
	}
	m.edit = cell
	m.editor.SetValue("")
	m.editor.Placeholder = "value"
	return m.editor.Focus()
}

// EditBar returns the label and the text input of the bar that adds a value.
func (m *MatrixModel) EditBar() (string, *textinput.Model) {
	return fmt.Sprintf("Add %s to %s: ", m.edit.Key, m.edit.File.Name), &m.editor
}

// ConfirmEdit adds the variable to the target file and returns that file, or
// nil when no edit is open. The cursor stays on the key it added.
func (m *MatrixModel) ConfirmEdit() *model.EnvFile {
	f, key := m.edit.File, m.edit.Key
	if f == nil {
		return nil
	}
	f.AddVar(key, m.editor.Value())
	m.edit = MatrixCell{}
	m.recompute(key)
	return f
}

// DeleteAtCursor deletes the key under the cursor from the file under the
// cursor, every occurrence of it (see EnvFile.DeleteKey).
// Returns the deleted key and the file it was deleted from, or a nil file if
// nothing was deleted. The cursor stays on the key while it keeps a row.
func (m *MatrixModel) DeleteAtCursor() (string, *model.EnvFile) {
	cell, ok := m.CursorCell()
	if !ok || !cell.Present {
		return "", nil
	}
	cell.File.DeleteKey(cell.Key)
	m.recompute(cell.Key)
	return cell.Key, cell.File
}

// CancelEdit cancels inline editing.
func (m *MatrixModel) CancelEdit() {
	m.edit = MatrixCell{}
}

func (m *MatrixModel) renderHeaderRow(endCol int, theme Theme) string {
	var hdr strings.Builder
	hdr.WriteString(padRight("KEY", m.layout.MatrixKeyWidth))
	for ci := m.cols.Offset; ci < endCol; ci++ {
		name := fitName(m.files[ci].Name, m.layout.MatrixColWidth-2)
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

	present := 0
	for _, p := range entry.Present {
		if p {
			present++
		}
	}
	total := len(entry.Present)
	counter := rowCounter(present, total)
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
	endCol := min(m.cols.Offset+visCols, len(m.files))

	b.WriteString(m.renderHeaderRow(endCol, theme))
	b.WriteString("\n")

	b.WriteString(theme.MutedItem.Render(strings.Repeat("─", max(m.Width-matrixPaddingWidth, 0))))
	b.WriteString("\n")

	visible := m.visibleLines()
	endRow := min(m.rows.Offset+visible, len(m.entries))

	for ri := m.rows.Offset; ri < endRow; ri++ {
		b.WriteString(m.renderBodyRow(ri, endCol, theme))
		b.WriteString("\n")
	}

	missing := 0
	total := len(m.entries) * len(m.files)
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

	content += strings.Repeat("\n", max(0, matrixContentOffsetY+visible-strings.Count(content, "\n")))

	style := lipgloss.NewStyle().
		Width(m.Width).
		Padding(0, matrixSidePaddingWidth)

	return style.Render(content + stats)
}

func (m *MatrixModel) sortLabel() string {
	if m.sortMode == model.SortCompleteness {
		return "completeness"
	}
	return "alpha"
}
