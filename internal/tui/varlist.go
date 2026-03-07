package tui

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/util"
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
	if m.File != nil {
		fileName = m.File.Name
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

	visible := max(m.Height-4, 1)
	// Reserve a line for peek if active on a modified/new variable
	if m.Peeking && m.Cursor >= 0 && m.Cursor < len(m.displayIndices) {
		v := &m.File.Vars[m.displayIndices[m.Cursor]]
		if v.Modified || v.IsNew {
			visible = max(visible-1, 1)
		}
	}

	// Calculate column widths
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
		// space for padding, warnings, borders
		m.Width-keyWidth-m.layout.VarListPadding, m.layout.VarListMinValueWidth)

	var lines []string
	end := min(m.Offset+visible, len(m.displayIndices))

	for i := m.Offset; i < end; i++ {
		idx := m.displayIndices[i]
		v := &m.File.Vars[idx]

		// Key (truncate if longer than column)
		key := padRight(truncate(v.Key, keyWidth), keyWidth)

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

		// Warning/status indicator (2 slots: [modified][issue])
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
		warning := mod + issue

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
			k := padRight(truncate(v.Key, keyWidth), keyWidth)
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

		// Peek line: show original value under the selected variable
		if m.Peeking && i == m.Cursor && m.Focused {
			var peekLine string
			if v.IsNew {
				peekLine = theme.MutedItem.Render(fmt.Sprintf("  %s  ↳ new variable", padRight("", keyWidth)))
			} else if v.Modified {
				orig := v.OriginalValue
				if len(orig) > maxValWidth-4 {
					orig = orig[:maxValWidth-6] + ".."
				}
				peekLine = theme.MutedItem.Render(fmt.Sprintf("  %s  ↳ was: %s", padRight("", keyWidth), orig))
			}
			if peekLine != "" {
				lines = append(lines, peekLine)
			}
		}
	}

	// Render deleted variables at the bottom
	for _, dv := range m.File.DeletedVars {
		if len(lines) >= visible {
			break
		}
		key := padRight(truncate(dv.Key, keyWidth), keyWidth)
		value := dv.Value
		if dv.IsSecret && !m.ShowSecrets {
			value = util.MaskValue(dv.Value)
		}
		if len(value) > maxValWidth {
			value = value[:maxValWidth-2] + ".."
		}
		value = padRight(value, maxValWidth)
		marker := theme.DeletedMarker.Render("-")
		line := theme.MutedItem.Render(fmt.Sprintf("  %s  %s", key, value)) + marker + " "
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return m.renderPanel(title, content, theme)
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
