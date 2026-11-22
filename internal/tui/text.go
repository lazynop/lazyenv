package tui

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lazynop/lazyenv/internal/util"
)

func displayText(value string, masked bool) string {
	if masked {
		value = util.MaskValue(value)
	}
	return flattenText(value)
}

func flattenText(s string) string { return strings.Map(flattenRune, s) }

func flattenRune(r rune) rune {
	switch {
	case r == '\n':
		return '↵'
	case r == '\t':
		return '⇥'
	case r == '\r':
		return '↩'
	case r < 0x20:
		return 0x2400 + r // Control Pictures: ␀ … ␟, ␛ for ESC
	case r == 0x7f:
		return '␡'
	case r >= 0x80 && r < 0xa0: // C1 controls
		return utf8.RuneError
	}
	return r
}

func truncate(s string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxCols {
		return s
	}
	if maxCols <= 2 {
		// Not enough room for the ".." ellipsis — hard-truncate with no tail.
		return ansi.Truncate(s, maxCols, "")
	}
	return ansi.Truncate(s, maxCols, "..")
}

func fitName(name string, maxCols int) string {
	return truncate(flattenText(name), maxCols)
}

func wrapRows(s string, width int) []string {
	return strings.Split(ansi.Wrap(s, width, ""), "\n")
}

func padRight(s string, width int) string {
	visLen := lipgloss.Width(s)
	if visLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visLen)
}
