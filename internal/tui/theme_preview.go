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

// ThemePreviewModel is a standalone Bubble Tea model for browsing themes.
type ThemePreviewModel struct {
	themes   []string
	cursor   int
	selected string // non-empty if user pressed Enter
	width    int
	height   int
	isDark   bool
}

// NewThemePreview returns a new theme preview model.
func NewThemePreview() ThemePreviewModel {
	return ThemePreviewModel{
		themes: config.ThemeNames(),
	}
}

func (m ThemePreviewModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m ThemePreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter":
			m.selected = m.themes[m.cursor]
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.themes)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// Selected returns the theme name chosen by the user, or "" if they quit.
func (m ThemePreviewModel) Selected() string {
	return m.selected
}

func (m ThemePreviewModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	listWidth := m.width / 3
	if listWidth < 20 {
		listWidth = 20
	}
	showcaseWidth := m.width - listWidth - 4 // borders

	left := m.renderThemeList(listWidth)
	right := m.renderShowcase(showcaseWidth)

	// Status bar
	statusBar := m.renderStatusBar()

	// Join panels side by side
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, panels, statusBar))
}

func (m ThemePreviewModel) currentColors() themes.Colors {
	c, _ := themes.Lookup(m.themes[m.cursor])
	return c
}

func (m ThemePreviewModel) renderThemeList(width int) string {
	tc := m.currentColors()
	primary := lipgloss.Color(tc.Primary)
	fg := lipgloss.Color(tc.Fg)
	muted := lipgloss.Color(tc.Muted)
	border := lipgloss.Color(tc.Border)
	cursorBg := lipgloss.Color(tc.CursorBg)

	title := lipgloss.NewStyle().Bold(true).Foreground(primary).
		Render(fmt.Sprintf(" Themes (%d)", len(m.themes)))

	contentHeight := m.height - 4 // borders + title + status bar
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Scrolling: keep cursor visible
	offset := 0
	if m.cursor >= contentHeight {
		offset = m.cursor - contentHeight + 1
	}

	var lines []string
	for i := offset; i < len(m.themes) && i < offset+contentHeight; i++ {
		name := m.themes[i]
		if i == m.cursor {
			style := lipgloss.NewStyle().Bold(true).Foreground(fg).Background(cursorBg).
				Width(width - 2)
			lines = append(lines, style.Render("▸ "+name))
		} else {
			style := lipgloss.NewStyle().Foreground(muted).Width(width - 2)
			lines = append(lines, style.Render("  "+name))
		}
	}

	content := strings.Join(lines, "\n")

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Height(m.height - 2) // status bar

	return panel.Render(title + "\n" + content)
}

func (m ThemePreviewModel) renderShowcase(width int) string {
	tc := m.currentColors()
	themeName := m.themes[m.cursor]

	primary := lipgloss.Color(tc.Primary)
	fg := lipgloss.Color(tc.Fg)
	muted := lipgloss.Color(tc.Muted)
	border := lipgloss.Color(tc.Border)

	title := lipgloss.NewStyle().Bold(true).Foreground(primary).
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
		hexStr := lipgloss.NewStyle().Foreground(muted).Render(hex)
		colorLines = append(colorLines, fmt.Sprintf("  %s %s %s", swatch, name, hexStr))
	}

	colorSection := strings.Join(colorLines, "\n")

	// Config box
	configBox := m.renderConfigBox(width-4, themeName, primary, fg, muted, border)

	content := title + "\n\n" + colorSection + "\n\n" + configBox

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Height(m.height - 2) // status bar

	return panel.Render(content)
}

func (m ThemePreviewModel) renderConfigBox(width int, themeName string, primary, fg, muted, border color.Color) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(primary).
		Render("Config files (priority)")

	searchPaths := config.ConfigSearchPaths(".", "")

	var lines []string
	for i, p := range searchPaths {
		lines = append(lines, lipgloss.NewStyle().Foreground(fg).
			Render(fmt.Sprintf("  %d. %s", i+1, p)))
	}

	snippet := lipgloss.NewStyle().Foreground(muted).
		Render(fmt.Sprintf("  theme = %q", themeName))

	content := title + "\n" + strings.Join(lines, "\n") + "\n\n" + snippet

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Padding(0, 1)

	return box.Render(content)
}

func (m ThemePreviewModel) renderStatusBar() string {
	tc := m.currentColors()
	primary := lipgloss.Color(tc.Primary)
	muted := lipgloss.Color(tc.Muted)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(primary)
	descStyle := lipgloss.NewStyle().Foreground(muted)

	bar := fmt.Sprintf(" %s %s  %s %s  %s %s",
		keyStyle.Render("↑↓/jk"), descStyle.Render("navigate"),
		keyStyle.Render("enter"), descStyle.Render("select"),
		keyStyle.Render("q/esc"), descStyle.Render("quit"),
	)

	return lipgloss.NewStyle().Width(m.width).Render(bar)
}

// RunThemePreview runs the interactive theme preview and returns the selected theme name.
// Returns "" if the user quit without selecting.
func RunThemePreview() (string, error) {
	m := NewThemePreview()
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	return result.(ThemePreviewModel).Selected(), nil
}
