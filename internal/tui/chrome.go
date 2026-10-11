package tui

// Panel chrome dimensions. "Chrome" here means the fixed rows/columns of UI
// around a panel's renderable content: borders, title, status bar, etc.
// Use these constants when computing the visible area from a panel's total
// Width/Height so the layout stays in one place — the rule banning magic
// numbers in panel sizing is documented in CODING_STANDARDS.md.
const (
	// panelBorderHeight is top + bottom borders of a single panel.
	panelBorderHeight = 2

	// panelBorderWidth is left + right borders of a single panel.
	panelBorderWidth = 2

	// panelChromeHeight is the rows a bordered panel draws around its
	// content: borders (top + bottom) + title row. The status bar is not part
	// of any panel.
	panelChromeHeight = panelBorderHeight + 1

	// panelChromeWidth is the total horizontal overhead: borders + the
	// implicit left/right padding lipgloss adds = 4 columns.
	panelChromeWidth = 4

	// statusBarHeight is the status bar row at the bottom of the screen.
	statusBarHeight = 1

	// bottomChromeHeight is the vertical space reserved below the panels of
	// the normal and compare views: input bar (the editor bar in compare
	// mode) + status bar + one spare row (the status bar wraps to two rows
	// on narrow terminals).
	bottomChromeHeight = 3

	// matrixBottomChromeHeight is the vertical space reserved below the
	// matrix view: editor bar (in ModeMatrixEditing) + status bar.
	matrixBottomChromeHeight = 2

	// compareMarginWidth is the horizontal margin around the side-by-side
	// compare view.
	compareMarginWidth = 2

	// panelContentOffsetY is the number of rows between a panel's top edge
	// and its first content row: top border + title row. Mouse hit detection
	// subtracts it from the click's Y coordinate.
	panelContentOffsetY = 2

	// matrixContentOffsetY is the number of rows above the first matrix
	// row: header row + separator row (the matrix view has no border).
	matrixContentOffsetY = 2

	// matrixChromeHeight is the rows the matrix view draws around its body
	// rows: header + separator row + stats footer.
	matrixChromeHeight = matrixContentOffsetY + 1
)
