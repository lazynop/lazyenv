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
	}, got.values)
	assert.Equal(t, []string{"BAR", "FOO"}, got.order)
}

func TestSnapshot_Empty(t *testing.T) {
	got := snapshot(nil)
	assert.Empty(t, got.values)
	assert.Empty(t, got.order)
}

func TestStats_ExistingFile_SaveDiff(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
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
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary())
}

func TestStats_ExistingFile_EditedAndReverted_NotReported(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "2"}})
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary())
}

func TestStats_ExistingFile_ReorderOnly(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "1"},
	})

	assert.Equal(t, []string{"/p/.env — reordered"}, s.Summary())
}

func TestStats_ExistingFile_ReorderWithEdits(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "BAZ", Value: "3"},
		{Key: "FOO", Value: "1"},
	})

	assert.Equal(t, []string{"/p/.env — 1 added, 0 changed, 0 deleted, reordered"}, s.Summary())
}

func TestStats_ExistingFile_AddAndDelete_NotReordered(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
		{Key: "QUX", Value: "4"},
	})
	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "BAZ", Value: "3"},
		{Key: "FOO", Value: "1"},
		{Key: "QUX", Value: "4"},
	})

	assert.Equal(t, []string{"/p/.env — 1 added, 0 changed, 1 deleted"}, s.Summary())
}

func TestStats_ExistingFile_DuplicateKeyOrderFollowsLastOccurrence(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "old"},
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "1"},
	})
	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "old"},
		{Key: "FOO", Value: "1"},
	})
	assert.Empty(t, s.Summary(), "moving the shadowed FOO is no reorder: a key's position is its last occurrence, the one a shell keeps")

	s.RecordSave("/p/.env", []model.EnvVar{
		{Key: "FOO", Value: "old"},
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	assert.Equal(t, []string{"/p/.env — reordered"}, s.Summary(), "moving the last FOO is a reorder")
}

func TestStats_InitialLoadIdempotent(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "DIFFERENT"}})
	s.RecordSave("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Empty(t, s.Summary(), "saving the first snapshot's value must report nothing: against the second load's DIFFERENT it would report 1 changed")
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
	s.RecordCreateDuplicate("/p/.env.copy", "/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordSave("/p/.env.copy", []model.EnvVar{
		{Key: "FOO", Value: "99"},
		{Key: "BAZ", Value: "3"},
	})
	assert.Equal(t, []string{
		"/p/.env.copy — duplicated from /p/.env.local, 1 added, 1 changed, 1 deleted",
	}, s.Summary())
}

func TestStats_CreateDuplicate_Reordered(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateDuplicate("/p/.env.copy", "/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordSave("/p/.env.copy", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "1"},
	})
	assert.Equal(t, []string{
		"/p/.env.copy — duplicated from /p/.env.local, reordered",
	}, s.Summary())
}

func TestStats_CreateThenDelete_NetZero(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.new", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordDelete("/p/.env.new")
	assert.Empty(t, s.Summary())
}

func TestStats_Rename_NoEdits(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env.local", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/.env.local", "/p/.env.dev")
	assert.Equal(t, []string{
		"/p/.env.dev (renamed from /p/.env.local) — 0 added, 0 changed, 0 deleted",
	}, s.Summary(), "a renamed file is reported without a save after the rename, compared with its initial load")
}

func TestStats_RenameThenRecreateAtOriginPath(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/.env", "/p/.env.backup")
	s.RecordCreateScratch("/p/.env", []model.EnvVar{{Key: "NEW", Value: "v"}})
	assert.Equal(t, []string{
		"/p/.env — new file (1 variable)",
		"/p/.env.backup (renamed from /p/.env) — 0 added, 0 changed, 0 deleted",
	}, s.Summary())
}

func TestStats_Rename_WithEdits(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordSave("/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordRename("/p/.env.local", "/p/.env.dev")
	s.RecordSave("/p/.env.dev", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "changed"},
		{Key: "BAZ", Value: "3"},
	})
	assert.Equal(t, []string{
		"/p/.env.dev (renamed from /p/.env.local) — 1 added, 1 changed, 0 deleted",
	}, s.Summary())
}

