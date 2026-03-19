package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestUpdateVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.UpdateVar(-1, "nope")
	ef.UpdateVar(99, "nope")

	assert.Equal(t, "bar", ef.Vars[0].Value)
	assert.False(t, ef.Modified)
}

func TestAddVar(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.AddVar("NEW", "val", false)

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

func TestAddVarSetsIsSecret(t *testing.T) {
	ef := newTestFile()

	ef.AddVar("SECRET_KEY", "val", true)
	assert.True(t, ef.Vars[0].IsSecret, "IsSecret should be set from parameter")

	ef.AddVar("NORMAL", "val", false)
	assert.False(t, ef.Vars[1].IsSecret)
}

func TestAddVarEmpty(t *testing.T) {
	ef := newTestFile()

	ef.AddVar("KEY", "", false)

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
	ef.AddVar("NEW", "val", false)
	require.Len(t, ef.Vars, 2)

	ef.DeleteVar(1) // delete the newly added var

	assert.Empty(t, ef.DeletedVars, "newly added vars should not appear in DeletedVars")
}

func TestReAddDeletedVarRemovesFromDeleted(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "original"},
	)

	ef.DeleteVar(0)
	require.Len(t, ef.DeletedVars, 1)

	ef.AddVar("FOO", "new_value", false)
	assert.Empty(t, ef.DeletedVars, "re-adding should remove from DeletedVars")
}

func TestReAddDeletedVarPreservesOriginalValue(t *testing.T) {
	ef := newTestFile(
		EnvVar{Key: "FOO", Value: "original"},
	)

	ef.DeleteVar(0)
	ef.AddVar("FOO", "new_value", false)

	require.Len(t, ef.Vars, 1)
	assert.Equal(t, "new_value", ef.Vars[0].Value)
	assert.Equal(t, "original", ef.Vars[0].OriginalValue, "peek should show the original value")
	assert.False(t, ef.Vars[0].IsNew, "re-added var should be treated as modified, not new")
	assert.True(t, ef.Vars[0].Modified)
}

func TestDeleteVarOutOfBounds(t *testing.T) {
	ef := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	ef.DeleteVar(-1)
	ef.DeleteVar(99)

	require.Len(t, ef.Vars, 1)
	assert.False(t, ef.Modified)
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
