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
	"github.com/alecthomas/kong"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/tui"
)

// The tests in this file keep the docs in step with the code: each one fails
// when a doc names something the code no longer has, or when the code gains
// something its doc leaves out.

var (
	codeSpanRe   = regexp.MustCompile("`([^`]+)`")
	wordRe       = regexp.MustCompile(`[A-Za-z_]\w*`)
	identRefRe   = regexp.MustCompile(`^\*?([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)(\*?)(?:\(.*\))?$`)
	justRefRe    = regexp.MustCompile(`^just ([a-z][\w-]*)$`)
	justRecipeRe = regexp.MustCompile(`(?m)^([A-Za-z][\w-]*)(?:\s[^:\n]*)?:(?:\s|$)`)
	minimumRe    = regexp.MustCompile("Minimum `(\\d+)`")
	themeCountRe = regexp.MustCompile(`(\d+) built-in (?:color )?themes`)
	goVersionRe  = regexp.MustCompile(`Go (\d+\.\d+)\+`)
	goModRe      = regexp.MustCompile(`(?m)^go (\d+\.\d+)`)
	terminalRe   = regexp.MustCompile(`at least (\d+)x(\d+)`)
)

// TestUsageDocumentsEveryFlag checks docs/usage.md against the Kong flags.
// The flags table lists every spelling of every flag and nothing else. Every
// flag either maps to a real config key in the flag → config key table, or is
// named in the sentence on flags without a config equivalent.
func TestUsageDocumentsEveryFlag(t *testing.T) {
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	doc := readFile(t, "docs/usage.md")

	// Every spelling a user can type, mapped to the flag's name.
	spellings := map[string]string{}
	for _, f := range parser.Model.Flags {
		spellings["--"+f.Name] = f.Name
		if f.Short != 0 {
			spellings["-"+string(f.Short)] = f.Name
		}
		if f.Tag.Negatable != "" { // negatable:"" adds --no-<name>
			spellings["--no-"+f.Name] = f.Name
		}
	}

	listed := map[string]bool{}
	for _, row := range markdownTable(t, doc, "Long form") {
		for _, s := range slices.Concat(codeSpans(row[0]), codeSpans(row[1])) {
			listed[s] = true
			assert.Contains(t, spellings, s, "docs/usage.md lists %s in the flags table, but it is not a flag", s)
		}
	}
	for _, s := range slices.Sorted(maps.Keys(spellings)) {
		assert.True(t, listed[s], "docs/usage.md has no flags-table row for %s", s)
	}

	keys := configKeys("", reflect.TypeFor[config.Config]())
	accounted := map[string]bool{}
	for _, row := range markdownTable(t, doc, "Config key") {
		for _, s := range codeSpans(row[0]) {
			assert.Contains(t, spellings, s, "docs/usage.md maps %s to a config key, but it is not a flag", s)
			accounted[spellings[s]] = true
		}
		for _, k := range codeSpans(row[1]) {
			assert.True(t, keys[k], "docs/usage.md maps a flag to %q, which is not a config key (nested keys are dotted: layout.file-list-width)", k)
		}
	}
	for _, s := range codeSpans(lineStartingWith(t, doc, "Flags without a config equivalent")) {
		assert.Contains(t, spellings, s, "docs/usage.md lists %s among the flags without a config equivalent, but it is not a flag", s)
		accounted[spellings[s]] = true
	}
	for _, f := range parser.Model.Flags {
		assert.True(t, accounted[f.Name],
			"docs/usage.md neither maps --%s to a config key nor lists it among the flags without a config equivalent", f.Name)
	}
}

// layoutMinimums are the layout options the TUI raises to a minimum, which
// docs/configuration.md states as "Minimum `N`".
var layoutMinimums = map[string]int{
	"file-list-width":  config.FileListMinWidth,
	"var-list-padding": config.VarListMinPadding,
	"diff-padding":     config.DiffMinPadding,
}

