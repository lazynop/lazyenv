package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

func newTestFile(vars ...EnvVar) *EnvFile {
	ef := &EnvFile{Name: "test.env"}
	for i, v := range vars {
		v.LineNum = i + 1
		ef.Vars = append(ef.Vars, v)
		ef.Lines = append(ef.Lines, RawLine{
			Type:    LineVariable,
			Content: v.Key + "=" + v.Value,
			VarIdx:  i,
		})
	}
	return ef
}

func TestUpdateVar(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "old"},
		EnvVar{Key: "BAR", Value: "keep"},
	)

	ef.UpdateVar(0, "new")

	assert.Equal(t, "new", ef.Vars[0].Value)
	assert.True(t, ef.Vars[0].Modified)
	assert.Equal(t, "old", ef.Vars[0].OriginalValue)
	assert.True(t, ef.Modified)
	assert.Equal(t, "keep", ef.Vars[1].Value)
	assert.False(t, ef.Vars[1].Modified)
}

func TestUpdateVarPreservesOriginalOnSecondEdit(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "original"})

	ef.UpdateVar(0, "first_edit")
	ef.UpdateVar(0, "second_edit")

	assert.Equal(t, "second_edit", ef.Vars[0].Value)
	assert.Equal(t, "original", ef.Vars[0].OriginalValue, "OriginalValue should not change on subsequent edits")
}

func TestUpdateVarSetsIsEmpty(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "val"})

	ef.UpdateVar(0, "")

	assert.True(t, ef.Vars[0].IsEmpty)
}

func TestUpdateVarDetectsASecretValue(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.UpdateVar(0, "ghp_abc123")
	assert.True(t, ef.Vars[0].IsSecret, "a token should be a secret")

	ef.UpdateVar(0, "plain")
	assert.False(t, ef.Vars[0].IsSecret)
}

func TestUpdateVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.UpdateVar(-1, "nope")
	ef.UpdateVar(99, "nope")

	assert.Equal(t, "bar", ef.Vars[0].Value)
	assert.False(t, ef.Modified)
}

func TestAddVar(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.AddVar("NEW", "val")

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "NEW", ef.Vars[1].Key)
	assert.Equal(t, "val", ef.Vars[1].Value)
	assert.True(t, ef.Vars[1].Modified)
	assert.True(t, ef.Vars[1].IsNew)
	assert.Empty(t, ef.Vars[1].OriginalValue, "new vars should have no OriginalValue")
	assert.True(t, ef.Modified)
	require.Len(t, ef.Lines, 2)
	assert.Equal(t, LineVariable, ef.Lines[1].Type)
	assert.Equal(t, 1, ef.Lines[1].VarIdx)
}

func TestAddVarDetectsASecret(t *testing.T) {
	ef := newTestFile()

	ef.AddVar("SECRET_KEY", "val")
	assert.True(t, ef.Vars[0].IsSecret, "a secret key should be a secret")

	ef.AddVar("TOKENLESS", "ghp_abc123")
	assert.True(t, ef.Vars[1].IsSecret, "a token should be a secret")

	ef.AddVar("NORMAL", "val")
	assert.False(t, ef.Vars[2].IsSecret)
}

func TestAddVarEmpty(t *testing.T) {
	ef := newTestFile()

	ef.AddVar("KEY", "")

	require.Len(t, ef.Vars, 1)
	assert.True(t, ef.Vars[0].IsEmpty)
}

func TestDeleteVar(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAR", Value: "2"},
		EnvVar{Key: "BAZ", Value: "3"},
	)

	ef.DeleteVar(1) // delete BAR

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.Equal(t, "BAZ", ef.Vars[1].Key)
	assert.True(t, ef.Modified)

	// VarIdx references should be updated
	for _, line := range ef.Lines {
		if line.Type == LineVariable {
			assert.True(t, line.VarIdx >= 0 && line.VarIdx < len(ef.Vars),
				"VarIdx %d out of range", line.VarIdx)
		}
	}
}

