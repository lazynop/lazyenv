package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

// VarListModel manages the variable list panel.
type VarListModel struct {
	File        *model.EnvFile
	Cursor      int
	Offset      int // scroll offset
	Height      int
	Width       int
	Focused     bool
	ShowSecrets bool
	SortAlpha   bool
	SearchQuery string
	Peeking     bool

	layout config.LayoutConfig

	// Filtered/sorted indices into File.Vars
	displayIndices []int
}

// NewVarListModel creates a new variable list model.
func NewVarListModel(layout config.LayoutConfig) VarListModel {
	return VarListModel{layout: layout}
}

// SetFile sets the active file and recomputes display indices.
func (m *VarListModel) SetFile(f *model.EnvFile) {
	m.File = f
	m.Cursor = 0
	m.Offset = 0
	m.Peeking = false
	m.recomputeIndices()
}

// recomputeIndices builds the list of displayed variable indices.
func (m *VarListModel) recomputeIndices() {
	m.displayIndices = nil
	if m.File == nil {
		return
	}

	for i := range m.File.Vars {
		if m.SearchQuery != "" {
			v := &m.File.Vars[i]
			keyMatch := strings.Contains(strings.ToLower(v.Key), strings.ToLower(m.SearchQuery))
			valMatch := strings.Contains(strings.ToLower(v.Value), strings.ToLower(m.SearchQuery))
			if !keyMatch && !valMatch {
				continue
			}
		}
		m.displayIndices = append(m.displayIndices, i)
	}

	if m.SortAlpha {
		sort.Slice(m.displayIndices, func(a, b int) bool {
			return m.File.Vars[m.displayIndices[a]].Key < m.File.Vars[m.displayIndices[b]].Key
		})
	}

	if m.Cursor >= len(m.displayIndices) {
		m.Cursor = max(0, len(m.displayIndices)-1)
	}
}

// MoveUp moves the cursor up.
func (m *VarListModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		if m.Cursor < m.Offset {
			m.Offset = m.Cursor
		}
	}
}

// MoveDown moves the cursor down.
func (m *VarListModel) MoveDown() {
	if m.Cursor < len(m.displayIndices)-1 {
		m.Cursor++
		visible := m.Height - 4
		if visible > 0 && m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
}

// SetCursor positions the cursor at the given index with bounds checking.
func (m *VarListModel) SetCursor(index int) {
	if len(m.displayIndices) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.displayIndices)-1))
	visible := m.Height - 4
	if visible > 0 {
		if m.Cursor < m.Offset {
			m.Offset = m.Cursor
		}
		if m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
}

// DisplayCount returns the number of displayed items.
func (m *VarListModel) DisplayCount() int {
	return len(m.displayIndices)
}

// SelectedVar returns the currently selected variable, or nil.
func (m *VarListModel) SelectedVar() *model.EnvVar {
	if m.File == nil || len(m.displayIndices) == 0 {
		return nil
	}
	if m.Cursor < 0 || m.Cursor >= len(m.displayIndices) {
		return nil
	}
	idx := m.displayIndices[m.Cursor]
	return &m.File.Vars[idx]
}

// SelectedVarIndex returns the real index of the selected var in File.Vars.
func (m *VarListModel) SelectedVarIndex() int {
	if m.Cursor < 0 || m.Cursor >= len(m.displayIndices) {
		return -1
	}
	return m.displayIndices[m.Cursor]
}

// ToggleSort toggles alphabetical sorting.
func (m *VarListModel) ToggleSort() {
	m.SortAlpha = !m.SortAlpha
	m.recomputeIndices()
}

// SetSearch sets the search query and refilters.
func (m *VarListModel) SetSearch(query string) {
	m.SearchQuery = query
	m.recomputeIndices()
}

// Refresh recomputes indices after file changes.
func (m *VarListModel) Refresh() {
	m.recomputeIndices()
}

// renderWarningIndicators renders the modification and issue warning indicators for a variable.
func (m *VarListModel) renderWarningIndicators(v *model.EnvVar, theme Theme) string {
	mod := " "
	issue := " "

	if v.IsNew {
		mod = theme.AddedMarker.Render("+")
	} else if v.Modified {
		mod = theme.ModifiedMarker.Render("*")
	}

	if v.IsDuplicate {
		issue = theme.DuplicateWarn.Render("D")
	} else if v.IsEmpty {
		issue = theme.EmptyWarning.Render("○")
	} else if v.IsPlaceholder {
		issue = theme.PlaceholderWarn.Render("…")
	}

	return mod + issue
}

// formatValue returns the masked, flattened, truncated and padded display
// value for a variable, ready to be emitted into a single-row table cell.
func (m *VarListModel) formatValue(v *model.EnvVar, maxValWidth int) string {
	value := v.Value
	if v.IsSecret && !m.ShowSecrets {
		value = util.MaskValue(value)
	}
	return padRight(truncate(flattenValue(value), maxValWidth), maxValWidth)
}

// controlCharReplacer maps control chars to width-1 glyphs from the Unicode
// Arrows block, so values render on a single row without breaking the panel
// layout. It is zero-alloc for inputs that contain none of the targets.
var controlCharReplacer = strings.NewReplacer(
	"\n", "↵", // U+21B5
	"\t", "⇥", // U+21E5
	"\r", "↩", // U+21A9
)

func flattenValue(s string) string { return controlCharReplacer.Replace(s) }

