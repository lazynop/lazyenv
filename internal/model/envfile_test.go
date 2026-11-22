package model

import (
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

	ef.DeleteVar(ef.LastIndexOfKey("BAR"))

	require.Len(t, ef.Vars, 2)
	assert.Equal(t, "FOO", ef.Vars[0].Key)
	assert.Equal(t, "BAZ", ef.Vars[1].Key)
	assert.True(t, ef.Modified)

	for _, line := range ef.Lines {
		if line.Type == LineVariable {
			assert.True(t, line.VarIdx >= 0 && line.VarIdx < len(ef.Vars),
				"VarIdx %d should be updated into range", line.VarIdx)
		}
	}
}

func TestDeleteVarTracksDeleted(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAR", Value: "2"},
	)

	ef.DeleteVar(ef.LastIndexOfKey("BAR"))

	require.Len(t, ef.DeletedVars, 1)
	assert.Equal(t, "BAR", ef.DeletedVars[0].Key)
	assert.Equal(t, "2", ef.DeletedVars[0].Value)
}

func TestDeleteNewVarNotTracked(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "1"})
	ef.AddVar("NEW", "val")
	require.Len(t, ef.Vars, 2)

	ef.DeleteVar(ef.LastIndexOfKey("NEW"))

	assert.Empty(t, ef.DeletedVars, "newly added vars should not appear in DeletedVars")
}

func TestDeleteVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.DeleteVar(-1)
	ef.DeleteVar(99)

	require.Len(t, ef.Vars, 1)
	assert.False(t, ef.Modified)
}

func TestDeleteKeyRemovesEveryOccurrence(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "A", Value: "1"},
		EnvVar{Key: "B", Value: "2"},
		EnvVar{Key: "A", Value: "3"},
		EnvVar{Key: "C", Value: "4"},
	)

	ef.DeleteKey("A")

	assert.Len(t, ef.Vars, 2, "only B and C should be left")
	assert.Equal(t, []string{"B=2", "C=4"}, lineVars(ef), "the A lines should go, the others stay in order")
	assert.Equal(t, []string{"A=3", "A=1"}, deletedVars(ef), "the A in effect should be listed first")
	assert.True(t, ef.Modified)
}

func TestDeleteKeyWithoutTheKeyLeavesTheFile(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "A", Value: "1"})

	ef.DeleteKey("B")

	assert.Equal(t, []string{"A=1"}, lineVars(ef))
	assert.Empty(t, ef.DeletedVars)
	assert.False(t, ef.Modified)
}

func TestTakingAKeyBackAfterDeleteKeyRevivesTheVarInEffect(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "A", Value: "1"},
		EnvVar{Key: "B", Value: "2"},
		EnvVar{Key: "A", Value: "3"},
	)
	ef.DeleteKey("A")

	ef.AddVar("A", "5")

	v := ef.VarByKey("A")
	require.NotNil(t, v)
	assert.Equal(t, "3", v.OriginalValue, "the original should be the A in effect")
	assert.Equal(t, []string{"A=1"}, deletedVars(ef))
}

func lineVars(ef *EnvFile) []string {
	var vars []string
	for _, l := range ef.Lines {
		if l.Type == LineVariable {
			v := ef.Vars[l.VarIdx]
			vars = append(vars, v.Key+"="+v.Value)
		}
	}
	return vars
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

func TestChangingASecretKeepsItsOriginalSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret EnvVar
		change func(ef *EnvFile)
	}{
		{"value edited to a plain one", EnvVar{Key: "GREETING", Value: "ghp_abc123", IsSecret: true},
			func(ef *EnvFile) { ef.UpdateVar(0, "hello") }},
		{"renamed to a plain key", EnvVar{Key: "API_TOKEN", Value: "hunter2", IsSecret: true},
			func(ef *EnvFile) { ef.RenameVar(0, "GREETING") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := newTestFile(tt.secret)

			tt.change(ef)

			assert.False(t, ef.Vars[0].IsSecret, "the changed var should no longer be a secret")
			assert.True(t, ef.Vars[0].OriginalIsSecret, "its original should keep its secret status, so peek still masks it")
		})
	}
}

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

