package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

func TestCheckGitIgnoreMarksUncoveredFiles(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":       "FOO=bar\n",
		".env.local": "LOCAL=1\n",
		".gitignore": ".env.local\n",
	})
	initGitRepo(t, dir)

	files, err := ScanDir(dir, false, config.DefaultConfig().Files)
	require.NoError(t, err)
	require.Len(t, files, 2)

	CheckGitIgnore(files)

	for _, f := range files {
		if f.Name == ".env" {
			assert.True(t, f.GitWarning, ".env should have GitWarning (not in .gitignore)")
		}
		if f.Name == ".env.local" {
			assert.False(t, f.GitWarning, ".env.local should NOT have GitWarning (in .gitignore)")
		}
	}
}

func TestCheckGitIgnoreAllIgnored(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":       "FOO=bar\n",
		".env.local": "LOCAL=1\n",
		".gitignore": ".env\n.env.local\n",
	})
	initGitRepo(t, dir)

	files, err := ScanDir(dir, false, config.DefaultConfig().Files)
	require.NoError(t, err)

	CheckGitIgnore(files)

	for _, f := range files {
		assert.False(t, f.GitWarning, "%s should NOT have GitWarning", f.Name)
	}
}

func TestCheckGitIgnoreNotAGitRepo(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env": "FOO=bar\n",
	})
	// NOT a git repo — no git init

	files, err := ScanDir(dir, false, config.DefaultConfig().Files)
	require.NoError(t, err)

	CheckGitIgnore(files)

	// Should silently do nothing (no warnings)
	for _, f := range files {
		assert.False(t, f.GitWarning, "should not warn outside a git repo")
	}
}

func TestCheckGitIgnoreEmptyFiles(t *testing.T) {
	CheckGitIgnore(nil)
	// Should not panic
}

func TestCheckGitIgnoreWildcard(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":            "FOO=bar\n",
		".env.local":      "LOCAL=1\n",
		".env.production": "PROD=1\n",
		".gitignore":      ".env*\n",
	})
	initGitRepo(t, dir)

	files, err := ScanDir(dir, false, config.DefaultConfig().Files)
	require.NoError(t, err)
	require.Len(t, files, 3)

	CheckGitIgnore(files)

	for _, f := range files {
		assert.False(t, f.GitWarning, "%s should be covered by .env* wildcard", f.Name)
	}
}

func TestCheckGitIgnoreSubdirectory(t *testing.T) {
	dir := setupScanDir(t, map[string]string{
		".env":       "FOO=bar\n",
		"sub/.env":   "SUB=1\n",
		".gitignore": ".env\n",
	})
	initGitRepo(t, dir)

	files, err := ScanDir(dir, true, config.DefaultConfig().Files)
	require.NoError(t, err)
	require.Len(t, files, 2)

	CheckGitIgnore(files)

	for _, f := range files {
		name := filepath.Base(f.Path)
		if name == ".env" {
			assert.False(t, f.GitWarning, "%s should be covered by .gitignore", f.Path)
		}
	}
}
