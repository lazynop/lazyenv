package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
	ef := ParseBytes(".env", []byte("FOO='original'\n"))
	ef.UpdateVar(0, "updated value")

	output := string(Marshal(ef))
	assert.Equal(t, "FOO='updated value'\n", output)
}

func TestReconstructLineExportWithComment(t *testing.T) {
	ef := ParseBytes(".env", []byte("export FOO=bar # keep this\n"))
	ef.UpdateVar(0, "baz")

	output := string(Marshal(ef))
	assert.Equal(t, "export FOO=baz # keep this\n", output)
}

func TestReconstructLineDoubleQuotedEscape(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=\"old\"\n"))
	ef.UpdateVar(0, "has \"quotes\" and\nnewline")

	output := string(Marshal(ef))
	assert.Equal(t, "FOO=\"has \\\"quotes\\\" and\\nnewline\"\n", output)
}

func TestMarshalPreservesUnmodified(t *testing.T) {
	input := "# Header comment\n\nFOO=bar\nBAZ=qux\n"
	ef := ParseBytes(".env", []byte(input))
	// No modifications — output should be identical
	assert.Equal(t, input, string(Marshal(ef)))
}

func TestWriteFileNoPath(t *testing.T) {
	ef := ParseBytes(".env", []byte("FOO=bar\n"))
	// Path is empty by default from ParseBytes
	err := WriteFile(ef)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no file path set")
}
