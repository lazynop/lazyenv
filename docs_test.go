package main

import (
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/x/ansi"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/config/themes"
	"github.com/lazynop/lazyenv/internal/docstest"
	"github.com/lazynop/lazyenv/internal/parser"
	"github.com/lazynop/lazyenv/internal/tui"
	"github.com/lazynop/lazyenv/internal/util"
)

var (
	wordRe                       = regexp.MustCompile(`[A-Za-z_]\w*`)
	identRefRe                   = regexp.MustCompile(`^\*?([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)(\*?)(?:\(.*\))?$`)
	justRefRe                    = regexp.MustCompile(`^just ([a-z][\w-]*)(?:\s|$)`)
	justRecipeRe                 = regexp.MustCompile(`(?m)^([A-Za-z][\w-]*)(?:\s[^:\n]*)?:(?:\s|$)`)
	minimumRe                    = regexp.MustCompile("Minimum `(\\d+)`")
	themeCountRe                 = regexp.MustCompile(`(\d+) built-in (?:color )?themes`)
	goVersionRe                  = regexp.MustCompile(`Go (\d+\.\d+)\+`)
	goModRe                      = regexp.MustCompile(`(?m)^go (\d+\.\d+)`)
	terminalRe                   = regexp.MustCompile(`at least (\d+)x(\d+)`)
	varListMinRe                 = regexp.MustCompile("terminal width (?:minus|-) `?(\\d+)")
	helpKeyColumnToLastWideGapRe = regexp.MustCompile(`^ {2}(\S.*\S|\S) {2,}\S`)
)

func TestUsageDocumentsEveryFlag(t *testing.T) {
	flagParser, err := kong.New(&cli)
	require.NoError(t, err)
	doc := docstest.ReadFile(t, "docs/usage.md")

	flagNameBySpelling := map[string]string{}
	for _, f := range flagParser.Model.Flags {
		flagNameBySpelling["--"+f.Name] = f.Name
		if f.Short != 0 {
			flagNameBySpelling["-"+string(f.Short)] = f.Name
		}
		if f.Tag.Negatable != "" {
			flagNameBySpelling["--no-"+f.Name] = f.Name
		}
	}

	listed := map[string]bool{}
	for _, row := range docstest.TableRows(t, doc, "Long form") {
		for _, s := range slices.Concat(docstest.CodeSpans(row[0]), docstest.CodeSpans(row[1])) {
			listed[s] = true
			assert.Contains(t, flagNameBySpelling, s, "docs/usage.md lists %s in the flags table, but it is not a flag", s)
		}
	}
	for _, s := range slices.Sorted(maps.Keys(flagNameBySpelling)) {
		assert.True(t, listed[s], "docs/usage.md has no flags-table row for %s", s)
	}

	keys := configKeys("", reflect.TypeFor[config.Config]())
	accounted := map[string]bool{}
	for _, row := range docstest.TableRows(t, doc, "Config key") {
		for _, s := range docstest.CodeSpans(row[0]) {
			assert.Contains(t, flagNameBySpelling, s, "docs/usage.md maps %s to a config key, but it is not a flag", s)
			accounted[flagNameBySpelling[s]] = true
		}
		for _, k := range docstest.CodeSpans(row[1]) {
			assert.True(t, keys[k], "docs/usage.md maps a flag to %q, which is not a config key (nested keys are dotted: layout.file-list-width)", k)
		}
	}
	for _, s := range docstest.CodeSpans(docstest.LineStartingWith(t, doc, "Flags without a config equivalent")) {
		assert.Contains(t, flagNameBySpelling, s, "docs/usage.md lists %s among the flags without a config equivalent, but it is not a flag", s)
		accounted[flagNameBySpelling[s]] = true
	}
	for _, f := range flagParser.Model.Flags {
		assert.True(t, accounted[f.Name],
			"docs/usage.md neither maps --%s to a config key nor lists it among the flags without a config equivalent", f.Name)
	}
}

var layoutMinimums = map[string]int{
	"file-list-width":  config.FileListMinWidth,
	"var-list-padding": config.VarListMinPadding,
	"diff-padding":     config.DiffMinPadding,
}

func TestConfigurationDocumentsDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	defaultByOption := map[string]reflect.Value{}
	configDefaults(t, reflect.ValueOf(cfg), defaultByOption)
	defaultByOption["value_heuristic"] = reflect.ValueOf(cfg.Secrets.ValueHeuristicEnabled())

	documented := map[string]bool{}
	for _, table := range docstest.Tables(docstest.ReadFile(t, "docs/configuration.md")) {
		if !slices.Equal(table.Header, []string{"Key", "Default", "Description"}) {
			continue
		}
		for _, row := range table.Rows {
			name := docstest.CodeSpans(row[0])[0]
			documented[name] = true
			want, ok := defaultByOption[name]
			if !assert.True(t, ok, "docs/configuration.md documents option %s, which Config does not have", name) {
				continue
			}
			for _, span := range docstest.CodeSpans(row[1]) {
				assertDocDefault(t, name, span, want)
			}
			if m := minimumRe.FindStringSubmatch(row[2]); m != nil {
				minimum, isRaised := layoutMinimums[name]
				assert.True(t, isRaised, "docs/configuration.md gives %s a minimum, but the TUI raises no minimum for it", name)
				assert.Equal(t, strconv.Itoa(minimum), m[1], "docs/configuration.md states the wrong minimum for %s", name)
			} else if _, isRaised := layoutMinimums[name]; isRaised {
				t.Errorf("docs/configuration.md does not state the minimum (\"Minimum `%d`\") of %s", layoutMinimums[name], name)
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(defaultByOption)) {
		assert.True(t, documented[name], "docs/configuration.md has no options-table row for %s", name)
	}
}

func configDefaults(t *testing.T, v reflect.Value, out map[string]reflect.Value) {
	t.Helper()
	for field := range v.Type().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "" || name == "-" || name == "colors" {
			continue
		}
		if field.Type.Kind() == reflect.Struct {
			configDefaults(t, v.FieldByIndex(field.Index), out)
			continue
		}
		_, dup := out[name]
		require.False(t, dup, "option name %s appears in two tables: key defaults by table", name)
		out[name] = v.FieldByIndex(field.Index)
	}
}

func assertDocDefault(t *testing.T, name, span string, want reflect.Value) {
	t.Helper()
	holder := reflect.New(reflect.StructOf([]reflect.StructField{{Name: "V", Type: want.Type(), Tag: `toml:"v"`}}))
	if !assert.NoError(t, toml.Unmarshal([]byte("v = "+span), holder.Interface()), "docs/configuration.md: the default of %s is not a TOML value", name) {
		return
	}
	got := holder.Elem().Field(0)
	if want.Kind() == reflect.Slice && want.Len() == 0 && got.Len() == 0 {
		return
	}
	assert.Equal(t, want.Interface(), got.Interface(), "docs/configuration.md states the wrong default for %s", name)
}

