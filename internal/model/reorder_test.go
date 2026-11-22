package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func parseEnv(content string) *model.EnvFile {
	return parser.ParseBytes("test.env", []byte(content), config.SecretsConfig{})
}

func marshalEnv(ef *model.EnvFile) string {
	return string(parser.Marshal(ef))
}

func varKeys(ef *model.EnvFile) []string {
	out := make([]string, len(ef.Vars))
	for i, v := range ef.Vars {
		out[i] = v.Key
	}
	return out
}

func TestReorderAlphabetical(t *testing.T) {
	ef := parseEnv("C=3\nA=1\nB=2")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "A=1\nB=2\nC=3", marshalEnv(ef))
	assert.Equal(t, []string{"A", "B", "C"}, varKeys(ef))
	assert.True(t, ef.Modified)
}

func TestReorderAttachedCommentsTravel(t *testing.T) {
	ef := parseEnv("# c comment\nC=3\n# a comment\nA=1")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "# a comment\nA=1\n# c comment\nC=3", marshalEnv(ef))
}

func TestReorderLeadingBlockPreserved(t *testing.T) {
	ef := parseEnv("# banner\n\nB=2\nA=1")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "# banner\n\nA=1\nB=2", marshalEnv(ef))
}

func TestReorderTrailingBlockPreserved(t *testing.T) {
	ef := parseEnv("B=2\nA=1\n# trailing")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "A=1\nB=2\n# trailing", marshalEnv(ef))
}

func TestReorderFloatingCommentFloatsToTop(t *testing.T) {
	ef := parseEnv("B=2\n# note\n\nA=1")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "# note\nA=1\nB=2", marshalEnv(ef))
}

func TestReorderGrouped(t *testing.T) {
	ef := parseEnv("REDIS_PORT=1\nDB_HOST=localhost\nDB_PORT=5432\nREDIS_HOST=r")
	ef.Reorder(model.ReorderGrouped)

	want := "DB_HOST=localhost\nDB_PORT=5432\n\nREDIS_HOST=r\nREDIS_PORT=1"
	assert.Equal(t, want, marshalEnv(ef))
}

func TestReorderGroupedUngroupedLast(t *testing.T) {
	ef := parseEnv("ZEBRA=1\nDB_HOST=h\nDB_PORT=p")
	ef.Reorder(model.ReorderGrouped)

	want := "DB_HOST=h\nDB_PORT=p\n\nZEBRA=1"
	assert.Equal(t, want, marshalEnv(ef))
}

func TestReorderDuplicatesKeepTheirOrderSoTheLastStillWins(t *testing.T) {
	ef := parseEnv("KEY=first\nKEY=second\nAAA=x")
	ef.Reorder(model.ReorderAlphabetical)

	assert.Equal(t, "AAA=x\nKEY=first\nKEY=second", marshalEnv(ef))
}

func TestReorderNoVariablesIsNoOp(t *testing.T) {
	ef := parseEnv("# just a comment\n")
	ef.Reorder(model.ReorderAlphabetical)

	assert.False(t, ef.Modified)
	assert.Equal(t, "# just a comment\n", marshalEnv(ef))
}

func assertLinesLinkToVars(t *testing.T, ef *model.EnvFile) {
	t.Helper()
	varLines := 0
	for i, l := range ef.Lines {
		if l.Type != model.LineVariable {
			continue
		}
		varLines++
		require.GreaterOrEqualf(t, l.VarIdx, 0, "line %d %q", i, l.Content)
		require.Lessf(t, l.VarIdx, len(ef.Vars), "line %d %q", i, l.Content)
		parsed := parseEnv(l.Content).Vars
		require.Lenf(t, parsed, 1, "line %d %q", i, l.Content)
		assert.Equalf(t, parsed[0].Key, ef.Vars[l.VarIdx].Key, "line %d %q", i, l.Content)
	}
	assert.Equal(t, len(ef.Vars), varLines)
}

func TestReorderLinksEachLineToItsVariable(t *testing.T) {
	tests := []struct {
		name string
		mode model.ReorderMode
		in   string
	}{
		{"alphabetical", model.ReorderAlphabetical, "# banner\n\nB=2\nC=3\nA=1"},
		{"grouped", model.ReorderGrouped, "# banner\n\nREDIS_PORT=1\nDB_HOST=h\nDB_PORT=5432\nREDIS_HOST=r"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := parseEnv(tt.in)
			ef.Reorder(tt.mode)

			assertLinesLinkToVars(t, ef)
		})
	}
}

func TestReorderThenEditLandsOnTheEditedLine(t *testing.T) {
	ef := parseEnv("# banner\n\nB=2\nC=3\nA=1")
	ef.Reorder(model.ReorderAlphabetical)

	ef.UpdateVar(ef.LastIndexOfKey("B"), "20")

	assert.Equal(t, "# banner\n\nA=1\nB=20\nC=3", marshalEnv(ef))
}

func TestReorderKeepsTheTrailingNewline(t *testing.T) {
	tests := []struct {
		name string
		mode model.ReorderMode
		in   string
		want string
	}{
		{"alphabetical", model.ReorderAlphabetical, "B=2\nC=3\nA=1\n", "A=1\nB=2\nC=3\n"},
		{"grouped", model.ReorderGrouped, "REDIS_PORT=1\nDB_HOST=h\nDB_PORT=p\nREDIS_HOST=r\n", "DB_HOST=h\nDB_PORT=p\n\nREDIS_HOST=r\nREDIS_PORT=1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := parseEnv(tt.in)
			ef.Reorder(tt.mode)

			assert.Equal(t, tt.want, marshalEnv(ef))
		})
	}
}
