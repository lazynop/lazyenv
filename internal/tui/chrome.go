package tui

const (
	panelBorderHeight = 2

	panelBorderWidth = 2

	// panelChromeHeight is the rows a bordered panel draws around its
	// content: borders (top + bottom) + title row. The status bar is not part
	// of any panel.
	panelChromeHeight = panelBorderHeight + 1

	panelRowMarginWidth = 2

	panelChromeWidth = panelBorderWidth + panelRowMarginWidth

	statusBarHeight = 1

	statusBarMarginWidth = 2

	// bottomChromeHeight is the rows reserved below the body of the normal,
	// compare and matrix views in every mode, drawn or not, so the body keeps
	// its height when an input bar opens or the status bar wraps: one for the
	// input bar, which never wraps, and two for the status bar, which wraps
	// on narrow terminals and takes the input bar's row too while no input
	// bar is open. App.withBottom cuts a status bar taller than its rows.
	bottomChromeHeight = 3

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

	inputCursorWidth = 1

	matrixPaddingWidth = 2

	helpPaddingHeight = 1
	helpPaddingWidth  = 2

	helpFooterGapHeight = 1

	// helpFooterHeight is the rows below the help text: the gap + the footer
	// hint.
	helpFooterHeight = helpFooterGapHeight + 1

	helpChromeHeight = 2*helpPaddingHeight + helpFooterHeight
)

// MinTerminalWidth and MinTerminalHeight are the smallest terminal the TUI
// lays its screens out in: below either, every screen outgrows the terminal
// (the status bar wraps past the rows reserved for it, matrix rows wrap), so
// the App and the theme preview draw tooSmallView instead.
const (
	MinTerminalWidth  = 60
	MinTerminalHeight = 10
)