func TestConfigurationDocumentsThemeColors(t *testing.T) {
	themes := map[string]map[string]string{}
	for _, name := range []string{"default-dark", "default-light"} {
		colors, ok := config.LookupTheme(name)
		require.True(t, ok, "no %s theme", name)
		themes[name] = colorsByKey(colors)
	}
	rows := docstest.TableRows(t, docstest.ReadFile(t, "docs/configuration.md"), "Dark default")
	documented := map[string]bool{}
	for _, row := range rows {
		name := docstest.CodeSpans(row[0])[0]
		documented[name] = true
		for i, theme := range []string{"default-dark", "default-light"} {
			want, ok := themes[theme][name]
			if !assert.True(t, ok, "docs/configuration.md lists color %s, which ColorConfig does not have", name) {
				break
			}
			got := strings.Join(docstest.CodeSpans(row[1+i]), "")
			assert.Equal(t, want, got, "docs/configuration.md states the wrong %s color for %s (an empty color is written (none), outside a code span)", theme, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(themes["default-dark"])) {
		assert.True(t, documented[name], "docs/configuration.md has no colors-table row for %s", name)
	}
}

func colorsByKey(c config.ColorConfig) map[string]string {
	colors := map[string]string{}
	v := reflect.ValueOf(c)
	for field := range v.Type().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		colors[name] = v.FieldByIndex(field.Index).String()
	}
	return colors
}

func TestDocsStateCurrentCounts(t *testing.T) {
	m := goModRe.FindStringSubmatch(docstest.ReadFile(t, "go.mod"))
	require.NotNil(t, m, "go.mod has no go directive")
	goVersion := m[1]
	themeCount := strconv.Itoa(len(config.ThemeNames()))
	terminal := []string{strconv.Itoa(tui.MinTerminalWidth), strconv.Itoa(tui.MinTerminalHeight)}
	assert.Regexp(t, terminalRe, docstest.ReadFile(t, "docs/installation.md"), "docs/installation.md should state the minimum terminal size")

	docs, err := filepath.Glob("docs/*.md")
	require.NoError(t, err)
	for _, path := range append(docs, "README.md") {
		recordsPastReleases := path == "docs/changelog.md"
		if recordsPastReleases {
			continue
		}
		doc := docstest.ReadFile(t, path)
		for _, m := range themeCountRe.FindAllStringSubmatch(doc, -1) {
			assert.Equal(t, themeCount, m[1], "%s states the wrong number of built-in themes", path)
		}
		for _, m := range goVersionRe.FindAllStringSubmatch(doc, -1) {
			assert.Equal(t, goVersion, m[1], "%s states a Go version other than go.mod's", path)
		}
		for _, m := range terminalRe.FindAllStringSubmatch(doc, -1) {
			assert.Equal(t, terminal, m[1:], "%s states the wrong minimum terminal size", path)
		}
	}
}

func TestDocsStateTheVarListMinimum(t *testing.T) {
	flag, ok := reflect.TypeOf(cli).FieldByName("FileListWidth")
	require.True(t, ok, "cli has no FileListWidth field to read the --file-list-width help from")
	surfaces := map[string]string{"the --file-list-width help": flag.Tag.Get("help")}
	for _, path := range []string{"docs/configuration.md", "docs/usage.md", "examples/config/full.toml"} {
		surfaces[path] = docstest.ReadFile(t, path)
	}
	for _, name := range slices.Sorted(maps.Keys(surfaces)) {
		m := varListMinRe.FindStringSubmatch(surfaces[name])
		if assert.NotNil(t, m, "%s should state how wide file-list-width can get", name) {
			assert.Equal(t, strconv.Itoa(config.VarListMinWidth), m[1], "%s states the wrong width the variables panel keeps", name)
		}
	}
}

var helpDocSections = map[string][]string{
	"Main view — File list (left panel)":      {"Files panel", "Both panels"},
	"Main view — Variable list (right panel)": {"Variables panel", "Both panels"},
	"Compare — Pick the second file":          {"Compare: pick the second file"},
	"Compare — Diff view":                     {"Compare view"},
	"Completeness matrix":                     {"Matrix"},
	"Prompts":                                 {"Prompts"},
	"Help screen (`?`)":                       {"This help"},
}

func TestHelpScreenMatchesKeybindingsDoc(t *testing.T) {
	help := map[string][][]string{}
	for _, section := range helpScreenSections(t) {
		help[section.title] = section.rows
	}
	mapped := map[string]bool{}
	doc := docstest.ReadFile(t, "docs/keybindings.md")
	for _, heading := range slices.Sorted(maps.Keys(helpDocSections)) {
		helpKeys := map[string]bool{}
		for _, title := range helpDocSections[heading] {
			require.Contains(t, help, title, "the help screen has no section %q", title)
			mapped[title] = true
			for _, row := range help[title] {
				for _, k := range row {
					helpKeys[k] = true
				}
			}
		}
		assert.Equal(t, slices.Sorted(maps.Keys(helpKeys)), slices.Sorted(maps.Keys(docSectionKeys(t, doc, heading))),
			"the help screen (%s) and docs/keybindings.md (%s) name different keys", strings.Join(helpDocSections[heading], " + "), heading)
	}
	for title := range help {
		assert.True(t, mapped[title], "help section %q maps to no docs/keybindings.md section in helpDocSections", title)
	}
	for line := range strings.Lines(doc) {
		heading, ok := strings.CutPrefix(strings.TrimSuffix(line, "\n"), docstest.SectionHeadingPrefix)
		if !ok || helpDocSections[heading] != nil || docOnlySections[heading] != "" {
			continue
		}
		assert.Empty(t, docSectionKeys(t, doc, heading),
			"docs/keybindings.md section %q lists keys but is in neither helpDocSections nor docOnlySections", heading)
	}
	for heading := range docOnlySections {
		assert.NotContains(t, helpDocSections, heading, "docOnlySections exempts %q, which helpDocSections maps", heading)
		assert.NotEmpty(t, docSectionKeys(t, doc, heading), "docOnlySections exempts %q, which lists no keys", heading)
	}
}

var docOnlySections = map[string]string{
	"Theme preview (`--themes`)": "a separate program mode with no help screen",
}

func docSectionKeys(t *testing.T, doc, heading string) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	for _, table := range docstest.Tables(docstest.Section(t, doc, heading)) {
		for _, row := range table.Rows {
			require.Len(t, row, len(table.Header), "a row of the %q table has the wrong number of cells: %v", heading, row)
		}
		for i, column := range table.Header {
			if column != "Key" && column != "Confirm" && column != "Cancel" {
				continue
			}
			for _, row := range table.Rows {
				for _, s := range docstest.CodeSpans(row[i]) {
					keys[s] = true
				}
			}
		}
	}
	return keys
}

