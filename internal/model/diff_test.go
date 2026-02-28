package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeDiffAllEqual(t *testing.T) {
	a := newTestFile(
		EnvVar{Key: "FOO", Value: "bar"},
		EnvVar{Key: "BAZ", Value: "qux"},
	)
	b := newTestFile(
		EnvVar{Key: "FOO", Value: "bar"},
		EnvVar{Key: "BAZ", Value: "qux"},
	)

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 2)
	assert.Equal(t, DiffEqual, entries[0].Status)
	assert.Equal(t, DiffEqual, entries[1].Status)
}

func TestComputeDiffChanged(t *testing.T) {
	a := newTestFile(EnvVar{Key: "FOO", Value: "old"})
	b := newTestFile(EnvVar{Key: "FOO", Value: "new"})

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 1)
	assert.Equal(t, DiffChanged, entries[0].Status)
	assert.Equal(t, "old", entries[0].ValueA)
	assert.Equal(t, "new", entries[0].ValueB)
}

func TestComputeDiffAdded(t *testing.T) {
	a := newTestFile(
		EnvVar{Key: "FOO", Value: "bar"},
		EnvVar{Key: "ONLY_A", Value: "val"},
	)
	b := newTestFile(EnvVar{Key: "FOO", Value: "bar"})

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 2)
	assert.Equal(t, DiffEqual, entries[0].Status)
	assert.Equal(t, DiffAdded, entries[1].Status)
	assert.Equal(t, "ONLY_A", entries[1].Key)
}

func TestComputeDiffRemoved(t *testing.T) {
	a := newTestFile(EnvVar{Key: "FOO", Value: "bar"})
	b := newTestFile(
		EnvVar{Key: "FOO", Value: "bar"},
		EnvVar{Key: "ONLY_B", Value: "val"},
	)

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 2)
	assert.Equal(t, DiffEqual, entries[0].Status)
	assert.Equal(t, DiffRemoved, entries[1].Status)
	assert.Equal(t, "ONLY_B", entries[1].Key)
}

func TestComputeDiffDuplicateKeys(t *testing.T) {
	a := newTestFile(
		EnvVar{Key: "FOO", Value: "first"},
		EnvVar{Key: "FOO", Value: "second"},
	)
	b := newTestFile(EnvVar{Key: "FOO", Value: "second"})

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 1)
	assert.Equal(t, DiffEqual, entries[0].Status, "should use last occurrence")
}

func TestComputeDiffEmpty(t *testing.T) {
	a := newTestFile()
	b := newTestFile()

	entries := ComputeDiff(a, b)

	assert.Empty(t, entries)
}

func TestComputeDiffMixed(t *testing.T) {
	a := newTestFile(
		EnvVar{Key: "SAME", Value: "val"},
		EnvVar{Key: "CHANGED", Value: "old"},
		EnvVar{Key: "ONLY_A", Value: "a"},
	)
	b := newTestFile(
		EnvVar{Key: "SAME", Value: "val"},
		EnvVar{Key: "CHANGED", Value: "new"},
		EnvVar{Key: "ONLY_B", Value: "b"},
	)

	entries := ComputeDiff(a, b)

	require.Len(t, entries, 4)

	statusMap := make(map[string]DiffStatus)
	for _, e := range entries {
		statusMap[e.Key] = e.Status
	}
	assert.Equal(t, DiffEqual, statusMap["SAME"])
	assert.Equal(t, DiffChanged, statusMap["CHANGED"])
	assert.Equal(t, DiffAdded, statusMap["ONLY_A"])
	assert.Equal(t, DiffRemoved, statusMap["ONLY_B"])
}