// renderVarLine renders a single variable line including cursor highlighting, secret masking, and warnings.
func (m *VarListModel) renderVarLine(i int, v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	key := padRight(truncate(v.Key, keyWidth), keyWidth)
	value := m.formatValue(v, maxValWidth)
	warning := m.renderWarningIndicators(v, theme)

	if i == m.Cursor && m.Focused {
		// Cursor line: render as a single styled block (no ANSI nesting)
		return theme.CursorItem.Render(fmt.Sprintf("  %s  %s%s", key, value, stripAnsi(warning)))
	}

	// Normal line: style key and value segments individually
	valueStyle := theme.ValueStyle
	if v.IsSecret && !m.ShowSecrets {
		valueStyle = theme.SecretValue
	}
	return fmt.Sprintf("  %s  %s%s",
		theme.KeyStyle.Render(key),
		valueStyle.Render(value),
		warning)
}

// renderPeekLine renders the peek line for modified/new variables, or empty string if not applicable.
func (m *VarListModel) renderPeekLine(v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	if v.IsNew {
		return theme.MutedItem.Render(fmt.Sprintf("  %s  ↳ new variable", padRight("", keyWidth)))
	}
	if v.Modified {
		orig := truncate(v.OriginalValue, maxValWidth-4)
		return theme.MutedItem.Render(fmt.Sprintf("  %s  ↳ was: %s", padRight("", keyWidth), orig))
	}
	return ""
}

// renderDeletedVars renders the deleted variables section.
func (m *VarListModel) renderDeletedVars(visible, keyWidth, maxValWidth int, currentLines int, theme Theme) []string {
	var lines []string
	for _, dv := range m.File.DeletedVars {
		if currentLines+len(lines) >= visible {
			break
		}
		key := padRight(truncate(dv.Key, keyWidth), keyWidth)
		value := m.formatValue(&dv, maxValWidth)
		marker := theme.DeletedMarker.Render("-")
		line := theme.MutedItem.Render(fmt.Sprintf("  %s  %s", key, value)) + marker + " "
		lines = append(lines, line)
	}
	return lines
}

// visibleLines returns how many variable lines can be displayed.
func (m *VarListModel) visibleLines() int {
	visible := max(m.Height-4, 1)
	if m.Peeking && m.Cursor >= 0 && m.Cursor < len(m.displayIndices) {
		v := &m.File.Vars[m.displayIndices[m.Cursor]]
		if v.Modified || v.IsNew {
			visible = max(visible-1, 1)
		}
	}
	return visible
}

// calcColumnWidths returns the key column width and max value width.
func (m *VarListModel) calcColumnWidths() (int, int) {
	keyWidth := 0
	for _, idx := range m.displayIndices {
		kw := len(m.File.Vars[idx].Key)
		if kw > keyWidth {
			keyWidth = kw
		}
	}
	if keyWidth > m.layout.VarListMaxKeyWidth {
		keyWidth = m.layout.VarListMaxKeyWidth
	}
	maxValWidth := max(
		m.Width-keyWidth-m.layout.VarListPadding, m.layout.VarListMinValueWidth)
	return keyWidth, maxValWidth
}

// renderVarLines renders all visible variable lines including peek lines.
func (m *VarListModel) renderVarLines(visible, keyWidth, maxValWidth int, theme Theme) []string {
	var lines []string
	end := min(m.Offset+visible, len(m.displayIndices))

	for i := m.Offset; i < end; i++ {
		idx := m.displayIndices[i]
		v := &m.File.Vars[idx]

		lines = append(lines, m.renderVarLine(i, v, keyWidth, maxValWidth, theme))

		if m.Peeking && i == m.Cursor && m.Focused {
			if peekLine := m.renderPeekLine(v, keyWidth, maxValWidth, theme); peekLine != "" {
				lines = append(lines, peekLine)
			}
		}
	}
	return lines
}

// viewTitle returns the panel title string.
func (m *VarListModel) viewTitle(theme Theme) string {
	if m.File == nil {
		return theme.PanelTitle.Render("Variables")
	}
	total := len(m.File.Vars)
	displayed := len(m.displayIndices)
	count := fmt.Sprintf("%d", total)
	if m.SearchQuery != "" && displayed != total {
		count = fmt.Sprintf("%d/%d", displayed, total)
	}
	return theme.PanelTitle.Render(m.File.Name) + " " + theme.MutedItem.Render(fmt.Sprintf("(%s vars)", count))
}

// View renders the variable list panel.
func (m *VarListModel) View(theme Theme) string {
	title := m.viewTitle(theme)

	if m.File == nil {
		content := theme.MutedItem.Render("  Select a file")
		return m.renderPanel(title, content, theme)
	}

	if len(m.displayIndices) == 0 {
		msg := "  No variables"
		if m.SearchQuery != "" {
			msg = fmt.Sprintf("  No matches for %q", m.SearchQuery)
		}
		return m.renderPanel(title, theme.MutedItem.Render(msg), theme)
	}

	visible := m.visibleLines()
	keyWidth, maxValWidth := m.calcColumnWidths()

	lines := m.renderVarLines(visible, keyWidth, maxValWidth, theme)
	lines = append(lines, m.renderDeletedVars(visible, keyWidth, maxValWidth, len(lines), theme)...)

	return m.renderPanel(title, strings.Join(lines, "\n"), theme)
}

func (m *VarListModel) renderPanel(title, content string, theme Theme) string {
	style := theme.VarPanel.
		Width(m.Width).
		Height(m.Height)

	if m.Focused {
		style = style.BorderForeground(theme.ColorPrimary)
	}

	inner := fmt.Sprintf("%s\n%s", title, content)
	return style.Render(inner)
}

// stripAnsi is a simple helper to remove ANSI escape codes for width calculation.
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