type helpSection struct {
	title string
	rows  [][]string
}

const (
	helpScreenWidth  = 300
	helpScreenHeight = 200
)

func helpScreenSections(t *testing.T) []helpSection {
	t.Helper()
	screen := helpScreen(helpScreenHeight)
	require.Equal(t, strings.Fields(helpScreen(2*helpScreenHeight)), strings.Fields(screen),
		"the help does not fit in %d rows: raise helpScreenHeight", helpScreenHeight)

	var lines []string
	titleColumn := len(screen)
	for line := range strings.Lines(screen) {
		line = strings.TrimRight(line, " \n")
		if line != "" {
			lines = append(lines, line)
			titleColumn = min(titleColumn, len(line)-len(strings.TrimLeft(line, " ")))
		}
	}
	var sections []helpSection
	for _, line := range lines {
		line = line[titleColumn:]
		isSectionTitle := !strings.HasPrefix(line, " ")
		if isSectionTitle {
			sections = append(sections, helpSection{title: line})
			continue
		}
		require.NotEmpty(t, sections, "the help screen has a key row before any section title: %q", line)
		section := &sections[len(sections)-1]
		listsGlyphsNotKeys := strings.Contains(section.title, "Indicators")
		if listsGlyphsNotKeys {
			continue
		}
		m := helpKeyColumnToLastWideGapRe.FindStringSubmatch(line)
		require.NotNil(t, m, "row %q of help section %s has no key column; if it is a wrapped line, raise helpScreenWidth (%d)", line, section.title, helpScreenWidth)
		section.rows = append(section.rows, helpKeys(m[1]))
	}
	return slices.DeleteFunc(sections, func(s helpSection) bool { return len(s.rows) == 0 })
}

func helpScreen(height int) string {
	var app tea.Model = tui.NewApp(config.DefaultConfig(), tui.StartupMessages{})
	app, _ = app.Update(tea.WindowSizeMsg{Width: helpScreenWidth, Height: height})
	app, _ = app.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	return ansi.Strip(app.View().Content)
}

func helpKeys(column string) []string {
	var keys []string
	for alternative := range strings.SplitSeq(column, " / ") {
		for part := range strings.FieldsSeq(alternative) {
			if part == "/" {
				keys = append(keys, part)
				continue
			}
			for k := range strings.SplitSeq(part, "/") {
				if k != "" {
					keys = append(keys, k)
				}
			}
		}
	}
	return keys
}

func TestKeybindingsDocumentsEveryKey(t *testing.T) {
	documented := map[string]bool{}
	for _, table := range docstest.Tables(docstest.ReadFile(t, "docs/keybindings.md")) {
		if table.Header[0] != "Key" {
			continue
		}
		for _, row := range table.Rows {
			for _, s := range docstest.CodeSpans(row[0]) {
				documented[s] = true
			}
		}
	}

	bound := map[string]bool{}
	keyMap := reflect.ValueOf(tui.DefaultKeyMap())
	for field := range keyMap.Type().Fields() {
		binding, ok := keyMap.FieldByIndex(field.Index).Interface().(key.Binding)
		require.True(t, ok, "KeyMap.%s is not a key.Binding", field.Name)
		for _, k := range binding.Keys() {
			notation := tui.KeyLabel(k)
			bound[notation] = true
			assert.True(t, documented[notation], "KeyMap.%s binds %q, but no Key column in docs/keybindings.md has `%s`", field.Name, k, notation)
		}
	}
	for _, s := range slices.Sorted(maps.Keys(documented)) {
		assert.True(t, bound[s], "docs/keybindings.md documents `%s`, but no KeyMap binding uses it", s)
	}
}