// TestConfigurationDocumentsDefaults checks the options tables of
// docs/configuration.md: every option has a row, each Default matches
// DefaultConfig, and each "Minimum `N`" matches the minimum the TUI applies.
func TestConfigurationDocumentsDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	defaults := map[string]reflect.Value{} // option name → default
	configDefaults(t, reflect.ValueOf(cfg), defaults)
	// A nil *bool stands for a default; compare the doc against the value it means.
	defaults["value_heuristic"] = reflect.ValueOf(cfg.Secrets.ValueHeuristicEnabled())

	documented := map[string]bool{}
	for _, table := range markdownTables(readFile(t, "docs/configuration.md")) {
		if !slices.Equal(table.header, []string{"Key", "Default", "Description"}) {
			continue
		}
		for _, row := range table.rows {
			name := codeSpans(row[0])[0]
			documented[name] = true
			want, ok := defaults[name]
			if !assert.True(t, ok, "docs/configuration.md documents option %s, which Config does not have", name) {
				continue
			}
			for _, span := range codeSpans(row[1]) { // none for "See description"
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
	for _, name := range slices.Sorted(maps.Keys(defaults)) {
		assert.True(t, documented[name], "docs/configuration.md has no options-table row for %s", name)
	}
}

// configDefaults collects the value of every option in v (a Config or one of
// its tables) by option name, skipping [colors]: its defaults come from the
// built-in themes, see TestConfigurationDocumentsThemeColors.
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

// assertDocDefault decodes span, a TOML value from the Default column, as the
// option's type and compares it to want.
func assertDocDefault(t *testing.T, name, span string, want reflect.Value) {
	t.Helper()
	holder := reflect.New(reflect.StructOf([]reflect.StructField{{Name: "V", Type: want.Type(), Tag: `toml:"v"`}}))
	if !assert.NoError(t, toml.Unmarshal([]byte("v = "+span), holder.Interface()), "docs/configuration.md: the default of %s is not a TOML value", name) {
		return
	}
	got := holder.Elem().Field(0)
	if want.Kind() == reflect.Slice && want.Len() == 0 && got.Len() == 0 {
		return // nil and [] are both "empty"
	}
	assert.Equal(t, want.Interface(), got.Interface(), "docs/configuration.md states the wrong default for %s", name)
}

// TestConfigurationDocumentsThemeColors checks the Dark default and Light
// default columns of the colors table against the default-dark and
// default-light themes; "(none)" stands for an empty color.
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
			assert.Equal(t, want, got, "docs/configuration.md states the wrong %s color for %s", theme, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(themes["default-dark"])) {
		assert.True(t, documented[name], "docs/configuration.md has no colors-table row for %s", name)
	}
}

// colorsByKey returns each color of c by its config key.
func colorsByKey(c config.ColorConfig) map[string]string {
	colors := map[string]string{}
	v := reflect.ValueOf(c)
	for field := range v.Type().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		colors[name] = v.FieldByIndex(field.Index).String()
	}
	return colors
}

