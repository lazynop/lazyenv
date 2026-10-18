package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
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
		if q != "" && !m.matchesQuery(&m.File.Vars[i], q) {
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
// lowercased) query. A masked value is skipped, so the search never
// confirms text the panel hides.
func (m *VarListModel) matchesQuery(v *model.EnvVar, qLower string) bool {
	if strings.Contains(strings.ToLower(v.Key), qLower) {
		return true
	}
	if m.isMasked(v) {
		return false
	}
	return strings.Contains(strings.ToLower(v.Value), qLower)
}

// isMasked reports whether the panel hides v's value: v is a secret and
// secrets are not revealed.
func (m *VarListModel) isMasked(v *model.EnvVar) bool {
	return v.IsSecret && !m.ShowSecrets
}

// isOriginalMasked reports whether the panel hides v's original value: it or
// v's current value is a secret, and secrets are not revealed.
func (m *VarListModel) isOriginalMasked(v *model.EnvVar) bool {
	return (v.IsSecret || v.OriginalIsSecret) && !m.ShowSecrets
}

func (m *VarListModel) clampCursor() {
	if m.Cursor >= len(m.displayItems) {
		m.Cursor = max(0, len(m.displayItems)-1)
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	// Keep the scroll offset within the (possibly shrunken) list, and the
	// cursor row (now perhaps modified, with a peek line) inside the viewport.
	m.ensureCursorVisible()
}

// SetSize sizes the panel and scrolls the cursor back into it.
func (m *VarListModel) SetSize(width, height int) {
	m.Width, m.Height = width, height
	m.ensureCursorVisible()
}

// ensureCursorVisible scrolls so the cursor row, and its peek line when one
// is drawn, sit inside the viewport, and the viewport shows no blank rows
// while rows above it are scrolled off. The deleted vars, drawn below the
// last item, count as rows too: on the last item the list scrolls on to show
// as many of them as fit.
func (m *VarListModel) ensureCursorVisible() {
	if m.Height <= panelChromeHeight {
		return // not laid out yet
	}
	visible := m.visibleLines()
	rows := len(m.displayItems)
	if m.File != nil {
		rows += len(m.File.DeletedVars)
	}
	m.Offset = scrollOffset(m.Cursor, m.Offset, visible, rows)
	if m.Cursor == len(m.displayItems)-1 {
		m.Offset = max(m.Offset, min(m.Cursor, rows-visible))
	}
}

// MoveUp moves the cursor up.
func (m *VarListModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		m.ensureCursorVisible()
	}
}

// MoveDown moves the cursor down.
func (m *VarListModel) MoveDown() {
	if m.Cursor < len(m.displayItems)-1 {
		m.Cursor++
		m.ensureCursorVisible()
	}
}

// SetFocused gives or takes the panel's focus, scrolling so the cursor row and
// the peek line drawn under it while focused stay inside the viewport.
func (m *VarListModel) SetFocused(focused bool) {
	m.Focused = focused
	m.ensureCursorVisible()
}

// TogglePeek turns the peek line under the cursor on or off.
func (m *VarListModel) TogglePeek() {
	m.Peeking = !m.Peeking
	m.ensureCursorVisible()
}

// SetCursor positions the cursor at the given displayItems index with bounds
// checking and adjusts the scroll offset to keep it visible.
func (m *VarListModel) SetCursor(index int) {
	if len(m.displayItems) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.displayItems)-1))
	m.ensureCursorVisible()
}

// IndexAtRow returns the index in displayItems of the item drawn on row, the
// list's rows counted from the first one shown, or -1 for a row that draws no
// item. The peek line maps to the item at the cursor, which it belongs to.
func (m *VarListModel) IndexAtRow(row int) int {
	rows := m.drawnRows()
	if row < 0 || row >= len(rows) {
		return -1
	}
	return rows[row].item
}

// drawnRow is a row the list draws: the index in displayItems of the item it
// shows, and whether it is the peek line under that item.
type drawnRow struct {
	item int
	peek bool
}