var notInRepo = map[string]string{
	"foo_test.go":         "an example file name",
	"vX.Y.Z":              "the shape of a version tag",
	"docs/changelog.md":   "generated in CI from CHANGELOG.md",
	"dist/page-media/":    "build output",
	"site/lazyenv/media/": "a path in the lazynop.dev repo",
}

func TestCodingStandardsNamesExistingCode(t *testing.T) {
	repo := newRepoIndex(t)
	doc := docstest.ReadFile(t, "CODING_STANDARDS.md")
	for ref := range notInRepo {
		assert.Contains(t, doc, "`"+ref+"`", "notInRepo lists `%s`, which CODING_STANDARDS.md no longer names: drop it", ref)
	}
	for _, ref := range docstest.CodeSpans(doc) {
		if _, ok := notInRepo[ref]; ok {
			continue
		}
		assert.Empty(t, repo.missing(ref), "CODING_STANDARDS.md names `%s`", ref)
	}
}

func TestDocsNameExistingJustRecipes(t *testing.T) {
	repo := newRepoIndex(t)
	pages, err := filepath.Glob("docs/*.md")
	require.NoError(t, err)
	spans, commands := 0, 0
	for _, path := range append(pages, "README.md", "CONTRIBUTING.md") {
		doc := docstest.ReadFile(t, path)
		for _, ref := range docstest.CodeSpans(doc) {
			if justRefRe.MatchString(ref) {
				spans++
				assert.Empty(t, repo.missing(ref), "%s names `%s`", path, ref)
			}
		}
		for _, command := range justCommandLines(doc) {
			commands++
			assert.Empty(t, repo.missing(command), "%s runs `%s` in a fenced block", path, command)
		}
	}
	assert.NotZero(t, spans, "the docs scan found no `just` span to check")
	assert.NotZero(t, commands, "the docs scan found no `just` command in a fenced block to check")
}

func justCommandLines(doc string) []string {
	var commands []string
	for _, line := range docstest.FencedLines(doc) {
		if justRefRe.MatchString(line) {
			commands = append(commands, line)
		}
	}
	return commands
}

const fencedJustDoc = "Run `just run` first.\n\n" +
	"```bash\n" +
	"just build        # build to bin/lazyenv\n" +
	"echo done\n" +
	"```\n\n" +
	"```toml\n" +
	"[colors]\n" +
	"primary = \"#FF6600\"    # override just primary\n" +
	"```\n"

func TestJustCommandLinesReadAFencedCommandWithATrailingComment(t *testing.T) {
	assert.Equal(t, []string{"just build        # build to bin/lazyenv"}, justCommandLines(fencedJustDoc), "the just command lines of the fenced blocks")
}

func TestJustCommandLinesSkipFencedProseHoldingJust(t *testing.T) {
	assert.NotContains(t, justCommandLines(fencedJustDoc), `primary = "#FF6600"    # override just primary`, "a TOML comment that holds the word just is read as a command")
}

func TestRepoIndexChecksTheRecipeAJustSpanNames(t *testing.T) {
	repo := repoIndex{justRecipes: map[string]bool{"mutant": true, "comment-baseline": true}}
	for _, tc := range []struct {
		ref, problem string
	}{
		{"just mutant", ""},
		{"just mutnt", "the justfile has no mutnt recipe"},
		{"just mutant <file> <old> <new> '^TestName$'", ""},
		{"just mutnt <file> <old> <new> '^TestName$'", "the justfile has no mutnt recipe"},
		{"just comment-baseline -allow-raise", ""},
		{"just comment-baselin -allow-raise", "the justfile has no comment-baselin recipe"},
		{"just --list", ""},
		{"just os=linux mutnt", ""},
	} {
		assert.Equal(t, tc.problem, repo.missing(tc.ref), "missing(%q)", tc.ref)
	}
}

type repoIndex struct {
	sourceWords map[string]bool
	baseNames   map[string]bool
	justRecipes map[string]bool
	goMod       string
}

