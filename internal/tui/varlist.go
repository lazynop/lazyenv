package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

// ungroupedHeaderLabel is the label shown above the trailing bucket of
// variables with a unique prefix, no '_', or a leading '_'.
const ungroupedHeaderLabel = "UNGROUPED"

// displayKind distinguishes a variable row from a group header row.
type displayKind int

const (
	displayItemVar displayKind = iota
	displayItemHeader
)

// displayItem is a single row in the rendered var list — either a variable
// or a collapsible group header.
type displayItem struct {
	Kind     displayKind
	VarIdx   int // index into File.Vars; valid when Kind == displayItemVar
	GroupIdx int // index into groups;    valid when Kind == displayItemHeader
}

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
	Grouping    bool

	layout config.LayoutConfig

	// Computed view: groups + flattened display rows.
	groups       []model.VarGroup
	displayItems []displayItem
	collapsed    map[string]bool // group prefix → collapsed flag (preserved across toggles)
}

// NewVarListModel creates a new variable list model.
func NewVarListModel(layout config.LayoutConfig) VarListModel {
	return VarListModel{
		layout:    layout,
		collapsed: make(map[string]bool),
	}
}

// SetFile sets the active file and recomputes display rows.
func (m *VarListModel) SetFile(f *model.EnvFile) {
	m.File = f
	m.Cursor = 0
	m.Offset = 0
	m.Peeking = false
	m.recomputeDisplay()
}

// recomputeDisplay rebuilds groups and the flat displayItems list.
//
// Linear view (no headers) is used when grouping is off OR when a search
// query is active. Otherwise: emit one displayItemHeader per non-Ungrouped
// group followed by its (optionally collapsed) vars; the Ungrouped bucket
// is appended last with no header.
func (m *VarListModel) recomputeDisplay() {
	m.groups = nil
	m.displayItems = nil
	if m.File == nil {
		return
	}

	if !m.Grouping || m.SearchQuery != "" {
		m.buildLinearDisplay()
		m.clampCursor()
		return
	}

	m.groups = model.ComputeGroups(m.File.Vars)
	if m.SortAlpha {
		sort.SliceStable(m.groups, func(i, j int) bool {
			// Ungrouped pinned last, even when sorting alphabetically.
			if m.groups[i].IsUngrouped() != m.groups[j].IsUngrouped() {
				return !m.groups[i].IsUngrouped()
			}
			return m.groups[i].Prefix < m.groups[j].Prefix
		})
	}
	// Show the Ungrouped header only when a named group also exists, so the
	// single-Ungrouped case (no real groupings) keeps the linear-view feel.
	showUngroupedHeader := m.namedGroupCount() > 0
	for gi, g := range m.groups {
		if g.IsUngrouped() && !showUngroupedHeader {
			m.appendGroupVars(g)
			continue
		}
		m.displayItems = append(m.displayItems, displayItem{
			Kind:     displayItemHeader,
			GroupIdx: gi,
		})
		if m.collapsed[g.Prefix] {
			continue
		}
		m.appendGroupVars(g)
	}
	m.clampCursor()
}

func (m *VarListModel) buildLinearDisplay() {
	q := strings.ToLower(m.SearchQuery)
	indices := make([]int, 0, len(m.File.Vars))
	for i := range m.File.Vars {
		if q != "" && !matchesQuery(&m.File.Vars[i], q) {
			continue
		}
		indices = append(indices, i)
	}
	m.emitVarRows(indices)
}

func (m *VarListModel) appendGroupVars(g model.VarGroup) {
	m.emitVarRows(slices.Clone(g.Vars))
}

// emitVarRows appends one displayItemVar per index in the given slice,
// optionally sorting alphabetically by Key first. Mutates indices in place.
func (m *VarListModel) emitVarRows(indices []int) {
	if m.SortAlpha {
		sort.Slice(indices, func(a, b int) bool {
			return m.File.Vars[indices[a]].Key < m.File.Vars[indices[b]].Key
		})
	}
	for _, idx := range indices {
		m.displayItems = append(m.displayItems, displayItem{Kind: displayItemVar, VarIdx: idx})
	}
}