// TestDocsStateCurrentCounts checks the numbers the docs quote against their
// source: the built-in theme count, the minimum Go version and the minimum
// terminal size, which docs/installation.md must state.
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
		if path == "docs/changelog.md" {
			continue // generated from CHANGELOG.md, which records history
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

// TestKeybindingsDocumentsEveryKey checks docs/keybindings.md against the
// KeyMap: every bound key has a row in some Key column, and every key in a Key
// column is bound.
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

// notInRepo are the code spans CODING_STANDARDS.md uses on purpose although
// the repo holds nothing by that name, each with the reason.
var notInRepo = map[string]string{
	"foo_test.go":         "an example file name",
	"vX.Y.Z":              "the shape of a version tag",
	"docs/changelog.md":   "generated in CI from CHANGELOG.md",
	"dist/page-media/":    "build output",
	"site/lazyenv/media/": "a path in the lazynop.dev repo",
}

// TestCodingStandardsNamesExistingCode checks that what CODING_STANDARDS.md
// names in code spans still exists: paths and file names on disk, module paths
// in go.mod, just recipes in the justfile, identifiers in the Go sources or
// build files. Spans that are none of these (snippets, flags, TOML) are not
// checked.
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

// repoIndex is what the repo holds, to check the names a doc uses against.
type repoIndex struct {
	words   map[string]bool // identifier-like words of the Go sources and build files
	files   map[string]bool // base names of the files at the root and under internal/
	recipes map[string]bool // just recipes
	goMod   string
}

func newRepoIndex(t *testing.T) repoIndex {
	t.Helper()
	repo := repoIndex{words: map[string]bool{}, files: map[string]bool{}, recipes: map[string]bool{}, goMod: readFile(t, "go.mod")}
	sources, err := filepath.Glob(".github/workflows/*.yml")
	require.NoError(t, err)
	sources = append(sources, "justfile", "go.mod", ".goreleaser.yml")

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, e := range entries {
		if !e.IsDir() {
			repo.files[e.Name()] = true
		}
		if filepath.Ext(e.Name()) == ".go" {
			sources = append(sources, e.Name())
		}
	}
	err = filepath.WalkDir("internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		repo.files[d.Name()] = true
		if filepath.Ext(path) == ".go" {
			sources = append(sources, path)
		}
		return nil
	})
	require.NoError(t, err)

	for _, path := range sources {
		for _, w := range wordRe.FindAllString(readFile(t, path), -1) {
			repo.words[w] = true
		}
	}
	for _, m := range justRecipeRe.FindAllStringSubmatch(readFile(t, "justfile"), -1) {
		repo.recipes[m[1]] = true
	}
	return repo
}

// missing returns why ref names nothing in the repo, or "" when it names
// something that exists or is not a name (a snippet, a flag, TOML).
func (r repoIndex) missing(ref string) string {
	if m := justRefRe.FindStringSubmatch(ref); m != nil {
		if !r.recipes[m[1]] {
			return "the justfile has no " + m[1] + " recipe"
		}
		return ""
	}
	if strings.Contains(ref, "/") {
		return r.missingPath(ref)
	}
	if r.files[ref] {
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

// missingPath is missing for a ref with a slash: a module path or a path in
// the repo.
func (r repoIndex) missingPath(ref string) string {
	if strings.Contains(ref, "...") || strings.ContainsAny(ref, " \"") {
		return "" // an elided path or a snippet
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

// missingIdent is missing for a dotted identifier chain; with prefix, its
// last segment only has to start a word.
func (r repoIndex) missingIdent(chain string, prefix bool) string {
	segments := strings.Split(chain, ".")
	for i, seg := range segments {
		if len(seg) == 1 {
			continue // a receiver or placeholder such as a or X
		}
		if i == len(segments)-1 && prefix {
			if !slices.ContainsFunc(slices.Collect(maps.Keys(r.words)), func(w string) bool { return strings.HasPrefix(w, seg) }) {
				return "no word in a Go source or build file starts with " + seg
			}
			continue
		}
		if !r.words[seg] {
			return seg + " appears in no Go source or build file"
		}
	}
	return ""
}

// configKeys returns the dotted path of every key the config file accepts
// under table ("" is the top level), recursing into nested tables.
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

// docTable is one Markdown table: its header cells and the cells of each body
// row, trimmed.
type docTable struct {
	header []string
	rows   [][]string
}

// markdownTables returns every table in doc, in order.
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
		case strings.Trim(line, "|-: ") == "": // the |---|---| separator
		default:
			last := &tables[len(tables)-1]
			last.rows = append(last.rows, cells)
		}
	}
	return tables
}

// markdownTable returns the body rows of the first table in doc with a column
// named header.
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

// lineStartingWith returns the first line of doc that starts with prefix.
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

// codeSpans returns the text of each `code` span in s.
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