func TestDeleteVarTracksDeleted(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAR", Value: "2"},
	)

	ef.DeleteVar(1) // delete BAR

	require.Len(t, ef.DeletedVars, 1)
	assert.Equal(t, "BAR", ef.DeletedVars[0].Key)
	assert.Equal(t, "2", ef.DeletedVars[0].Value)
}

func TestDeleteNewVarNotTracked(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "1"})
	ef.AddVar("NEW", "val")
	require.Len(t, ef.Vars, 2)

	ef.DeleteVar(1) // delete the newly added var

	assert.Empty(t, ef.DeletedVars, "newly added vars should not appear in DeletedVars")
}

func TestDeleteVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.DeleteVar(-1)
	ef.DeleteVar(99)

	require.Len(t, ef.Vars, 1)
	assert.False(t, ef.Modified)
}

func TestRenameVar(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.RenameVar(0, "BAZ")

	assert.Equal(t, "BAZ", ef.Vars[0].Key)
	assert.True(t, ef.Vars[0].Modified, "renamed var must be marked modified")
	assert.True(t, ef.Modified, "file must be marked modified")
}

func TestRenameVarDetectsASecretKey(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.RenameVar(0, "API_TOKEN")
	assert.True(t, ef.Vars[0].IsSecret, "a secret key should be a secret")

	ef.RenameVar(0, "PLAIN")
	assert.False(t, ef.Vars[0].IsSecret)
}

// TestChangesDetectPlaceholders edits and adds values: a variable is a
// placeholder exactly when its new value is one.
func TestChangesDetectPlaceholders(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "API_URL", Value: "changeme", IsPlaceholder: true},
		EnvVar{Key: "GREETING", Value: "hello"},
	)

	ef.UpdateVar(0, "https://api.example.com")
	ef.UpdateVar(1, "changeme")
	ef.AddVar("NEW_VAR", "TODO")

	assert.False(t, ef.Vars[0].IsPlaceholder, "a real value should drop the placeholder flag")
	assert.True(t, ef.Vars[1].IsPlaceholder, "a placeholder value should set it")
	assert.True(t, ef.Vars[2].IsPlaceholder, "an added placeholder should have it")
}

// TestChangesDetectSecretsWithTheFileConfig changes the variables of a file
// whose [secrets] config makes keys ending in _HOST secrets and keys starting
// with PUBLIC_ never secrets: each changed variable is a secret as that config
// judges it, the other way round from the built-in rules.
func TestChangesDetectSecretsWithTheFileConfig(t *testing.T) {
	tests := []struct {
		name   string
		change func(ef *EnvFile) // changes the var that then has key want
		want   string
		secret bool
	}{
		{"edit, secret key", func(ef *EnvFile) { ef.UpdateVar(0, "alpha") }, "DB_HOST", true},
		{"edit, safe key", func(ef *EnvFile) { ef.UpdateVar(1, "ghp_abc123") }, "PUBLIC_TOKEN", false},
		{"rename to a secret key", func(ef *EnvFile) { ef.RenameVar(2, "API_HOST") }, "API_HOST", true},
		{"rename to a safe key", func(ef *EnvFile) { ef.RenameVar(1, "PUBLIC_KEY") }, "PUBLIC_KEY", false},
		{"add, secret key", func(ef *EnvFile) { ef.AddVar("CACHE_HOST", "alpha") }, "CACHE_HOST", true},
		{"add, safe key", func(ef *EnvFile) { ef.AddVar("PUBLIC_KEY", "ghp_abc123") }, "PUBLIC_KEY", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := newTestFile(
				EnvVar{Key: "DB_HOST", Value: "localhost"},
				EnvVar{Key: "PUBLIC_TOKEN", Value: "ghp_xyz789"},
				EnvVar{Key: "GREETING", Value: "hello"},
			)
			ef.Secrets = config.SecretsConfig{ExtraPatterns: []string{"_HOST"}, SafePatterns: []string{"PUBLIC_"}}

			tt.change(ef)

			v := ef.VarByKey(tt.want)
			require.NotNil(t, v)
			assert.Equal(t, tt.secret, v.IsSecret)
		})
	}
}

