package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

// DiffStats holds counts of each diff type.
type DiffStats struct {
	Equal   int
	Changed int
	Added   int
	Removed int
}

// DiffViewModel manages the diff/compare view.
type DiffViewModel struct {
	FileA       *model.EnvFile
	FileB       *model.EnvFile
	allEntries  []model.DiffEntry
	Entries     []model.DiffEntry // visible entries (after filtering)
	Cursor      int
	Offset      int
	Width       int
	Height      int
	HideEqual   bool
	ShowSecrets bool
	Stats       DiffStats

	layout  config.LayoutConfig
	secrets config.SecretsConfig
}

// NewDiffViewModel creates a new diff view model.
func NewDiffViewModel(layout config.LayoutConfig, secrets config.SecretsConfig) DiffViewModel {
	return DiffViewModel{layout: layout, secrets: secrets}
}

// SetFiles computes the diff between two files.
func (m *DiffViewModel) SetFiles(a, b *model.EnvFile) {
	m.FileA, m.FileB = a, b
	m.HideEqual = false
	m.Rediff()
}

// SetSize sizes the panels and scrolls the cursor back into them.
func (m *DiffViewModel) SetSize(width, height int) {
	m.Width, m.Height = width, height
	m.ensureVisible()
}

func (m *DiffViewModel) recompute() {
	m.Stats = DiffStats{}
	m.Entries = nil

	for _, e := range m.allEntries {
		switch e.Status {
		case model.DiffEqual:
			m.Stats.Equal++
		case model.DiffChanged:
			m.Stats.Changed++
		case model.DiffAdded:
			m.Stats.Added++
		case model.DiffRemoved:
			m.Stats.Removed++
		}

		if m.HideEqual && e.Status == model.DiffEqual {
			continue
		}
		m.Entries = append(m.Entries, e)
	}

	if m.Cursor >= len(m.Entries) {
		m.Cursor = max(0, len(m.Entries)-1)
	}
	m.ensureVisible()
}

// Rediff recomputes the diff after FileA or FileB changed.
func (m *DiffViewModel) Rediff() {
	m.allEntries = model.ComputeDiff(m.FileA, m.FileB)
	m.recompute()
}

// ToggleFilter toggles hiding of equal entries.
func (m *DiffViewModel) ToggleFilter() {
	m.HideEqual = !m.HideEqual
	m.recompute()
}

// MoveUp moves the cursor up.
func (m *DiffViewModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		m.ensureVisible()
	}
}

// MoveDown moves the cursor down.
func (m *DiffViewModel) MoveDown() {
	if m.Cursor < len(m.Entries)-1 {
		m.Cursor++
		m.ensureVisible()
	}
}

// SetCursor positions the cursor at the given index with bounds checking.
func (m *DiffViewModel) SetCursor(index int) {
	if len(m.Entries) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.Entries)-1))
	m.ensureVisible()
}

// NextDiff jumps to the next non-equal entry.
func (m *DiffViewModel) NextDiff() {
	for i := m.Cursor + 1; i < len(m.Entries); i++ {
		if m.Entries[i].Status != model.DiffEqual {
			m.Cursor = i
			m.ensureVisible()
			return
		}
	}
}

// PrevDiff jumps to the previous non-equal entry.
func (m *DiffViewModel) PrevDiff() {
	for i := m.Cursor - 1; i >= 0; i-- {
		if m.Entries[i].Status != model.DiffEqual {
			m.Cursor = i
			m.ensureVisible()
			return
		}
	}
}

// visibleLines returns how many entries fit in a panel.
func (m *DiffViewModel) visibleLines() int {
	return max(m.Height-panelChromeHeight, 1)
}

// IndexAt returns the index of the entry drawn on screen row y, the panels
// being drawn from the top of the screen, or -1 for a row that draws no entry.
func (m *DiffViewModel) IndexAt(y int) int {
	return visibleIndex(y-panelContentOffsetY, m.Offset, m.visibleLines(), len(m.Entries))
}

