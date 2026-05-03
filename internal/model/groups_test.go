package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeGroups_EmptyList(t *testing.T) {
	groups := ComputeGroups(nil)
	assert.Nil(t, groups)

	groups = ComputeGroups([]EnvVar{})
	assert.Nil(t, groups)
}

func TestComputeGroups_ThresholdEnforced(t *testing.T) {
	// All prefixes are unique (count==1) → single Ungrouped group with all vars.
	vars := []EnvVar{
		{Key: "A_X"},
		{Key: "B_Y"},
		{Key: "C_Z"},
	}
	groups := ComputeGroups(vars)
	require.Len(t, groups, 1)
	assert.True(t, groups[0].IsUngrouped(), "single group should be Ungrouped")
	assert.Equal(t, "", groups[0].Prefix)
	assert.Equal(t, []int{0, 1, 2}, groups[0].Vars)
}

func TestComputeGroups_SimpleGrouping(t *testing.T) {
	vars := []EnvVar{
		{Key: "DB_HOST"},
		{Key: "DB_PORT"},
		{Key: "REDIS_URL"},
		{Key: "REDIS_PORT"},
		{Key: "PORT"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 3)
	assert.Equal(t, "DB", groups[0].Prefix)
	assert.Equal(t, []int{0, 1}, groups[0].Vars)
	assert.Equal(t, "REDIS", groups[1].Prefix)
	assert.Equal(t, []int{2, 3}, groups[1].Vars)
	assert.True(t, groups[2].IsUngrouped())
	assert.Equal(t, []int{4}, groups[2].Vars)
}

func TestComputeGroups_PreservesFileOrder(t *testing.T) {
	// REDIS appears first (index 0), DB second (index 1) → groups in that order.
	vars := []EnvVar{
		{Key: "REDIS_URL"},
		{Key: "DB_HOST"},
		{Key: "REDIS_PORT"},
		{Key: "DB_PORT"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 2)
	assert.Equal(t, "REDIS", groups[0].Prefix)
	assert.Equal(t, []int{0, 2}, groups[0].Vars)
	assert.Equal(t, "DB", groups[1].Prefix)
	assert.Equal(t, []int{1, 3}, groups[1].Vars)
}

func TestComputeGroups_NoUnderscore_Ungrouped(t *testing.T) {
	vars := []EnvVar{
		{Key: "PORT"},
		{Key: "DEBUG"},
		{Key: "DB_HOST"},
		{Key: "DB_PORT"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 2)
	assert.Equal(t, "DB", groups[0].Prefix, "named group comes first")
	assert.Equal(t, []int{2, 3}, groups[0].Vars)
	assert.True(t, groups[1].IsUngrouped(), "Ungrouped is always last")
	assert.Equal(t, []int{0, 1}, groups[1].Vars)
}

func TestComputeGroups_LeadingUnderscore(t *testing.T) {
	// Vars beginning with '_' have prefix "" → Ungrouped, regardless of count.
	vars := []EnvVar{
		{Key: "_HIDDEN"},
		{Key: "_SECRET"},
		{Key: "DB_HOST"},
		{Key: "DB_PORT"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 2)
	assert.Equal(t, "DB", groups[0].Prefix)
	assert.Equal(t, []int{2, 3}, groups[0].Vars)
	assert.True(t, groups[1].IsUngrouped())
	assert.Equal(t, []int{0, 1}, groups[1].Vars,
		"leading-underscore vars must go to Ungrouped, never form a group")
}

func TestComputeGroups_DuplicateKeys(t *testing.T) {
	// Duplicate keys are kept in the same group, in insertion order.
	vars := []EnvVar{
		{Key: "DB_HOST", Value: "a"},
		{Key: "DB_HOST", Value: "b"},
		{Key: "DB_PORT", Value: "5432"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 1)
	assert.Equal(t, "DB", groups[0].Prefix)
	assert.Equal(t, []int{0, 1, 2}, groups[0].Vars)
}

func TestComputeGroups_CaseSensitive(t *testing.T) {
	// "DB" and "db" are distinct prefixes; with one each, both go to Ungrouped.
	vars := []EnvVar{
		{Key: "DB_HOST"},
		{Key: "db_host"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 1)
	assert.True(t, groups[0].IsUngrouped())
	assert.Equal(t, []int{0, 1}, groups[0].Vars)
}

func TestComputeGroups_AllInOneGroup_NoUngrouped(t *testing.T) {
	vars := []EnvVar{
		{Key: "DB_HOST"},
		{Key: "DB_PORT"},
		{Key: "DB_USER"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 1)
	assert.Equal(t, "DB", groups[0].Prefix)
	assert.Equal(t, []int{0, 1, 2}, groups[0].Vars)
}

func TestComputeGroups_InterleavedGroups(t *testing.T) {
	// Vars from the same group are NOT contiguous in the file. The group
	// still collects them all, in their original order.
	vars := []EnvVar{
		{Key: "DB_HOST"},
		{Key: "REDIS_URL"},
		{Key: "DB_PORT"},
		{Key: "REDIS_PORT"},
		{Key: "DB_USER"},
	}
	groups := ComputeGroups(vars)

	require.Len(t, groups, 2)
	assert.Equal(t, "DB", groups[0].Prefix, "DB has lower first-occurrence index")
	assert.Equal(t, []int{0, 2, 4}, groups[0].Vars)
	assert.Equal(t, "REDIS", groups[1].Prefix)
	assert.Equal(t, []int{1, 3}, groups[1].Vars)
}

func TestComputeGroups_PrefixOf(t *testing.T) {
	cases := []struct {
		key, want string
	}{
		{"DB_HOST", "DB"},
		{"DB", ""},
		{"_HIDDEN", ""},
		{"", ""},
		{"A_B_C", "A"},
		{"A__B", "A"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, prefixOf(tc.key), "prefixOf(%q)", tc.key)
	}
}