func newRepoIndex(t *testing.T) repoIndex {
	t.Helper()
	repo := repoIndex{sourceWords: map[string]bool{}, baseNames: map[string]bool{}, justRecipes: map[string]bool{}, goMod: docstest.ReadFile(t, "go.mod")}
	sources, err := filepath.Glob(".github/workflows/*.yml")
	require.NoError(t, err)
	sources = append(sources, "justfile", "go.mod", ".goreleaser.yml")

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, e := range entries {
		if !e.IsDir() {
			repo.baseNames[e.Name()] = true
		}
		if filepath.Ext(e.Name()) == ".go" {
			sources = append(sources, e.Name())
		}
	}
	err = filepath.WalkDir("internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		repo.baseNames[d.Name()] = true
		if filepath.Ext(path) == ".go" {
			sources = append(sources, path)
		}
		return nil
	})
	require.NoError(t, err)

	for _, path := range sources {
		for _, w := range wordRe.FindAllString(docstest.ReadFile(t, path), -1) {
			repo.sourceWords[w] = true
		}
	}
	for _, m := range justRecipeRe.FindAllStringSubmatch(docstest.ReadFile(t, "justfile"), -1) {
		repo.justRecipes[m[1]] = true
	}
	return repo
}

func (r repoIndex) missing(ref string) string {
	if m := justRefRe.FindStringSubmatch(ref); m != nil {
		if !r.justRecipes[m[1]] {
			return "the justfile has no " + m[1] + " recipe"
		}
		return ""
	}
	if strings.Contains(ref, "/") {
		return r.missingPath(ref)
	}
	if r.baseNames[ref] {
		return ""
	}
	switch filepath.Ext(ref) {
	case ".go", ".md", ".toml", ".yml":
		return "no file by that name is at the root or under internal/"
	}
	if m := identRefRe.FindStringSubmatch(ref); m != nil {
		return r.missingIdent(m[1], m[2] == "*")
	}
	return ""
}

func (r repoIndex) missingPath(ref string) string {
	elidedOrSnippet := strings.Contains(ref, "...") || strings.ContainsAny(ref, " \"")
	if elidedOrSnippet {
		return ""
	}
	if first, _, _ := strings.Cut(ref, "/"); strings.Contains(first, ".") && !strings.HasPrefix(first, ".") {
		if !strings.Contains(r.goMod, ref) {
			return "go.mod does not require that module"
		}
		return ""
	}
	if matches, err := filepath.Glob(strings.TrimSuffix(ref, "/")); err != nil || len(matches) == 0 {
		return "it is not in the repo"
	}
	return ""
}

func (r repoIndex) missingIdent(chain string, lastSegmentIsPrefix bool) string {
	segments := strings.Split(chain, ".")
	for i, seg := range segments {
		receiverOrPlaceholder := len(seg) == 1
		if receiverOrPlaceholder {
			continue
		}
		if i == len(segments)-1 && lastSegmentIsPrefix {
			if !slices.ContainsFunc(slices.Collect(maps.Keys(r.sourceWords)), func(w string) bool { return strings.HasPrefix(w, seg) }) {
				return "no word in a Go source or build file starts with " + seg
			}
			continue
		}
		if !r.sourceWords[seg] {
			return seg + " appears in no Go source or build file"
		}
	}
	return ""
}

func configKeys(table string, typ reflect.Type) map[string]bool {
	keys := map[string]bool{}
	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "" || name == "-" {
			continue
		}
		path := name
		if table != "" {
			path = table + "." + name
		}
		if field.Type.Kind() == reflect.Struct {
			maps.Copy(keys, configKeys(path, field.Type))
			continue
		}
		keys[path] = true
	}
	return keys
}

func TestConfigurationCreditsEveryTheme(t *testing.T) {
	var credited []string
	for _, table := range docstest.Tables(docstest.ReadFile(t, "docs/configuration.md")) {
		if !slices.Equal(table.Header, []string{"Theme", "Palette source"}) {
			continue
		}
		for _, row := range table.Rows {
			name := docstest.CodeSpans(row[0])[0]
			credited = append(credited, name)
			assert.Regexp(t, `^(<https://\S+>|lazyenv original)$`, row[1], "credit of %s", name)
		}
	}
	slices.Sort(credited)
	assert.Equal(t, themes.Names(), credited)
}

