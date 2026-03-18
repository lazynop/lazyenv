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
}

// NewThemePreview returns a new theme preview model.
func NewThemePreview() ThemePreviewModel {
	return ThemePreviewModel{
		themes:      config.ThemeNames(),
		searchPaths: config.ConfigSearchPaths(".", ""),
	}
}

func (m ThemePreviewModel) Init() tea.Cmd {
	return nil
}

func (m ThemePreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			listWidth := max(m.width/3, config.FileListMinWidth)
			if msg.X < listWidth {
				index := msg.Y - 2 + m.scrollOffset()
				if index >= 0 && index < len(m.themes) {
					m.cursor = index
				}
			}
		}
		return m, nil
	case tea.MouseWheelMsg:
		listWidth := max(m.width/3, config.FileListMinWidth)
		if msg.X < listWidth {
			if msg.Button == tea.MouseWheelUp {
				m.cursor = max(0, m.cursor-3)
			} else {
				m.cursor = min(len(m.themes)-1, m.cursor+3)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m ThemePreviewModel) scrollOffset() int {
	contentHeight := max(m.height-4, 1)
	if m.cursor >= contentHeight {
		return m.cursor - contentHeight + 1
	}
	return 0
}

// Selected returns the theme name chosen by the user, or "" if they quit.
func (m ThemePreviewModel) Selected() string {
	return m.selected
}

func (m ThemePreviewModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	listWidth := max(m.width/3, config.FileListMinWidth)
	showcaseWidth := m.width - listWidth - 4 // borders

	tc := m.currentColors()
	rc := resolveThemeColors(tc)

	left := m.renderThemeList(listWidth, rc)
	right := m.renderShowcase(showcaseWidth, tc, rc)

	// Status bar
	statusBar := m.renderStatusBar(rc)

	// Join panels side by side
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, panels, statusBar))
	view.BackgroundColor = rc.bg
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

func (m ThemePreviewModel) currentColors() themes.Colors {
	c, _ := themes.Lookup(m.themes[m.cursor])
	return c
}

func (m ThemePreviewModel) renderThemeList(width int, rc resolvedColors) string {

	title := lipgloss.NewStyle().Bold(true).Foreground(rc.primary).
		Render(fmt.Sprintf(" Themes (%d)", len(m.themes)))

	contentHeight := max(m.height-4, 1) // borders + title + status bar
	offset := m.scrollOffset()

	var lines []string
	for i := offset; i < len(m.themes) && i < offset+contentHeight; i++ {
		name := m.themes[i]
		if i == m.cursor {
			style := lipgloss.NewStyle().Bold(true).Foreground(rc.fg).Background(rc.cursorBg).
				Width(width - 2)
			lines = append(lines, style.Render("▸ "+name))
		} else {
			style := lipgloss.NewStyle().Foreground(rc.muted).Width(width - 2)
			lines = append(lines, style.Render("  "+name))
		}
	}

	content := strings.Join(lines, "\n")

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(rc.border).
		Width(width).
		Height(m.height - 2) // status bar

	return panel.Render(title + "\n" + content)
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

	// Config box
	configBox := m.renderConfigBox(width-4, themeName, rc)

	content := title + "\n\n" + colorSection + "\n\n" + configBox

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(rc.border).
		Width(width).
		Height(m.height - 2) // status bar

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
