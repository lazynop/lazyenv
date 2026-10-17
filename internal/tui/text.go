package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/lazynop/lazyenv/internal/util"
)

// This file holds the text helpers every view draws its values with.

// displayText returns value as the panel shows it: masked when masked is
// set, and flattened to a single row.
func displayText(value string, masked bool) string {
	if masked {
		value = util.MaskValue(value)
	}
	return flattenValue(value)
}

// controlCharReplacer maps control chars to width-1 glyphs from the Unicode
// Arrows block, so values render on a single row without breaking the panel
// layout. It is zero-alloc for inputs that contain none of the targets.
var controlCharReplacer = strings.NewReplacer(
	"\n", "↵", // U+21B5
	"\t", "⇥", // U+21E5
	"\r", "↩", // U+21A9
)

func flattenValue(s string) string { return controlCharReplacer.Replace(s) }

// truncate fits s into maxCols display columns, appending ".." when the
// string is actually shortened. It operates on Unicode display width (via
// go-runewidth, the same oracle lipgloss.Width uses), so multi-byte runes
// like secret bullets or flatten glyphs are never sliced mid-sequence.
func truncate(s string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxCols {
		return s
	}
	if maxCols <= 2 {
		// Not enough room for the ".." ellipsis — hard-truncate with no tail.
		return runewidth.Truncate(s, maxCols, "")
	}
	return runewidth.Truncate(s, maxCols, "..")
}

func padRight(s string, width int) string {
	visLen := lipgloss.Width(s)
	if visLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visLen)
}