func TestConfigurationDocumentsTheSecretMask(t *testing.T) {
	row := docstest.TableRow(t, docstest.ReadFile(t, "docs/configuration.md"), "Key", "show-secrets")
	assert.Contains(t, docstest.CodeSpans(row[2]), util.MaskValue("secret"), "docs/configuration.md, show-secrets row, states the wrong mask")
}

var themeNoticeRe = regexp.MustCompile("unknown `--theme` .* for (\\d+) seconds")

func TestConfigurationDocumentsTheThemeNoticeDuration(t *testing.T) {
	section := docstest.Section(t, docstest.ReadFile(t, "docs/configuration.md"), "Invalid configuration")
	m := themeNoticeRe.FindStringSubmatch(section)
	require.NotNil(t, m, "docs/configuration.md § Invalid configuration should state how long the unknown --theme notice shows")
	timeout := config.DefaultConfig().Layout.ErrorMessageTimeout
	assert.Equal(t, strconv.Itoa(int(timeout.Seconds())), m[1], "docs/configuration.md § Invalid configuration states the wrong duration of the unknown --theme notice")
}

var numberedItemRe = regexp.MustCompile(`(?m)^\d+\. (.*)$`)

func TestConfigurationDocumentsTheConfigSearchOrder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	userConfigDir, err := os.UserConfigDir()
	require.NoError(t, err)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	projectDir := filepath.Join(root, "project")
	configFlag := filepath.Join(root, "flag.toml")

	asDocumented := func(path string) string {
		if path == configFlag {
			return "--config"
		}
		for _, base := range []struct{ dir, prefix string }{{projectDir, ""}, {userConfigDir, ""}, {home, "~/"}} {
			if rel, err := filepath.Rel(base.dir, path); err == nil && !strings.HasPrefix(rel, "..") {
				return base.prefix + filepath.ToSlash(rel)
			}
		}
		require.Failf(t, "unexpected path", "ConfigSearchPaths returned %s, outside every directory this test knows", path)
		return ""
	}

	items := numberedItemRe.FindAllStringSubmatch(docstest.Section(t, docstest.ReadFile(t, "docs/configuration.md"), "Config file location"), -1)
	paths := config.ConfigSearchPaths(projectDir, configFlag)
	require.Len(t, items, len(paths), "docs/configuration.md § Config file location lists another number of places than ConfigSearchPaths searches")
	for i, path := range paths {
		assert.Contains(t, docstest.CodeSpans(items[i][1]), asDocumented(path), "docs/configuration.md § Config file location states the wrong place at step %d", i+1)
	}
}

var (
	releasePlatformsRe = regexp.MustCompile(`Builds are available for ([^\n]*)\.`)
	releaseOSRe        = regexp.MustCompile(`(\w+) \(([^)]*)\)`)
	goreleaserKeyRe    = regexp.MustCompile(`^(?: {2}- | {4})(\w+):`)
	goreleaserItemRe   = regexp.MustCompile(`^ {6}- (?:(\w+): )?"?([^"]*)"?$`)
	goreleaserPairRe   = regexp.MustCompile(`^ {8}(\w+): "?([^"]*)"?$`)
)

var releaseOSNames = map[string]string{"linux": "Linux", "darwin": "macOS", "windows": "Windows", "freebsd": "FreeBSD"}

func TestInstallationListsTheReleasePlatforms(t *testing.T) {
	m := releasePlatformsRe.FindStringSubmatch(docstest.Section(t, docstest.ReadFile(t, "docs/installation.md"), "From releases"))
	require.NotNil(t, m, "docs/installation.md § From releases should list the platforms, as \"Builds are available for ...\"")
	var documented []string
	for _, platform := range releaseOSRe.FindAllStringSubmatch(m[1], -1) {
		for arch := range strings.SplitSeq(platform[2], ", ") {
			documented = append(documented, platform[1]+" "+arch)
		}
	}

	var built []string
	for _, p := range goreleaserPlatforms(t, docstest.ReadFile(t, ".goreleaser.yml")) {
		name, ok := releaseOSNames[p["goos"]]
		require.True(t, ok, ".goreleaser.yml builds for %s, which releaseOSNames does not name", p["goos"])
		arch := p["goarch"]
		if arch == "arm" {
			arch += "v" + p["goarm"]
		}
		built = append(built, name+" "+arch)
	}
	assert.ElementsMatch(t, built, documented, "docs/installation.md § From releases lists other platforms than .goreleaser.yml builds")
}

