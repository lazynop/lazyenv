package tui

import (
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/docstest"
	"github.com/lazynop/lazyenv/internal/model"
)

var (
	groupHeaderRe     = regexp.MustCompile(`(?m)^(\S+) ([A-Z]+) \(\d+\)$`)
	autoWidthRe       = regexp.MustCompile(`auto \(1/(\d+) `)
	skippedDirsRe     = regexp.MustCompile("skipping hidden directories, ([^.]*)\\.")
	standInSentenceRe = regexp.MustCompile(`with a visible stand-in for each: ([^\n]*?)\. The file keeps`)
)

func TestFeaturesDocumentsTheChangeMarkers(t *testing.T) {
	glyphByMeaning := map[string]string{
		"Added":             newVarGlyph,
		"Modified":          modifiedVarGlyph,
		"Deleted":           deletedVarGlyph,
		"Duplicate key":     duplicateKeyGlyph,
		"Empty value":       emptyValueGlyph,
		"Placeholder value": placeholderGlyph,
	}
	section := docstest.Section(t, docstest.ReadFile(t, "docs/features.md"), "Change tracking")
	documented := map[string]bool{}
	for _, row := range docstest.TableRows(t, section, "Marker") {
		meaning := row[1]
		documented[meaning] = true
		want, ok := glyphByMeaning[meaning]
		if assert.True(t, ok, "docs/features.md § Change tracking lists a %q marker, which this test does not map to a glyph", meaning) {
			assert.Equal(t, []string{want}, docstest.CodeSpans(row[0]), "docs/features.md § Change tracking states the wrong marker for %q", meaning)
		}
	}
	for _, meaning := range slices.Sorted(maps.Keys(glyphByMeaning)) {
		assert.True(t, documented[meaning], "docs/features.md § Change tracking has no row for the %q marker", meaning)
	}
}

func TestFeaturesDocumentsTheGroupHeaders(t *testing.T) {
	section := docstest.Section(t, docstest.ReadFile(t, "docs/features.md"), "Variable grouping")
	headers := groupHeaderRe.FindAllStringSubmatch(section, -1)
	require.NotEmpty(t, headers, "docs/features.md § Variable grouping should show group headers")
	for _, h := range headers {
		assert.Equal(t, expandedGroupGlyph, h[1], "docs/features.md § Variable grouping draws the header %q with the wrong glyph", h[0])
	}
	assert.Equal(t, ungroupedHeaderLabel, headers[len(headers)-1][2], "docs/features.md § Variable grouping should end its example on the ungrouped section's header")
	assert.Contains(t, docstest.CodeSpans(docstest.LineStartingWith(t, section, "Press ")), ungroupedHeaderLabel, "docs/features.md § Variable grouping should name the ungrouped section")
}

func TestFeaturesDocumentsTheControlCharacterStandIns(t *testing.T) {
	section := docstest.Section(t, docstest.ReadFile(t, "docs/features.md"), "Browse and edit")
	m := standInSentenceRe.FindStringSubmatch(section)
	require.NotNil(t, m, "docs/features.md § Browse and edit should list the stand-in of each control character")
	standIns := docstest.CodeSpans(m[1])
	inputsByStandIn := [][]string{
		{"\n"},
		{"\t"},
		{"\r"},
		{"\x7f"},
		{"\x00"},
		{"\x1f"},
		{"\x1b"},
		{"\u0080", "\u009f", "\xff"},
	}
	require.Len(t, standIns, len(inputsByStandIn), "docs/features.md § Browse and edit lists other stand-ins than this test checks: %q", standIns)
	for i, inputs := range inputsByStandIn {
		for _, input := range inputs {
			assert.Equal(t, standIns[i], flattenText(input), "docs/features.md § Browse and edit states the wrong stand-in for %q", input)
		}
	}
}

