package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMatrixEmpty(t *testing.T) {
	entries, names := ComputeMatrix(nil)
	assert.Empty(t, entries)
	assert.Empty(t, names)
}

func TestComputeMatrixSingleFile(t *testing.T) {
	f := newTestFile(
		EnvVar{Key: "A", Value: "1"},
		EnvVar{Key: "B", Value: "2"},
	)
	f.Name = "one.env"
	entries, names := ComputeMatrix([]*EnvFile{f})
	assert.Equal(t, []string{"one.env"}, names)
	require.Len(t, entries, 2)
	// alphabetical order
	assert.Equal(t, "A", entries[0].Key)
	assert.Equal(t, "B", entries[1].Key)
	assert.Equal(t, []bool{true}, entries[0].Present)
	assert.Equal(t, []string{"1"}, entries[0].Values)
}

func TestComputeMatrixMultipleFiles(t *testing.T) {
	f1 := newTestFile(
		EnvVar{Key: "DB_HOST", Value: "localhost"},
		EnvVar{Key: "API_KEY", Value: "abc"},
	)
	f1.Name = ".env"
	f2 := newTestFile(
		EnvVar{Key: "DB_HOST", Value: "prod-host"},
		EnvVar{Key: "REDIS_URL", Value: "redis://x"},
	)
	f2.Name = ".env.prod"
	entries, names := ComputeMatrix([]*EnvFile{f1, f2})
	assert.Equal(t, []string{".env", ".env.prod"}, names)
	require.Len(t, entries, 3)

	// alphabetical: API_KEY, DB_HOST, REDIS_URL
	assert.Equal(t, "API_KEY", entries[0].Key)
	assert.Equal(t, []bool{true, false}, entries[0].Present)
	assert.Equal(t, []string{"abc", ""}, entries[0].Values)

	assert.Equal(t, "DB_HOST", entries[1].Key)
	assert.Equal(t, []bool{true, true}, entries[1].Present)

	assert.Equal(t, "REDIS_URL", entries[2].Key)
	assert.Equal(t, []bool{false, true}, entries[2].Present)
	assert.Equal(t, []string{"", "redis://x"}, entries[2].Values)
}

func TestComputeMatrixDuplicateKeys(t *testing.T) {
	// When a file has duplicate keys, last value wins (shell semantics)
	f := newTestFile(
		EnvVar{Key: "A", Value: "first"},
		EnvVar{Key: "A", Value: "second"},
	)
	f.Name = "dup.env"
	entries, _ := ComputeMatrix([]*EnvFile{f})
	require.Len(t, entries, 1)
	assert.Equal(t, "A", entries[0].Key)
	assert.Equal(t, []string{"second"}, entries[0].Values)
}

func TestSortByCompleteness(t *testing.T) {
	entries := []MatrixEntry{
		{Key: "ALL", Present: []bool{true, true, true}},
		{Key: "NONE", Present: []bool{false, false, false}},
		{Key: "SOME", Present: []bool{true, false, true}},
	}
	SortEntries(entries, SortCompleteness)
	// most gaps first: NONE (0/3), SOME (2/3), ALL (3/3)
	assert.Equal(t, "NONE", entries[0].Key)
	assert.Equal(t, "SOME", entries[1].Key)
	assert.Equal(t, "ALL", entries[2].Key)
}

func TestSortByAlpha(t *testing.T) {
	entries := []MatrixEntry{
		{Key: "C"},
		{Key: "A"},
		{Key: "B"},
	}
	SortEntries(entries, SortAlpha)
	assert.Equal(t, "A", entries[0].Key)
	assert.Equal(t, "B", entries[1].Key)
	assert.Equal(t, "C", entries[2].Key)
}