func readGoreleaserBuild(t *testing.T, yml string) (lists map[string][]string, ignored []map[string]string) {
	t.Helper()
	_, builds, ok := strings.Cut(yml, "\nbuilds:\n")
	require.True(t, ok, ".goreleaser.yml has no builds section")
	lists = map[string][]string{}
	key := ""
	for line := range strings.Lines(builds) {
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			break
		}
		if strings.HasPrefix(line, "  - ") {
			require.Empty(t, lists, ".goreleaser.yml has a second build: readGoreleaserBuild reads only one")
		}
		if m := goreleaserKeyRe.FindStringSubmatch(line); m != nil {
			key = m[1]
			continue
		}
		if m := goreleaserItemRe.FindStringSubmatch(line); m != nil {
			if key == "ignore" {
				require.NotEmpty(t, m[1], ".goreleaser.yml: an ignore entry should start with a key: %q", line)
				ignored = append(ignored, map[string]string{m[1]: m[2]})
			} else {
				lists[key] = append(lists[key], m[2])
			}
			continue
		}
		if m := goreleaserPairRe.FindStringSubmatch(line); m != nil && key == "ignore" && len(ignored) > 0 {
			ignored[len(ignored)-1][m[1]] = m[2]
			continue
		}
		require.Failf(t, "unknown line", ".goreleaser.yml: readGoreleaserBuild does not know the shape of the builds line %q", line)
	}
	return lists, ignored
}

func goreleaserPlatforms(t *testing.T, yml string) []map[string]string {
	t.Helper()
	lists, ignored := readGoreleaserBuild(t, yml)
	supported := goTargets(t)
	var platforms []map[string]string
	for _, goos := range lists["goos"] {
		for _, goarch := range lists["goarch"] {
			if !supported[goos+"/"+goarch] {
				continue
			}
			goarms := []string{""}
			if goarch == "arm" {
				goarms = lists["goarm"]
			}
			for _, goarm := range goarms {
				p := map[string]string{"goos": goos, "goarch": goarch, "goarm": goarm}
				if !slices.ContainsFunc(ignored, func(rule map[string]string) bool { return isSubset(rule, p) }) {
					platforms = append(platforms, p)
				}
			}
		}
	}
	return platforms
}

func goTargets(t *testing.T) map[string]bool {
	t.Helper()
	out, err := exec.Command("go", "tool", "dist", "list").Output()
	require.NoError(t, err)
	targets := map[string]bool{}
	for _, target := range strings.Fields(string(out)) {
		targets[target] = true
	}
	return targets
}

func isSubset(rule, platform map[string]string) bool {
	for k, v := range rule {
		if platform[k] != v {
			return false
		}
	}
	return true
}

func TestDocsStateTheBackupName(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(original, []byte("A=1\n"), 0o600))
	require.NoError(t, parser.CreateBackup(original))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var backup string
	for _, e := range entries {
		if e.Name() != ".env" {
			backup = e.Name()
		}
	}
	require.Len(t, entries, 2, "CreateBackup should write one backup next to the file")
	suffix, ok := strings.CutPrefix(backup, ".env")
	require.True(t, ok, "the backup %s of .env should start with .env", backup)

	features := docstest.Section(t, docstest.ReadFile(t, "docs/features.md"), "Automatic backup")
	assert.Contains(t, docstest.CodeSpans(features), backup, "docs/features.md § Automatic backup states the wrong backup name of .env")
	surfaces := map[string]string{
		"docs/usage.md, --no-backup row":       docstest.RowText(t, docstest.ReadFile(t, "docs/usage.md"), "Long form", "--no-backup"),
		"docs/configuration.md, no-backup row": docstest.RowText(t, docstest.ReadFile(t, "docs/configuration.md"), "Key", "no-backup"),
	}
	for _, name := range slices.Sorted(maps.Keys(surfaces)) {
		assert.Contains(t, docstest.CodeSpans(surfaces[name]), suffix, "%s states the wrong backup suffix", name)
	}
}