// ensureVisible scrolls just enough to keep the cursor on screen.
func (m *DiffViewModel) ensureVisible() {
	m.Offset = scrollOffset(m.Cursor, m.Offset, m.visibleLines(), len(m.Entries))
}

// CopyToRight copies the value of the selected entry from file A to file B.
// Returns the key name if successful, empty string otherwise.
func (m *DiffViewModel) CopyToRight() string {
	if m.Cursor < 0 || m.Cursor >= len(m.Entries) {
		return ""
	}
	e := m.Entries[m.Cursor]
	if e.Status == model.DiffEqual {
		return ""
	}

	switch e.Status {
	case model.DiffChanged:
		// Update existing key in B with A's value
		if i := m.FileB.LastIndexOfKey(e.Key); i >= 0 {
			m.FileB.UpdateVar(i, e.ValueA, util.IsSecret(e.Key, e.ValueA, m.secrets))
		}
	case model.DiffAdded:
		// Key only in A, add to B
		m.FileB.AddVar(e.Key, e.ValueA, util.IsSecret(e.Key, e.ValueA, m.secrets))
	case model.DiffRemoved:
		// Key only in B — copying "right" means overwrite with nothing;
		// more useful: delete from B so they match (both gone)
		if i := m.FileB.LastIndexOfKey(e.Key); i >= 0 {
			m.FileB.DeleteVar(i)
		}
	}

	m.Rediff()
	return e.Key
}

// CopyToLeft copies the value of the selected entry from file B to file A.
// Returns the key name if successful, empty string otherwise.
func (m *DiffViewModel) CopyToLeft() string {
	if m.Cursor < 0 || m.Cursor >= len(m.Entries) {
		return ""
	}
	e := m.Entries[m.Cursor]
	if e.Status == model.DiffEqual {
		return ""
	}

	switch e.Status {
	case model.DiffChanged:
		// Update existing key in A with B's value
		if i := m.FileA.LastIndexOfKey(e.Key); i >= 0 {
			m.FileA.UpdateVar(i, e.ValueB, util.IsSecret(e.Key, e.ValueB, m.secrets))
		}
	case model.DiffRemoved:
		// Key only in B, add to A
		m.FileA.AddVar(e.Key, e.ValueB, util.IsSecret(e.Key, e.ValueB, m.secrets))
	case model.DiffAdded:
		// Key only in A — delete from A
		if i := m.FileA.LastIndexOfKey(e.Key); i >= 0 {
			m.FileA.DeleteVar(i)
		}
	}

	m.Rediff()
	return e.Key
}

// Reset re-parses both files from disk, discarding in-memory edits.
// Returns error message on failure, empty string on success.
func (m *DiffViewModel) Reset() string {
	newA, errA := reparse(m.FileA, m.secrets)
	newB, errB := reparse(m.FileB, m.secrets)
	if errA != nil {
		return "Failed to reload " + m.FileA.Name
	}
	if errB != nil {
		return "Failed to reload " + m.FileB.Name
	}
	m.FileA = newA
	m.FileB = newB
	m.Rediff()
	return ""
}

