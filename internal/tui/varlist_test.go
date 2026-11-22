package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func TestVarListMoveUp(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 20

	vl.MoveDown()
	vl.MoveDown()
	assert.Equal(t, 2, vl.Cursor)

	vl.MoveUp()
	assert.Equal(t, 1, vl.Cursor)

	vl.MoveUp()
	assert.Equal(t, 0, vl.Cursor)

	vl.MoveUp()
	assert.Equal(t, 0, vl.Cursor, "MoveUp at the top should keep the cursor at 0")
}

func TestVarListMoveDownBound(t *testing.T) {
	f := makeTestFile(".env", "A", "B")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 20

	vl.MoveDown()
	vl.MoveDown()
	vl.MoveDown()
	assert.Equal(t, 1, vl.Cursor, "should not exceed last index")
}

func TestNoMatchesNoteCutsTheQueryAsQuoted(t *testing.T) {
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(makeTestFile(".env", "A"))
	vl.Width = config.VarListMinWidth
	escapedQuotes := strings.Repeat(`"`, 30)
	vl.SetSearch(escapedQuotes)

	note := vl.note()
	assert.Equal(t, vl.contentCols(), lipgloss.Width(note), "the note should fill the panel's width, no more: %s", note)
	assert.True(t, strings.HasSuffix(note, `.."`), "the cut query should keep its closing quote: %s", note)
}

func TestVarListSelectedVarNoFile(t *testing.T) {
	vl := NewVarListModel(config.DefaultConfig().Layout)
	assert.Nil(t, vl.SelectedVar())
	assert.Equal(t, -1, vl.SelectedVarIndex())
}

func TestVarListSelectedVarEmpty(t *testing.T) {
	f := makeTestFile(".env")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	assert.Nil(t, vl.SelectedVar())
}

func TestVarListSearchFilter(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "API_KEY", "DB_PORT")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	assert.Equal(t, 3, len(vl.displayItems))

	vl.SetSearch("DB")
	assert.Equal(t, 2, len(vl.displayItems), "should match DB_HOST and DB_PORT")

	vl.SetSearch("API")
	assert.Equal(t, 1, len(vl.displayItems), "should match API_KEY only")

	vl.SetSearch("NOMATCH")
	assert.Equal(t, 0, len(vl.displayItems))

	vl.SetSearch("")
	assert.Equal(t, 3, len(vl.displayItems), "clear search should show all")
}

func TestVarListSearchCaseInsensitive(t *testing.T) {
	f := makeTestFile(".env", "MY_KEY")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	vl.SetSearch("my_key")
	assert.Equal(t, 1, len(vl.displayItems), "search should be case-insensitive")
}

func TestVarListSearchSkipsMaskedSecretValues(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN", "HOST")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	vl.SetSearch("hunter")
	assert.Equal(t, 0, len(vl.displayItems), "a masked secret's value must not match")

	vl.SetSearch("token")
	assert.Equal(t, 1, len(vl.displayItems), "a masked secret's key still matches")
}

func TestVarListToggleSecretsRefiltersActiveSearch(t *testing.T) {
	f := makeTestFile(".env", "A_TOKEN", "HUNTER_NAME", "B_TOKEN")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	f.Vars[2].Value, f.Vars[2].IsSecret = "hunter3", true
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 20
	vl.SetSearch("hunter")
	require.Equal(t, 1, len(vl.displayItems), "only the key match while secrets are masked")

	vl.SetShowSecrets(true)
	assert.Equal(t, 3, len(vl.displayItems), "revealed secret values match")
	require.NotNil(t, vl.SelectedVar())
	assert.Equal(t, "HUNTER_NAME", vl.SelectedVar().Key, "cursor stays on the same var")

	vl.SetShowSecrets(false)
	assert.Equal(t, 1, len(vl.displayItems), "masking again drops the value matches")
	require.NotNil(t, vl.SelectedVar())
	assert.Equal(t, "HUNTER_NAME", vl.SelectedVar().Key, "cursor stays on the same var")
}