// matchesQuery reports whether v's key or value contains the (already
// lowercased) query.
func matchesQuery(v *model.EnvVar, qLower string) bool {
	return strings.Contains(strings.ToLower(v.Key), qLower) ||
		strings.Contains(strings.ToLower(v.Value), qLower)
}

func (m *VarListModel) clampCursor() {
	if m.Cursor >= len(m.displayItems) {
		m.Cursor = max(0, len(m.displayItems)-1)
	}
	if m.Cursor < 0 {
		m.Cursor = 0
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
	if m.Cursor < len(m.displayItems)-1 {
		m.Cursor++
		visible := m.Height - 4
		if visible > 0 && m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
}

// SetCursor positions the cursor at the given displayItems index with bounds
// checking and adjusts the scroll offset to keep it visible.
func (m *VarListModel) SetCursor(index int) {
	if len(m.displayItems) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.displayItems)-1))
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

// DisplayCount returns the number of displayed rows (vars + headers).
func (m *VarListModel) DisplayCount() int {
	return len(m.displayItems)
}

// SelectedVar returns the currently selected variable, or nil if the cursor
// is on a header (or out of range).
func (m *VarListModel) SelectedVar() *model.EnvVar {
	idx := m.SelectedVarIndex()
	if idx < 0 {
		return nil
	}
	return &m.File.Vars[idx]
}

// SelectedVarIndex returns the index into File.Vars for the currently
// selected variable, or -1 if the cursor is on a header (or out of range).
func (m *VarListModel) SelectedVarIndex() int {
	if m.File == nil || m.Cursor < 0 || m.Cursor >= len(m.displayItems) {
		return -1
	}
	item := m.displayItems[m.Cursor]
	if item.Kind != displayItemVar {
		return -1
	}
	return item.VarIdx
}

// IsHeaderAtCursor reports whether the row under the cursor is a group header.
func (m *VarListModel) IsHeaderAtCursor() bool {
	if m.Cursor < 0 || m.Cursor >= len(m.displayItems) {
		return false
	}
	return m.displayItems[m.Cursor].Kind == displayItemHeader
}

func (m *VarListModel) isCollapsed(prefix string) bool {
	return m.collapsed[prefix]
}

func (m *VarListModel) ToggleSort() {
	m.SortAlpha = !m.SortAlpha
	m.recomputeDisplay()
}

func (m *VarListModel) SetSearch(query string) {
	m.SearchQuery = query
	m.recomputeDisplay()
}

func (m *VarListModel) Refresh() {
	m.recomputeDisplay()
}

// ToggleGrouping flips Grouping and keeps the cursor on the previously-
// selected var (or its containing header if the var is now hidden).
// Returns the count of non-Ungrouped groups now visible.
func (m *VarListModel) ToggleGrouping() int {
	prevVarIdx := m.SelectedVarIndex()
	m.Grouping = !m.Grouping
	m.recomputeDisplay()

	if prevVarIdx >= 0 {
		for i, item := range m.displayItems {
			if item.Kind == displayItemVar && item.VarIdx == prevVarIdx {
				m.SetCursor(i)
				return m.namedGroupCount()
			}
		}
		// Var hidden in a collapsed group: fall back to that group's header.
		for i, item := range m.displayItems {
			if item.Kind != displayItemHeader {
				continue
			}
			if slices.Contains(m.groups[item.GroupIdx].Vars, prevVarIdx) {
				m.SetCursor(i)
				return m.namedGroupCount()
			}
		}
	}
	return m.namedGroupCount()
}

// ToggleCollapseAtCursor toggles the collapsed state of the group whose
// header is under the cursor. No-op (returns false) when the cursor is
// not on a header.
func (m *VarListModel) ToggleCollapseAtCursor() bool {
	if !m.IsHeaderAtCursor() {
		return false
	}
	gi := m.displayItems[m.Cursor].GroupIdx
	if gi < 0 || gi >= len(m.groups) {
		return false
	}
	prefix := m.groups[gi].Prefix
	m.collapsed[prefix] = !m.collapsed[prefix]
	m.recomputeDisplay()
	m.clampCursor()
	return true
}

