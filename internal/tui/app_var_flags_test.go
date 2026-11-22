package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

var secretsOppositeToTheBuiltIns = config.SecretsConfig{
	ExtraPatterns: []string{"_HOST"},
	SafePatterns:  []string{"PUBLIC_"},
}

var customSecretsFixture = map[string]string{
	".env":      "DB_HOST=alpha\nPUBLIC_TOKEN=ghp_first1\nNOTE=hello\nMEMO=ghp_memo44\n",
	".env.prod": "DB_HOST=beta\nCACHE_HOST=delta\nGREETING=ghp_second2\n",
}

type shownValue struct {
	value  string
	masked bool
}

var customSecretsLoaded = map[string][]shownValue{
	".env":      {{"alpha", true}, {"ghp_first1", false}, {"hello", false}, {"ghp_memo44", true}},
	".env.prod": {{"beta", true}, {"delta", true}, {"ghp_second2", true}},
}

func customSecretsApp(t *testing.T) App {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Secrets = secretsOppositeToTheBuiltIns
	return loadFixtureApp(t, cfg, customSecretsFixture, 120, 40)
}

func assertShows(t *testing.T, app App, values ...shownValue) {
	t.Helper()
	view := stripAnsi(app.View().Content)
	for _, v := range values {
		assertMaskedInView(t, view, v.value, v.masked)
	}
}

func editValueKeys(row int, value string) []string {
	return slices.Concat([]string{"right"}, down(row), replaceInput("e", value))
}

func renameVarKeys(row int, name string) []string {
	return slices.Concat([]string{"right"}, down(row), renameKeys(name))
}

func addVarKeys(name, value string) []string {
	return slices.Concat([]string{"right", "a"}, typing(name), []string{"enter"}, typing(value), []string{"enter"})
}

var compareProdKeys = []string{"c", "j", "enter"}

func copyIntoEnvKeys(row int) []string {
	return slices.Concat(compareProdKeys, down(row), []string{"h", "q"})
}

func copyIntoProdKeys(row int) []string {
	return slices.Concat(compareProdKeys, down(row), []string{"l", "q", "j"})
}

func matrixAddKeys(sortedKeyRow, fileCol int, value string) []string {
	return slices.Concat([]string{"m"}, down(sortedKeyRow), slices.Repeat([]string{"l"}, fileCol),
		[]string{"a"}, typing(value), []string{"enter", "esc"}, down(fileCol))
}

