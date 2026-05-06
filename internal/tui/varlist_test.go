package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

func TestVarListMoveUp(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 20

	vl.MoveDown() // cursor=1
	vl.MoveDown() // cursor=2
	assert.Equal(t, 2, vl.Cursor)

	vl.MoveUp()
	assert.Equal(t, 1, vl.Cursor)

	vl.MoveUp()
	assert.Equal(t, 0, vl.Cursor)

	// Already at top
	vl.MoveUp()
	assert.Equal(t, 0, vl.Cursor)
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

	assert.Equal(t, 3, vl.DisplayCount())

	vl.SetSearch("DB")
	assert.Equal(t, 2, vl.DisplayCount(), "should match DB_HOST and DB_PORT")

	vl.SetSearch("API")
	assert.Equal(t, 1, vl.DisplayCount(), "should match API_KEY only")

	vl.SetSearch("NOMATCH")
	assert.Equal(t, 0, vl.DisplayCount())

	vl.SetSearch("")
	assert.Equal(t, 3, vl.DisplayCount(), "clear search should show all")
}

func TestVarListSearchCaseInsensitive(t *testing.T) {
	f := makeTestFile(".env", "MY_KEY")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	vl.SetSearch("my_key")
	assert.Equal(t, 1, vl.DisplayCount(), "search should be case-insensitive")
}

func TestVarListToggleSort(t *testing.T) {
	f := makeTestFile(".env", "ZZZ", "AAA", "MMM")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	// Default order: file order
	assert.False(t, vl.SortAlpha)
	assert.Equal(t, 0, vl.displayItems[0].VarIdx) // ZZZ
	assert.Equal(t, 1, vl.displayItems[1].VarIdx) // AAA
	assert.Equal(t, 2, vl.displayItems[2].VarIdx) // MMM

	vl.ToggleSort()
	assert.True(t, vl.SortAlpha)
	// Alphabetical: AAA, MMM, ZZZ
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
	assert.Equal(t, 1, vl.DisplayCount())

	f.AddVar("BAR", "val", false)
	vl.Refresh()
	assert.Equal(t, 2, vl.DisplayCount())
}

func TestFlattenValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain_ascii", "hello world", "hello world"},
		{"empty", "", ""},
		{"single_newline", "line1\nline2", "line1↵line2"},
		{"single_tab", "col1\tcol2", "col1⇥col2"},
		{"single_cr", "before\rafter", "before↩after"},
		{"mixed_all_three", "a\nb\tc\rd", "a↵b⇥c↩d"},
		{"multiple_newlines", "a\nb\nc", "a↵b↵c"},
		{"trailing_newline", "end\n", "end↵"},
		{"leading_newline", "\nstart", "↵start"},
		{"crlf_pair", "line\r\nnext", "line↩↵next"},
		{"no_control_chars_fast_path", "unchanged utf8 àèìòù", "unchanged utf8 àèìòù"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, flattenValue(tt.in))
		})
	}
}

