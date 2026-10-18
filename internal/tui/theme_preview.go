package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/config/themes"
)

// resolvedColors holds theme colors converted to color.Color for rendering.
type resolvedColors struct {
	primary  color.Color
	warning  color.Color
	err      color.Color
	success  color.Color
	muted    color.Color
	fg       color.Color
	bg       color.Color
	border   color.Color
	cursorBg color.Color
	modified color.Color
	added    color.Color
	deleted  color.Color
}

func resolveThemeColors(tc themes.Colors) resolvedColors {
	return resolvedColors{
		primary:  lipgloss.Color(tc.Primary),
		warning:  lipgloss.Color(tc.Warning),
		err:      lipgloss.Color(tc.Error),
		success:  lipgloss.Color(tc.Success),
		muted:    lipgloss.Color(tc.Muted),
		fg:       lipgloss.Color(tc.Fg),
		bg:       lipgloss.Color(tc.Bg),
		border:   lipgloss.Color(tc.Border),
		cursorBg: lipgloss.Color(tc.CursorBg),
		modified: lipgloss.Color(tc.Modified),
		added:    lipgloss.Color(tc.Added),
		deleted:  lipgloss.Color(tc.Deleted),
	}
}

// ThemePreviewModel is a standalone Bubble Tea model for browsing themes.
type ThemePreviewModel struct {
	themes      []string
	searchPaths []string
	cursor      int
	selected    string // non-empty if user pressed Enter
	width       int
	height      int
	noMouse     bool
	scrollLines int // themes a wheel tick moves through: mouse-scroll-lines
}

// NewThemePreview returns a new theme preview model, with the mouse settings
// from the config.
func NewThemePreview(noMouse bool, mouseScrollLines int) ThemePreviewModel {
	return ThemePreviewModel{
		themes:      config.ThemeNames(),
		searchPaths: config.ConfigSearchPaths(".", ""),
		noMouse:     noMouse,
		scrollLines: mouseScrollLines,
	}
}

func (m ThemePreviewModel) Init() tea.Cmd {
	return nil
}

func (m ThemePreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, cmd := tooSmallInput(m.width, m.height, msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg), nil
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg), nil
	}
	return m, nil
}

func (m ThemePreviewModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.selected = m.themes[m.cursor]
		return m, tea.Quit
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	}
	return m, nil
}

func (m *ThemePreviewModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *ThemePreviewModel) moveDown() {
	if m.cursor < len(m.themes)-1 {
		m.cursor++
	}
}

func (m ThemePreviewModel) handleMouseClick(msg tea.MouseClickMsg) ThemePreviewModel {
	if msg.Button == tea.MouseLeft {
		listWidth := max(m.width/3, config.FileListMinWidth)
		if msg.X < listWidth {
			index := visibleIndex(msg.Y-panelContentOffsetY, m.scrollOffset(), m.listRows(), len(m.themes))
			if index >= 0 {
				m.cursor = index
			}
		}
	}
	return m
}

func (m ThemePreviewModel) handleMouseWheel(msg tea.MouseWheelMsg) ThemePreviewModel {
	listWidth := max(m.width/3, config.FileListMinWidth)
	if msg.X < listWidth {
		wheelScroll(m.scrollLines, msg.Button == tea.MouseWheelUp, m.moveUp, m.moveDown)
	}
	return m
}

func (m ThemePreviewModel) scrollOffset() int {
	contentHeight := m.listRows()
	if m.cursor >= contentHeight {
		return m.cursor - contentHeight + 1
	}
	return 0
}

// panelHeight is the height of both panels: the rows above the status bar.
func (m ThemePreviewModel) panelHeight() int {
	return m.height - statusBarHeight
}

// listRows returns how many themes the list panel shows.
func (m ThemePreviewModel) listRows() int {
	return max(m.panelHeight()-panelChromeHeight, 0)
}

// Selected returns the theme name chosen by the user, or "" if they quit.
func (m ThemePreviewModel) Selected() string {
	return m.selected
}

func (m ThemePreviewModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	tc := m.currentColors()
	rc := resolveThemeColors(tc)

	if terminalTooSmall(m.width, m.height) {
		title := lipgloss.NewStyle().Foreground(rc.primary).Bold(true)
		muted := lipgloss.NewStyle().Foreground(rc.muted)
		return m.newView(tooSmallView(m.width, m.height, title, muted), rc)
	}

	listWidth := max(m.width/3, config.FileListMinWidth)
	// A Lip Gloss Width includes the border, so the two panels split the
	// terminal width between them.
	showcaseWidth := m.width - listWidth

	left := m.renderThemeList(listWidth, rc)
	right := m.renderShowcase(showcaseWidth, tc, rc)

	// Status bar
	statusBar := m.renderStatusBar(rc)

	// Join panels side by side
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return m.newView(lipgloss.JoinVertical(lipgloss.Left, panels, statusBar), rc)
}

