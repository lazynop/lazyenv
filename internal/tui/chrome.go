package tui

// Panel chrome dimensions. "Chrome" here means the fixed rows/columns of UI
// around a panel's renderable content: borders, title, status bar, etc.
// Use these constants when computing the visible area from a panel's total
// Width/Height so the layout stays in one place — the rule banning magic
// numbers in panel sizing is documented in CLAUDE.md.
const (
	// panelBorderHeight is top + bottom borders of a single panel.
	panelBorderHeight = 2

	// panelBorderWidth is left + right borders of a single panel.
	panelBorderWidth = 2

	// panelChromeHeight is the total vertical overhead around a panel:
	// borders (top + bottom) + title row + status bar = 4 rows.
	panelChromeHeight = 4

	// panelChromeWidth is the total horizontal overhead: borders + the
	// implicit left/right padding lipgloss adds = 4 columns.
	panelChromeWidth = 4

	// diffChromeHeight is the vertical overhead of the side-by-side diff
	// view: panel chrome plus the two-row diff header.
	diffChromeHeight = panelChromeHeight + 2
)