func TestVarListFormatValueFlattensControlChars(t *testing.T) {
	// A multiline or tabbed value must never contribute raw control chars to
	// the rendered line — that would break panel layout and push the warning
	// indicators to the wrong row.
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
	vl.Height = 8 // visible = 8 - 4 = 4

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

	// The full key should NOT appear (it exceeds the 30-char cap)
	assert.NotContains(t, view, longKey, "long key should be truncated")
	// The truncated version with ".." should appear
	assert.Contains(t, view, "THIS_IS_A_VERY_LONG_ENVIRONM..", "truncated key with .. should appear")
	// Short key should still be fully visible
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

// --- Variable grouping (E009) ---

func newGroupingFixture() *model.EnvFile {
	// DB (3), REDIS (2), Ungrouped: PORT, DEBUG
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
	assert.Equal(t, 7, vl.DisplayCount())

	count := vl.ToggleGrouping()
	assert.Equal(t, 2, count, "should report 2 named groups (DB, REDIS)")
	assert.True(t, vl.Grouping)
	// 3 headers (DB, REDIS, UNGROUPED) + 7 vars = 10 items
	assert.Equal(t, 10, vl.DisplayCount())
	assert.Equal(t, 3, countHeaders(vl.displayItems))
	// First item must be the DB header (lowest first-occurrence index).
	assert.Equal(t, displayItemHeader, vl.displayItems[0].Kind)
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
}

func TestVarListGrouping_CollapseHidesVars(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	// After toggle the cursor follows DB_HOST (var 0) to row 1; the DB
	// header is at row 0. Move the cursor onto the header to collapse it.
	vl.SetCursor(0)
	require.True(t, vl.IsHeaderAtCursor())
	ok := vl.ToggleCollapseAtCursor()
	assert.True(t, ok)
	assert.True(t, vl.isCollapsed("DB"))
	// 10 items - 3 DB vars = 7 items now (DB header still present).
	assert.Equal(t, 7, vl.DisplayCount())
	// No DB_* var rows must remain.
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
	vl.SetCursor(0) // DB header

	vl.ToggleCollapseAtCursor() // collapse DB
	assert.True(t, vl.isCollapsed("DB"))
	vl.ToggleCollapseAtCursor() // expand DB
	assert.False(t, vl.isCollapsed("DB"))
	assert.Equal(t, 10, vl.DisplayCount())
}

func TestVarListGrouping_ToggleCollapseNoOpOnVar(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	// Move cursor to a var (skip the DB header).
	vl.MoveDown() // cursor=1, first DB var
	assert.False(t, vl.IsHeaderAtCursor())
	ok := vl.ToggleCollapseAtCursor()
	assert.False(t, ok, "no toggle when cursor is on a var")
	assert.False(t, vl.isCollapsed("DB"))
}

func TestVarListGrouping_CursorFollowsVarOnToggle(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30

	// Linear view: place cursor on REDIS_URL (index 3 in displayItems).
	vl.SetCursor(3)
	assert.Equal(t, "REDIS_URL", vl.SelectedVar().Key)

	vl.ToggleGrouping()

	// After grouping the cursor must still resolve to REDIS_URL,
	// not to the REDIS header.
	got := vl.SelectedVar()
	require.NotNil(t, got)
	assert.Equal(t, "REDIS_URL", got.Key)
}

func TestVarListGrouping_CursorOnHeaderAfterCollapse(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	// Move cursor to the REDIS header. With Grouping on, layout is:
	//   0: DB header, 1..3: DB vars, 4: REDIS header, 5..6: REDIS vars,
	//   7: UNGROUPED header, 8..9: ungrouped vars.
	vl.SetCursor(4)
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
	assert.Equal(t, 3, vl.DisplayCount(), "3 DB_* matches")
	assert.True(t, vl.Grouping, "Grouping flag is preserved across search")

	vl.SetSearch("")
	assert.Equal(t, 3, countHeaders(vl.displayItems),
		"clearing search restores headers")
}

func TestVarListGrouping_SortAlphaWithinGroups(t *testing.T) {
	// Vars deliberately not in alphabetical order within DB.
	f := makeTestFile(".env", "DB_PORT", "DB_HOST", "DB_USER", "PORT", "DEBUG")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort() // alpha on

	// Layout: [DB header, DB_HOST, DB_PORT, DB_USER, UNGROUPED header, DEBUG, PORT]
	require.Equal(t, 7, vl.DisplayCount())
	assert.Equal(t, displayItemHeader, vl.displayItems[0].Kind)
	assert.Equal(t, "DB_HOST", f.Vars[vl.displayItems[1].VarIdx].Key)
	assert.Equal(t, "DB_PORT", f.Vars[vl.displayItems[2].VarIdx].Key)
	assert.Equal(t, "DB_USER", f.Vars[vl.displayItems[3].VarIdx].Key)
	// UNGROUPED header, then PORT/DEBUG sorted alpha → DEBUG, PORT.
	assert.Equal(t, displayItemHeader, vl.displayItems[4].Kind)
	assert.True(t, vl.groups[vl.displayItems[4].GroupIdx].IsUngrouped())
	assert.Equal(t, "DEBUG", f.Vars[vl.displayItems[5].VarIdx].Key)
	assert.Equal(t, "PORT", f.Vars[vl.displayItems[6].VarIdx].Key)
}

func TestVarListGrouping_GroupsSortedAlphaUnderSort(t *testing.T) {
	// REDIS appears first in the file but DB must come first under alpha
	// sort, since `o` now reorders groups alphabetically too. Ungrouped
	// (none here) would still pin last.
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	require.Equal(t, 6, vl.DisplayCount())
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix,
		"DB group must come first by alpha order under sort")
	assert.Equal(t, "REDIS", vl.groups[vl.displayItems[3].GroupIdx].Prefix,
		"REDIS group must come second by alpha order under sort")
}

