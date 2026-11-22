package tui

const (
	panelBorderHeight    = 2
	panelTopBorderHeight = 1
	panelTitleHeight     = 1

	panelBorderWidth = 2

	panelChromeHeight = panelBorderHeight + panelTitleHeight

	panelRowMarginWidth = 2

	panelChromeWidth = panelBorderWidth + panelRowMarginWidth

	statusBarHeight = 1

	statusBarMarginWidth = 2

	inputBarHeight         = 1
	statusBarWrappedHeight = 2
	bottomChromeHeight     = inputBarHeight + statusBarWrappedHeight

	compareMarginWidth = 2

	panelContentOffsetY = panelTopBorderHeight + panelTitleHeight

	matrixHeaderHeight      = 1
	matrixSeparatorHeight   = 1
	matrixStatsFooterHeight = 1
	matrixContentOffsetY    = matrixHeaderHeight + matrixSeparatorHeight
	matrixChromeHeight      = matrixContentOffsetY + matrixStatsFooterHeight

	inputCursorWidth = 1

	matrixPaddingWidth     = 2
	matrixSidePaddingWidth = matrixPaddingWidth / 2

	helpPaddingHeight = 1
	helpPaddingWidth  = 2

	helpFooterGapHeight = 1

	helpFooterHintHeight = 1
	helpFooterHeight     = helpFooterGapHeight + helpFooterHintHeight

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
