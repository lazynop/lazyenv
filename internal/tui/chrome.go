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

	// bottomChromeHeight is the rows reserved below the body of the normal,
	// compare and matrix views in every mode, drawn or not, so the body keeps
	// its height when an input bar opens or the status bar wraps: one for the
	// input bar, which never wraps, and two for the status bar, which wraps
	// on narrow terminals and takes the input bar's row too while no input
	// bar is open. App.withBottom cuts a status bar taller than its rows.
	bottomChromeHeight = 3

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

	// inputCursorWidth is the column a text input draws after its value for
	// the cursor at the end.
	inputCursorWidth = 1

	// matrixPaddingWidth is the left + right padding of the matrix view,
	// which has no border.
	matrixPaddingWidth = 2
)