func TestStats_Rename_Reordered(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env.local", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordRename("/p/.env.local", "/p/.env.dev")
	s.RecordSave("/p/.env.dev", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "1"},
	})
	assert.Equal(t, []string{
		"/p/.env.dev (renamed from /p/.env.local) — reordered",
	}, s.Summary())
}

func TestStats_RenameChain(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/a", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/a", "/p/b")
	s.RecordRename("/p/b", "/p/c")
	s.RecordSave("/p/c", []model.EnvVar{{Key: "FOO", Value: "1"}})

	assert.Equal(t, []string{
		"/p/c (renamed from /p/a) — 0 added, 0 changed, 0 deleted",
	}, s.Summary())
}

func TestStats_RenameBackToOrigin(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/a", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/a", "/p/b")
	s.RecordRename("/p/b", "/p/a")
	assert.Empty(t, s.Summary())
}

func TestStats_RenameBackToOrigin_WithEdits(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/a", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/a", "/p/b")
	s.RecordRename("/p/b", "/p/a")
	s.RecordSave("/p/a", []model.EnvVar{{Key: "FOO", Value: "changed"}})

	assert.Equal(t, []string{"/p/a — 0 added, 1 changed, 0 deleted"}, s.Summary())
}

func TestStats_RenameBackToOrigin_AfterReorder(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/a", []model.EnvVar{
		{Key: "FOO", Value: "1"},
		{Key: "BAR", Value: "2"},
	})
	s.RecordRename("/p/a", "/p/b")
	s.RecordSave("/p/b", []model.EnvVar{
		{Key: "BAR", Value: "2"},
		{Key: "FOO", Value: "1"},
	})
	s.RecordRename("/p/b", "/p/a")

	assert.Equal(t, []string{"/p/a — reordered"}, s.Summary())
}

func TestStats_RenameThenDelete_ReportsOriginDeleted(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/a", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/a", "/p/b")
	s.RecordDelete("/p/b")

	assert.Equal(t, []string{"/p/a — deleted"}, s.Summary())
}

func TestStats_CreateThenRename_KeepsCreateOrigin(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.fresh", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/.env.fresh", "/p/.env.renamed")

	assert.Equal(t, []string{
		"/p/.env.renamed — new file (1 variable)",
	}, s.Summary())
}

func TestStats_CreateThenRenameThenDelete_NetZero(t *testing.T) {
	s := NewSessionStats()
	s.RecordCreateScratch("/p/.env.fresh", []model.EnvVar{{Key: "FOO", Value: "1"}})
	s.RecordRename("/p/.env.fresh", "/p/.env.renamed")
	s.RecordDelete("/p/.env.renamed")

	assert.Empty(t, s.Summary())
}

func TestStats_Format_Empty(t *testing.T) {
	s := NewSessionStats()
	assert.Equal(t, "", s.Format())
}

func TestStats_Format_MultipleFilesAlphabetical(t *testing.T) {
	s := NewSessionStats()
	s.RecordInitialLoad("/p/.env.staging", []model.EnvVar{{Key: "X", Value: "1"}})
	s.RecordInitialLoad("/p/.env.local", []model.EnvVar{{Key: "Y", Value: "1"}})
	s.RecordSave("/p/.env.staging", []model.EnvVar{{Key: "X", Value: "2"}})
	s.RecordCreateScratch("/p/.env.new", []model.EnvVar{{Key: "Z", Value: "1"}})
	s.RecordDelete("/p/.env.local")

	got := s.Format()
	want := "Session summary:\n" +
		"  /p/.env.local — deleted\n" +
		"  /p/.env.new — new file (1 variable)\n" +
		"  /p/.env.staging — 0 added, 1 changed, 0 deleted\n"
	assert.Equal(t, want, got)
}
