package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

// Regression tests for unquoted values that cannot survive a write/re-parse
// round trip unless NormalizeForWrite upgrades them to a quoted style.
// Each case feeds raw user input through the same path the TUI editor uses
// (UpdateVar on a QuoteNone var), serializes, re-parses, and requires the
// value — and the rest of the file — to come back intact.
func TestWriteRoundTripPreservesUnsafeUnquotedValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"hash_after_space_becomes_comment", "hello #world"},
		{"leading_double_quote_swallows_following_lines", `"oops`},
		{"leading_single_quote_swallows_following_lines", "'oops"},
		{"trailing_whitespace_stripped_on_reparse", "val  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := ParseBytes(".env", []byte("KEY=initial\nOTHER=keep\n"), config.SecretsConfig{})
			require.Len(t, ef.Vars, 2)

			ef.UpdateVar(0, tt.value)
			NormalizeForWrite(ef)
			data := Marshal(ef)

			re := ParseBytes(".env", data, config.SecretsConfig{})
			require.Len(t, re.Vars, 2,
				"re-parse must yield the same two vars; file was %q", string(data))

			v := re.VarByKey("KEY")
			require.NotNil(t, v, "KEY must survive; file was %q", string(data))
			assert.Equal(t, tt.value, v.Value,
				"KEY value must round-trip; file was %q", string(data))

			other := re.VarByKey("OTHER")
			require.NotNil(t, other, "OTHER must survive; file was %q", string(data))
			assert.Equal(t, "keep", other.Value)
		})
	}
}
