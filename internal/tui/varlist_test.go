package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

	assert.Equal(t, 3, len(vl.displayIndices))

	vl.SetSearch("DB")
	assert.Equal(t, 2, len(vl.displayIndices), "should match DB_HOST and DB_PORT")

	vl.SetSearch("API")
	assert.Equal(t, 1, len(vl.displayIndices), "should match API_KEY only")

	vl.SetSearch("NOMATCH")
	assert.Equal(t, 0, len(vl.displayIndices))

	vl.SetSearch("")
	assert.Equal(t, 3, len(vl.displayIndices), "clear search should show all")
}

func TestVarListSearchCaseInsensitive(t *testing.T) {
	f := makeTestFile(".env", "MY_KEY")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	vl.SetSearch("my_key")
	assert.Equal(t, 1, len(vl.displayIndices), "search should be case-insensitive")
}

func TestVarListToggleSort(t *testing.T) {
	f := makeTestFile(".env", "ZZZ", "AAA", "MMM")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)

	// Default order: file order
	assert.False(t, vl.SortAlpha)
	assert.Equal(t, 0, vl.displayIndices[0]) // ZZZ
	assert.Equal(t, 1, vl.displayIndices[1]) // AAA
	assert.Equal(t, 2, vl.displayIndices[2]) // MMM

	vl.ToggleSort()
	assert.True(t, vl.SortAlpha)
	// Alphabetical: AAA(1), MMM(2), ZZZ(0)
	assert.Equal(t, "AAA", f.Vars[vl.displayIndices[0]].Key)
	assert.Equal(t, "MMM", f.Vars[vl.displayIndices[1]].Key)
	assert.Equal(t, "ZZZ", f.Vars[vl.displayIndices[2]].Key)

	vl.ToggleSort()
	assert.False(t, vl.SortAlpha)
}

func TestVarListRefreshAfterAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vl := NewVarListModel(config.DefaultConfig().Layout)
	vl.SetFile(f)
	assert.Equal(t, 1, len(vl.displayIndices))

	f.AddVar("BAR", "val", false)
	vl.Refresh()
	assert.Equal(t, 2, len(vl.displayIndices))
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