// View renders the diff view.
func (m *DiffViewModel) View(theme Theme) string {
	if m.FileA == nil || m.FileB == nil {
		return ""
	}

	halfWidth := m.Width / 2

	// Headers
	leftName := m.FileA.Name
	rightName := m.FileB.Name
	if m.FileA.Modified {
		leftName += "*"
	}
	if m.FileB.Modified {
		rightName += "*"
	}
	leftTitle := theme.PanelTitle.Render(leftName)
	rightTitle := theme.PanelTitle.Render(rightName)

	// Calculate column widths
	longestKey := 0
	for _, e := range m.Entries {
		longestKey = max(longestKey, len(e.Key))
	}
	keyWidth, valWidth := columnLimits{
		maxKey:   m.layout.DiffMaxKeyWidth,
		padding:  max(m.layout.DiffPadding, config.DiffMinPadding),
		minValue: m.layout.DiffMinValueWidth,
	}.split(longestKey, halfWidth)

	// Render entries
	end := min(m.Offset+m.visibleLines(), len(m.Entries))

	var leftLines, rightLines []string

	for i := m.Offset; i < end; i++ {
		leftLine, rightLine := m.renderDiffEntry(m.Entries[i], i == m.Cursor, keyWidth, valWidth, halfWidth, theme)
		leftLines = append(leftLines, leftLine)
		rightLines = append(rightLines, rightLine)
	}

	leftContent := strings.Join(leftLines, "\n")
	rightContent := strings.Join(rightLines, "\n")

	panel := lipgloss.NewStyle().
		Width(halfWidth).
		Height(m.Height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		panel.Render(fmt.Sprintf("%s\n%s", leftTitle, leftContent)),
		panel.Render(fmt.Sprintf("%s\n%s", rightTitle, rightContent)))
}

// renderDiffEntry renders a single diff entry, returning the left and right
// panel lines.
func (m *DiffViewModel) renderDiffEntry(e model.DiffEntry, isCursor bool, keyWidth, valWidth, halfWidth int, theme Theme) (string, string) {
	var statusChar string
	var style lipgloss.Style

	switch e.Status {
	case model.DiffEqual:
		statusChar = "="
		style = theme.DiffEqual
	case model.DiffChanged:
		statusChar = "≠"
		style = theme.DiffChanged
	case model.DiffAdded:
		statusChar = "+"
		style = theme.DiffAdded
	case model.DiffRemoved:
		statusChar = "-"
		style = theme.DiffRemoved
	}

	key := padRight(truncate(e.Key, keyWidth), keyWidth)
	masked := e.IsSecret && !m.ShowSecrets
	valA := padRight(truncate(displayText(e.ValueA, masked), valWidth), valWidth)
	valB := padRight(truncate(displayText(e.ValueB, masked), valWidth), valWidth)

	leftLine := fmt.Sprintf("  %s  %s", key, valA)
	rightLine := fmt.Sprintf("  %s  %s  %s", key, valB, statusChar)

	if e.Status == model.DiffRemoved || e.Status == model.DiffAdded {
		missing := padRight(truncate("(missing)", valWidth), valWidth)
		if e.Status == model.DiffRemoved {
			leftLine = fmt.Sprintf("  %s  %s", key, missing)
		} else {
			rightLine = fmt.Sprintf("  %s  %s  %s", key, missing, statusChar)
		}
	}

	if isCursor {
		leftLine = theme.CursorItem.Render(padRight(leftLine, halfWidth-panelChromeWidth))
		rightLine = theme.CursorItem.Render(padRight(rightLine, halfWidth-panelChromeWidth))
	} else {
		leftLine = style.Render(leftLine)
		rightLine = style.Render(rightLine)
	}

	return leftLine, rightLine
}

// columnLimits holds a key/value table's width settings from the layout config.
type columnLimits struct {
	maxKey   int // widest the key column may grow
	padding  int // a row's fixed overhead: borders, gaps and markers
	minValue int // value width kept before the key column gives up any more
}

// split divides a panel width between the key column, no wider than
// longestKey or maxKey, and the value column. When both don't fit, the key
// column shrinks first, down to nothing, so the value keeps minValue columns
// or all the room when there is less than that. As long as padding covers the
// row's fixed overhead (callers floor it at the config minimums, which do),
// the row never grows past width, so it never wraps.
func (l columnLimits) split(longestKey, width int) (keyWidth, valWidth int) {
	room := max(width-l.padding, 0)
	keyWidth = min(longestKey, l.maxKey, max(room-l.minValue, 0))
	return keyWidth, room - keyWidth
}
