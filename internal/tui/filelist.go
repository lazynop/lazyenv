package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

// The file list's indicators, which the help lists under File Indicators.
const (
	selectedFileGlyph = "●"
	modifiedFileGlyph = "*" // also after a file's name in the compare view
	gitWarningGlyph   = "!"
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
		m.ensureCursorVisible()
	}
}

// MoveDown moves the cursor down.
func (m *FileListModel) MoveDown() {
	if m.Cursor < len(m.Files)-1 {
		m.Cursor++
		m.ensureCursorVisible()
	}
}

// SetCursor positions the cursor at the given index with bounds checking and
// selects the file there.
func (m *FileListModel) SetCursor(index int) {
	if len(m.Files) == 0 {
		return
	}
	m.MoveCursorTo(index)
	m.Select()
}

// MoveCursorTo positions the cursor at the given index with bounds checking,
// keeping the selected file: compare-select points at a file without making it
// the active one.
func (m *FileListModel) MoveCursorTo(index int) {
	if len(m.Files) == 0 {
		return
	}
	m.Cursor = max(0, min(index, len(m.Files)-1))
	m.ensureCursorVisible()
}

// SetSize sizes the panel and scrolls the cursor back into it.
func (m *FileListModel) SetSize(width, height int) {
	m.Width, m.Height = width, height
	m.ensureCursorVisible()
}

// ensureCursorVisible scrolls so the cursor row sits inside the viewport, and
// the viewport shows no blank rows while rows above it are scrolled off.
func (m *FileListModel) ensureCursorVisible() {
	if m.Height > panelChromeHeight { // not before the first layout
		m.Offset = scrollOffset(m.Cursor, m.Offset, m.visibleLines(), len(m.Files))
	}
}

// visibleLines returns how many files fit in the panel.
func (m *FileListModel) visibleLines() int {
	return max(m.Height-panelChromeHeight, 1)
}

// IndexAt returns the index of the file drawn on screen row y, the panel
// being drawn from the top of the screen, or -1 for a row that draws no file.
func (m *FileListModel) IndexAt(y int) int {
	return visibleIndex(y-panelContentOffsetY, m.Offset, m.visibleLines(), len(m.Files))
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
		content := theme.MutedItem.Render("  No .env files")
		return renderPanel(m.Width, m.Height, m.Focused, title, content, theme)
	}

	var lines []string
	end := min(m.Offset+m.visibleLines(), len(m.Files))
	inner := m.Width - panelChromeWidth

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
		name := truncate(flattenText(f.Name), nameMax)
		// Gap fills remaining space to push varCount to the right
		gap := max(inner-2-warnLen-lipgloss.Width(name)-len(varCount)-modLen, 1)

		var line string
		if i == m.Cursor && m.Focused {
			warn := ""
			if f.GitWarning {
				warn = gitWarningGlyph + " "
			}
			mod := ""
			if f.Modified {
				mod = modifiedFileGlyph
			}
			sel := "  "
			if i == m.Selected {
				sel = selectedFileGlyph + " "
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
				gitWarn = theme.GitWarning.Render(gitWarningGlyph + " ")
			}

			indicator := "  "
			if i == m.Selected {
				indicator = selectedFileGlyph + " "
			}

			modified := ""
			if f.Modified {
				modified = theme.ModifiedMarker.Render(modifiedFileGlyph)
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
	return renderPanel(m.Width, m.Height, m.Focused, title, content, theme)
}