func TestVarListToggleSort(t *testing.T) {
	f := makeTestFile(".env", "ZZZ", "AAA", "MMM")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	assert.False(t, vl.SortAlpha, "the list should start in file order")
	assert.Equal(t, 0, vl.displayItems[0].VarIdx)
	assert.Equal(t, 1, vl.displayItems[1].VarIdx)
	assert.Equal(t, 2, vl.displayItems[2].VarIdx)

	vl.ToggleSort()
	assert.True(t, vl.SortAlpha)
	assert.Equal(t, "AAA", f.Vars[vl.displayItems[0].VarIdx].Key)
	assert.Equal(t, "MMM", f.Vars[vl.displayItems[1].VarIdx].Key)
	assert.Equal(t, "ZZZ", f.Vars[vl.displayItems[2].VarIdx].Key)

	vl.ToggleSort()
	assert.False(t, vl.SortAlpha)
}

func TestVarListRefreshAfterAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	assert.Equal(t, 1, len(vl.displayItems))

	f.AddVar("BAR", "val")
	vl.Refresh()
	assert.Equal(t, 2, len(vl.displayItems))
}

func TestVarListFormatValueFlattensControlCharsToKeepTheRowLayout(t *testing.T) {
	vl := NewVarListModel(config.DefaultConfig().Layout)
	v := &model.EnvVar{Value: "first line\nsecond line\tcol\rend"}

	out := vl.formatValue(v, 60)

	assert.NotContains(t, out, "\n", "rendered value must not contain raw newlines")
	assert.NotContains(t, out, "\t", "rendered value must not contain raw tabs")
	assert.NotContains(t, out, "\r", "rendered value must not contain raw carriage returns")
	assert.Contains(t, out, "↵", "newlines must be replaced with the visible marker")
	assert.Contains(t, out, "⇥", "tabs must be replaced with the visible marker")
	assert.Contains(t, out, "↩", "carriage returns must be replaced with the visible marker")
}

func TestVarListScrolling(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F", "G", "H")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 8

	for range 6 {
		vl.MoveDown()
	}
	assert.Equal(t, 6, vl.Cursor)
	assert.Greater(t, vl.Offset, 0, "should scroll")
}

func TestVarListViewNoFile(t *testing.T) {
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.Width = 60
	vl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := vl.View(theme)
	assert.Contains(t, view, "Select a file")
}

func TestVarListViewNoMatches(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.SetSearch("ZZZZZ")
	vl.Width = 60
	vl.Height = 10

	theme := BuildTheme(true, config.ColorConfig{})
	view := vl.View(theme)
	assert.Contains(t, view, "No matches")
}

func TestVarListViewRendersVars(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Width = 60
	vl.Height = 20
	vl.Focused = true

	theme := BuildTheme(true, config.ColorConfig{})
	view := vl.View(theme)
	assert.Contains(t, view, "FOO")
	assert.Contains(t, view, "BAR")
}

func TestVarListViewTruncatesLongKeys(t *testing.T) {
	longKey := "THIS_IS_A_VERY_LONG_ENVIRONMENT_VARIABLE_NAME_THAT_EXCEEDS_THIRTY"
	f := makeTestFile(".env", longKey, "SHORT")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Width = 80
	vl.Height = 20
	vl.Focused = true

	theme := BuildTheme(true, config.ColorConfig{})
	view := vl.View(theme)

	assert.NotContains(t, view, longKey, "long key should be truncated")
	assert.Contains(t, view, "THIS_IS_A_VERY_LONG_ENVIRONM..", "truncated key with .. should appear")
	assert.Contains(t, view, "SHORT")
}

