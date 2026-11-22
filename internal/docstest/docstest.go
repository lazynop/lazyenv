// Package docstest reads the repo's Markdown docs for the tests that check
// the facts they state against the code.
package docstest

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// SectionHeadingPrefix starts the heading of a section that Section finds.
const SectionHeadingPrefix = "## "

var codeSpanRe = regexp.MustCompile("`([^`]+)`")

// ReadFile returns the content of the file at path, relative to the repo
// root, whichever package's directory the test runs in.
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	require.NoError(t, err)
	return string(data)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "no go.mod above the test's directory")
		dir = parent
	}
}

// CodeSpans returns the text of each inline code span in s, in order.
func CodeSpans(s string) []string {
	var spans []string
	for _, m := range codeSpanRe.FindAllStringSubmatch(s, -1) {
		spans = append(spans, m[1])
	}
	return spans
}

// Table is a Markdown table: the header cells and, for each row, its cells,
// trimmed.
type Table struct {
	Header []string
	Rows   [][]string
}

// Tables returns every table in doc, in order.
func Tables(doc string) []Table {
	var tables []Table
	inTable := false
	for line := range strings.Lines(doc) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			inTable = false
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		switch {
		case !inTable:
			tables = append(tables, Table{Header: cells})
			inTable = true
		case isSeparatorRow(line):
		default:
			last := &tables[len(tables)-1]
			last.Rows = append(last.Rows, cells)
		}
	}
	return tables
}

func isSeparatorRow(line string) bool {
	return strings.Trim(line, "|-: ") == ""
}

// TableRows returns the rows of the first table in doc with a column named
// header, and fails the test when there is none.
func TableRows(t *testing.T, doc, header string) [][]string {
	t.Helper()
	for _, table := range Tables(doc) {
		if slices.Contains(table.Header, header) {
			return table.Rows
		}
	}
	require.Failf(t, "missing table", "no table with a %q column", header)
	return nil
}

// LineStartingWith returns the first line of doc that starts with prefix, and
// fails the test when there is none.
func LineStartingWith(t *testing.T, doc, prefix string) string {
	t.Helper()
	for line := range strings.Lines(doc) {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	require.Failf(t, "missing line", "no line starts with %q", prefix)
	return ""
}

// Section returns the body of the section of doc under the heading
// SectionHeadingPrefix + heading, up to the next such heading, and fails the
// test when there is none.
func Section(t *testing.T, doc, heading string) string {
	t.Helper()
	_, body, ok := strings.Cut(doc, "\n"+SectionHeadingPrefix+heading+"\n")
	require.True(t, ok, "no section %q", SectionHeadingPrefix+heading)
	body, _, _ = strings.Cut(body, "\n"+SectionHeadingPrefix)
	return body
}