func TestFeaturesDocumentsThePeekLines(t *testing.T) {
	section := docstest.Section(t, docstest.ReadFile(t, "docs/features.md"), "Peek original values")
	var documented []string
	for _, span := range docstest.CodeSpans(section) {
		if strings.HasPrefix(span, "↳") {
			documented = append(documented, span)
		}
	}
	m := NewVarListModel(config.DefaultConfig().Layout)
	theme := BuildTheme(true, config.ColorConfig{})
	peekText := func(v *model.EnvVar) string {
		return strings.TrimSpace(ansi.Strip(m.renderPeekLine(v, 4, 80, theme)))
	}

	newLine := peekText(&model.EnvVar{Key: "NEW", Value: "v", IsNew: true})
	const original = "old"
	wasLine, ok := strings.CutSuffix(peekText(&model.EnvVar{Key: "MOD", Value: "v", OriginalValue: original}), " "+original)
	require.True(t, ok, "the peek line of a modified variable should end with its original value")
	assert.ElementsMatch(t, []string{wasLine, newLine}, documented, "docs/features.md § Peek original values states other peek lines than renderPeekLine draws")
}

func TestDocsStateTheAutoFileListWidth(t *testing.T) {
	surfaces := map[string]string{
		"docs/configuration.md, file-list-width row": docstest.RowText(t, docstest.ReadFile(t, "docs/configuration.md"), "Key", "file-list-width"),
		"docs/usage.md, --file-list-width row":       docstest.RowText(t, docstest.ReadFile(t, "docs/usage.md"), "Long form", "--file-list-width"),
		"examples/config/full.toml, file-list-width": docstest.LineStartingWith(t, docstest.ReadFile(t, "examples/config/full.toml"), "file-list-width ="),
	}
	const width = 400
	var app tea.Model = NewApp(config.DefaultConfig(), StartupMessages{})
	app, _ = app.Update(tea.WindowSizeMsg{Width: width, Height: 40})
	for _, name := range slices.Sorted(maps.Keys(surfaces)) {
		m := autoWidthRe.FindStringSubmatch(surfaces[name])
		if !assert.NotNil(t, m, "%s should state the share of the screen the auto width takes, as \"auto (1/N ...\"", name) {
			continue
		}
		divisor, err := strconv.Atoi(m[1])
		require.NoError(t, err)
		assert.Equal(t, width/divisor, app.(App).fileWidth, "%s states the wrong share of the screen for the auto file-list width", name)
	}
}

func TestDocsStateTheDirectoriesARecursiveScanSkips(t *testing.T) {
	surfaces := map[string]string{
		"docs/configuration.md, recursive row": docstest.RowText(t, docstest.ReadFile(t, "docs/configuration.md"), "Key", "recursive"),
		"docs/usage.md, --recursive row":       docstest.RowText(t, docstest.ReadFile(t, "docs/usage.md"), "Long form", "--recursive"),
	}
	var named []string
	for _, name := range skippedDirNames {
		if !strings.HasPrefix(name, ".") {
			named = append(named, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(surfaces)) {
		m := skippedDirsRe.FindStringSubmatch(surfaces[name])
		if assert.NotNil(t, m, "%s should state the directories a recursive scan skips, as \"skipping hidden directories, ...\"", name) {
			assert.ElementsMatch(t, named, docstest.CodeSpans(m[1]), "%s names other skipped directories than skippedDirNames", name)
		}
	}

	fixture := map[string]string{".env": "A=1\n", "sub/.env": "A=1\n", ".hidden/.env": "A=1\n"}
	for _, name := range named {
		fixture[filepath.Join(name, ".env")] = "A=1\n"
	}
	dir := setupScanDir(t, fixture)
	files, err := ScanDir(dir, true, config.DefaultConfig().Files, config.SecretsConfig{})
	require.NoError(t, err)
	var found []string
	for _, f := range files {
		rel, err := filepath.Rel(dir, f.Path)
		require.NoError(t, err)
		found = append(found, filepath.ToSlash(rel))
	}
	assert.ElementsMatch(t, []string{".env", "sub/.env"}, found, "a recursive scan should skip hidden directories and the named ones, as the docs state")
}
