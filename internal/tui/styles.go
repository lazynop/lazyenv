package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all Lip Gloss styles with adaptive colors for light/dark terminals.
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

	// Value styles
	SecretValue      lipgloss.Style
	EmptyWarning     lipgloss.Style
	PlaceholderWarn  lipgloss.Style
	DuplicateWarn    lipgloss.Style
	KeyStyle         lipgloss.Style
	ValueStyle       lipgloss.Style
	CommentStyle     lipgloss.Style

	// Diff styles
	DiffEqual   lipgloss.Style
	DiffChanged lipgloss.Style
	DiffAdded   lipgloss.Style
	DiffRemoved lipgloss.Style

	// Help
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
}

var (
	colorPrimary = lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#BD93F9"}
	colorWarning = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FFB86C"}
	colorError   = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FF5555"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#50FA7B"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6272A4"}
	colorFg      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F8F8F2"}
	colorBg      = lipgloss.AdaptiveColor{Light: "#F9FAFB", Dark: "#282A36"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#44475A"}
)

// DefaultTheme returns the default theme.
func DefaultTheme() Theme {
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
			Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#44475A"}),
		MutedItem: lipgloss.NewStyle().
			Foreground(colorMuted),
		ModifiedMarker: lipgloss.NewStyle().
			Foreground(colorWarning),

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
	}
}