func TestVarListViewIndicators(t *testing.T) {
	f := &model.EnvFile{
		Name: ".env",
		Vars: []model.EnvVar{
			{Key: "NEW_VAR", Value: "val", IsNew: true, Modified: true},
			{Key: "DUP_KEY", Value: "val", IsDuplicate: true},
			{Key: "EMPTY", Value: "", IsEmpty: true},
		},
		Lines: []model.RawLine{
			{Type: model.LineVariable, Content: "NEW_VAR=val", VarIdx: 0},
			{Type: model.LineVariable, Content: "DUP_KEY=val", VarIdx: 1},
			{Type: model.LineVariable, Content: "EMPTY=", VarIdx: 2},
		},
	}
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Width = 60
	vl.Height = 20
	vl.Focused = true

	theme := BuildTheme(true, config.ColorConfig{})
	view := vl.View(theme)
	assert.Contains(t, view, "NEW_VAR")
	assert.Contains(t, view, "DUP_KEY")
	assert.Contains(t, view, "EMPTY")
}

const (
	groupingDBHeaderRow    = 0
	groupingDBHostRow      = 1
	groupingRedisHeaderRow = 4
)

func newGroupingFixture() *model.EnvFile {
	return makeTestFile(".env",
		"DB_HOST", "DB_PORT", "DB_USER",
		"REDIS_URL", "REDIS_PORT",
		"PORT", "DEBUG",
	)
}

func countHeaders(items []displayItem) int {
	n := 0
	for _, it := range items {
		if it.Kind == displayItemHeader {
			n++
		}
	}
	return n
}

func TestVarListGrouping_ToggleProducesHeaders(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30

	assert.Equal(t, 0, countHeaders(vl.displayItems), "no headers before toggle")
	assert.Equal(t, 7, len(vl.displayItems))

	count := vl.ToggleGrouping()
	assert.Equal(t, 2, count, "should report 2 named groups (DB, REDIS)")
	assert.True(t, vl.Grouping)
	assert.Equal(t, 10, len(vl.displayItems), "3 headers (DB, REDIS, UNGROUPED) plus 7 vars")
	assert.Equal(t, 3, countHeaders(vl.displayItems))
	assert.Equal(t, displayItemHeader, vl.displayItems[0].Kind, "the DB header should come first: its group occurs first in the file")
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
}

func TestVarListGrouping_CollapseHidesVars(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	vl.SetCursor(groupingDBHeaderRow)
	require.True(t, vl.IsHeaderAtCursor())
	ok := vl.ToggleCollapseAtCursor()
	assert.True(t, ok)
	assert.True(t, vl.collapsed["DB"])
	assert.Equal(t, 7, len(vl.displayItems), "10 items less the 3 DB vars; the DB header stays")
	for _, item := range vl.displayItems {
		if item.Kind == displayItemVar {
			assert.NotContains(t, f.Vars[item.VarIdx].Key, "DB_")
		}
	}
}

func TestVarListGrouping_ToggleCollapseExpandsAgain(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.SetCursor(groupingDBHeaderRow)

	vl.ToggleCollapseAtCursor()
	assert.True(t, vl.collapsed["DB"])
	vl.ToggleCollapseAtCursor()
	assert.False(t, vl.collapsed["DB"])
	assert.Equal(t, 10, len(vl.displayItems))
}

func TestVarListGrouping_ToggleCollapseNoOpOnVar(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	vl.MoveDown()
	assert.False(t, vl.IsHeaderAtCursor())
	ok := vl.ToggleCollapseAtCursor()
	assert.False(t, ok, "no toggle when cursor is on a var")
	assert.False(t, vl.collapsed["DB"])
}

func TestVarListGrouping_CursorFollowsVarOnToggle(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30

	vl.SetCursor(3)
	assert.Equal(t, "REDIS_URL", vl.SelectedVar().Key)

	vl.ToggleGrouping()

	got := vl.SelectedVar()
	require.NotNil(t, got)
	assert.Equal(t, "REDIS_URL", got.Key, "the cursor should follow REDIS_URL, not land on the REDIS header")
}

func TestVarListGrouping_CursorOnHeaderAfterCollapse(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	vl.SetCursor(groupingRedisHeaderRow)
	assert.True(t, vl.IsHeaderAtCursor())
	assert.Equal(t, -1, vl.SelectedVarIndex(), "header has no var index")
	assert.Nil(t, vl.SelectedVar())
}

