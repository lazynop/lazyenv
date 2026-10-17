package tui

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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

// flattenValue returns s drawable on one panel row: line breaks and tabs
// become width-1 arrows, and every other control character a visible
// stand-in, so a value can neither break the panel layout nor send the
// terminal a control sequence (clear the screen, write the clipboard).
// strings.Map also replaces invalid UTF-8, which could carry raw C1 bytes, and
// allocates nothing when s needs no change.
func flattenValue(s string) string { return strings.Map(flattenRune, s) }

func flattenRune(r rune) rune {
	switch {
	case r == '\n':
		return '↵' // U+21B5
	case r == '\t':
		return '⇥' // U+21E5
	case r == '\r':
		return '↩' // U+21A9
	case r < 0x20:
		return 0x2400 + r // Control Pictures: ␀ … ␟, ␛ for ESC
	case r == 0x7f:
		return '␡' // U+2421
	case r >= 0x80 && r < 0xa0: // C1 controls
		return utf8.RuneError
	}
	return r
}

// truncate fits s into maxCols display columns, appending ".." when the
// string is actually shortened. It measures s by grapheme cluster with x/ansi,
// as lipgloss.Width does, so a row it fits is a row lipgloss draws in that
// width, and multi-byte runes like secret bullets or flatten glyphs are never
// sliced mid-sequence.
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

func padRight(s string, width int) string {
	visLen := lipgloss.Width(s)
	if visLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visLen)
}
