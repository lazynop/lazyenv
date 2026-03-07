package themes

// Default light color theme.
// Used when no theme is specified and the terminal has a light background.
var defaultLight = Colors{
	Primary:  "#7B2FBE", // purple
	Warning:  "#D97706", // amber
	Error:    "#DC2626", // red
	Success:  "#059669", // green
	Muted:    "#6B7280", // gray
	Fg:       "#1F2937", // foreground
	Bg:       "",        // terminal native
	Border:   "#D1D5DB", // border
	CursorBg: "#E5E7EB",
	Modified: "#D97706", // amber
	Added:    "#059669", // green
	Deleted:  "#DC2626", // red
}