func TestCustomSecretsConfigMasksChangedValues(t *testing.T) {
	const (
		compareRowDBHost      = 0
		compareRowPublicToken = 1
		compareRowCacheHost   = 4
		compareRowGreeting    = 5
		matrixRowCacheHost    = 0
		matrixRowGreeting     = 2
		matrixRowPublicToken  = 5
	)
	tests := []struct {
		name        string
		keys        []string
		changedFile string
		newValue    shownValue
	}{
		{"edit, secret key", editValueKeys(0, "gamma"), ".env", shownValue{"gamma", true}},
		{"edit, safe key", editValueKeys(1, "ghp_third3"), ".env", shownValue{"ghp_third3", false}},
		{"edit to a token", editValueKeys(2, "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
		{"rename to a secret key", renameVarKeys(2, "API_HOST"), ".env", shownValue{"hello", true}},
		{"rename to a safe key", renameVarKeys(3, "PUBLIC_MEMO"), ".env", shownValue{"ghp_memo44", false}},
		{"add, secret key", addVarKeys("CACHE_HOST", "gamma"), ".env", shownValue{"gamma", true}},
		{"add, safe key", addVarKeys("PUBLIC_KEY", "ghp_third3"), ".env", shownValue{"ghp_third3", false}},
		{"add a token", addVarKeys("GREETING", "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
		{"compare copy right, changed", copyIntoProdKeys(compareRowDBHost), ".env.prod", shownValue{"alpha", true}},
		{"compare copy right, added", copyIntoProdKeys(compareRowPublicToken), ".env.prod", shownValue{"ghp_first1", false}},
		{"compare copy left, changed", copyIntoEnvKeys(compareRowDBHost), ".env", shownValue{"beta", true}},
		{"compare copy left, removed", copyIntoEnvKeys(compareRowCacheHost), ".env", shownValue{"delta", true}},
		{"compare copy of a token", copyIntoEnvKeys(compareRowGreeting), ".env", shownValue{"ghp_second2", true}},
		{"matrix add, secret key", matrixAddKeys(matrixRowCacheHost, 0, "gamma"), ".env", shownValue{"gamma", true}},
		{"matrix add, safe key", matrixAddKeys(matrixRowPublicToken, 1, "ghp_third3"), ".env.prod", shownValue{"ghp_third3", false}},
		{"matrix add of a token", matrixAddKeys(matrixRowGreeting, 0, "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := press(customSecretsApp(t), tt.keys...)
			require.Equal(t, ModeNormal, changed.mode)
			require.Equal(t, tt.changedFile, changed.varList.File.Name)
			require.True(t, changed.varList.File.Modified, "the keys should change %s", tt.changedFile)
			assertShows(t, changed, tt.newValue)

			saved := press(changed, "w")
			require.Contains(t, saved.statusBar.Message, "Saved "+tt.changedFile)
			assertShows(t, saved, tt.newValue)

			reset := press(press(customSecretsApp(t), tt.keys...), "r")
			require.Contains(t, reset.statusBar.Message, "Reset to saved state")
			assertShows(t, reset, customSecretsLoaded[tt.changedFile]...)
		})
	}
}

func TestCustomSecretsConfigMasksAfterCompareSaveAndReset(t *testing.T) {
	tests := []struct {
		name           string
		saveOrResetKey string
		mainViewValues map[string][]shownValue
	}{
		{"save", "w", map[string][]shownValue{
			".env":      customSecretsLoaded[".env"],
			".env.prod": {{"alpha", true}, {"delta", true}, {"ghp_second2", true}},
		}},
		{"reset", "r", customSecretsLoaded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := press(customSecretsApp(t), slices.Concat(compareProdKeys, []string{"l", tt.saveOrResetKey, "q"})...)

			assertDiskFileShows(t, app, ".env", tt.mainViewValues[".env"])
			assertDiskFileShows(t, press(app, "j"), ".env.prod", tt.mainViewValues[".env.prod"])
		})
	}
}

func assertDiskFileShows(t *testing.T, app App, name string, values []shownValue) {
	t.Helper()
	require.Equal(t, name, app.varList.File.Name)
	require.False(t, app.varList.File.Modified, "%s should match the disk", name)
	assertShows(t, app, values...)
}

func TestCustomSecretsConfigMasksADuplicateLikeEveryNewFile(t *testing.T) {
	app := submitFileName(customSecretsApp(t), "C", ".env.copy")
	require.Equal(t, ".env.copy", app.varList.File.Name)
	assertShows(t, app, customSecretsLoaded[".env"]...)

	app = press(app, slices.Concat(editValueKeys(0, "gamma"), []string{"w"})...)

	require.Contains(t, app.statusBar.Message, "Saved .env.copy")
	assertShows(t, app, shownValue{"gamma", true})
}

func markersAfterSpacelessValues(t *testing.T, view, key string) []string {
	t.Helper()
	var markers []string
	for line := range strings.SplitSeq(view, "\n") {
		panels := strings.Split(line, "│")
		if len(panels) < 2 {
			continue
		}
		varListPanel := panels[len(panels)-2]
		fields := strings.Fields(varListPanel)
		if len(fields) >= 2 && fields[0] == key {
			markers = append(markers, strings.Join(fields[2:], ""))
		}
	}
	require.NotEmpty(t, markers, "no var list row for %s in:\n%s", key, view)
	return markers
}

func TestPlaceholderMarkerFollowsChanges(t *testing.T) {
	tests := []struct {
		name                               string
		keys                               []string
		key                                string
		markersAfterKeys, markersAfterSave string
	}{
		{"placeholder edited to a real value", editValueKeys(0, "https://api.example.com"), "API_URL", modifiedVarGlyph, ""},
		{"value edited to a placeholder", editValueKeys(1, "changeme"), "GREETING", modifiedVarGlyph + placeholderGlyph, placeholderGlyph},
		{"placeholder added", addVarKeys("NEW_VAR", "changeme"), "NEW_VAR", newVarGlyph + placeholderGlyph, placeholderGlyph},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := loadFixtureApp(t, config.DefaultConfig(), map[string]string{
				".env": "API_URL=changeme\nGREETING=hello\n",
			}, 120, 40)
			require.Equal(t, []string{placeholderGlyph}, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), "API_URL"))

			app = press(app, tt.keys...)
			assert.Equal(t, []string{tt.markersAfterKeys}, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), tt.key))

			app = press(app, "w")
			require.Contains(t, app.statusBar.Message, "Saved .env")
			assert.Equal(t, []string{tt.markersAfterSave}, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), tt.key))
		})
	}
}

func TestDuplicateMarkerFollowsChanges(t *testing.T) {
	tests := []struct {
		name                               string
		keys                               []string
		markersAfterKeys, markersAfterSave map[string][]string
	}{
		{
			"one renamed", renameVarKeys(0, "C"),
			map[string][]string{"A": {""}, "C": {modifiedVarGlyph}},
			map[string][]string{"A": {""}, "C": {""}},
		},
		{
			"one deleted", []string{"right", "d", "y"},
			map[string][]string{"A": {"", deletedVarGlyph}},
			map[string][]string{"A": {""}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := loadFixtureApp(t, config.DefaultConfig(), map[string]string{".env": "A=1\nA=2\nB=3\n"}, 120, 40)
			require.Equal(t, []string{duplicateKeyGlyph, duplicateKeyGlyph}, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), "A"))

			app = press(app, tt.keys...)
			for key, markers := range tt.markersAfterKeys {
				assert.Equal(t, markers, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), key), key)
			}

			app = press(app, "w")
			require.Contains(t, app.statusBar.Message, "Saved .env")
			for key, markers := range tt.markersAfterSave {
				assert.Equal(t, markers, markersAfterSpacelessValues(t, stripAnsi(app.View().Content), key), key)
			}
		})
	}
}
