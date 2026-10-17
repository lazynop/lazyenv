package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

// maskValueAllowed lists the functions that may call util.MaskValue directly,
// keyed by "file:function".
var maskValueAllowed = map[string]string{
	"text.go:displayText": "the helper itself",
}

// TestSecretsMaskOnlyThroughDisplayText fails on a util.MaskValue call outside
// displayText: one masking path keeps every view hiding the same values.
func TestSecretsMaskOnlyThroughDisplayText(t *testing.T) {
	assertRefsAllowed(t, sourceRefs(t, internalUtilPath, "MaskValue"), maskValueAllowed,
		"draw a variable value with displayText(value, masked), so every view masks it the same way")
}

func TestFlattenValue(t *testing.T) {
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
			assert.Equal(t, tt.want, flattenValue(tt.in))
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
		// ── Pure ASCII ──
		{"ascii_fits", "hello", 10, "hello"},
		{"ascii_exact", "hello", 5, "hello"},
		{"ascii_truncate", "hello world", 8, "hello .."},
		{"ascii_truncate_hello_world_5", "hello world", 5, "hel.."},
		{"ascii_truncate_tight", "hello", 4, "he.."},
		{"empty", "", 5, ""},

		// ── Bullets (U+2022, 1 col, 3 bytes) — the secret-masking case ──
		// The value is 15 bytes but 5 columns. The old byte-based truncate
		// would slice mid-sequence and produce tofu `•�..`.
		{"bullets_fit_exactly", "•••••", 5, "•••••"},
		{"bullets_byte_over_col_under", "•••••", 6, "•••••"},
		{"bullets_col_over_truncate", "••••••", 5, "•••.."},
		{"bullets_no_tofu_at_4_cols", "•••••", 4, "••.."},
		{"bullets_three_cols_keeps_ellipsis", "•••••", 3, "•.."},
		{"bullets_no_tofu_at_7_cols", "••••••••", 7, "•••••.."},

		// ── Arrow glyphs from flattenValue (U+21B5, U+21E5, U+21A9) ──
		{"arrow_truncate", "line1↵line2", 7, "line1.."},
		{"tab_arrow_truncate", "a⇥b⇥c⇥d", 5, "a⇥b.."},

		// ── Degenerate cases: maxCols ≤ 2 has no room for ".." ellipsis ──
		{"maxlen_zero", "anything", 0, ""},
		{"maxlen_negative", "anything", -5, ""},
		{"maxlen_one_ascii", "hello", 1, "h"},
		{"maxlen_two_ascii", "hello", 2, "he"},
		{"maxlen_two_bullets", "•••••", 2, "••"},
		{"maxlen_one_bullet", "•••••", 1, "•"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.in, tt.maxCols))
		})
	}
}

// TestTruncateMeasuresLikeLipgloss fits strings whose width go-runewidth and
// lipgloss disagree on. truncate must measure them as lipgloss draws them, or
// a row it fits wraps (an emoji with VS16 takes two columns, not one) or loses
// text that fits (a ZWJ sequence is one two-column cluster).
func TestTruncateMeasuresLikeLipgloss(t *testing.T) {
	for _, s := range []string{strings.Repeat("\u2764\uFE0F", 10), "\U0001F468\u200D\U0001F469\u200D\U0001F467 family"} {
		width := lipgloss.Width(s)
		for cols := 1; cols < width; cols++ {
			assert.LessOrEqual(t, lipgloss.Width(truncate(s, cols)), cols, "%q in %d columns", s, cols)
		}
		assert.Equal(t, s, truncate(s, width), "%q fits in its own width", s)
	}
}

// TestDisplayTextNeutralizesTerminalControls draws values holding terminal
// control sequences: no control byte reaches the screen, so a value can't
// write the clipboard, clear the screen or restyle the panel.
func TestDisplayTextNeutralizesTerminalControls(t *testing.T) {
	for _, value := range []string{
		"\x1b]52;c;ZXZpbA==\x07", // OSC 52: write the clipboard
		"\x1b[2J",                // clear the screen
		"\x9b31m",                // C1 CSI: red text
		"bell\x07 del\x7f nul\x00",
	} {
		shown := displayText(value, false)
		assert.True(t, utf8.ValidString(shown), "%q is drawn as %q", value, shown)
		assert.False(t, strings.ContainsFunc(shown, func(r rune) bool {
			return r < 0x20 || (r >= 0x7f && r < 0xa0)
		}), "%q is drawn as %q", value, shown)
	}
}
