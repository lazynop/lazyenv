package tui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
)

// Theme holds all Lip Gloss styles.
type Theme struct {
	// Panel styles
	Panel            lipgloss.Style
	PanelTitle       lipgloss.Style
	StatusBar        lipgloss.Style
	StatusBarKey     lipgloss.Style
	StatusBarWarning lipgloss.Style
	StatusBarAlert   lipgloss.Style

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
}

// BuildTheme returns a theme based on the terminal background and optional color overrides.
// Each empty color falls back to the default-dark palette on a dark terminal
// and to the default-light palette on a light one.
func BuildTheme(isDark bool, colors config.ColorConfig) Theme {
	palette := config.OverlayColors(defaultPalette(isDark), colors)

	colorPrimary := lipgloss.Color(palette.Primary)
	colorWarning := lipgloss.Color(palette.Warning)
	colorError := lipgloss.Color(palette.Error)
	colorSuccess := lipgloss.Color(palette.Success)
	colorMuted := lipgloss.Color(palette.Muted)
	colorFg := lipgloss.Color(palette.Fg)
	colorBorder := lipgloss.Color(palette.Border)
	colorCursorBg := lipgloss.Color(palette.CursorBg)

	colorModified := lipgloss.Color(palette.Modified)
	colorAdded := lipgloss.Color(palette.Added)
	colorDeleted := lipgloss.Color(palette.Deleted)

	var colorBg color.Color
	if palette.Bg != "" {
		colorBg = lipgloss.Color(palette.Bg)
	}

	return Theme{
		Panel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder),
		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1, 0, 1+statusBarMarginWidth),
		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),
		StatusBarWarning: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning),
		StatusBarAlert: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError),

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
	}
}

func defaultPalette(isDark bool) config.ColorConfig {
	name := "default-light"
	if isDark {
		name = "default-dark"
	}
	palette, ok := config.LookupTheme(name)
	if !ok {
		panic("built-in theme " + name + " is missing")
	}
	return palette
}

func (t *Theme) barRoom(width int) int {
	return width - t.StatusBar.GetHorizontalFrameSize()
}

func renderPanel(width, height int, focused bool, title, content string, theme Theme) string {
	style := theme.Panel.
		Width(width).
		Height(height)

	if focused {
		style = style.BorderForeground(theme.ColorPrimary)
	}

	inner := fmt.Sprintf("%s\n%s", title, content)
	return style.Render(inner)
}
