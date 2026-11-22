package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

var maskValueAllowed = map[string]string{
	"text.go:displayText": "the helper itself",
}

func TestSecretsMaskOnlyThroughDisplayText(t *testing.T) {
	assertRefsAllowed(t, sourceRefs(t, internalUtilPath, "MaskValue"), maskValueAllowed,
		"draw a variable value with displayText(value, masked), so every view masks it the same way")
}

func TestFlattenText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain_ascii", "hello world", "hello world"},
		{"empty", "", ""},
		{"single_newline", "line1\nline2", "line1↵line2"},
		{"single_tab", "col1\tcol2", "col1⇥col2"},
		{"single_cr", "before\rafter", "before↩after"},
		{"mixed_all_three", "a\nb\tc\rd", "a↵b⇥c↩d"},
		{"multiple_newlines", "a\nb\nc", "a↵b↵c"},
		{"trailing_newline", "end\n", "end↵"},
		{"leading_newline", "\nstart", "↵start"},
		{"crlf_pair", "line\r\nnext", "line↩↵next"},
		{"no_control_chars_fast_path", "unchanged utf8 àèìòù", "unchanged utf8 àèìòù"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, flattenText(tt.in))
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		maxCols int
		want    string
	}{
		{"ascii_fits", "hello", 10, "hello"},
		{"ascii_exact", "hello", 5, "hello"},
		{"ascii_truncate", "hello world", 8, "hello .."},
		{"ascii_truncate_hello_world_5", "hello world", 5, "hel.."},
		{"ascii_truncate_tight", "hello", 4, "he.."},
		{"empty", "", 5, ""},

		{"bullets_fit_exactly", "•••••", 5, "•••••"},
		{"bullets_byte_over_col_under", "•••••", 6, "•••••"},
		{"bullets_col_over_truncate", "••••••", 5, "•••.."},
		{"bullets_no_tofu_at_4_cols", "•••••", 4, "••.."},
		{"bullets_three_cols_keeps_ellipsis", "•••••", 3, "•.."},
		{"bullets_no_tofu_at_7_cols", "••••••••", 7, "•••••.."},

		{"arrow_truncate", "line1↵line2", 7, "line1.."},
		{"tab_arrow_truncate", "a⇥b⇥c⇥d", 5, "a⇥b.."},

		{"maxlen_zero", "anything", 0, ""},
		{"maxlen_negative", "anything", -5, ""},
		{"maxlen_one_ascii_no_room_for_dots", "hello", 1, "h"},
		{"maxlen_two_ascii_no_room_for_dots", "hello", 2, "he"},
		{"maxlen_two_bullets_no_room_for_dots", "•••••", 2, "••"},
		{"maxlen_one_bullet_no_room_for_dots", "•••••", 1, "•"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.in, tt.maxCols))
		})
	}
}

func TestTruncateMeasuresLikeLipglossSoRowsNeitherWrapNorLoseText(t *testing.T) {
	emojiWithVS16TakingTwoColumns := strings.Repeat("\u2764\uFE0F", 10)
	zwjSequenceDrawnAsOneCluster := "\U0001F468\u200D\U0001F469\u200D\U0001F467 family"
	for _, s := range []string{emojiWithVS16TakingTwoColumns, zwjSequenceDrawnAsOneCluster} {
		width := lipgloss.Width(s)
		for cols := 1; cols < width; cols++ {
			assert.LessOrEqual(t, lipgloss.Width(truncate(s, cols)), cols, "%q in %d columns", s, cols)
		}
		assert.Equal(t, s, truncate(s, width), "%q fits in its own width", s)
	}
}

func TestDisplayTextNeutralizesTerminalControls(t *testing.T) {
	for _, tt := range []struct{ does, value string }{
		{"OSC 52 writes the clipboard", "\x1b]52;c;ZXZpbA==\x07"},
		{"CSI clears the screen", "\x1b[2J"},
		{"C1 CSI turns text red", "\x9b31m"},
		{"bell, del and nul", "bell\x07 del\x7f nul\x00"},
	} {
		shown := displayText(tt.value, false)
		assert.True(t, utf8.ValidString(shown), "%s: %q is drawn as %q", tt.does, tt.value, shown)
		assert.False(t, strings.ContainsFunc(shown, func(r rune) bool {
			return r < 0x20 || (r >= 0x7f && r < 0xa0)
		}), "%s: %q is drawn as %q", tt.does, tt.value, shown)
	}
}