// newView wraps content in a view painted with the theme's background.
func (m ThemePreviewModel) newView(content string, rc resolvedColors) tea.View {
	view := tea.NewView(content)
	view.BackgroundColor = rc.bg
	if !m.noMouse {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
}

func (m ThemePreviewModel) currentColors() themes.Colors {
	c, _ := themes.Lookup(m.themes[m.cursor])
	return c
}

func (m ThemePreviewModel) renderThemeList(width int, rc resolvedColors) string {

	title := lipgloss.NewStyle().Bold(true).Foreground(rc.primary).
		Render(fmt.Sprintf(" Themes (%d)", len(m.themes)))

	contentHeight := m.listRows()
	offset := m.scrollOffset()

	innerWidth := width - panelBorderWidth
	itemStyle := lipgloss.NewStyle().Foreground(rc.muted).Width(innerWidth)
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(rc.fg).Background(rc.cursorBg).Width(innerWidth)
	lines := []string{title}
	for i := offset; i < len(m.themes) && i < offset+contentHeight; i++ {
		prefix, style := "  ", itemStyle
		if i == m.cursor {
			prefix, style = "▸ ", cursorStyle
		}
		// Truncate rather than wrap, so each theme keeps to one row.
		lines = append(lines, style.Render(truncate(prefix+m.themes[i], innerWidth)))
	}

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(rc.border).
		Width(width).
		Height(m.panelHeight())

	return panel.Render(strings.Join(lines, "\n"))
}

func (m ThemePreviewModel) renderShowcase(width int, tc themes.Colors, rc resolvedColors) string {
	themeName := m.themes[m.cursor]

	title := lipgloss.NewStyle().Bold(true).Foreground(rc.primary).
		Render(" " + themeName)

	// Color list: swatch + name + hex
	type colorEntry struct {
		name  string
		value string
	}
	colors := []colorEntry{
		{"primary", tc.Primary},
		{"warning", tc.Warning},
		{"error", tc.Error},
		{"success", tc.Success},
		{"muted", tc.Muted},
		{"fg", tc.Fg},
		{"bg", tc.Bg},
		{"border", tc.Border},
		{"cursor-bg", tc.CursorBg},
		{"modified", tc.Modified},
		{"added", tc.Added},
		{"deleted", tc.Deleted},
	}

	var colorLines []string
	for _, c := range colors {
		hex := c.value
		if hex == "" {
			hex = "(none)"
		}
		swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(c.value)).Render("██")
		name := lipgloss.NewStyle().Foreground(lipgloss.Color(c.value)).
			Width(10).Render(c.name)
		hexStr := lipgloss.NewStyle().Foreground(rc.muted).Render(hex)
		colorLines = append(colorLines, fmt.Sprintf("  %s %s %s", swatch, name, hexStr))
	}

	colorSection := strings.Join(colorLines, "\n")

	innerWidth := width - panelBorderWidth
	configBox := m.renderConfigBox(innerWidth, themeName, rc)

	// Fit the content to the rows inside the border: show the config box only
	// when it fits, and cut the colors short on the shortest terminals.
	inner := lipgloss.NewStyle().Width(innerWidth)
	innerRows := max(m.panelHeight()-panelBorderHeight, 1)
	content := inner.Render(title + "\n\n" + colorSection)
	if box := inner.Render(configBox); lipgloss.Height(content)+1+lipgloss.Height(box) <= innerRows {
		content += "\n\n" + box
	}
	content = inner.MaxHeight(innerRows).Render(content)

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(rc.border).
		Width(width).
		Height(m.panelHeight())

	return panel.Render(content)
}

func (m ThemePreviewModel) renderConfigBox(width int, themeName string, rc resolvedColors) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(rc.primary).
		Render("Config files (priority)")

	var lines []string
	for i, p := range m.searchPaths {
		lines = append(lines, lipgloss.NewStyle().Foreground(rc.fg).
			Render(fmt.Sprintf("  %d. %s", i+1, p)))
	}

	snippet := lipgloss.NewStyle().Foreground(rc.muted).
		Render(fmt.Sprintf("  theme = %q", themeName))

	content := title + "\n" + strings.Join(lines, "\n") + "\n\n" + snippet

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(rc.border).
		Width(width).
		Padding(0, 1)

	return box.Render(content)
}

func (m ThemePreviewModel) renderStatusBar(rc resolvedColors) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(rc.primary)
	descStyle := lipgloss.NewStyle().Foreground(rc.muted)

	bar := fmt.Sprintf(" %s %s  %s %s  %s %s",
		keyStyle.Render("↑↓/jk"), descStyle.Render("navigate"),
		keyStyle.Render("enter"), descStyle.Render("select"),
		keyStyle.Render("q/esc"), descStyle.Render("quit"),
	)

	// Inline keeps the bar on one row, cut at the terminal width.
	return lipgloss.NewStyle().Width(m.width).MaxWidth(m.width).Inline(true).Render(bar)
}

// RunThemePreview runs the interactive theme preview and returns the selected theme name.
// Returns "" if the user quit without selecting.
func RunThemePreview(noMouse bool, mouseScrollLines int) (string, error) {
	m := NewThemePreview(noMouse, mouseScrollLines)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	return result.(ThemePreviewModel).Selected(), nil
}
