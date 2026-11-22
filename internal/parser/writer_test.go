package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
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
	assert.Equal(t, input, string(Marshal(ef)))
}

func TestRoundTripPreservesNoTrailingNewline(t *testing.T) {
	src := []byte("FOO=bar")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripPreservesTrailingNewline(t *testing.T) {
	src := []byte("FOO=bar\n")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripEmptyFile(t *testing.T) {
	src := []byte{}
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, src, Marshal(ef))
}

func TestRoundTripCRLFNormalizedToLF(t *testing.T) {
	src := []byte("FOO=bar\r\nBAZ=qux\r\n")
	expected := []byte("FOO=bar\nBAZ=qux\n")
	ef := ParseBytes(".env", src, config.SecretsConfig{})
	assert.Equal(t, expected, Marshal(ef))
}

func assertNoTempLeftover(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".lazyenv-*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, matches, "no orphan temp file must remain")
}

func TestWriteFileAtomicCreatesFileWithPerm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.new")
	data := []byte("FOO=bar\n")

	require.NoError(t, WriteFileAtomic(path, data, 0640))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm())

	assertNoTempLeftover(t, dir)
}

func TestWriteFileAtomicOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("OLD=1\n"), 0644))

	require.NoError(t, WriteFileAtomic(path, []byte("NEW=2\n"), 0644))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("NEW=2\n"), got)
	assertNoTempLeftover(t, dir)
}

func TestWriteFileAtomicTempCreateFails(t *testing.T) {
	err := WriteFileAtomic("/nonexistent/dir/.env", []byte("x"), 0644)
	require.Error(t, err)
	assert.ErrorContains(t, err, "creating temp file")
}

func TestWriteFileAtomicRenameFailsCleansUpTemp(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "iam-a-dir")
	require.NoError(t, os.Mkdir(target, 0755))

	err := WriteFileAtomic(target, []byte("x"), 0644)
	require.Error(t, err)
	assert.ErrorContains(t, err, "renaming temp file")
	assertNoTempLeftover(t, dir)
}

func TestWriteFileNoPath(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n"), config.SecretsConfig{})
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
	NormalizeForWrite(ef)

	assert.Equal(t, model.QuoteDouble, ef.Vars[0].QuoteStyle)
}

func TestNormalizeForWriteQuoteNoneRoundTrips(t *testing.T) {
	cases := []struct {
		name  string
		value string
		quote model.QuoteStyle
	}{
		{"newline", "line1\nline2\nline3", model.QuoteDouble},
		{"cr", "before\rafter", model.QuoteDouble},
		{"crlf", "line1\r\nline2\r\nline3", model.QuoteDouble},
		{"inner tab", "col1\tcol2", model.QuoteNone},
		{"leading tab", "\tvalue", model.QuoteNone},
		{"tab only", "\t", model.QuoteDouble},
		{"trailing tab", "value\t", model.QuoteDouble},
		{"tab before hash", "value\t#not-a-comment", model.QuoteDouble},
		{"leading tab before hash", "\t#not-a-comment", model.QuoteDouble},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ef := ParseBytes(".env", []byte("EXISTING=ok\n"), config.SecretsConfig{})
			ef.AddVar("V", tc.value)

			NormalizeForWrite(ef)
			out := string(Marshal(ef))
			re := ParseBytes(".env", []byte(out), config.SecretsConfig{})

			require.Len(t, re.Vars, 2, "no phantom vars must leak out of the round-trip")
			v := re.VarByKey("V")
			require.NotNil(t, v)
			assert.Equal(t, tc.value, v.Value)
			assert.Equal(t, tc.quote, v.QuoteStyle)
			assert.Empty(t, v.Comment)
		})
	}
}

func TestAnUntouchedUnquotedValueWithATabSavesVerbatim(t *testing.T) {
	ef := ParseBytes(".env", []byte("A=1\nK=a\tb\nL=\tc\n"), config.SecretsConfig{})
	ef.UpdateVar(0, "2")

	NormalizeForWrite(ef)

	assert.Equal(t, "A=2\nK=a\tb\nL=\tc\n", string(Marshal(ef)))
}

func TestAVarAddedToAnEmptyFileEndsWithANewline(t *testing.T) {
	ef := ParseBytes(".env", nil, config.SecretsConfig{})
	ef.AddVar("KEY", "value")
	assert.Equal(t, "KEY=value\n", string(Marshal(ef)))
}

func TestAnEmptiedFileSavesEmpty(t *testing.T) {
	ef := ParseBytes(".env", nil, config.SecretsConfig{})
	ef.AddVar("KEY", "value")
	ef.DeleteVar(0)
	assert.Empty(t, Marshal(ef))
}

func TestAParsedFileEmptiedSavesEmpty(t *testing.T) {
	ef := ParseBytes(".env", []byte("A=1\n"), config.SecretsConfig{})
	ef.DeleteVar(0)
	assert.Empty(t, Marshal(ef))
}