// revivals are the ways a variable revives API_KEY, deleted from the file
// deletedAPIKeyFile returns: adding the key again, renaming BAR onto it, or
// renaming onto it a variable added this session.
var revivals = map[string]func(ef *EnvFile){
	"add":    func(ef *EnvFile) { ef.AddVar("API_KEY", "2") },
	"rename": func(ef *EnvFile) { ef.RenameVar(0, "API_KEY") }, // BAR
	"rename a var added this session": func(ef *EnvFile) {
		ef.AddVar("NEW", "2")
		ef.RenameVar(ef.LastIndexOfKey("NEW"), "API_KEY")
	},
}

// deletedAPIKeyFile returns a file that held API_KEY=1, a secret, and BAR=2,
// with API_KEY deleted.
func deletedAPIKeyFile() *EnvFile {
	ef := newTestFile(
		EnvVar{Key: "API_KEY", Value: "1", IsSecret: true},
		EnvVar{Key: "BAR", Value: "2"},
	)
	ef.DeleteVar(0)
	return ef
}

// deletedVars returns the variables ef lists as deleted, in order, each as
// KEY=value.
func deletedVars(ef *EnvFile) []string {
	var deleted []string
	for _, d := range ef.DeletedVars {
		deleted = append(deleted, d.Key+"="+d.Value)
	}
	return deleted
}

// TestTakingADeletedKeyRevivesIt gives a key deleted this session back to a
// variable: the deleted entry goes, and the variable takes the deleted one's
// original and is no longer new, as the deleted one was not.
func TestTakingADeletedKeyRevivesIt(t *testing.T) {
	for name, revive := range revivals {
		t.Run(name, func(t *testing.T) {
			ef := deletedAPIKeyFile()

			revive(ef)

			assert.Empty(t, ef.DeletedVars, "the key should leave the deleted vars")
			v := ef.VarByKey("API_KEY")
			require.NotNil(t, v)
			assert.Equal(t, "2", v.Value)
			assert.Equal(t, "1", v.OriginalValue, "the original should be the deleted var's")
			assert.True(t, v.OriginalIsSecret, "the original should keep the deleted var's secret status")
			assert.False(t, v.IsNew, "the var should no longer be new")
			assert.True(t, v.Modified)
		})
	}
}

// TestRenamingARevivedVarOffItsKeyUndoesTheRevival renames the variable that
// revived a deleted key to another key, then to a third one: the deleted
// variable is listed again, once, and the variable gets back the original it
// had before the revival.
func TestRenamingARevivedVarOffItsKeyUndoesTheRevival(t *testing.T) {
	want := map[string]struct {
		original string
		isNew    bool
	}{
		"add":                             {original: "", isNew: true},
		"rename":                          {original: "2", isNew: false},
		"rename a var added this session": {original: "", isNew: true},
	}
	for name, revive := range revivals {
		t.Run(name, func(t *testing.T) {
			ef := deletedAPIKeyFile()
			revive(ef)
			idx := ef.LastIndexOfKey("API_KEY")

			ef.RenameVar(idx, "BAZ")
			ef.RenameVar(idx, "QUX")

			assert.Equal(t, []string{"API_KEY=1"}, deletedVars(ef), "the deleted var should be listed again, once")
			v := ef.Vars[idx]
			assert.Equal(t, want[name].original, v.OriginalValue, "the var should get back its own original")
			assert.False(t, v.OriginalIsSecret)
			assert.Equal(t, want[name].isNew, v.IsNew)
		})
	}
}

// TestDeletingARevivedVarUndoesTheRevival deletes the variable that revived a
// deleted key: the variable deleted first is listed again with its value, and
// the reviving one is listed after it only if it was not added this session.
func TestDeletingARevivedVarUndoesTheRevival(t *testing.T) {
	want := map[string][]string{
		"add":                             {"API_KEY=1"},
		"rename":                          {"API_KEY=1", "API_KEY=2"},
		"rename a var added this session": {"API_KEY=1"},
	}
	for name, revive := range revivals {
		t.Run(name, func(t *testing.T) {
			ef := deletedAPIKeyFile()
			revive(ef)

			ef.DeleteVar(ef.LastIndexOfKey("API_KEY"))

			assert.Equal(t, want[name], deletedVars(ef))
		})
	}
}

