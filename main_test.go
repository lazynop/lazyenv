package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
