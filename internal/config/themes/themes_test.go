package themes

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	c, ok := Lookup("dracula")
	assert.True(t, ok, "known theme should be found")
	assert.NotEqual(t, Colors{}, c, "found theme must have a non-zero palette")
	assert.NotEmpty(t, c.Primary)

	_, ok = Lookup("does-not-exist")
	assert.False(t, ok, "unknown theme should not be found")
}

func TestNames(t *testing.T) {
	names := Names()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "dracula")
	assert.Contains(t, names, "default-dark")
	assert.Equal(t, len(registry), len(names), "Names must list every registered theme")
	assert.True(t, sort.StringsAreSorted(names), "Names must be sorted")
}

func TestNamesReturnsDefensiveCopy(t *testing.T) {
	names := Names()
	first := names[0]
	names[0] = "zzz-mutated"

	assert.Equal(t, first, Names()[0],
		"mutating the returned slice must not affect subsequent calls")
}