func TestVarListGrouping_SearchDisablesHeaders(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	assert.Equal(t, 3, countHeaders(vl.displayItems),
		"DB + REDIS + UNGROUPED = 3 headers")

	vl.SetSearch("DB")
	assert.Equal(t, 0, countHeaders(vl.displayItems),
		"search must disable headers in rendering")
	assert.Equal(t, 3, len(vl.displayItems), "3 DB_* matches")
	assert.True(t, vl.Grouping, "Grouping flag is preserved across search")

	vl.SetSearch("")
	assert.Equal(t, 3, countHeaders(vl.displayItems),
		"clearing search restores headers")
}

func TestVarListGrouping_SortAlphaWithinGroups(t *testing.T) {
	f := makeTestFile(".env", "DB_PORT", "DB_HOST", "DB_USER", "PORT", "DEBUG")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, 7, len(vl.displayItems))
	assert.Equal(t, displayItemHeader, vl.displayItems[0].Kind)
	assert.Equal(t, "DB_HOST", f.Vars[vl.displayItems[1].VarIdx].Key)
	assert.Equal(t, "DB_PORT", f.Vars[vl.displayItems[2].VarIdx].Key)
	assert.Equal(t, "DB_USER", f.Vars[vl.displayItems[3].VarIdx].Key)
	assert.Equal(t, displayItemHeader, vl.displayItems[4].Kind)
	assert.True(t, vl.groups[vl.displayItems[4].GroupIdx].IsUngrouped())
	assert.Equal(t, "DEBUG", f.Vars[vl.displayItems[5].VarIdx].Key)
	assert.Equal(t, "PORT", f.Vars[vl.displayItems[6].VarIdx].Key)
}

func TestVarListGrouping_GroupsSortedAlphaUnderSort(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, 6, len(vl.displayItems))
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix,
		"DB group must come first by alpha order under sort")
	assert.Equal(t, "REDIS", vl.groups[vl.displayItems[3].GroupIdx].Prefix,
		"REDIS group must come second by alpha order under sort")
}

func TestVarListGrouping_GroupsFileOrderWithoutSortAlpha(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	require.Equal(t, 6, len(vl.displayItems))
	assert.Equal(t, "REDIS", vl.groups[vl.displayItems[0].GroupIdx].Prefix,
		"REDIS group must come first by file order when sort is off")
	assert.Equal(t, "DB", vl.groups[vl.displayItems[3].GroupIdx].Prefix,
		"DB group must come second by file order when sort is off")
}

