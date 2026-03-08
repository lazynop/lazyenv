package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
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
	AddedMarker    lipgloss.Style
	DeletedMarker  lipgloss.Style
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
	ColorError   color.Color
	ColorMuted   color.Color
	ColorFg      color.Color
	ColorBg      color.Color // nil when no theme bg; used by View.BackgroundColor
	ColorBorder  color.Color
}

// BuildTheme returns a theme based on the terminal background and optional color overrides.
func BuildTheme(isDark bool, colors config.ColorConfig) Theme {
	ld := lipgloss.LightDark(isDark)

	resolve := func(override, light, dark string) color.Color {
		if override != "" {
			return lipgloss.Color(override)
		}
		return ld(lipgloss.Color(light), lipgloss.Color(dark))
	}

	colorPrimary := resolve(colors.Primary, "#7B2FBE", "#BD93F9")
	colorWarning := resolve(colors.Warning, "#D97706", "#FFB86C")
	colorError := resolve(colors.Error, "#DC2626", "#FF5555")
	colorSuccess := resolve(colors.Success, "#059669", "#50FA7B")
	colorMuted := resolve(colors.Muted, "#6B7280", "#6272A4")
	colorFg := resolve(colors.Fg, "#1F2937", "#F8F8F2")
	colorBorder := resolve(colors.Border, "#D1D5DB", "#44475A")
	colorCursorBg := resolve(colors.CursorBg, "#E5E7EB", "#44475A")

	colorModified := resolve(colors.Modified, "#D97706", "#FFB86C")
	colorAdded := resolve(colors.Added, "#059669", "#50FA7B")
	colorDeleted := resolve(colors.Deleted, "#DC2626", "#FF5555")

	var colorBg color.Color
	if colors.Bg != "" {
		colorBg = lipgloss.Color(colors.Bg)
	}

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
			Foreground(colorModified),
		AddedMarker: lipgloss.NewStyle().
			Foreground(colorAdded),
		DeletedMarker: lipgloss.NewStyle().
			Foreground(colorDeleted),
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
		ColorError:   colorError,
		ColorMuted:   colorMuted,
		ColorFg:      colorFg,
		ColorBg:      colorBg,
		ColorBorder:  colorBorder,
	}
}
