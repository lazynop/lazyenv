package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
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
	FileA      *model.EnvFile
	FileB      *model.EnvFile
	allEntries []model.DiffEntry
	Entries    []model.DiffEntry // visible entries (after filtering)
	Cursor     int
	Offset     int
	Width      int
	Height     int
	HideEqual  bool
	Stats      DiffStats
}

// NewDiffViewModel creates a new diff view model.
func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{}
}

// SetFiles computes the diff between two files.
func (m *DiffViewModel) SetFiles(a, b *model.EnvFile) {
	m.FileA = a
	m.FileB = b
	m.allEntries = model.ComputeDiff(a, b)
	m.HideEqual = false
	m.recompute()
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
	if m.Offset > m.Cursor {
		m.Offset = m.Cursor
	}
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
		if m.Cursor < m.Offset {
			m.Offset = m.Cursor
		}
	}
}

// MoveDown moves the cursor down.
func (m *DiffViewModel) MoveDown() {
	if m.Cursor < len(m.Entries)-1 {
		m.Cursor++
		visible := m.Height - 6
		if visible > 0 && m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
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

func (m *DiffViewModel) ensureVisible() {
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}
	visible := m.Height - 6
	if visible > 0 && m.Cursor >= m.Offset+visible {
		m.Offset = m.Cursor - visible + 1
	}
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
		for i := len(m.FileB.Vars) - 1; i >= 0; i-- {
			if m.FileB.Vars[i].Key == e.Key {
				m.FileB.UpdateVar(i, e.ValueA)
				break
			}
		}
	case model.DiffAdded:
		// Key only in A, add to B
		m.FileB.AddVar(e.Key, e.ValueA)
	case model.DiffRemoved:
		// Key only in B — copying "right" means overwrite with nothing;
		// more useful: delete from B so they match (both gone)
		for i := len(m.FileB.Vars) - 1; i >= 0; i-- {
			if m.FileB.Vars[i].Key == e.Key {
				m.FileB.DeleteVar(i)
				break
			}
		}
	}

	m.allEntries = model.ComputeDiff(m.FileA, m.FileB)
	m.recompute()
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
		for i := len(m.FileA.Vars) - 1; i >= 0; i-- {
			if m.FileA.Vars[i].Key == e.Key {
				m.FileA.UpdateVar(i, e.ValueB)
				break
			}
		}
	case model.DiffRemoved:
		// Key only in B, add to A
		m.FileA.AddVar(e.Key, e.ValueB)
	case model.DiffAdded:
		// Key only in A — delete from A
		for i := len(m.FileA.Vars) - 1; i >= 0; i-- {
			if m.FileA.Vars[i].Key == e.Key {
				m.FileA.DeleteVar(i)
				break
			}
		}
	}

	m.allEntries = model.ComputeDiff(m.FileA, m.FileB)
	m.recompute()
	return e.Key
}

// Reset re-parses both files from disk, discarding in-memory edits.
// Returns error message on failure, empty string on success.
func (m *DiffViewModel) Reset() string {
	newA, errA := parser.ParseFile(m.FileA.Path)
	newB, errB := parser.ParseFile(m.FileB.Path)
	if errA != nil {
		return "Failed to reload " + m.FileA.Name
	}
	if errB != nil {
		return "Failed to reload " + m.FileB.Name
	}
	m.FileA = newA
	m.FileB = newB
	m.allEntries = model.ComputeDiff(m.FileA, m.FileB)
	m.recompute()
	return ""
}

// View renders the diff view.
func (m *DiffViewModel) View(theme Theme) string {
	if m.FileA == nil || m.FileB == nil {
		return ""
	}

	halfWidth := (m.Width - 2) / 2

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
	keyWidth := 0
	for _, e := range m.Entries {
		if len(e.Key) > keyWidth {
			keyWidth = len(e.Key)
		}
	}
	if keyWidth > 25 {
		keyWidth = 25
	}

	valWidth := max(halfWidth-keyWidth-10, 8)

	// Render entries
	visible := max(m.Height-6, 1)

	end := min(m.Offset+visible, len(m.Entries))

	var leftLines, rightLines []string

	for i := m.Offset; i < end; i++ {
		e := m.Entries[i]
		isCursor := i == m.Cursor

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

		key := padRight(e.Key, keyWidth)
		valA := truncate(e.ValueA, valWidth)
		valB := truncate(e.ValueB, valWidth)
		valA = padRight(valA, valWidth)
		valB = padRight(valB, valWidth)

		leftLine := fmt.Sprintf("  %s  %s", key, valA)
		rightLine := fmt.Sprintf("  %s  %s  %s", key, valB, statusChar)

		if e.Status == model.DiffRemoved {
			leftLine = padRight("", halfWidth-4)
		}
		if e.Status == model.DiffAdded {
			rightLine = fmt.Sprintf("  %s  %s", padRight("", keyWidth+valWidth+2), statusChar)
		}

		if isCursor {
			leftLine = theme.CursorItem.Render(padRight(leftLine, halfWidth-4))
			rightLine = theme.CursorItem.Render(padRight(rightLine, halfWidth-4))
		} else {
			leftLine = style.Render(leftLine)
			rightLine = style.Render(rightLine)
		}

		leftLines = append(leftLines, leftLine)
		rightLines = append(rightLines, rightLine)
	}

	leftContent := strings.Join(leftLines, "\n")
	rightContent := strings.Join(rightLines, "\n")

	leftPanel := lipgloss.NewStyle().
		Width(halfWidth).
		Height(m.Height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Render(fmt.Sprintf("%s\n%s", leftTitle, leftContent))

	rightPanel := lipgloss.NewStyle().
		Width(halfWidth).
		Height(m.Height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Render(fmt.Sprintf("%s\n%s", rightTitle, rightContent))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return s[:maxLen]
	}
	return s[:maxLen-2] + ".."
}