// TestUndoingARevivalListsTheDeletedVarInItsPlace deletes FOO, then BAZ,
// renames BAR to FOO and back: FOO is listed again before BAZ, where it was.
func TestUndoingARevivalListsTheDeletedVarInItsPlace(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAZ", Value: "3"},
		EnvVar{Key: "BAR", Value: "2"},
	)
	ef.DeleteVar(0) // FOO
	ef.DeleteVar(0) // BAZ

	ef.RenameVar(0, "FOO") // BAR
	ef.RenameVar(0, "BAR")

	assert.Equal(t, []string{"FOO=1", "BAZ=3"}, deletedVars(ef))
}

// TestUndoingRevivalsInReverseOrderListsBothDeletedVars deletes FOO, then BAZ,
// renames X to BAZ and Y to FOO, then X and Y back, the first undone revival
// being BAZ's, whose place is past the end of the emptied DeletedVars: both
// are listed again, in their order.
func TestUndoingRevivalsInReverseOrderListsBothDeletedVars(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAZ", Value: "3"},
		EnvVar{Key: "X", Value: "4"},
		EnvVar{Key: "Y", Value: "5"},
	)
	ef.DeleteVar(0)        // FOO
	ef.DeleteVar(0)        // BAZ
	ef.RenameVar(0, "BAZ") // X
	ef.RenameVar(1, "FOO") // Y
	require.Empty(t, ef.DeletedVars)

	ef.RenameVar(0, "X")
	ef.RenameVar(1, "Y")

	assert.Equal(t, []string{"FOO=1", "BAZ=3"}, deletedVars(ef))
}

// TestRenameVarOntoAKeyNotDeletedKeepsItsOwnOriginal renames a variable onto a
// key no deleted variable holds, while another key is deleted: the variable's
// original is its own value, and the deleted variable stays.
func TestRenameVarOntoAKeyNotDeletedKeepsItsOwnOriginal(t *testing.T) {
	ef := deletedAPIKeyFile()

	ef.RenameVar(0, "BAZ") // BAR

	assert.Equal(t, "2", ef.Vars[0].OriginalValue)
	assert.False(t, ef.Vars[0].OriginalIsSecret)
	assert.Equal(t, []string{"API_KEY=1"}, deletedVars(ef))
}

func TestRenameVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.RenameVar(-1, "X")
	ef.RenameVar(99, "X")

	assert.Equal(t, "FOO", ef.Vars[0].Key, "out-of-range index must be a no-op")
	assert.False(t, ef.Modified, "no-op rename must not mark the file modified")
}

func TestGitWarningDefaultFalse(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	assert.False(t, ef.GitWarning, "GitWarning should default to false")
}

func TestVarByKey(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "first"},
		EnvVar{Key: "BAR", Value: "only"},
		EnvVar{Key: "FOO", Value: "second"},
	)

	v := ef.VarByKey("FOO")
	require.NotNil(t, v)
	assert.Equal(t, "second", v.Value, "returns last occurrence (shell semantics)")

	v = ef.VarByKey("BAR")
	require.NotNil(t, v)
	assert.Equal(t, "only", v.Value)

	assert.Nil(t, ef.VarByKey("MISSING"))
}

func TestLastIndexOfKey(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "first"},
		EnvVar{Key: "BAR", Value: "only"},
		EnvVar{Key: "FOO", Value: "second"},
	)

	assert.Equal(t, 2, ef.LastIndexOfKey("FOO"), "returns the last occurrence (shell semantics)")
	assert.Equal(t, 1, ef.LastIndexOfKey("BAR"))
	assert.Equal(t, -1, ef.LastIndexOfKey("MISSING"))
}

