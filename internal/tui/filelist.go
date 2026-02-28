package tui

import (
	"fmt"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"strings"

	"charm.land/lipgloss/v2"
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
	title := theme.PanelTitle.Render("Files")

	if len(m.Files) == 0 {
		content := theme.MutedItem.Render("  No .env files found")
		return m.renderPanel(title, content, theme)
	}

	visible := m.Height - 4 // borders + title + padding
	if visible < 1 {
		visible = 1
	}

	var lines []string
	end := m.Offset + visible
	if end > len(m.Files) {
		end = len(m.Files)
	}

	for i := m.Offset; i < end; i++ {
		f := m.Files[i]
		name := f.Name

		// Indicators
		indicator := "  "
		if i == m.Selected {
			indicator = "● "
		}

		modified := ""
		if f.Modified {
			modified = theme.ModifiedMarker.Render("*")
		}

		line := fmt.Sprintf("%s%s%s", indicator, name, modified)

		if i == m.Cursor && m.Focused {
			line = theme.CursorItem.Render(padRight(line, m.Width-4))
		} else if i == m.Selected {
			line = theme.SelectedItem.Render(line)
		} else {
			line = theme.NormalItem.Render(line)
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
