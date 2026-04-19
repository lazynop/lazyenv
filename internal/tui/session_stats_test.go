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

func TestStats_DeleteExistingFile(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordDelete("/p/.env")

	assert.Equal(t, []string{"/p/.env — deleted"}, s.Summary())
}

func TestStats_DeleteUnloadedPath_NotReported(t *testing.T) {
	// Deleting a file we never had in `initial` (shouldn't happen in practice).
	s := NewSessionStats()
	s.RecordDelete("/p/.env")
	assert.Empty(t, s.Summary())
}

func TestStats_CreateScratch(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.new", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
		{Key: "BAZ", Value: "3"},
	})
	assert.Equal(t, []string{"/p/.env.new — new file (3 variables)"}, s.Summary())
}

func TestStats_CreateScratch_OneVariable_SingularWording(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.new", []model.EnvVar{{Key: "FOO", Value: "1"}})
	assert.Equal(t, []string{"/p/.env.new — new file (1 variable)"}, s.Summary())
}

func TestStats_CreateTemplate(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateTemplate("/p/.env.example", "/p/.env.prod", []model.EnvVar{
		{Key: "FOO"},
		{Key: "BAR"},
	})
	assert.Equal(t, []string{
		"/p/.env.example — from template /p/.env.prod (2 variables)",
	}, s.Summary())
}

func TestStats_CreateScratch_ThenSaveMoreVars(t *testing.T) {
	// Post-create saves update final; count reflects final.
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.new", nil)
	s.RecordSave("/p/.env.new", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	assert.Equal(t, []string{"/p/.env.new — new file (2 variables)"}, s.Summary())
}

func TestStats_CreateDuplicate_NoEdits(t *testing.T) {
	s := NewSessionStats()
	src := []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	}
	s.RecordCreateDuplicate("/p/.env.copy", "/p/.env.local", src)
	assert.Equal(t, []string{
		"/p/.env.copy — duplicated from /p/.env.local (2 variables)",
	}, s.Summary())
}

func TestStats_CreateDuplicate_WithEdits(t *testing.T) {
	s := NewSessionStats()
	// Create a duplicate of a 2-var file.
	s.RecordCreateDuplicate("/p/.env.copy", "/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	// After edits: changed FOO, deleted BAR, added BAZ.
	s.RecordSave("/p/.env.copy", []model.EnvVar{
		{Key: "FOO", Value: "99"},
		{Key: "BAZ", Value: "3"},
	})
	assert.Equal(t, []string{
		"/p/.env.copy — duplicated from /p/.env.local, 1 added, 1 changed, 1 deleted",
	}, s.Summary())
}

func TestStats_CreateThenDelete_NetZero(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.new", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordDelete("/p/.env.new")
	assert.Empty(t, s.Summary())
}