// drawnRows returns the rows the list draws from Offset down: the items that
// fit (visibleLines), and the peek line under the cursor's when one is drawn.
// renderRows draws them and IndexAtRow maps clicks through them, so both follow
// one layout.
func (m *VarListModel) drawnRows() []drawnRow {
	var rows []drawnRow
	for i := m.Offset; i < min(m.Offset+m.visibleLines(), len(m.displayItems)); i++ {
		rows = append(rows, drawnRow{item: i})
		if i == m.Cursor && m.drawsPeekLine() {
			rows = append(rows, drawnRow{item: i, peek: true})
		}
	}
	return rows
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

// cursorAnchor remembers the user's logical cursor position before a
// recomputeDisplay so it can be restored even if the row index changes.
//
// At most one of hasHeader/hasVar is true at a time (since a row is either
// a header or a var). Both false means "no anchor" — restoreCursor is a
// no-op and the cursor stays wherever clampCursor leaves it.
type cursorAnchor struct {
	hasHeader    bool
	headerPrefix string
	hasVar       bool
	varIdx       int
}

func (m *VarListModel) captureCursor() cursorAnchor {
	if m.Cursor < 0 || m.Cursor >= len(m.displayItems) {
		return cursorAnchor{}
	}
	item := m.displayItems[m.Cursor]
	switch item.Kind {
	case displayItemHeader:
		gi := item.GroupIdx
		if gi >= 0 && gi < len(m.groups) {
			return cursorAnchor{hasHeader: true, headerPrefix: m.groups[gi].Prefix}
		}
	case displayItemVar:
		return cursorAnchor{hasVar: true, varIdx: item.VarIdx}
	}
	return cursorAnchor{}
}

// restoreCursor positions the cursor on the row that matches the anchor.
// If the original row was a header, look for a header with the same prefix.
// If it was a var, look for the var first; if hidden in a collapsed group,
// fall back to that group's header.
func (m *VarListModel) restoreCursor(a cursorAnchor) {
	if a.hasHeader {
		for i, item := range m.displayItems {
			if item.Kind == displayItemHeader && m.groups[item.GroupIdx].Prefix == a.headerPrefix {
				m.SetCursor(i)
				return
			}
		}
	}
	if a.hasVar {
		for i, item := range m.displayItems {
			if item.Kind == displayItemVar && item.VarIdx == a.varIdx {
				m.SetCursor(i)
				return
			}
		}
		for i, item := range m.displayItems {
			if item.Kind == displayItemHeader && slices.Contains(m.groups[item.GroupIdx].Vars, a.varIdx) {
				m.SetCursor(i)
				return
			}
		}
	}
}

func (m *VarListModel) ToggleSort() {
	a := m.captureCursor()
	m.SortAlpha = !m.SortAlpha
	m.recomputeDisplay()
	m.restoreCursor(a)
}

// SetShowSecrets sets secret masking and re-filters an active search, since
// a masked secret's value doesn't match. The cursor stays on the same var
// while it is still listed.
func (m *VarListModel) SetShowSecrets(show bool) {
	a := m.captureCursor()
	m.ShowSecrets = show
	m.recomputeDisplay()
	m.restoreCursor(a)
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
	a := m.captureCursor()
	m.Grouping = !m.Grouping
	m.recomputeDisplay()
	m.restoreCursor(a)
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
	return padRight(truncate(displayText(v.Value, m.isMasked(v)), maxValWidth), maxValWidth)
}

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
	if m.isMasked(v) {
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

// renderPeekLine renders the peek line under a modified or new variable.
func (m *VarListModel) renderPeekLine(v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	text := "↳ new variable"
	if !v.IsNew {
		text = "↳ was: " + displayText(v.OriginalValue, m.isOriginalMasked(v))
	}
	// The text sits in the value column, so it fits wherever the row does.
	return theme.MutedItem.Render(fmt.Sprintf("  %s  %s", padRight("", keyWidth), truncate(text, maxValWidth)))
}

// renderDeletedVars renders the deleted variables section in the panel rows
// the currentLines already drawn leave.
func (m *VarListModel) renderDeletedVars(keyWidth, maxValWidth int, currentLines int, theme Theme) []string {
	var lines []string
	for _, dv := range m.File.DeletedVars {
		if currentLines+len(lines) >= m.contentRows() {
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

// contentRows returns how many rows the panel has inside its chrome.
func (m *VarListModel) contentRows() int {
	return max(m.Height-panelChromeHeight, 1)
}

// visibleLines returns how many display items fit in the viewport, leaving a
// row for the peek line when one is drawn.
func (m *VarListModel) visibleLines() int {
	visible := m.contentRows()
	if m.drawsPeekLine() {
		visible = max(visible-1, 1)
	}
	return visible
}

// drawsPeekLine reports whether a peek line is drawn under the cursor row:
// peek is on, the panel is focused and the cursor is on a modified or new var.
func (m *VarListModel) drawsPeekLine() bool {
	if !m.Peeking || !m.Focused || m.Cursor < 0 || m.Cursor >= len(m.displayItems) {
		return false
	}
	item := m.displayItems[m.Cursor]
	if item.Kind != displayItemVar {
		return false
	}
	v := &m.File.Vars[item.VarIdx]
	return v.Modified || v.IsNew
}

// calcColumnWidths returns the key column width and max value width.
// Headers are excluded from key-width measurement.
func (m *VarListModel) calcColumnWidths() (int, int) {
	longestKey := 0
	for _, item := range m.displayItems {
		if item.Kind != displayItemVar {
			continue
		}
		longestKey = max(longestKey, len(m.File.Vars[item.VarIdx].Key))
	}
	return columnLimits{
		maxKey:   m.layout.VarListMaxKeyWidth,
		padding:  max(m.layout.VarListPadding, config.VarListMinPadding),
		minValue: m.layout.VarListMinValueWidth,
	}.split(longestKey, m.Width)
}

// renderRows renders all visible rows (vars + headers) plus peek lines.
func (m *VarListModel) renderRows(keyWidth, maxValWidth int, theme Theme) []string {
	var lines []string
	for _, row := range m.drawnRows() {
		item := m.displayItems[row.item]
		switch {
		case item.Kind == displayItemHeader:
			lines = append(lines, m.renderHeaderLine(row.item, m.groups[item.GroupIdx], theme))
		case row.peek:
			lines = append(lines, m.renderPeekLine(&m.File.Vars[item.VarIdx], keyWidth, maxValWidth, theme))
		default: // displayItemVar
			lines = append(lines, m.renderVarLine(row.item, &m.File.Vars[item.VarIdx], keyWidth, maxValWidth, theme))
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

	keyWidth, maxValWidth := m.calcColumnWidths()

	lines := m.renderRows(keyWidth, maxValWidth, theme)
	lines = append(lines, m.renderDeletedVars(keyWidth, maxValWidth, len(lines), theme)...)

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

// stripAnsi removes ANSI escape sequences from s.
func stripAnsi(s string) string { return ansi.Strip(s) }
