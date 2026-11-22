package main

import (
	"io/fs"
	"maps"
	"os"
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
	"github.com/lazynop/lazyenv/internal/tui"
)

var (
	codeSpanRe                   = regexp.MustCompile("`([^`]+)`")
	wordRe                       = regexp.MustCompile(`[A-Za-z_]\w*`)
	identRefRe                   = regexp.MustCompile(`^\*?([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)(\*?)(?:\(.*\))?$`)
	justRefRe                    = regexp.MustCompile(`^just ([a-z][\w-]*)$`)
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
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	doc := readFile(t, "docs/usage.md")

	flagNameBySpelling := map[string]string{}
	for _, f := range parser.Model.Flags {
		flagNameBySpelling["--"+f.Name] = f.Name
		if f.Short != 0 {
			flagNameBySpelling["-"+string(f.Short)] = f.Name
		}
		if f.Tag.Negatable != "" {
			flagNameBySpelling["--no-"+f.Name] = f.Name
		}
	}

	listed := map[string]bool{}
	for _, row := range markdownTable(t, doc, "Long form") {
		for _, s := range slices.Concat(codeSpans(row[0]), codeSpans(row[1])) {
			listed[s] = true
			assert.Contains(t, flagNameBySpelling, s, "docs/usage.md lists %s in the flags table, but it is not a flag", s)
		}
	}
	for _, s := range slices.Sorted(maps.Keys(flagNameBySpelling)) {
		assert.True(t, listed[s], "docs/usage.md has no flags-table row for %s", s)
	}

	keys := configKeys("", reflect.TypeFor[config.Config]())
	accounted := map[string]bool{}
	for _, row := range markdownTable(t, doc, "Config key") {
		for _, s := range codeSpans(row[0]) {
			assert.Contains(t, flagNameBySpelling, s, "docs/usage.md maps %s to a config key, but it is not a flag", s)
			accounted[flagNameBySpelling[s]] = true
		}
		for _, k := range codeSpans(row[1]) {
			assert.True(t, keys[k], "docs/usage.md maps a flag to %q, which is not a config key (nested keys are dotted: layout.file-list-width)", k)
		}
	}
	for _, s := range codeSpans(lineStartingWith(t, doc, "Flags without a config equivalent")) {
		assert.Contains(t, flagNameBySpelling, s, "docs/usage.md lists %s among the flags without a config equivalent, but it is not a flag", s)
		accounted[flagNameBySpelling[s]] = true
	}
	for _, f := range parser.Model.Flags {
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
	for _, table := range markdownTables(readFile(t, "docs/configuration.md")) {
		if !slices.Equal(table.header, []string{"Key", "Default", "Description"}) {
			continue
		}
		for _, row := range table.rows {
			name := codeSpans(row[0])[0]
			documented[name] = true
			want, ok := defaultByOption[name]
			if !assert.True(t, ok, "docs/configuration.md documents option %s, which Config does not have", name) {
				continue
			}
			for _, span := range codeSpans(row[1]) {
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
	rows := markdownTable(t, readFile(t, "docs/configuration.md"), "Dark default")
	documented := map[string]bool{}
	for _, row := range rows {
		name := codeSpans(row[0])[0]
		documented[name] = true
		for i, theme := range []string{"default-dark", "default-light"} {
			want, ok := themes[theme][name]
			if !assert.True(t, ok, "docs/configuration.md lists color %s, which ColorConfig does not have", name) {
				break
			}
			got := strings.Join(codeSpans(row[1+i]), "")
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
	m := goModRe.FindStringSubmatch(readFile(t, "go.mod"))
	require.NotNil(t, m, "go.mod has no go directive")
	goVersion := m[1]
	themeCount := strconv.Itoa(len(config.ThemeNames()))
	terminal := []string{strconv.Itoa(tui.MinTerminalWidth), strconv.Itoa(tui.MinTerminalHeight)}
	assert.Regexp(t, terminalRe, readFile(t, "docs/installation.md"), "docs/installation.md should state the minimum terminal size")

	docs, err := filepath.Glob("docs/*.md")
	require.NoError(t, err)
	for _, path := range append(docs, "README.md") {
		recordsPastReleases := path == "docs/changelog.md"
		if recordsPastReleases {
			continue
		}
		doc := readFile(t, path)
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
		surfaces[path] = readFile(t, path)
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
	doc := readFile(t, "docs/keybindings.md")
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
		heading, ok := strings.CutPrefix(strings.TrimSuffix(line, "\n"), sectionHeadingPrefix)
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
	for _, table := range markdownTables(markdownSection(t, doc, heading)) {
		for _, row := range table.rows {
			require.Len(t, row, len(table.header), "a row of the %q table has the wrong number of cells: %v", heading, row)
		}
		for i, column := range table.header {
			if column != "Key" && column != "Confirm" && column != "Cancel" {
				continue
			}
			for _, row := range table.rows {
				for _, s := range codeSpans(row[i]) {
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
	for _, table := range markdownTables(readFile(t, "docs/keybindings.md")) {
		if table.header[0] != "Key" {
			continue
		}
		for _, row := range table.rows {
			for _, s := range codeSpans(row[0]) {
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
	doc := readFile(t, "CODING_STANDARDS.md")
	for ref := range notInRepo {
		assert.Contains(t, doc, "`"+ref+"`", "notInRepo lists `%s`, which CODING_STANDARDS.md no longer names: drop it", ref)
	}
	for _, ref := range codeSpans(doc) {
		if _, ok := notInRepo[ref]; ok {
			continue
		}
		if problem := repo.missing(ref); problem != "" {
			t.Errorf("CODING_STANDARDS.md names `%s`, but %s", ref, problem)
		}
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
	repo := repoIndex{sourceWords: map[string]bool{}, baseNames: map[string]bool{}, justRecipes: map[string]bool{}, goMod: readFile(t, "go.mod")}
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
		for _, w := range wordRe.FindAllString(readFile(t, path), -1) {
			repo.sourceWords[w] = true
		}
	}
	for _, m := range justRecipeRe.FindAllStringSubmatch(readFile(t, "justfile"), -1) {
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

type docTable struct {
	header []string
	rows   [][]string
}

func markdownTables(doc string) []docTable {
	var tables []docTable
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
			tables = append(tables, docTable{header: cells})
			inTable = true
		case isSeparatorRow(line):
		default:
			last := &tables[len(tables)-1]
			last.rows = append(last.rows, cells)
		}
	}
	return tables
}

func isSeparatorRow(line string) bool {
	return strings.Trim(line, "|-: ") == ""
}

func markdownTable(t *testing.T, doc, header string) [][]string {
	t.Helper()
	for _, table := range markdownTables(doc) {
		if slices.Contains(table.header, header) {
			return table.rows
		}
	}
	require.Failf(t, "missing table", "no table with a %q column", header)
	return nil
}

func lineStartingWith(t *testing.T, doc, prefix string) string {
	t.Helper()
	for line := range strings.Lines(doc) {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	require.Failf(t, "missing line", "no line starts with %q", prefix)
	return ""
}

const sectionHeadingPrefix = "## "

func markdownSection(t *testing.T, doc, heading string) string {
	t.Helper()
	_, body, ok := strings.Cut(doc, "\n"+sectionHeadingPrefix+heading+"\n")
	require.True(t, ok, "no section %q", sectionHeadingPrefix+heading)
	body, _, _ = strings.Cut(body, "\n"+sectionHeadingPrefix)
	return body
}

func codeSpans(s string) []string {
	var spans []string
	for _, m := range codeSpanRe.FindAllStringSubmatch(s, -1) {
		spans = append(spans, m[1])
	}
	return spans
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestConfigurationCreditsEveryTheme(t *testing.T) {
	var credited []string
	for _, table := range markdownTables(readFile(t, "docs/configuration.md")) {
		if !slices.Equal(table.header, []string{"Theme", "Palette source"}) {
			continue
		}
		for _, row := range table.rows {
			name := codeSpans(row[0])[0]
			credited = append(credited, name)
			assert.Regexp(t, `^(<https://\S+>|lazyenv original)$`, row[1], "credit of %s", name)
		}
	}
	slices.Sort(credited)
	assert.Equal(t, themes.Names(), credited)
}