func TestVarListGrouping_UngroupedAlwaysLastWithSortAlpha(t *testing.T) {
	f := makeTestFile(".env",
		"ZZZ_A", "ZZZ_B",
		"AAA",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, 5, len(vl.displayItems))
	assert.Equal(t, "ZZZ", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
	assert.True(t, vl.groups[vl.displayItems[3].GroupIdx].IsUngrouped(),
		"Ungrouped must remain last under alpha sort")
	assert.Equal(t, "AAA", f.Vars[vl.displayItems[4].VarIdx].Key)
}

func TestVarListGrouping_SortsPrefixesCaseSensitivelyLikeTheGrouping(t *testing.T) {
	f := makeTestFile(".env",
		"db_host", "db_port",
		"AWS_KEY", "AWS_SECRET",
		"aws_region", "aws_profile",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, 3, len(vl.groups))
	assert.Equal(t, "AWS", vl.groups[0].Prefix)
	assert.Equal(t, "aws", vl.groups[1].Prefix)
	assert.Equal(t, "db", vl.groups[2].Prefix)
}

func TestVarListGrouping_ToggleSortFollowsVarCursor(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	vl.SetCursor(1)
	require.Equal(t, "REDIS_URL", f.Vars[vl.SelectedVarIndex()].Key)

	vl.ToggleSort()

	require.NotEqual(t, -1, vl.SelectedVarIndex(), "cursor must be on a var")
	assert.Equal(t, "REDIS_URL", f.Vars[vl.SelectedVarIndex()].Key)
}

func TestVarListGrouping_ToggleSortFollowsHeaderCursor(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	vl.SetCursor(0)
	require.True(t, vl.IsHeaderAtCursor())
	require.Equal(t, "REDIS", vl.groups[vl.displayItems[vl.Cursor].GroupIdx].Prefix)

	vl.ToggleSort()

	require.True(t, vl.IsHeaderAtCursor(), "cursor must still be on a header")
	assert.Equal(t, "REDIS", vl.groups[vl.displayItems[vl.Cursor].GroupIdx].Prefix,
		"cursor must follow REDIS header to its new position")
}

func TestVarListGrouping_UngroupedAloneSkipsAHeaderOverTheWholeList(t *testing.T) {
	f := makeTestFile(".env", "PORT", "DEBUG", "HOST")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	assert.Equal(t, 0, countHeaders(vl.displayItems),
		"single-Ungrouped case must skip the header")
	assert.Equal(t, 3, len(vl.displayItems))
}

func TestVarListGrouping_UngroupedHeaderShownWithNamedGroups(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "DB_PORT", "PORT", "DEBUG")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	require.Equal(t, 6, len(vl.displayItems))
	require.Equal(t, 2, countHeaders(vl.displayItems))
	last := vl.displayItems[3]
	require.Equal(t, displayItemHeader, last.Kind, "last section starts with a header")
	assert.True(t, vl.groups[last.GroupIdx].IsUngrouped())

	vl.SetCursor(3)
	require.True(t, vl.IsHeaderAtCursor())
	vl.ToggleCollapseAtCursor()
	assert.True(t, vl.collapsed[""], "the empty prefix, Ungrouped's, should be collapsed")
	assert.Equal(t, 4, len(vl.displayItems), "collapsed Ungrouped: DB hdr + 2 vars + UNGROUPED hdr")
}

func TestVarListGrouping_PreservesCollapsedAcrossDisable(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30

	vl.ToggleGrouping()
	vl.SetCursor(groupingDBHeaderRow)
	vl.ToggleCollapseAtCursor()
	assert.True(t, vl.collapsed["DB"])

	vl.ToggleGrouping()
	assert.False(t, vl.Grouping)
	assert.True(t, vl.collapsed["DB"])

	vl.ToggleGrouping()
	assert.True(t, vl.Grouping)
	assert.True(t, vl.collapsed["DB"], "DB stays collapsed after round-trip")
	assert.Equal(t, 7, len(vl.displayItems), "10 items less the 3 DB vars")
}

func TestVarListGrouping_RefreshAfterAddRecomputesGroups(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "PORT")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	assert.Equal(t, 0, countHeaders(vl.displayItems), "a single DB_ var should form no group")

	f.AddVar("DB_PORT", "5432")
	vl.Refresh()
	assert.Equal(t, 2, countHeaders(vl.displayItems), "DB_PORT should form a DB group and PORT go under the UNGROUPED header")
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
}

func ungroupedHeaderRow(t *testing.T, vl *VarListModel) int {
	t.Helper()
	for i, item := range vl.displayItems {
		if item.Kind == displayItemHeader && vl.groups[item.GroupIdx].IsUngrouped() {
			return i
		}
	}
	require.Fail(t, "no UNGROUPED header is listed")
	return -1
}

func TestVarListRestoreCursorFollowsTheEmptyPrefixUngroupedHeaderToItsNewRow(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
		"PORT", "DEBUG",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	before := ungroupedHeaderRow(t, &vl)
	vl.SetCursor(before)
	anchor := vl.captureCursor()
	require.Equal(t, cursorAnchor{kind: anchorHeader, headerPrefix: ""}, anchor)

	f.AddVar("REDIS_HOST", "localhost")
	vl.Refresh()
	after := ungroupedHeaderRow(t, &vl)
	require.NotEqual(t, before, after, "REDIS_HOST must move the UNGROUPED header to another row")

	vl.restoreCursor(anchor)

	assert.Equal(t, after, vl.Cursor, "the cursor must follow the UNGROUPED header to its new row")
}

