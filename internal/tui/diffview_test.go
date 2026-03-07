package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
)

func makeDiffFiles() (*model.EnvFile, *model.EnvFile) {
	a := makeTestFile(".env", "SHARED", "ONLY_A", "CHANGED")
	b := makeTestFile(".env.prod", "SHARED", "ONLY_B", "CHANGED")
	// Make CHANGED have different values
	b.Vars[2].Value = "different"
	return a, b
}

func TestDiffViewSetFiles(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.SetFiles(a, b)

	assert.Equal(t, a, dv.FileA)
	assert.Equal(t, b, dv.FileB)
	assert.Greater(t, len(dv.Entries), 0)
	assert.False(t, dv.HideEqual)
}

func TestDiffViewStats(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.SetFiles(a, b)

	assert.Equal(t, 1, dv.Stats.Equal, "SHARED is equal")
	assert.Equal(t, 1, dv.Stats.Changed, "CHANGED has different values")
	assert.Equal(t, 1, dv.Stats.Added, "ONLY_A is only in A")
	assert.Equal(t, 1, dv.Stats.Removed, "ONLY_B is only in B")
}

func TestDiffViewMoveUpDown(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	assert.Equal(t, 0, dv.Cursor)

	// Move down
	dv.MoveDown()
	assert.Equal(t, 1, dv.Cursor)

	dv.MoveDown()
	assert.Equal(t, 2, dv.Cursor)

	// Move up
	dv.MoveUp()
	assert.Equal(t, 1, dv.Cursor)

	// Move up past beginning — stays at 0
	dv.MoveUp()
	dv.MoveUp()
	dv.MoveUp()
	assert.Equal(t, 0, dv.Cursor)
}

func TestDiffViewMoveDownBound(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	total := len(dv.Entries)
	for i := 0; i < total+5; i++ {
		dv.MoveDown()
	}
	assert.Equal(t, total-1, dv.Cursor, "cursor should not exceed last entry")
}

func TestDiffViewToggleFilter(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.SetFiles(a, b)

	totalBefore := len(dv.Entries)
	assert.False(t, dv.HideEqual)

	dv.ToggleFilter()
	assert.True(t, dv.HideEqual)
	assert.Less(t, len(dv.Entries), totalBefore, "should hide equal entries")

	// Verify no equal entries are visible
	for _, e := range dv.Entries {
		assert.NotEqual(t, model.DiffEqual, e.Status)
	}

	// Toggle back
	dv.ToggleFilter()
	assert.False(t, dv.HideEqual)
	assert.Equal(t, totalBefore, len(dv.Entries))
}

