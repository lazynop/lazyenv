package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

// FileListModel manages the file list panel.
type FileListModel struct {
	Files    []*model.EnvFile
	Cursor   int
	Selected int // index of the selected (active) file
	Offset   int // scroll offset
	Height   int
	Width    int
	Focused  bool
}

// NewFileListModel creates a new file list model.
func NewFileListModel() FileListModel {
	return FileListModel{
		Selected: 0,
		Cursor:   0,
	}
}

// SetFiles sets the file list.
func (m *FileListModel) SetFiles(files []*model.EnvFile) {
	m.Files = files
	if m.Selected >= len(files) {
		m.Selected = 0
	}
	if m.Cursor >= len(files) {
		m.Cursor = 0
	}
}

// MoveUp moves the cursor up.
func (m *FileListModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		if m.Cursor < m.Offset {
			m.Offset = m.Cursor
		}
	}
}

// MoveDown moves the cursor down.
func (m *FileListModel) MoveDown() {
	if m.Cursor < len(m.Files)-1 {
		m.Cursor++
		visible := m.Height - 2 // account for border/title
		if visible > 0 && m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
}

// SetCursor positions the cursor at the given index with bounds checking.
func (m *FileListModel) SetCursor(index int) {
	if len(m.Files) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.Files)-1))
	m.Select()
	visible := m.Height - 2
	if visible > 0 {
		if m.Cursor < m.Offset {
			m.Offset = m.Cursor
		}
		if m.Cursor >= m.Offset+visible {
			m.Offset = m.Cursor - visible + 1
		}
	}
}

// Select selects the current cursor position.
func (m *FileListModel) Select() {
	m.Selected = m.Cursor
}

// SelectedFile returns the currently selected file.
func (m *FileListModel) SelectedFile() *model.EnvFile {
	if m.Selected >= 0 && m.Selected < len(m.Files) {
		return m.Files[m.Selected]
	}
	return nil
}

// CursorFile returns the file under the cursor.
func (m *FileListModel) CursorFile() *model.EnvFile {
	if m.Cursor >= 0 && m.Cursor < len(m.Files) {
		return m.Files[m.Cursor]
	}
	return nil
}

// View renders the file list panel.
func (m *FileListModel) View(theme Theme) string {
	title := theme.PanelTitle.Render("Files") + " " + theme.MutedItem.Render(fmt.Sprintf("(%d)", len(m.Files)))

	if len(m.Files) == 0 {
		content := theme.MutedItem.Render("  No .env files found")
		return m.renderPanel(title, content, theme)
	}

	visible := max(
		// borders + title + padding
		m.Height-4, 1)

	var lines []string
	end := min(m.Offset+visible, len(m.Files))
	inner := m.Width - 4 // panel width minus borders

	for i := m.Offset; i < end; i++ {
		f := m.Files[i]
		varCount := fmt.Sprintf("%d", len(f.Vars))

		// Layout: indicator(2) + warn(0|2) + name + gap + varCount + modified(0|1)
		warnLen := 0
		if f.GitWarning {
			warnLen = 2
		}
		modLen := 0
		if f.Modified {
			modLen = 1
		}
		// Space for name: inner - indicator - warn - gap(1 min) - varCount - modified
		nameMax := max(inner-2-warnLen-1-len(varCount)-modLen, 1)
		name := truncate(f.Name, nameMax)
		// Gap fills remaining space to push varCount to the right
		gap := max(inner-2-warnLen-len(name)-len(varCount)-modLen, 1)

		var line string
		if i == m.Cursor && m.Focused {
			warn := ""
			if f.GitWarning {
				warn = "! "
			}
			mod := ""
			if f.Modified {
				mod = "*"
			}
			sel := "  "
			if i == m.Selected {
				sel = "● "
			}
			raw := fmt.Sprintf("%s%s%s%s%s%s", sel, warn, name, strings.Repeat(" ", gap), varCount, mod)
			line = theme.CursorItem.Render(padRight(raw, inner))
		} else {
			// Style each segment individually to avoid ANSI reset leaking
			var itemStyle lipgloss.Style
			if i == m.Selected {
				itemStyle = theme.SelectedItem
			} else {
				itemStyle = theme.NormalItem
			}

			gitWarn := ""
			if f.GitWarning {
				gitWarn = theme.GitWarning.Render("! ")
			}

			indicator := "  "
			if i == m.Selected {
				indicator = "● "
			}

			modified := ""
			if f.Modified {
				modified = theme.ModifiedMarker.Render("*")
			}

			line = fmt.Sprintf("%s%s%s%s%s%s",
				itemStyle.Render(indicator),
				gitWarn,
				itemStyle.Render(name),
				itemStyle.Render(strings.Repeat(" ", gap)),
				theme.MutedItem.Render(varCount),
				modified)
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return m.renderPanel(title, content, theme)
}

func (m *FileListModel) renderPanel(title, content string, theme Theme) string {
	style := theme.FilePanel.
		Width(m.Width).
		Height(m.Height)

	if m.Focused {
		style = style.BorderForeground(theme.ColorPrimary)
	}

	inner := fmt.Sprintf("%s\n%s", title, content)
	return style.Render(inner)
}

func padRight(s string, width int) string {
	visLen := lipgloss.Width(s)
	if visLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visLen)
}