func (m *VarListModel) namedGroupCount() int {
	n := 0
	for _, g := range m.groups {
		if !g.IsUngrouped() {
			n++
		}
	}
	return n
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
func (m *VarListModel) renderVarLine(rowIdx int, v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	key := padRight(truncate(v.Key, keyWidth), keyWidth)
	value := m.formatValue(v, maxValWidth)
	warning := m.renderWarningIndicators(v, theme)

	if rowIdx == m.Cursor && m.Focused {
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

// renderHeaderLine renders a group header row (▾ PREFIX (N) or ▸ PREFIX (N)).
//
// When the cursor is on this row and the panel is focused, the entire line is
// re-rendered as a single CursorItem block (ANSI-stripped first) to avoid
// nesting reset codes from inner Render() calls.
func (m *VarListModel) renderHeaderLine(rowIdx int, group model.VarGroup, theme Theme) string {
	arrow := "▾"
	if m.collapsed[group.Prefix] {
		arrow = "▸"
	}
	label := group.Prefix
	if group.IsUngrouped() {
		label = ungroupedHeaderLabel
	}
	countStr := fmt.Sprintf("(%d)", len(group.Vars))

	if rowIdx == m.Cursor && m.Focused {
		return theme.CursorItem.Render(fmt.Sprintf("  %s %s %s", arrow, label, countStr))
	}
	return fmt.Sprintf("  %s %s %s",
		theme.MutedItem.Render(arrow),
		theme.KeyStyle.Render(label),
		theme.MutedItem.Render(countStr))
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

// visibleLines returns how many display rows can be shown.
func (m *VarListModel) visibleLines() int {
	visible := max(m.Height-4, 1)
	if m.Peeking && m.Cursor >= 0 && m.Cursor < len(m.displayItems) {
		item := m.displayItems[m.Cursor]
		if item.Kind == displayItemVar {
			v := &m.File.Vars[item.VarIdx]
			if v.Modified || v.IsNew {
				visible = max(visible-1, 1)
			}
		}
	}
	return visible
}

// calcColumnWidths returns the key column width and max value width.
// Headers are excluded from key-width measurement.
func (m *VarListModel) calcColumnWidths() (int, int) {
	keyWidth := 0
	for _, item := range m.displayItems {
		if item.Kind != displayItemVar {
			continue
		}
		kw := len(m.File.Vars[item.VarIdx].Key)
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

// renderRows renders all visible rows (vars + headers) plus peek lines.
func (m *VarListModel) renderRows(visible, keyWidth, maxValWidth int, theme Theme) []string {
	var lines []string
	end := min(m.Offset+visible, len(m.displayItems))

	for i := m.Offset; i < end; i++ {
		item := m.displayItems[i]
		switch item.Kind {
		case displayItemHeader:
			lines = append(lines, m.renderHeaderLine(i, m.groups[item.GroupIdx], theme))
		default: // displayItemVar
			v := &m.File.Vars[item.VarIdx]
			lines = append(lines, m.renderVarLine(i, v, keyWidth, maxValWidth, theme))

			if m.Peeking && i == m.Cursor && m.Focused {
				if peekLine := m.renderPeekLine(v, keyWidth, maxValWidth, theme); peekLine != "" {
					lines = append(lines, peekLine)
				}
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
	displayed := 0
	for _, item := range m.displayItems {
		if item.Kind == displayItemVar {
			displayed++
		}
	}
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

	if len(m.displayItems) == 0 {
		msg := "  No variables"
		if m.SearchQuery != "" {
			msg = fmt.Sprintf("  No matches for %q", m.SearchQuery)
		}
		return m.renderPanel(title, theme.MutedItem.Render(msg), theme)
	}

	visible := m.visibleLines()
	keyWidth, maxValWidth := m.calcColumnWidths()

	lines := m.renderRows(visible, keyWidth, maxValWidth, theme)
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
