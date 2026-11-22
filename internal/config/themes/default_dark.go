package themes

// Default dark color theme.
// Used when no theme is specified and the terminal has a dark background.
var defaultDark = Colors{
	Primary:  "#BD93F9",
	Warning:  "#FFB86C",
	Error:    "#FF5555",
	Success:  "#50FA7B",
	Muted:    "#6272A4",
	Fg:       "#F8F8F2",
	Bg:       "", // terminal native
	Border:   "#44475A",
	CursorBg: "#44475A",
	Modified: "#FFB86C",
	Added:    "#50FA7B",
	Deleted:  "#FF5555",
}
