package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicKeyValue(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.Equal(t, "bar", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteNone, ef.Vars[0].QuoteStyle)
}

func TestUTF8BOMStripped(t *testing.T) {
	// Files saved by Windows editors may start with a UTF-8 BOM (EF BB BF).
	// Without stripping, the first key parses with BOM bytes prepended and the
	// line is misclassified as a comment.
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("FOO=bar\n")...)
	ef := ParseBytes(".env", data, config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.Equal(t, "bar", ef.Vars[0].Value)
}

func TestDoubleQuotedValue(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=\"hello world\"\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "hello world", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
}

func TestSingleQuotedValue(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='hello world'\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "hello world", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)
}

func TestMultilineDoubleQuoted(t *testing.T) {
	ef := ParseBytes(".env", []byte("MULTI=\"line1\nline2\nline3\"\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "line1\nline2\nline3", ef.Vars[0].Value)
}

func TestExportPrefix(t *testing.T) {
	ef := ParseBytes(".env", []byte("export NODE_ENV=production\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "NODE_ENV", ef.Vars[0].Key)
	assert.Equal(t, "production", ef.Vars[0].Value)
	assert.True(t, ef.Vars[0].HasExport)
}

func TestEmptyValue(t *testing.T) {
	ef := ParseBytes(".env", []byte("EMPTY=\nALSO_EMPTY=\"\"\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 2)
	assert.Empty(t, ef.Vars[0].Value)
	assert.True(t, ef.Vars[0].IsEmpty)
	assert.Empty(t, ef.Vars[1].Value)
	assert.True(t, ef.Vars[1].IsEmpty)
}

func TestComments(t *testing.T) {
	ef := ParseBytes(".env", []byte("# This is a comment\nFOO=bar\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	require.Len(t, ef.Lines, 2)
	assert.Equal(t, model.LineComment, ef.Lines[0].Type)
}

func TestInlineComment(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar # this is inline\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "bar", ef.Vars[0].Value)
	assert.Equal(t, "this is inline", ef.Vars[0].Comment)
}

func TestBlankLines(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n\nBAZ=qux\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 2)
	require.Len(t, ef.Lines, 3)
	assert.Equal(t, model.LineEmpty, ef.Lines[1].Type)
}

func TestEscapeSequences(t *testing.T) {
	input := "FOO=\"hello \\\"world\\\" \\\\path\\nnewline\"\n"
	ef := ParseBytes(".env", []byte(input), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "hello \"world\" \\path\nnewline", ef.Vars[0].Value)
}

func TestDuplicateDetection(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=first\nBAR=baz\nFOO=second\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 3)
	assert.True(t, ef.Vars[0].IsDuplicate, "first FOO should be duplicate")
	assert.False(t, ef.Vars[1].IsDuplicate, "BAR should not be duplicate")
	assert.True(t, ef.Vars[2].IsDuplicate, "second FOO should be duplicate")
}

func TestInvalidLinesTreatedAsComments(t *testing.T) {
	ef := ParseBytes(".env", []byte("not a valid line\nFOO=bar\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, model.LineComment, ef.Lines[0].Type)
}

func TestRoundTripFidelity(t *testing.T) {
	input := "# Database config\nDB_HOST=localhost              # primary host\nDB_PORT=5432\nDB_PASSWORD=\"super secret\"\n\n# API settings\nexport API_KEY='sk-12345'\nEMPTY=\nQUOTED_EMPTY=\"\"\n\n# Multiline\nMULTI=\"line1\nline2\nline3\"\n"

	ef := ParseBytes(".env", []byte(input), config.SecretsConfig{})
	output := Marshal(ef)

	assert.Equal(t, input, string(output))
}

func TestModifiedVarWriteBack(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=old\nBAR=keep\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "new")

	assert.Equal(t, "FOO=new\nBAR=keep\n", string(Marshal(ef)))
}

func TestModifiedQuotedVarWriteBack(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=\"old value\"\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "new value")

	assert.Equal(t, "FOO=\"new value\"\n", string(Marshal(ef)))
}

func TestAddVar(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n"), config.SecretsConfig{})
	ef.AddVar("NEW", "val", false)

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "FOO=bar\nNEW=val\n", string(Marshal(ef)))
}

func TestDeleteVar(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\nBAZ=qux\nQUX=end\n"), config.SecretsConfig{})
	ef.DeleteVar(1) // delete BAZ

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "FOO=bar\nQUX=end\n", string(Marshal(ef)))
}

func TestSecretDetection(t *testing.T) {
	ef := ParseBytes(".env", []byte("DB_PASSWORD=secret123\nAPI_KEY=sk-12345\nNORMAL=hello\n"), config.SecretsConfig{})

	assert.True(t, ef.Vars[0].IsSecret, "DB_PASSWORD should be secret")
	assert.True(t, ef.Vars[1].IsSecret, "API_KEY should be secret")
	assert.False(t, ef.Vars[2].IsSecret, "NORMAL should not be secret")
}

func TestPlaceholderDetection(t *testing.T) {
	ef := ParseBytes(".env", []byte("A=changeme\nB=TODO\nC=your_api_key_here\nD=real_value\n"), config.SecretsConfig{})

	assert.True(t, ef.Vars[0].IsPlaceholder, "changeme should be placeholder")
	assert.True(t, ef.Vars[1].IsPlaceholder, "TODO should be placeholder")
	assert.True(t, ef.Vars[2].IsPlaceholder, "your_api_key_here should be placeholder")
	assert.False(t, ef.Vars[3].IsPlaceholder, "real_value should not be placeholder")
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\n"), 0644))

	ef, err := ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	ef.UpdateVar(0, "baz")
	require.NoError(t, WriteFile(ef))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "FOO=baz\n", string(data))
}

func TestExportPrefixWriteBack(t *testing.T) {
	ef := ParseBytes(".env", []byte("export FOO=bar\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "baz")

	assert.Equal(t, "export FOO=baz\n", string(Marshal(ef)))
}

func TestInlineCommentWriteBack(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar # comment\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "baz")

	assert.Equal(t, "FOO=baz # comment\n", string(Marshal(ef)))
}

func TestVarByKey(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=first\nFOO=second\n"), config.SecretsConfig{})

	v := ef.VarByKey("FOO")
	require.NotNil(t, v)
	assert.Equal(t, "second", v.Value, "VarByKey should return last occurrence")

	assert.Nil(t, ef.VarByKey("MISSING"))
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"FOO", true},
		{"foo_bar", true},
		{"A1", true},
		{"_PRIVATE", true},
		{"my.config", true},
		{"my-var", true},
		{"FOO_BAR_123", true},

		// Invalid cases
		{"", false},
		{"1STARTS_WITH_DIGIT", false},
		{"has space", false},
		{"has@sign", false},
		{"has#hash", false},
		{"has$dollar", false},
		{"with=equals", false},
	}

	for _, tt := range tests {
		t.Run("key_"+tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidKey(tt.key))
		})
	}
}

func TestParseSingleQuotedEdgeCases(t *testing.T) {
	// Unterminated single quote
	ef := ParseBytes(".env", []byte("FOO='no closing quote\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "no closing quote", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)

	// Empty single-quoted value
	ef = ParseBytes(".env", []byte("FOO=''\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)
	assert.True(t, ef.Vars[0].IsEmpty)

	// Single-quoted value with inline comment
	ef = ParseBytes(".env", []byte("FOO='bar' # comment\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "bar", ef.Vars[0].Value)
	assert.Equal(t, "comment", ef.Vars[0].Comment)
}

func TestProcessEscapesEdgeCases(t *testing.T) {
	// Unrecognized escapes are preserved literally
	assert.Equal(t, "\\x", processEscapes(`\x`))
	assert.Equal(t, "\\z", processEscapes(`\z`))

	// Trailing backslash (no char after it) — preserved as-is
	assert.Equal(t, "end\\", processEscapes(`end\`))

	// \r escape
	assert.Equal(t, "\r", processEscapes(`\r`))

	// \t escape
	assert.Equal(t, "\t", processEscapes(`\t`))

	// Mixed escapes
	assert.Equal(t, "a\nb\\c\"d", processEscapes(`a\nb\\c\"d`))
}

func TestParseDoubleQuotedUnterminated(t *testing.T) {
	// Unterminated double quote: value extends to EOF
	ef := ParseBytes(".env", []byte("FOO=\"no closing\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "no closing", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
}

func TestExportWithTab(t *testing.T) {
	ef := ParseBytes(".env", []byte("export\tFOO=bar\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.True(t, ef.Vars[0].HasExport)
}

func TestKeyWithNoEquals(t *testing.T) {
	// A line with no = sign should be treated as a comment
	ef := ParseBytes(".env", []byte("NOEQUALS\nFOO=bar\n"), config.SecretsConfig{})
	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.Equal(t, model.LineComment, ef.Lines[0].Type)
}

func TestMultilineSingleQuoted(t *testing.T) {
	ef := ParseBytes(".env", []byte("MULTI='line1\nline2\nline3'\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "MULTI", ef.Vars[0].Key)
	assert.Equal(t, "line1\nline2\nline3", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)

	// One LineVariable, no phantom comment lines for the continuations.
	require.Len(t, ef.Lines, 1)
	assert.Equal(t, model.LineVariable, ef.Lines[0].Type)
}

func TestMultilineSingleQuotedRoundTrip(t *testing.T) {
	input := "MULTI='line1\nline2\nline3'\n"
	ef := ParseBytes(".env", []byte(input), config.SecretsConfig{})
	assert.Equal(t, input, string(Marshal(ef)))
}

func TestMultilineSingleQuotedFollowedByVar(t *testing.T) {
	ef := ParseBytes(".env", []byte("MULTI='line1\nline2'\nNEXT=value\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "MULTI", ef.Vars[0].Key)
	assert.Equal(t, "line1\nline2", ef.Vars[0].Value)
	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)
	assert.Equal(t, "NEXT", ef.Vars[1].Key)
	assert.Equal(t, "value", ef.Vars[1].Value)
}

func TestMultilineSingleQuotedNoPhantomVariable(t *testing.T) {
	// A continuation line containing an '=' must NOT be parsed as a separate
	// phantom variable. It is part of the single-quoted value.
	ef := ParseBytes(".env", []byte("NOTE='Short summary\nitem=value'\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "NOTE", ef.Vars[0].Key)
	assert.Equal(t, "Short summary\nitem=value", ef.Vars[0].Value)
}

func TestMultilineSingleQuotedWithInlineComment(t *testing.T) {
	ef := ParseBytes(".env", []byte("MULTI='line1\nline2' # trailing comment\n"), config.SecretsConfig{})

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "line1\nline2", ef.Vars[0].Value)
	assert.Equal(t, "trailing comment", ef.Vars[0].Comment)
}

func TestModifiedMultilineSingleQuotedWriteBack(t *testing.T) {
	ef := ParseBytes(".env", []byte("MULTI='line1\nline2'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "newline1\nnewline2\nnewline3")

	assert.Equal(t, "MULTI='newline1\nnewline2\nnewline3'\n", string(Marshal(ef)))
}

func TestMultilineSingleQuotedCompareValuesDiffer(t *testing.T) {
	// Regression test for the compare false-negative: two single-quoted
	// multiline values that share the first line but differ on the second
	// must end up with different Value fields after parse.
	efA := ParseBytes("a.env", []byte("MSG='first line\nsecond A'\n"), config.SecretsConfig{})
	efB := ParseBytes("b.env", []byte("MSG='first line\nsecond B'\n"), config.SecretsConfig{})

	require.Len(t, efA.Vars, 1)
	require.Len(t, efB.Vars, 1)
	assert.NotEqual(t, efA.Vars[0].Value, efB.Vars[0].Value,
		"values that differ only on the second line must not parse equal")
}
