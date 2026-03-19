package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

// MatrixModel manages the completeness matrix view.
type MatrixModel struct {
	files     []*model.EnvFile
	entries   []model.MatrixEntry
	fileNames []string

	cursorRow int // selected key (row)
	cursorCol int // selected file (column)
	offsetRow int // vertical scroll offset
	offsetCol int // horizontal scroll offset

	sortMode model.SortMode

	layout  config.LayoutConfig
	secrets config.SecretsConfig

	Width  int
	Height int

	editing  bool
	editKey  string // key being added
	editFile int    // file index target
	editor   textinput.Model
	message  string // transient message
}

// NewMatrixModel creates a matrix model from the loaded files.
func NewMatrixModel(files []*model.EnvFile, layout config.LayoutConfig, secrets config.SecretsConfig) MatrixModel {
	entries, names := model.ComputeMatrix(files)
	ti := textinput.New()
	ti.CharLimit = 256
	return MatrixModel{
		files:     files,
		entries:   entries,
		fileNames: names,
		layout:    layout,
		secrets:   secrets,
		editor:    ti,
	}
}

func (m *MatrixModel) recompute() {
	m.entries, m.fileNames = model.ComputeMatrix(m.files)
	model.SortEntries(m.entries, m.sortMode)
	if m.cursorRow >= len(m.entries) {
		m.cursorRow = max(0, len(m.entries)-1)
	}
}

// MoveUp moves the cursor up.
func (m *MatrixModel) MoveUp() {
	if m.cursorRow > 0 {
		m.cursorRow--
	}
	m.ensureVisible()
}

// MoveDown moves the cursor down.
func (m *MatrixModel) MoveDown() {
	if m.cursorRow < len(m.entries)-1 {
		m.cursorRow++
	}
	m.ensureVisible()
}

// MoveLeft moves the column cursor left.
func (m *MatrixModel) MoveLeft() {
	if m.cursorCol > 0 {
		m.cursorCol--
	}
	m.ensureHorizontalVisible()
}

// MoveRight moves the column cursor right.
func (m *MatrixModel) MoveRight() {
	if m.cursorCol < len(m.fileNames)-1 {
		m.cursorCol++
	}
	m.ensureHorizontalVisible()
}

// SetCursor positions the cursor at the given row and column with bounds checking.
func (m *MatrixModel) SetCursor(row, col int) {
	if len(m.entries) > 0 {
		m.cursorRow = max(0, min(row, len(m.entries)-1))
	}
	if len(m.fileNames) > 0 {
		m.cursorCol = max(0, min(col, len(m.fileNames)-1))
	}
	m.ensureVisible()
	m.ensureHorizontalVisible()
}

func (m *MatrixModel) ensureVisible() {
	viewRows := max(1, m.Height-4) // header + footer + borders
	if m.cursorRow < m.offsetRow {
		m.offsetRow = m.cursorRow
	}
	if m.cursorRow >= m.offsetRow+viewRows {
		m.offsetRow = m.cursorRow - viewRows + 1
	}
}

func (m *MatrixModel) ensureHorizontalVisible() {
	visibleCols := m.visibleCols()
	if visibleCols < 1 {
		return
	}
	if m.cursorCol < m.offsetCol {
		m.offsetCol = m.cursorCol
	}
	if m.cursorCol >= m.offsetCol+visibleCols {
		m.offsetCol = m.cursorCol - visibleCols + 1
	}
}