func TestChangesMarkDuplicates(t *testing.T) {
	tests := []struct {
		name            string
		change          func(ef *EnvFile)
		wantIsDuplicate []bool
	}{
		{"rename one of two A away", func(ef *EnvFile) { ef.RenameVar(0, "C") }, []bool{false, false, false}},
		{"rename B onto A", func(ef *EnvFile) { ef.RenameVar(2, "A") }, []bool{true, true, true}},
		{"add a third A", func(ef *EnvFile) { ef.AddVar("A", "4") }, []bool{true, true, false, true}},
		{"add a second B", func(ef *EnvFile) { ef.AddVar("B", "4") }, []bool{true, true, true, true}},
		{"delete one of two A", func(ef *EnvFile) { ef.DeleteVar(0) }, []bool{false, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := newTestFile(
				EnvVar{Key: "A", Value: "1", IsDuplicate: true},
				EnvVar{Key: "A", Value: "2", IsDuplicate: true},
				EnvVar{Key: "B", Value: "3"},
			)

			tt.change(ef)

			var got []bool
			for _, v := range ef.Vars {
				got = append(got, v.IsDuplicate)
			}
			assert.Equal(t, tt.wantIsDuplicate, got)
		})
	}
}

func TestChangesDetectSecretsWithTheFileConfig(t *testing.T) {
	tests := []struct {
		name       string
		change     func(ef *EnvFile)
		changedKey string
		secret     bool
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

			v := ef.VarByKey(tt.changedKey)
			require.NotNil(t, v)
			assert.Equal(t, tt.secret, v.IsSecret, "the file's [secrets] config, not the built-in rules, should judge the changed var")
		})
	}
}

var revivals = map[string]func(ef *EnvFile){
	"add":    func(ef *EnvFile) { ef.AddVar("API_KEY", "2") },
	"rename": func(ef *EnvFile) { ef.RenameVar(ef.LastIndexOfKey("BAR"), "API_KEY") },
	"rename a var added this session": func(ef *EnvFile) {
		ef.AddVar("NEW", "2")
		ef.RenameVar(ef.LastIndexOfKey("NEW"), "API_KEY")
	},
}

func deletedAPIKeyFile() *EnvFile {
	ef := newTestFile(
		EnvVar{Key: "API_KEY", Value: "1", IsSecret: true},
		EnvVar{Key: "BAR", Value: "2"},
	)
	ef.DeleteVar(0)
	return ef
}

func deletedVars(ef *EnvFile) []string {
	var deleted []string
	for _, d := range ef.DeletedVars {
		deleted = append(deleted, d.Key+"="+d.Value)
	}
	return deleted
}

func TestTakingADeletedKeyRevivesIt(t *testing.T) {
	for name, revive := range revivals {
		t.Run(name, func(t *testing.T) {
			ef := deletedAPIKeyFile()

			revive(ef)

			assert.Empty(t, ef.DeletedVars, "the key should leave the deleted vars")
			v := ef.VarByKey("API_KEY")
			require.NotNil(t, v)
			assert.Equal(t, "2", v.Value)
			assert.Equal(t, "1", v.OriginalValue, "the original, which peek shows, should be the deleted var's")
			assert.True(t, v.OriginalIsSecret, "the original should keep the deleted var's secret status")
			assert.False(t, v.IsNew, "the var should no longer be new")
			assert.True(t, v.Modified)
		})
	}
}

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

func TestUndoingARevivalListsTheDeletedVarInItsPlace(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAZ", Value: "3"},
		EnvVar{Key: "BAR", Value: "2"},
	)
	ef.DeleteVar(ef.LastIndexOfKey("FOO"))
	ef.DeleteVar(ef.LastIndexOfKey("BAZ"))
	bar := ef.LastIndexOfKey("BAR")

	ef.RenameVar(bar, "FOO")
	ef.RenameVar(bar, "BAR")

	assert.Equal(t, []string{"FOO=1", "BAZ=3"}, deletedVars(ef))
}

func TestUndoingRevivalsInReverseOrderListsBothDeletedVars(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "1"},
		EnvVar{Key: "BAZ", Value: "3"},
		EnvVar{Key: "X", Value: "4"},
		EnvVar{Key: "Y", Value: "5"},
	)
	ef.DeleteVar(ef.LastIndexOfKey("FOO"))
	ef.DeleteVar(ef.LastIndexOfKey("BAZ"))
	x, y := ef.LastIndexOfKey("X"), ef.LastIndexOfKey("Y")
	ef.RenameVar(x, "BAZ")
	ef.RenameVar(y, "FOO")
	require.Empty(t, ef.DeletedVars, "BAZ's place should now be past the end of the emptied DeletedVars")

	ef.RenameVar(x, "X")
	ef.RenameVar(y, "Y")

	assert.Equal(t, []string{"FOO=1", "BAZ=3"}, deletedVars(ef))
}

func TestRenameVarOntoAKeyNotDeletedKeepsItsOwnOriginal(t *testing.T) {
	ef := deletedAPIKeyFile()

	bar := ef.LastIndexOfKey("BAR")

	ef.RenameVar(bar, "BAZ")

	assert.Equal(t, "2", ef.Vars[bar].OriginalValue)
	assert.False(t, ef.Vars[bar].OriginalIsSecret)
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
