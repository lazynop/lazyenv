package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
