package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/tui"
)

func TestVersionIsDevSentinel(t *testing.T) {
	assert.Equal(t, "dev", version,
		`main.go must keep version = "dev"; releases inject the git tag at link time via GoReleaser ldflags (-X main.version=...)`)
}

func TestFileListWidthFlagBelowMinimumReturnsNoNotice(t *testing.T) {
	width := 5
	cli.FileListWidth = &width
	t.Cleanup(func() { cli.FileListWidth = nil })

	cfg := config.DefaultConfig()
	notices := applyCLIOverrides(&cfg)

	assert.Empty(t, notices, "the TUI raises the width on its own, so there is nothing to report")
	assert.Equal(t, 5, cfg.Layout.FileListWidth)
}

func TestUnknownThemeFlagReturnsNotice(t *testing.T) {
	theme := "does-not-exist"
	cli.Theme = &theme
	t.Cleanup(func() { cli.Theme = nil })

	cfg := config.DefaultConfig()
	notices := applyCLIOverrides(&cfg)

	assert.Equal(t, []string{`unknown theme "does-not-exist" (see --list-themes)`}, notices)
}

func TestShowConfigReportsRejectedFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := config.DefaultConfig()

	startup := tui.StartupMessages{ConfigErrors: []string{`invalid sort value: "random"`}}

	status := showConfig(&stdout, &stderr, cfg, ".lazyenvrc", startup)

	assert.Equal(t, 1, status)
	assert.Contains(t, stderr.String(), "Config errors in .lazyenvrc:")
	assert.Contains(t, stderr.String(), `✗ invalid sort value: "random"`)
	assert.Contains(t, stderr.String(), "Showing the defaults with the command-line flags applied.")
	assert.Contains(t, stdout.String(), `sort = 'position'`, "the defaults are still printed")
}

func TestShowConfigReportsNoticesWithoutFailing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := config.DefaultConfig()

	startup := tui.StartupMessages{Notices: []string{`unknown theme "nope" (see --list-themes)`}}

	status := showConfig(&stdout, &stderr, cfg, ".lazyenvrc", startup)

	assert.Equal(t, 0, status, "a notice adjusts the run, it does not reject the file")
	assert.Contains(t, stderr.String(), `unknown theme "nope" (see --list-themes)`)
	assert.NotContains(t, stderr.String(), "Config errors")
	assert.NotContains(t, stderr.String(), "Showing the defaults")
}

func TestShowConfigReportsBeforeTheTOML(t *testing.T) {
	var terminal bytes.Buffer
	cfg := config.DefaultConfig()

	startup := tui.StartupMessages{
		ConfigErrors: []string{`invalid sort value: "random"`},
		Notices:      []string{`unknown theme "nope" (see --list-themes)`},
	}

	showConfig(&terminal, &terminal, cfg, ".lazyenvrc", startup)

	out := terminal.String()
	tomlStart := strings.Index(out, "sort = ")
	require.NotEqual(t, -1, tomlStart, "the TOML is printed")
	for _, report := range []string{"invalid sort value", "unknown theme"} {
		at := strings.Index(out, report)
		require.NotEqual(t, -1, at, "%q is printed", report)
		assert.Less(t, at, tomlStart, "%q comes before the TOML", report)
	}
}

func TestCheckConfigRejectsAnInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".lazyenvrc")
	require.NoError(t, os.WriteFile(path, []byte("sort = \"random\"\n"), 0o644))
	var stdout, stderr bytes.Buffer

	status := checkConfig(&stdout, &stderr, dir, "")

	assert.Equal(t, 1, status)
	assert.Contains(t, stdout.String(), "✓ "+path, "the file in use is marked")
	assert.Contains(t, stderr.String(), "Config errors in "+path+":")
	assert.Contains(t, stderr.String(), `✗ invalid sort value: "random"`)
}

func TestCheckConfigAcceptsAValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".lazyenvrc")
	require.NoError(t, os.WriteFile(path, []byte("sort = \"alphabetical\"\n"), 0o644))
	var stdout, stderr bytes.Buffer

	status := checkConfig(&stdout, &stderr, dir, "")

	assert.Equal(t, 0, status)
	assert.Contains(t, stdout.String(), "Config OK: "+path)
	assert.Empty(t, stderr.String())
}

func TestCheckConfigWithoutAFile(t *testing.T) {
	// Isolate from the user's global config
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer

	status := checkConfig(&stdout, &stderr, t.TempDir(), "")

	assert.Equal(t, 0, status)
	assert.Contains(t, stdout.String(), "No configuration file found (defaults will be used).")
	assert.NotContains(t, stdout.String(), "✓", "no path is marked as in use")
}
