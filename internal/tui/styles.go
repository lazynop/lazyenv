package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds all Lip Gloss styles.
type Theme struct {
	// Panel styles
	FilePanel    lipgloss.Style
	VarPanel     lipgloss.Style
	PanelTitle   lipgloss.Style
	StatusBar    lipgloss.Style
	StatusBarKey lipgloss.Style

	// Item styles
	SelectedItem   lipgloss.Style
	NormalItem     lipgloss.Style
	CursorItem     lipgloss.Style
	MutedItem      lipgloss.Style
	ModifiedMarker lipgloss.Style
	GitWarning     lipgloss.Style

	// Value styles
	SecretValue     lipgloss.Style
	EmptyWarning    lipgloss.Style
	PlaceholderWarn lipgloss.Style
	DuplicateWarn   lipgloss.Style
	KeyStyle        lipgloss.Style
	ValueStyle      lipgloss.Style
	CommentStyle    lipgloss.Style

	// Diff styles
	DiffEqual   lipgloss.Style
	DiffChanged lipgloss.Style
	DiffAdded   lipgloss.Style
	DiffRemoved lipgloss.Style

	// Help
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Colors (for direct use)
	ColorPrimary color.Color
	ColorFg      color.Color
	ColorBorder  color.Color
}

// BuildTheme returns a theme based on the terminal background.
func BuildTheme(isDark bool) Theme {
	ld := lipgloss.LightDark(isDark)

	colorPrimary := ld(lipgloss.Color("#7B2FBE"), lipgloss.Color("#BD93F9"))
	colorWarning := ld(lipgloss.Color("#D97706"), lipgloss.Color("#FFB86C"))
	colorError := ld(lipgloss.Color("#DC2626"), lipgloss.Color("#FF5555"))
	colorSuccess := ld(lipgloss.Color("#059669"), lipgloss.Color("#50FA7B"))
	colorMuted := ld(lipgloss.Color("#6B7280"), lipgloss.Color("#6272A4"))
	colorFg := ld(lipgloss.Color("#1F2937"), lipgloss.Color("#F8F8F2"))
	colorBorder := ld(lipgloss.Color("#D1D5DB"), lipgloss.Color("#44475A"))
	colorCursorBg := ld(lipgloss.Color("#E5E7EB"), lipgloss.Color("#44475A"))

	return Theme{
		FilePanel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder),
		VarPanel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder),
		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1),
		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),
		NormalItem: lipgloss.NewStyle().
			Foreground(colorFg),
		CursorItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFg).
			Background(colorCursorBg),
		MutedItem: lipgloss.NewStyle().
			Foreground(colorMuted),
		ModifiedMarker: lipgloss.NewStyle().
			Foreground(colorWarning),
		GitWarning: lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true),

		SecretValue: lipgloss.NewStyle().
			Foreground(colorMuted),
		EmptyWarning: lipgloss.NewStyle().
			Foreground(colorWarning),
		PlaceholderWarn: lipgloss.NewStyle().
			Foreground(colorWarning),
		DuplicateWarn: lipgloss.NewStyle().
			Foreground(colorError),
		KeyStyle: lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true),
		ValueStyle: lipgloss.NewStyle().
			Foreground(colorFg),
		CommentStyle: lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true),

		DiffEqual: lipgloss.NewStyle().
			Foreground(colorMuted),
		DiffChanged: lipgloss.NewStyle().
			Foreground(colorWarning),
		DiffAdded: lipgloss.NewStyle().
			Foreground(colorSuccess),
		DiffRemoved: lipgloss.NewStyle().
			Foreground(colorError),

		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),
		HelpDesc: lipgloss.NewStyle().
			Foreground(colorMuted),

		ColorPrimary: colorPrimary,
		ColorFg:      colorFg,
		ColorBorder:  colorBorder,
	}
}