func TestVarListGrouping_GroupsFileOrderWithoutSortAlpha(t *testing.T) {
	// With grouping ON but sort OFF, groups must stay in file order.
	f := makeTestFile(".env",
		"REDIS_URL", "REDIS_PORT",
		"DB_HOST", "DB_PORT",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	// SortAlpha intentionally left off.

	require.Equal(t, 6, vl.DisplayCount())
	assert.Equal(t, "REDIS", vl.groups[vl.displayItems[0].GroupIdx].Prefix,
		"REDIS group must come first by file order when sort is off")
	assert.Equal(t, "DB", vl.groups[vl.displayItems[3].GroupIdx].Prefix,
		"DB group must come second by file order when sort is off")
}

func TestVarListGrouping_UngroupedAlwaysLastWithSortAlpha(t *testing.T) {
	// AAA (Ungrouped — single occurrence, no shared prefix with another)
	// would alphabetically come before ZZZ, but it must stay pinned last.
	f := makeTestFile(".env",
		"ZZZ_A", "ZZZ_B",
		"AAA",
	)
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	vl.ToggleSort()

	// Layout under sort: [ZZZ header, ZZZ_A, ZZZ_B, UNGROUPED header, AAA]
	require.Equal(t, 5, vl.DisplayCount())
	assert.Equal(t, "ZZZ", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
	assert.True(t, vl.groups[vl.displayItems[3].GroupIdx].IsUngrouped(),
		"Ungrouped must remain last under alpha sort")
	assert.Equal(t, "AAA", f.Vars[vl.displayItems[4].VarIdx].Key)
}

func TestVarListGrouping_CaseSensitiveAlphaOrder(t *testing.T) {
	// Prefix matching in ComputeGroups is case-sensitive, so AWS and aws
	// are distinct groups. Alpha sort must be case-sensitive too:
	// uppercase letters sort before lowercase in Go string comparison.
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

	// Expected order: AWS, aws, db (uppercase < lowercase).
	require.Equal(t, 3, len(vl.groups))
	assert.Equal(t, "AWS", vl.groups[0].Prefix)
	assert.Equal(t, "aws", vl.groups[1].Prefix)
	assert.Equal(t, "db", vl.groups[2].Prefix)
}

func TestVarListGrouping_UngroupedAloneNoHeader(t *testing.T) {
	// Only Ungrouped (no prefix shared by ≥2): no header — degenerates to
	// the linear view to avoid wrapping the entire list under one section.
	f := makeTestFile(".env", "PORT", "DEBUG", "HOST")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	assert.Equal(t, 0, countHeaders(vl.displayItems),
		"single-Ungrouped case must skip the header")
	assert.Equal(t, 3, vl.DisplayCount())
}

func TestVarListGrouping_UngroupedHeaderShownWithNamedGroups(t *testing.T) {
	// When at least one named group exists, the trailing Ungrouped section
	// gets its own header (collapsible like the others).
	f := makeTestFile(".env", "DB_HOST", "DB_PORT", "PORT", "DEBUG")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()

	// Layout: DB hdr, DB_HOST, DB_PORT, UNGROUPED hdr, PORT, DEBUG
	require.Equal(t, 6, vl.DisplayCount())
	require.Equal(t, 2, countHeaders(vl.displayItems))
	last := vl.displayItems[3]
	require.Equal(t, displayItemHeader, last.Kind, "last section starts with a header")
	assert.True(t, vl.groups[last.GroupIdx].IsUngrouped())

	// And it's collapsible: collapse hides the trailing PORT and DEBUG.
	vl.SetCursor(3)
	require.True(t, vl.IsHeaderAtCursor())
	vl.ToggleCollapseAtCursor()
	assert.True(t, vl.isCollapsed("")) // empty prefix = Ungrouped
	assert.Equal(t, 4, vl.DisplayCount(), "collapsed Ungrouped: DB hdr + 2 vars + UNGROUPED hdr")
}

func TestVarListGrouping_PreservesCollapsedAcrossDisable(t *testing.T) {
	f := newGroupingFixture()
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30

	vl.ToggleGrouping()
	vl.SetCursor(0)             // DB header
	vl.ToggleCollapseAtCursor() // collapse DB
	assert.True(t, vl.isCollapsed("DB"))

	vl.ToggleGrouping() // disable
	assert.False(t, vl.Grouping)
	// collapsed state must survive
	assert.True(t, vl.isCollapsed("DB"))

	vl.ToggleGrouping() // re-enable
	assert.True(t, vl.Grouping)
	assert.True(t, vl.isCollapsed("DB"), "DB stays collapsed after round-trip")
	// 10 items - 3 DB vars = 7
	assert.Equal(t, 7, vl.DisplayCount())
}

func TestVarListGrouping_RefreshAfterAddRecomputesGroups(t *testing.T) {
	// Single DB_HOST: not yet a group.
	f := makeTestFile(".env", "DB_HOST", "PORT")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	vl.Height = 30
	vl.ToggleGrouping()
	assert.Equal(t, 0, countHeaders(vl.displayItems))

	// Adding DB_PORT must form a real DB group; PORT then sits under the
	// trailing UNGROUPED header.
	f.AddVar("DB_PORT", "5432", false)
	vl.Refresh()
	assert.Equal(t, 2, countHeaders(vl.displayItems))
	assert.Equal(t, "DB", vl.groups[vl.displayItems[0].GroupIdx].Prefix)
}
