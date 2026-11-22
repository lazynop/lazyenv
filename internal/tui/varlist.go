package tui

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

const (
	newVarGlyph         = "+"
	modifiedVarGlyph    = "*"
	deletedVarGlyph     = "-"
	duplicateKeyGlyph   = "D"
	emptyValueGlyph     = "○"
	placeholderGlyph    = "…"
	collapsedGroupGlyph = "▸"
	expandedGroupGlyph  = "▾"
)

const ungroupedHeaderLabel = "UNGROUPED"

type displayKind int

const (
	displayItemVar displayKind = iota
	displayItemHeader
)

type displayItem struct {
	Kind     displayKind
	VarIdx   int
	GroupIdx int
}

// VarListModel manages the variable list panel.
type VarListModel struct {
	scrollCursor

	File        *model.EnvFile
	Height      int
	Width       int
	Focused     bool
	ShowSecrets bool
	SortAlpha   bool
	SearchQuery string
	Peeking     bool
	Grouping    bool

	layout config.LayoutConfig

	groups       []model.VarGroup
	displayItems []displayItem
	collapsed    map[string]bool
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
	m.scrollCursor = scrollCursor{}
	m.Peeking = false
	m.recomputeDisplay()
}

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
			if m.groups[i].IsUngrouped() != m.groups[j].IsUngrouped() {
				return !m.groups[i].IsUngrouped()
			}
			return m.groups[i].Prefix < m.groups[j].Prefix
		})
	}
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

func (m *VarListModel) matchesQuery(v *model.EnvVar, qLower string) bool {
	if strings.Contains(strings.ToLower(v.Key), qLower) {
		return true
	}
	if m.isMasked(v) {
		return false
	}
	return strings.Contains(strings.ToLower(v.Value), qLower)
}

func (m *VarListModel) isMasked(v *model.EnvVar) bool {
	return v.IsSecret && !m.ShowSecrets
}

func (m *VarListModel) isOriginalMasked(v *model.EnvVar) bool {
	return (v.IsSecret || v.OriginalIsSecret) && !m.ShowSecrets
}

func (m *VarListModel) clampCursor() {
	m.clamp(len(m.displayItems))
	m.ensureCursorVisible()
}

// SetSize sizes the panel and scrolls the cursor back into it.
func (m *VarListModel) SetSize(width, height int) {
	m.Width, m.Height = width, height
	m.ensureCursorVisible()
}

func (m *VarListModel) ensureCursorVisible() {
	if !panelLaidOut(m.Height) {
		return
	}
	visible := m.visibleLines()
	rows := len(m.displayItems)
	if m.File != nil {
		rows += len(m.File.DeletedVars)
	}
	m.follow(visible, rows)
	if m.Cursor == len(m.displayItems)-1 {
		m.Offset = max(m.Offset, min(m.Cursor, rows-visible))
	}
}

// MoveUp moves the cursor up.
func (m *VarListModel) MoveUp() {
	m.prev()
	m.ensureCursorVisible()
}

// MoveDown moves the cursor down.
func (m *VarListModel) MoveDown() {
	m.next(len(m.displayItems))
	m.ensureCursorVisible()
}

// PageUp scrolls up a page, the cursor moving with the view (pageMove).
func (m *VarListModel) PageUp() {
	m.pageMove(m.visibleLines(), len(m.displayItems), -1)
	m.ensureCursorVisible()
}

// PageDown scrolls down a page, the cursor moving with the view (pageMove).
func (m *VarListModel) PageDown() {
	m.pageMove(m.visibleLines(), len(m.displayItems), 1)
	m.ensureCursorVisible()
}

// MoveToFirst moves the cursor to the first item.
func (m *VarListModel) MoveToFirst() {
	m.SetCursor(0)
}

// MoveToLast moves the cursor to the last item.
func (m *VarListModel) MoveToLast() {
	m.SetCursor(len(m.displayItems) - 1)
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
	m.moveTo(index, len(m.displayItems))
	m.ensureCursorVisible()
}

// IndexAt returns the index in displayItems of the item drawn on screen row y,
// the panel being drawn from the top of the screen, or -1 for a row that draws
// no item: the note and the deleted vars, which no cursor reaches, draw none.
// The peek line maps to the item at the cursor, which it belongs to.
func (m *VarListModel) IndexAt(y int) int {
	rows := m.drawnRows()
	row := y - panelContentOffsetY
	if row < 0 || row >= len(rows) {
		return -1
	}
	if kind := rows[row].kind; kind == itemRow || kind == peekRow {
		return rows[row].index
	}
	return -1
}

