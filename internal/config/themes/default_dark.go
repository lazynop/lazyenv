package themes

// Default dark color theme.
// Used when no theme is specified and the terminal has a dark background.
var defaultDark = Colors{
	Primary:  "#BD93F9", // purple
	Warning:  "#FFB86C", // orange
	Error:    "#FF5555", // red
	Success:  "#50FA7B", // green
	Muted:    "#6272A4", // gray
	Fg:       "#F8F8F2", // foreground
	Bg:       "",        // terminal native
	Border:   "#44475A", // border
	CursorBg: "#44475A",
	Modified: "#FFB86C", // orange
	Added:    "#50FA7B", // green
	Deleted:  "#FF5555", // red
}