func TestDiffViewCopyToRightChanged(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the changed entry
	for i, e := range dv.Entries {
		if e.Status == model.DiffChanged {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToRight()
	assert.Equal(t, "CHANGED", key)

	// After copy, B should have A's value
	vB := dv.FileB.VarByKey("CHANGED")
	require.NotNil(t, vB)
	assert.Equal(t, "val_CHANGED", vB.Value)
}

func TestDiffViewCopyToRightAdded(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the added entry (only in A)
	for i, e := range dv.Entries {
		if e.Status == model.DiffAdded {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToRight()
	assert.Equal(t, "ONLY_A", key)

	// B should now have ONLY_A
	vB := dv.FileB.VarByKey("ONLY_A")
	require.NotNil(t, vB)
	assert.Equal(t, "val_ONLY_A", vB.Value)
}

func TestDiffViewCopyToRightRemoved(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the removed entry (only in B)
	for i, e := range dv.Entries {
		if e.Status == model.DiffRemoved {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToRight()
	assert.Equal(t, "ONLY_B", key)

	// B should no longer have ONLY_B
	vB := dv.FileB.VarByKey("ONLY_B")
	assert.Nil(t, vB, "ONLY_B should be deleted from B")
}

func TestDiffViewCopyToLeftChanged(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the changed entry
	for i, e := range dv.Entries {
		if e.Status == model.DiffChanged {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToLeft()
	assert.Equal(t, "CHANGED", key)

	// A should now have B's value
	vA := dv.FileA.VarByKey("CHANGED")
	require.NotNil(t, vA)
	assert.Equal(t, "different", vA.Value)
}

func TestDiffViewCopyToLeftRemoved(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the removed entry (only in B)
	for i, e := range dv.Entries {
		if e.Status == model.DiffRemoved {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToLeft()
	assert.Equal(t, "ONLY_B", key)

	// A should now have ONLY_B
	vA := dv.FileA.VarByKey("ONLY_B")
	require.NotNil(t, vA)
}

func TestDiffViewCopyToLeftAdded(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the added entry (only in A)
	for i, e := range dv.Entries {
		if e.Status == model.DiffAdded {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToLeft()
	assert.Equal(t, "ONLY_A", key)

	// A should no longer have ONLY_A
	vA := dv.FileA.VarByKey("ONLY_A")
	assert.Nil(t, vA, "ONLY_A should be deleted from A")
}

func TestDiffViewCopyEqualNoOp(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Find the equal entry
	for i, e := range dv.Entries {
		if e.Status == model.DiffEqual {
			dv.Cursor = i
			break
		}
	}

	key := dv.CopyToRight()
	assert.Equal(t, "", key, "should not copy equal entries")

	key = dv.CopyToLeft()
	assert.Equal(t, "", key, "should not copy equal entries")
}

func TestDiffViewNextPrevDiff(t *testing.T) {
	a, b := makeDiffFiles()
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 40
	dv.SetFiles(a, b)

	// Start at 0 (SHARED, which is equal)
	assert.Equal(t, 0, dv.Cursor)
	assert.Equal(t, model.DiffEqual, dv.Entries[0].Status)

	// NextDiff should skip past equal
	dv.NextDiff()
	assert.NotEqual(t, model.DiffEqual, dv.Entries[dv.Cursor].Status)
	firstDiff := dv.Cursor

	// NextDiff again
	dv.NextDiff()
	assert.Greater(t, dv.Cursor, firstDiff)

	// PrevDiff should go back
	dv.PrevDiff()
	assert.Equal(t, firstDiff, dv.Cursor)
}

func TestDiffViewCopyOutOfBounds(t *testing.T) {
	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Cursor = -1
	assert.Equal(t, "", dv.CopyToRight())
	assert.Equal(t, "", dv.CopyToLeft())
}

func TestDiffViewTruncatesLongKeys(t *testing.T) {
	longKey := "THIS_IS_A_VERY_LONG_ENVIRONMENT_VARIABLE_NAME"
	a := makeTestFile(".env", longKey)
	b := makeTestFile(".env.prod", longKey)
	b.Vars[0].Value = "different"

	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Width = 100
	dv.Height = 20
	dv.SetFiles(a, b)

	theme := BuildTheme(true, config.ColorConfig{})
	view := dv.View(theme)

	// The full key should NOT appear (it exceeds the 25-char cap)
	assert.NotContains(t, view, longKey, "long key should be truncated")
	// The truncated version with ".." should appear
	assert.Contains(t, view, "THIS_IS_A_VERY_LONG_ENV..", "truncated key with .. should appear")
}

func TestDiffViewResetPreservesGitWarning(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, ".env")
	pathB := filepath.Join(dir, ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0644))

	fA, err := parser.ParseFile(pathA)
	require.NoError(t, err)
	fA.GitWarning = true

	fB, err := parser.ParseFile(pathB)
	require.NoError(t, err)
	fB.GitWarning = true

	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.SetFiles(fA, fB)

	// Modify and then reset
	fA.UpdateVar(0, "changed")
	errMsg := dv.Reset()
	assert.Empty(t, errMsg)

	assert.True(t, dv.FileA.GitWarning, "GitWarning on file A must survive DiffViewModel.Reset")
	assert.True(t, dv.FileB.GitWarning, "GitWarning on file B must survive DiffViewModel.Reset")
}

func TestDiffViewScrolling(t *testing.T) {
	// Create files with many entries to test scrolling
	keys := make([]string, 20)
	for i := range 20 {
		keys[i] = "VAR_" + string(rune('A'+i))
	}
	a := makeTestFile(".env", keys...)
	b := makeTestFile(".env.prod") // empty — all will be "added"

	dv := NewDiffViewModel(config.DefaultConfig().Layout)
	dv.Height = 12 // visible = 12 - 6 = 6
	dv.SetFiles(a, b)

	// Move down past visible area
	for range 10 {
		dv.MoveDown()
	}
	assert.Equal(t, 10, dv.Cursor)
	assert.Greater(t, dv.Offset, 0, "offset should scroll")
}
