package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionIsDevSentinel(t *testing.T) {
	assert.Equal(t, "dev", version,
		`main.go must keep version = "dev"; releases inject the git tag at link time via GoReleaser ldflags (-X main.version=...)`)
}
