package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCreateBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, ".env")
	content := []byte("FOO=bar\nBAZ=qux\n")
	require.NoError(t, os.WriteFile(src, content, 0640))

	err := CreateBackup(src)
	require.NoError(t, err)

	bak := src + ".bak"
	got, err := os.ReadFile(bak)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	info, err := os.Stat(bak)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm())
}

func TestCreateBackupOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, ".env")
	bak := src + ".bak"

	require.NoError(t, os.WriteFile(src, []byte("OLD=value\n"), 0644))
	require.NoError(t, os.WriteFile(bak, []byte("stale backup\n"), 0644))

	require.NoError(t, CreateBackup(src))

	got, err := os.ReadFile(bak)
	require.NoError(t, err)
	assert.Equal(t, []byte("OLD=value\n"), got)
}

func TestCreateBackupMissingSource(t *testing.T) {
	err := CreateBackup("/nonexistent/path/.env")
	require.Error(t, err)
	assert.ErrorContains(t, err, "reading file for backup")
}

func TestEscapeDouble(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"newline", "hello\nworld", `hello\nworld`},
		{"tab", "hello\tworld", `hello\tworld`},
		{"backslash", `a\b`, `a\\b`},
		{"double_quote", `say "hi"`, `say \"hi\"`},
		{"carriage_return", "line\rend", `line\rend`},
		{"plain", "nothing special", "nothing special"},
		{"all_combined", "a\"\\\n\t\r", `a\"\\\n\t\r`},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeDouble(tt.in))
		})
	}
}

func TestReconstructLineSingleQuoted(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='original'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "updated value")

	output := string(Marshal(ef))
	assert.Equal(t, "FOO='updated value'\n", output)
}

func TestReconstructLineExportWithComment(t *testing.T) {
	ef := ParseBytes(".env", []byte("export FOO=bar # keep this\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "baz")

	output := string(Marshal(ef))
	assert.Equal(t, "export FOO=baz # keep this\n", output)
}

func TestReconstructLineDoubleQuotedEscape(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=\"old\"\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "has \"quotes\" and\nnewline")

	output := string(Marshal(ef))
	assert.Equal(t, "FOO=\"has \\\"quotes\\\" and\\nnewline\"\n", output)
}

func TestMarshalPreservesUnmodified(t *testing.T) {
	input := "# Header comment\n\nFOO=bar\nBAZ=qux\n"
	ef := ParseBytes(".env", []byte(input), config.SecretsConfig{})
	// No modifications — output should be identical
	assert.Equal(t, input, string(Marshal(ef)))
}

func TestRoundTripPreservesNoTrailingNewline(t *testing.T) {
	// A file without a trailing newline must round-trip without one being
	// silently injected.
	src := []byte("FOO=bar")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripPreservesTrailingNewline(t *testing.T) {
	// The common case: a file with a trailing newline keeps it on save.
	src := []byte("FOO=bar\n")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripEmptyFile(t *testing.T) {
	// A 0-byte file must round-trip as 0 bytes, not be turned into "\n".
	src := []byte{}
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripCRLFNormalizedToLF(t *testing.T) {
	// CRLF endings are normalized to LF on read. Round-trip keeps a trailing
	// newline (LF), not the original CRLF — line-ending normalization is by
	// design, not a fidelity regression.
	src := []byte("FOO=bar\r\nBAZ=qux\r\n")
	expected := []byte("FOO=bar\nBAZ=qux\n")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, expected, Marshal(ef))
}

func TestWriteFileNoPath(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n"), config.SecretsConfig{})
	// Path is empty by default from ParseBytes
	err := WriteFile(ef)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no file path set")
}

func TestNormalizeForWriteSingleQuoteWithApostrophe(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='original'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "it's a trap")

	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
	assert.True(t, ef.Vars[0].Modified)
}

func TestNormalizeForWriteNoApostropheNoOp(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='original'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "plain value")

	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteSingle, ef.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteDoubleQuoteUnchanged(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=\"original\"\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "it's fine")

	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteUnquotedSafeValueUnchanged(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=original\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "anything_safe")

	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteNone, ef.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteMultipleVars(t *testing.T) {
	ef := ParseBytes(".env", []byte("A='x'\nB='y'\nC='z'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "has ' apostrophe")
	ef.UpdateVar(2, "also 'quoted'")

	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
	assert.Equal(t, model.QuoteSingle, ef.Vars[1].QuoteStyle, "B untouched")
	assert.Equal(t, model.QuoteDouble, ef.Vars[2].QuoteStyle)
}

func TestNormalizeForWriteSingleQuoteToDoubleMarshal(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='original'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "it's a trap")

	NormalizeForWrite(ef)
	out := string(Marshal(ef))
	assert.Equal(t, "FOO=\"it's a trap\"\n", out)

	re := ParseBytes(".env", []byte(out), config.SecretsConfig{})
	require.Len(t, re.Vars, 1)
	assert.Equal(t, "it's a trap", re.Vars[0].Value)
	assert.Equal(t, model.QuoteDouble, re.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteIdempotent(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO='original'\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "it's a trap")

	NormalizeForWrite(ef)
	NormalizeForWrite(ef) // second call must be a no-op

	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteQuoteNoneControlChars(t *testing.T) {
	// Reproduces the compare-copy bug: a var added via AddVar (QuoteNone
	// by default) with any of \n, \r, \t in the value must round-trip
	// through marshal + re-parse without losing data past the first line.
	cases := []struct {
		name  string
		value string
	}{
		{"newline", "line1\nline2\nline3"},
		{"cr", "before\rafter"},
		{"tab", "col1\tcol2"},
		{"crlf", "line1\r\nline2\r\nline3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ef := ParseBytes(".env", []byte("EXISTING=ok\n"), config.SecretsConfig{})
			ef.AddVar("V", tc.value, false)

			NormalizeForWrite(ef)
			out := string(Marshal(ef))
			re := ParseBytes(".env", []byte(out), config.SecretsConfig{})

			v := re.VarByKey("V")
			require.NotNil(t, v)
			assert.Equal(t, tc.value, v.Value)
			assert.Equal(t, model.QuoteDouble, v.QuoteStyle)
			require.Len(t, re.Vars, 2, "no phantom vars must leak out of the round-trip")
		})
	}
}
