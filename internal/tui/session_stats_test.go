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

func TestStats_ExistingFile_SaveDiff(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	// Save: add BAZ, change FOO, delete BAR.
	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "99"},
		{Key: "BAZ", Value: "3"},
	})

	assert.Equal(t, []string{
		"/p/.env — 1 added, 1 changed, 1 deleted",
	}, s.Summary())
}

func TestStats_ExistingFile_NoChanges_NotReported(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	// Saved but identical content → nothing to report.
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary())
}

func TestStats_ExistingFile_EditedAndReverted_NotReported(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	// Multiple saves: 1 → 2 → 1. Net-zero.
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "2"}})
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary())
}

func TestStats_InitialLoadIdempotent(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	// A second call must NOT overwrite the first snapshot.
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "DIFFERENT"}})
	// Saving with the ORIGINAL value proves the first snapshot is preserved:
	// if idempotency were broken, diff(DIFFERENT, 1) would report 1 changed.
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary())
}

func TestStats_NilReceiver_Safe(t *testing.T) {
	var s *SessionStats
	s.RecordInitialLoad("/p/.env", nil)
	s.RecordSave("/p/.env", nil)
	assert.Empty(t, s.Summary())
}
