package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
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

	status := showConfig(&stdout, &stderr, cfg, []string{`invalid sort value: "random"`}, nil)

	assert.Equal(t, 1, status)
	assert.Contains(t, stderr.String(), `✗ invalid sort value: "random"`)
	assert.Contains(t, stdout.String(), `sort = 'position'`, "the defaults are still printed")
}

func TestShowConfigReportsNoticesWithoutFailing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := config.DefaultConfig()

	status := showConfig(&stdout, &stderr, cfg, nil, []string{`unknown theme "nope" (see --list-themes)`})

	assert.Equal(t, 0, status, "a notice adjusts the run, it does not reject the file")
	assert.Contains(t, stderr.String(), `unknown theme "nope" (see --list-themes)`)
	assert.NotContains(t, stderr.String(), "rejected")
}

func TestShowConfigReportsBeforeTheTOML(t *testing.T) {
	var terminal bytes.Buffer
	cfg := config.DefaultConfig()

	showConfig(&terminal, &terminal, cfg, []string{`invalid sort value: "random"`}, []string{`unknown theme "nope" (see --list-themes)`})

	out := terminal.String()
	tomlStart := strings.Index(out, "sort = ")
	require.NotEqual(t, -1, tomlStart, "the TOML is printed")
	for _, report := range []string{"invalid sort value", "unknown theme"} {
		at := strings.Index(out, report)
		require.NotEqual(t, -1, at, "%q is printed", report)
		assert.Less(t, at, tomlStart, "%q comes before the TOML", report)
	}
}
