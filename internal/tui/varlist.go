package tui

import (
	"fmt"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/util"
	"sort"
	"strings"
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

	// Filtered/sorted indices into File.Vars
	displayIndices []int
}

// NewVarListModel creates a new variable list model.
func NewVarListModel() VarListModel {
	return VarListModel{}
}

// SetFile sets the active file and recomputes display indices.
func (m *VarListModel) SetFile(f *model.EnvFile) {
	m.File = f
	m.Cursor = 0
	m.Offset = 0
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

// View renders the variable list panel.
func (m *VarListModel) View(theme Theme) string {
	fileName := ""
	varCount := 0
	if m.File != nil {
		fileName = m.File.Name
		varCount = len(m.File.Vars)
	}

	title := theme.PanelTitle.Render(fmt.Sprintf("Variables (%s)", fileName))

	if m.File == nil {
		content := theme.MutedItem.Render("  Select a file")
		return m.renderPanel(title, content, theme)
	}

	if len(m.displayIndices) == 0 {
		msg := "  No variables"
		if m.SearchQuery != "" {
			msg = fmt.Sprintf("  No matches for %q", m.SearchQuery)
		}
		content := theme.MutedItem.Render(msg)
		return m.renderPanel(title, content, theme)
	}

	visible := m.Height - 4
	if visible < 1 {
		visible = 1
	}

	// Calculate column widths
	keyWidth := 0
	for _, idx := range m.displayIndices {
		kw := len(m.File.Vars[idx].Key)
		if kw > keyWidth {
			keyWidth = kw
		}
	}
	if keyWidth > 30 {
		keyWidth = 30
	}

	maxValWidth := m.Width - keyWidth - 12 // space for padding, warnings, borders
	if maxValWidth < 10 {
		maxValWidth = 10
	}

	var lines []string
	end := m.Offset + visible
	if end > len(m.displayIndices) {
		end = len(m.displayIndices)
	}

	for i := m.Offset; i < end; i++ {
		idx := m.displayIndices[i]
		v := &m.File.Vars[idx]

		// Key
		key := padRight(v.Key, keyWidth)

		// Value (potentially masked)
		value := v.Value
		if v.IsSecret && !m.ShowSecrets {
			value = util.MaskValue(v.Value)
		}
		// Truncate long values
		if len(value) > maxValWidth {
			value = value[:maxValWidth-2] + ".."
		}
		value = padRight(value, maxValWidth)

		// Warning indicator
		warning := "  "
		if v.IsDuplicate {
			warning = theme.DuplicateWarn.Render("⚠ ")
		} else if v.IsEmpty {
			warning = theme.EmptyWarning.Render("⚠ ")
		} else if v.IsPlaceholder {
			warning = theme.PlaceholderWarn.Render("⚠ ")
		}

		var line string
		if v.IsSecret && !m.ShowSecrets {
			line = fmt.Sprintf("  %s  %s%s",
				theme.KeyStyle.Render(key),
				theme.SecretValue.Render(value),
				warning)
		} else {
			line = fmt.Sprintf("  %s  %s%s",
				theme.KeyStyle.Render(key),
				theme.ValueStyle.Render(value),
				warning)
		}

		if i == m.Cursor && m.Focused {
			line = theme.CursorItem.Render(padRight(stripAnsi(line), m.Width-4))
			// Re-apply on cursor: show key bold on highlighted bg
			k := padRight(v.Key, keyWidth)
			val := v.Value
			if v.IsSecret && !m.ShowSecrets {
				val = util.MaskValue(v.Value)
			}
			if len(val) > maxValWidth {
				val = val[:maxValWidth-2] + ".."
			}
			val = padRight(val, maxValWidth)
			line = theme.CursorItem.Render(fmt.Sprintf("  %s  %s%s", k, val, stripAnsi(warning)))
		}

		lines = append(lines, line)
	}

	_ = varCount
	content := strings.Join(lines, "\n")
	return m.renderPanel(title, content, theme)
}

func (m *VarListModel) renderPanel(title, content string, theme Theme) string {
	style := theme.VarPanel.
		Width(m.Width).
		Height(m.Height)

	if m.Focused {
		style = style.BorderForeground(colorPrimary)
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