// parseTestEnv builds an EnvFile from raw text using a minimal classifier
// (the model package cannot import the parser). It is good enough for the
// controlled inputs used by the reorder tests.
func parseTestEnv(content string) *EnvFile {
	ef := &EnvFile{Name: "test.env"}
	if content == "" {
		return ef
	}
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			ef.Lines = append(ef.Lines, RawLine{Type: LineEmpty, Content: line, VarIdx: -1})
		case strings.HasPrefix(trimmed, "#"):
			ef.Lines = append(ef.Lines, RawLine{Type: LineComment, Content: line, VarIdx: -1})
		default:
			key, value, _ := strings.Cut(line, "=")
			idx := len(ef.Vars)
			ef.Vars = append(ef.Vars, EnvVar{Key: key, Value: value, LineNum: len(ef.Lines) + 1})
			ef.Lines = append(ef.Lines, RawLine{Type: LineVariable, Content: line, VarIdx: idx})
		}
	}
	return ef
}

// dumpLines reconstructs the file text from RawLine.Content, mirroring what a
// write would produce for unmodified variables.
func dumpLines(ef *EnvFile) string {
	parts := make([]string, len(ef.Lines))
	for i, l := range ef.Lines {
		parts[i] = l.Content
	}
	return strings.Join(parts, "\n")
}

func varKeys(ef *EnvFile) []string {
	out := make([]string, len(ef.Vars))
	for i, v := range ef.Vars {
		out[i] = v.Key
	}
	return out
}

func TestReorderAlphabetical(t *testing.T) {
	ef := parseTestEnv("C=3\nA=1\nB=2")
	ef.Reorder(ReorderAlphabetical)

	assert.Equal(t, "A=1\nB=2\nC=3", dumpLines(ef))
	assert.Equal(t, []string{"A", "B", "C"}, varKeys(ef))
	assert.True(t, ef.Modified)
}

func TestReorderAttachedCommentsTravel(t *testing.T) {
	ef := parseTestEnv("# c comment\nC=3\n# a comment\nA=1")
	ef.Reorder(ReorderAlphabetical)

	assert.Equal(t, "# a comment\nA=1\n# c comment\nC=3", dumpLines(ef))
}

func TestReorderLeadingBlockPreserved(t *testing.T) {
	ef := parseTestEnv("# banner\n\nB=2\nA=1")
	ef.Reorder(ReorderAlphabetical)

	assert.Equal(t, "# banner\n\nA=1\nB=2", dumpLines(ef))
}

func TestReorderTrailingBlockPreserved(t *testing.T) {
	ef := parseTestEnv("B=2\nA=1\n# trailing")
	ef.Reorder(ReorderAlphabetical)

	assert.Equal(t, "A=1\nB=2\n# trailing", dumpLines(ef))
}

func TestReorderFloatingCommentFloatsToTop(t *testing.T) {
	ef := parseTestEnv("B=2\n# note\n\nA=1")
	ef.Reorder(ReorderAlphabetical)

	assert.Equal(t, "# note\nA=1\nB=2", dumpLines(ef))
}

func TestReorderGrouped(t *testing.T) {
	ef := parseTestEnv("REDIS_PORT=1\nDB_HOST=localhost\nDB_PORT=5432\nREDIS_HOST=r")
	ef.Reorder(ReorderGrouped)

	want := "DB_HOST=localhost\nDB_PORT=5432\n\nREDIS_HOST=r\nREDIS_PORT=1"
	assert.Equal(t, want, dumpLines(ef))
}

func TestReorderGroupedUngroupedLast(t *testing.T) {
	ef := parseTestEnv("ZEBRA=1\nDB_HOST=h\nDB_PORT=p")
	ef.Reorder(ReorderGrouped)

	want := "DB_HOST=h\nDB_PORT=p\n\nZEBRA=1"
	assert.Equal(t, want, dumpLines(ef))
}

func TestReorderDuplicatesKeepRelativeOrder(t *testing.T) {
	ef := parseTestEnv("KEY=first\nKEY=second\nAAA=x")
	ef.Reorder(ReorderAlphabetical)

	// Stable sort keeps first before second, so last-wins semantics survive.
	assert.Equal(t, "AAA=x\nKEY=first\nKEY=second", dumpLines(ef))
}

func TestReorderNoVariablesIsNoOp(t *testing.T) {
	ef := parseTestEnv("# just a comment\n")
	ef.Reorder(ReorderAlphabetical)

	assert.False(t, ef.Modified)
	assert.Equal(t, "# just a comment\n", dumpLines(ef))
}
