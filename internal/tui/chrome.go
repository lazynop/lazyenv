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

	// diffHeaderHeight is the two-row header above the side-by-side diff
	// panels.
	diffHeaderHeight = 2

	// diffChromeHeight is the vertical overhead of the side-by-side diff
	// view: panel chrome plus the diff header.
	diffChromeHeight = panelChromeHeight + diffHeaderHeight

	// statusBarHeight is the status bar row at the bottom of the screen.
	statusBarHeight = 1

	// bottomChromeHeight is the vertical space reserved below the two main
	// panels: input bar + status bar + one spare row (the status bar wraps
	// to two rows on narrow terminals).
	bottomChromeHeight = 3

	// compareBottomChromeHeight is bottomChromeHeight plus the editor bar
	// row shown in ModeEditingCompare.
	compareBottomChromeHeight = bottomChromeHeight + 1

	// compareMarginWidth is the horizontal margin around the side-by-side
	// compare view.
	compareMarginWidth = 2

	// panelContentOffsetY is the number of rows between a panel's top edge
	// and its first content row: top border + title row. Mouse hit detection
	// subtracts it from the click's Y coordinate.
	panelContentOffsetY = 2

	// diffContentOffsetY is panelContentOffsetY plus the column-header row
	// of the compare view.
	diffContentOffsetY = panelContentOffsetY + 1

	// matrixContentOffsetY is the number of rows above the first matrix
	// row: header row + separator row (the matrix view has no border).
	matrixContentOffsetY = 2
)