func (m *MatrixModel) visibleCols() int {
	available := m.Width - m.layout.MatrixKeyWidth - 2
	if available < m.layout.MatrixColWidth {
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
	entry := m.entries[m.cursorRow]
	if entry.Present[m.cursorCol] {
		m.message = "Variable already exists"
		return nil
	}
	m.editing = true
	m.editKey = entry.Key
	m.editFile = m.cursorCol
	m.editor.SetValue("")
	m.editor.Placeholder = "value"
	return m.editor.Focus()
}

// ConfirmEdit adds the variable to the target file.
func (m *MatrixModel) ConfirmEdit() {
	if !m.editing {
		return
	}
	f := m.files[m.editFile]
	f.AddVar(m.editKey, m.editor.Value(), util.IsSecret(m.editKey, m.editor.Value(), m.secrets))
	m.editing = false
	m.recompute()
}

// CancelEdit cancels inline editing.
func (m *MatrixModel) CancelEdit() {
	m.editing = false
}

func (m *MatrixModel) renderHeaderRow(endCol int, theme Theme) string {
	var hdr strings.Builder
	hdr.WriteString(fmt.Sprintf("%-*s", m.layout.MatrixKeyWidth, "KEY"))
	for ci := m.offsetCol; ci < endCol; ci++ {
		name := m.fileNames[ci]
		if len(name) > m.layout.MatrixColWidth-2 {
			name = name[:m.layout.MatrixColWidth-3] + "…"
		}
		if ci == m.cursorCol {
			hdr.WriteString(theme.SelectedItem.Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, name)))
		} else {
			hdr.WriteString(theme.HelpKey.Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, name)))
		}
	}
	return hdr.String()
}

func (m *MatrixModel) renderBodyRow(ri, endCol int, checkMark, crossMark string, theme Theme) string {
	entry := m.entries[ri]

	// Key name
	keyStr := entry.Key
	if len(keyStr) > m.layout.MatrixKeyWidth-2 {
		keyStr = keyStr[:m.layout.MatrixKeyWidth-3] + "…"
	}
	keyFormatted := fmt.Sprintf("%-*s", m.layout.MatrixKeyWidth, keyStr)
	if ri == m.cursorRow {
		keyFormatted = theme.SelectedItem.Render(keyFormatted)
	} else {
		keyFormatted = theme.NormalItem.Render(keyFormatted)
	}

	var row strings.Builder
	row.WriteString(keyFormatted)
	for ci := m.offsetCol; ci < endCol; ci++ {
		var cell string
		if ri == m.cursorRow && ci == m.cursorCol {
			// Highlight active cell
			if entry.Present[ci] {
				cell = theme.SelectedItem.Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, " "+checkMark))
			} else {
				cell = theme.DiffRemoved.Underline(true).Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, " "+crossMark))
			}
		} else if entry.Present[ci] {
			cell = theme.DiffEqual.Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, " "+checkMark))
		} else {
			cell = theme.DiffRemoved.Render(fmt.Sprintf("%-*s", m.layout.MatrixColWidth, " "+crossMark))
		}
		row.WriteString(cell)
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
	endCol := min(m.offsetCol+visCols, len(m.fileNames))

	// Header row
	b.WriteString(m.renderHeaderRow(endCol, theme))
	b.WriteString("\n")

	// Separator
	b.WriteString(theme.MutedItem.Render(strings.Repeat("─", m.Width-2)))
	b.WriteString("\n")

	// Body rows
	viewRows := max(1, m.Height-4)
	endRow := min(m.offsetRow+viewRows, len(m.entries))

	checkMark := "✓"
	crossMark := "✗"

	for ri := m.offsetRow; ri < endRow; ri++ {
		b.WriteString(m.renderBodyRow(ri, endCol, checkMark, crossMark, theme))
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
	stats := theme.MutedItem.Render(
		fmt.Sprintf("  %d keys  %d/%d cells missing  sort:%s",
			len(m.entries), missing, total, m.sortLabel()))

	content := b.String()

	// Pad to fill height
	lines := strings.Count(content, "\n")
	for lines < m.Height-3 {
		content += "\n"
		lines++
	}

	style := lipgloss.NewStyle().
		Width(m.Width).
		Padding(0, 1)

	return style.Render(content + stats)
}

func (m *MatrixModel) sortLabel() string {
	if m.sortMode == model.SortCompleteness {
		return "completeness"
	}
	return "alpha"
}
