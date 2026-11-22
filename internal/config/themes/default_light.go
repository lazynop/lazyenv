package themes

// Default light color theme.
// Used when no theme is specified and the terminal has a light background.
var defaultLight = Colors{
	Primary:  "#7B2FBE",
	Warning:  "#D97706",
	Error:    "#DC2626",
	Success:  "#059669",
	Muted:    "#6B7280",
	Fg:       "#1F2937",
	Bg:       "", // terminal native
	Border:   "#D1D5DB",
	CursorBg: "#E5E7EB",
	Modified: "#D97706",
	Added:    "#059669",
	Deleted:  "#DC2626",
}