type drawnRowKind int

const (
	itemRow drawnRowKind = iota
	peekRow
	noteRow
	deletedRow
)

type drawnRow struct {
	kind  drawnRowKind
	index int
}

func (m *VarListModel) drawnRows() []drawnRow {
	var rows []drawnRow
	if len(m.displayItems) == 0 {
		rows = append(rows, drawnRow{kind: noteRow})
	}
	for i := m.Offset; i < min(m.Offset+m.visibleLines(), len(m.displayItems)); i++ {
		rows = append(rows, drawnRow{itemRow, i})
		if i == m.Cursor && m.drawsPeekLine() {
			rows = append(rows, drawnRow{peekRow, i})
		}
	}
	if m.File != nil {
		for i := range min(len(m.File.DeletedVars), m.contentRows()-len(rows)) {
			rows = append(rows, drawnRow{deletedRow, i})
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

type anchorKind int

const (
	anchorNone anchorKind = iota
	anchorHeader
	anchorVar
)

type cursorAnchor struct {
	kind         anchorKind
	headerPrefix string
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
			return cursorAnchor{kind: anchorHeader, headerPrefix: m.groups[gi].Prefix}
		}
	case displayItemVar:
		return cursorAnchor{kind: anchorVar, varIdx: item.VarIdx}
	}
	return cursorAnchor{}
}

func (m *VarListModel) restoreCursor(a cursorAnchor) {
	if a.kind == anchorHeader {
		for i, item := range m.displayItems {
			if item.Kind == displayItemHeader && m.groups[item.GroupIdx].Prefix == a.headerPrefix {
				m.SetCursor(i)
				return
			}
		}
	}
	if a.kind == anchorVar {
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

func (m *VarListModel) renderWarningIndicators(v *model.EnvVar, theme Theme) string {
	mod := " "
	issue := " "

	if v.IsNew {
		mod = theme.AddedMarker.Render(newVarGlyph)
	} else if v.Modified {
		mod = theme.ModifiedMarker.Render(modifiedVarGlyph)
	}

	if v.IsDuplicate {
		issue = theme.DuplicateWarn.Render(duplicateKeyGlyph)
	} else if v.IsEmpty {
		issue = theme.EmptyWarning.Render(emptyValueGlyph)
	} else if v.IsPlaceholder {
		issue = theme.PlaceholderWarn.Render(placeholderGlyph)
	}

	return mod + issue
}

func (m *VarListModel) formatValue(v *model.EnvVar, maxValWidth int) string {
	return padRight(truncate(displayText(v.Value, m.isMasked(v)), maxValWidth), maxValWidth)
}

func (m *VarListModel) renderVarLine(rowIdx int, v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	key := padRight(truncate(v.Key, keyWidth), keyWidth)
	value := m.formatValue(v, maxValWidth)
	warning := m.renderWarningIndicators(v, theme)

	if rowIdx == m.Cursor && m.Focused {
		return theme.CursorItem.Render(fmt.Sprintf("  %s  %s%s", key, value, stripAnsi(warning)))
	}

	valueStyle := theme.ValueStyle
	if m.isMasked(v) {
		valueStyle = theme.SecretValue
	}
	return fmt.Sprintf("  %s  %s%s",
		theme.KeyStyle.Render(key),
		valueStyle.Render(value),
		warning)
}

func (m *VarListModel) renderHeaderLine(rowIdx int, group model.VarGroup, theme Theme) string {
	arrow := expandedGroupGlyph
	if m.collapsed[group.Prefix] {
		arrow = collapsedGroupGlyph
	}
	label := group.Prefix
	if group.IsUngrouped() {
		label = ungroupedHeaderLabel
	}
	countStr := fmt.Sprintf("(%d)", len(group.Vars))
	const format = "  %s %s %s"
	label = truncate(label, m.contentCols()-lipgloss.Width(fmt.Sprintf(format, arrow, "", countStr)))

	if rowIdx == m.Cursor && m.Focused {
		return theme.CursorItem.Render(fmt.Sprintf(format, arrow, label, countStr))
	}
	return fmt.Sprintf(format,
		theme.MutedItem.Render(arrow),
		theme.KeyStyle.Render(label),
		theme.MutedItem.Render(countStr))
}

func (m *VarListModel) renderPeekLine(v *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	text := "↳ new variable"
	if !v.IsNew {
		text = "↳ was: " + displayText(v.OriginalValue, m.isOriginalMasked(v))
	}
	return theme.MutedItem.Render(fmt.Sprintf("  %s  %s", padRight("", keyWidth), truncate(text, maxValWidth)))
}

func (m *VarListModel) renderDeletedLine(dv *model.EnvVar, keyWidth, maxValWidth int, theme Theme) string {
	key := padRight(truncate(dv.Key, keyWidth), keyWidth)
	value := m.formatValue(dv, maxValWidth)
	return theme.MutedItem.Render(fmt.Sprintf("  %s  %s", key, value)) + theme.DeletedMarker.Render(deletedVarGlyph) + " "
}

func (m *VarListModel) contentRows() int {
	return max(m.Height-panelChromeHeight, 1)
}

func (m *VarListModel) contentCols() int {
	return m.Width - panelBorderWidth
}

func (m *VarListModel) visibleLines() int {
	visible := m.contentRows()
	if m.drawsPeekLine() {
		visible = max(visible-1, 1)
	}
	return visible
}

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

func (m *VarListModel) calcColumnWidths() (keyWidth, maxValWidth int) {
	longestKey := 0
	for _, item := range m.displayItems {
		if item.Kind != displayItemVar {
			continue
		}
		longestKey = max(longestKey, len(m.File.Vars[item.VarIdx].Key))
	}
	if m.File != nil {
		for _, v := range m.File.DeletedVars {
			longestKey = max(longestKey, len(v.Key))
		}
	}
	return columnLimits{
		maxKey:   m.layout.VarListMaxKeyWidth,
		padding:  max(m.layout.VarListPadding, config.VarListMinPadding),
		minValue: m.layout.VarListMinValueWidth,
	}.split(longestKey, m.Width)
}

func (m *VarListModel) note() string {
	switch {
	case m.File == nil:
		return "  Select a file"
	case m.SearchQuery != "":
		quoted := strconv.Quote(m.SearchQuery)
		escapedQuery := quoted[1 : len(quoted)-1]
		const format = `  No matches for "%s"`
		room := m.contentCols() - lipgloss.Width(fmt.Sprintf(format, ""))
		return fmt.Sprintf(format, truncate(escapedQuery, room))
	}
	return "  No variables"
}

func (m *VarListModel) renderRows(keyWidth, maxValWidth int, theme Theme) []string {
	var lines []string
	for _, row := range m.drawnRows() {
		switch row.kind {
		case itemRow:
			item := m.displayItems[row.index]
			if item.Kind == displayItemHeader {
				lines = append(lines, m.renderHeaderLine(row.index, m.groups[item.GroupIdx], theme))
			} else {
				lines = append(lines, m.renderVarLine(row.index, &m.File.Vars[item.VarIdx], keyWidth, maxValWidth, theme))
			}
		case peekRow:
			v := &m.File.Vars[m.displayItems[row.index].VarIdx]
			lines = append(lines, m.renderPeekLine(v, keyWidth, maxValWidth, theme))
		case noteRow:
			lines = append(lines, theme.MutedItem.Render(truncate(m.note(), m.contentCols())))
		case deletedRow:
			lines = append(lines, m.renderDeletedLine(&m.File.DeletedVars[row.index], keyWidth, maxValWidth, theme))
		}
	}
	return lines
}

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
	count = fmt.Sprintf("(%s vars)", count)
	const sep = " "
	room := m.contentCols() - theme.PanelTitle.GetHorizontalFrameSize() - lipgloss.Width(sep+count)
	return theme.PanelTitle.Render(fitName(m.File.Name, room)) + sep + theme.MutedItem.Render(count)
}

// View renders the variable list panel.
func (m *VarListModel) View(theme Theme) string {
	title := m.viewTitle(theme)
	keyWidth, maxValWidth := m.calcColumnWidths()

	lines := m.renderRows(keyWidth, maxValWidth, theme)

	return renderPanel(m.Width, m.Height, m.Focused, title, strings.Join(lines, "\n"), theme)
}

func stripAnsi(s string) string { return ansi.Strip(s) }
