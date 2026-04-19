package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/model"
)

func TestSnapshot_LastWinsForDuplicateKeys(t *testing.T) {
	vars := []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "override"},
	}
	got := snapshot(vars)
	assert.Equal(t, map[string]string{
		"FOO": "override",
		"BAR": "2",
	}, got)
}

func TestSnapshot_Empty(t *testing.T) {
	got := snapshot(nil)
	assert.Equal(t, map[string]string{}, got)
}