func TestVarListGrouping_RefreshPreservesAlphaOrder(t *testing.T) {
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
		"AWS_KEY",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, "DB", vl.groups[0].Prefix, "AWS, a single var, should be Ungrouped")
	require.Equal(t, "REDIS", vl.groups[1].Prefix)

	f.AddVar("AWS_SECRET", "shh")
	vl.Refresh()

	require.GreaterOrEqual(t, len(vl.groups), 3)
	assert.Equal(t, "AWS", vl.groups[0].Prefix,
		"AWS_SECRET makes AWS a group, which should come first alphabetically after Refresh")
	assert.Equal(t, "DB", vl.groups[1].Prefix)
	assert.Equal(t, "REDIS", vl.groups[2].Prefix)
}

func TestVarListSearchFromBottomKeepsMatchVisible(t *testing.T) {
	keys := make([]string, 0, 51)
	for i := range 50 {
		keys = append(keys, fmt.Sprintf("VAR_%02d", i))
	}
	keys = append(keys, "TARGET")
	f := makeTestFile(".env", keys...)

	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	const renderedRows = 6
	vl.Height = panelChromeHeight + renderedRows
	vl.Width = 60

	for range 50 {
		vl.MoveDown()
	}
	require.Greater(t, vl.Offset, 0, "list must be scrolled down before searching")

	vl.SetSearch("TARGET")
	require.Equal(t, 1, len(vl.displayItems))

	theme := BuildTheme(true, config.ColorConfig{})
	view := stripAnsi(vl.View(theme))
	assert.Contains(t, view, "TARGET",
		"the only search match must be rendered inside the viewport")
}

func TestVarListNarrowWidthKeepsEachVarOnOneRow(t *testing.T) {
	keys := []string{"UPSTREAM_SERVICE_CIRCUIT_BREAKER_THRESHOLD", "DATABASE_URL", "REDIS_URL", "SENTRY_DSN", "LOG_LEVEL", "APP_ENV"}
	f := parser.ParseBytes(".env", []byte(strings.Join(keys, "=\n")+"=\n"), config.SecretsConfig{})
	theme := BuildTheme(true, config.ColorConfig{})
	layout := config.DefaultConfig().Layout
	const paddingBelowTheMinimum = 1
	layout.VarListPadding = paddingBelowTheMinimum
	const fewerRowsThanVars = 4
	height := panelChromeHeight + fewerRowsThanVars
	rowEndMarker := emptyValueGlyph

	assertNoRowWraps(t, 30, 100, height, []string{rowEndMarker}, func(width int) string {
		vl := NewVarListModel(layout)
		vl.SetFile(f)
		vl.Width = width
		vl.Height = height
		return vl.View(theme)
	})
}

func TestVarListNarrowWidthKeepsPeekLineOnOneRow(t *testing.T) {
	keys := []string{"UPSTREAM_SERVICE_CIRCUIT_BREAKER_THRESHOLD", "DATABASE_URL", "REDIS_URL", "SENTRY_DSN"}
	f := makeTestFile(".env", keys...)
	f.Vars[0].Value = strings.Repeat("x", 200)
	f.UpdateVar(0, "new_value")
	f.AddVar("NEW_VAR", "value")
	theme := BuildTheme(true, config.ColorConfig{})
	layout := config.DefaultConfig().Layout
	const paddingBelowTheMinimum = 1
	layout.VarListPadding = paddingBelowTheMinimum
	const fewerRowsThanVars = 3
	height := panelChromeHeight + fewerRowsThanVars

	for _, cursor := range []int{0, len(keys)} {
		assertNoRowWraps(t, 30, 100, height, []string{"↳"}, func(width int) string {
			vl := NewVarListModel(layout)
			vl.SetFile(f)
			vl.Focused = true
			vl.Peeking = true
			vl.Width = width
			vl.Height = height
			vl.SetCursor(cursor)
			return vl.View(theme)
		})
	}
}
