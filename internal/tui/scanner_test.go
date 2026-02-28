package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScanDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}
	return dir
}

func TestScanDirFindsEnvFiles(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":           "FOO=bar\n",
		".env.local":     "BAZ=qux\n",
		"app.env":        "APP=1\n",
		"not-an-env.txt": "ignored\n",
	})

	files, err := ScanDir(dir, false)

	require.NoError(t, err)
	require.Len(t, files, 3)

	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	assert.Contains(t, names, ".env")
	assert.Contains(t, names, ".env.local")
	assert.Contains(t, names, "app.env")
}

func TestScanDirNonRecursive(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":     "FOO=bar\n",
		"sub/.env": "SUB=val\n",
	})

	files, err := ScanDir(dir, false)

	require.NoError(t, err)
	require.Len(t, files, 1, "non-recursive should only find root .env")
}

func TestScanDirRecursive(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":          "FOO=bar\n",
		"sub/.env":      "SUB=val\n",
		"sub/deep/.env": "DEEP=val\n",
	})

	files, err := ScanDir(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 3)
}

func TestScanDirSkipsNodeModules(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":                  "FOO=bar\n",
		"node_modules/pkg/.env": "SKIP=me\n",
	})

	files, err := ScanDir(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestScanDirSkipsHiddenDirs(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":         "FOO=bar\n",
		".hidden/.env": "SKIP=me\n",
	})

	files, err := ScanDir(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestScanDirSortsByDepthThenName(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		"sub/.env.local": "A=1\n",
		"sub/.env":       "B=2\n",
		".env.prod":      "C=3\n",
		".env":           "D=4\n",
	})

	files, err := ScanDir(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 4)
	// Root files first (sorted by name), then sub/ files
	assert.Equal(t, ".env", files[0].Name)
	assert.Equal(t, ".env.prod", files[1].Name)
}

func TestScanDirEmptyDir(t *testing.T) {
	dir := t.TempDir()

	files, err := ScanDir(dir, false)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestIsEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"app.env", true},
		{"config.txt", false},
		{"env", false},
		{".environment", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isEnvFile(tt.name), "isEnvFile(%q)", tt.name)
	}
}
