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
