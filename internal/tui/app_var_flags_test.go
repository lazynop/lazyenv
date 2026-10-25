package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

// customSecrets is a [secrets] config under which a key ending in _HOST is a
// secret whatever its value (extra_patterns) and a key starting with PUBLIC_
// never is (safe_patterns): DB_HOST=alpha is masked and PUBLIC_TOKEN=ghp_first1
// shown, the other way round from the built-in rules.
var customSecrets = config.SecretsConfig{
	ExtraPatterns: []string{"_HOST"},
	SafePatterns:  []string{"PUBLIC_"},
}

// customSecretsFixture is two files to change under customSecrets. PUBLIC_TOKEN,
// NOTE, a plain var, and MEMO, a token under a plain key, are only in .env;
// CACHE_HOST and GREETING, a token under a plain key, only in .env.prod.
var customSecretsFixture = map[string]string{
	".env":      "DB_HOST=alpha\nPUBLIC_TOKEN=ghp_first1\nNOTE=hello\nMEMO=ghp_memo44\n",
	".env.prod": "DB_HOST=beta\nCACHE_HOST=delta\nGREETING=ghp_second2\n",
}

// shownValue is a value the screen shows, and whether it shows it masked.
type shownValue struct {
	value  string
	masked bool
}

// customSecretsLoaded is how the var list shows the values of each file of
// customSecretsFixture, loaded under customSecrets.
var customSecretsLoaded = map[string][]shownValue{
	".env":      {{"alpha", true}, {"ghp_first1", false}, {"hello", false}, {"ghp_memo44", true}},
	".env.prod": {{"beta", true}, {"delta", true}, {"ghp_second2", true}},
}

// customSecretsApp loads customSecretsFixture the way the program does, with
// customSecrets as the [secrets] config: the main view shows .env, with the
// file list focused.
func customSecretsApp(t *testing.T) App {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Secrets = customSecrets
	return loadFixtureApp(t, cfg, customSecretsFixture, 120, 40)
}

// assertShows checks app's screen shows each of values, masked or in clear.
func assertShows(t *testing.T, app App, values ...shownValue) {
	t.Helper()
	view := stripAnsi(app.View().Content)
	for _, v := range values {
		assertMaskedInView(t, view, v.value, v.masked)
	}
}

// The key scripts below start in the main view on .env with the file list
// focused, as loadFixtureApp leaves it, and end in the main view showing the
// file they change.

// editValueKeys edits the value of .env's var at row to value.
func editValueKeys(row int, value string) []string {
	return slices.Concat([]string{"right"}, down(row), []string{"e", "ctrl+u"}, typing(value), []string{"enter"})
}

// renameVarKeys renames .env's var at row to name.
func renameVarKeys(row int, name string) []string {
	return slices.Concat([]string{"right"}, down(row), renameKeys(name))
}

// addVarKeys adds name=value to .env.
func addVarKeys(name, value string) []string {
	return slices.Concat([]string{"right", "a"}, typing(name), []string{"enter"}, typing(value), []string{"enter"})
}

// compareProdKeys compares .env, on the left, with .env.prod.
var compareProdKeys = []string{"c", "j", "enter"}

// copyIntoEnvKeys compares .env with .env.prod and copies the entry at row
// into .env.
func copyIntoEnvKeys(row int) []string {
	return slices.Concat(compareProdKeys, down(row), []string{"h", "q"})
}

// copyIntoProdKeys compares .env with .env.prod and copies the entry at row
// into .env.prod, which it then shows.
func copyIntoProdKeys(row int) []string {
	return slices.Concat(compareProdKeys, down(row), []string{"l", "q", "j"})
}

// matrixAddKeys adds value from the matrix, under the key at row (keys sorted
// alphabetically) to the file at col (0 .env, 1 .env.prod).
func matrixAddKeys(row, col int, value string) []string {
	return slices.Concat([]string{"m"}, down(row), slices.Repeat([]string{"l"}, col),
		[]string{"a"}, typing(value), []string{"enter", "esc"}, down(col))
}

// TestCustomSecretsConfigMasksChangedValues changes a var in each way the app
// offers, under a custom [secrets] config: the var list masks the new value as
// the config judges it, before and after a save, and a reset masks the file's
// values as on load.
func TestCustomSecretsConfigMasksChangedValues(t *testing.T) {
	tests := []struct {
		name   string
		keys   []string
		target string     // the file the keys change, which the var list then shows
		shown  shownValue // the new value, as the var list shows it
	}{
		{"edit, secret key", editValueKeys(0, "gamma"), ".env", shownValue{"gamma", true}},
		{"edit, safe key", editValueKeys(1, "ghp_third3"), ".env", shownValue{"ghp_third3", false}},
		{"edit to a token", editValueKeys(2, "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
		{"rename to a secret key", renameVarKeys(2, "API_HOST"), ".env", shownValue{"hello", true}},
		{"rename to a safe key", renameVarKeys(3, "PUBLIC_MEMO"), ".env", shownValue{"ghp_memo44", false}},
		{"add, secret key", addVarKeys("CACHE_HOST", "gamma"), ".env", shownValue{"gamma", true}},
		{"add, safe key", addVarKeys("PUBLIC_KEY", "ghp_third3"), ".env", shownValue{"ghp_third3", false}},
		{"add a token", addVarKeys("GREETING", "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
		// Compare entries: DB_HOST changed, PUBLIC_TOKEN, NOTE and MEMO only
		// in .env, CACHE_HOST and GREETING only in .env.prod.
		{"compare copy right, changed", copyIntoProdKeys(0), ".env.prod", shownValue{"alpha", true}},
		{"compare copy right, added", copyIntoProdKeys(1), ".env.prod", shownValue{"ghp_first1", false}},
		{"compare copy left, changed", copyIntoEnvKeys(0), ".env", shownValue{"beta", true}},
		{"compare copy left, removed", copyIntoEnvKeys(4), ".env", shownValue{"delta", true}},
		{"compare copy of a token", copyIntoEnvKeys(5), ".env", shownValue{"ghp_second2", true}},
		// Matrix rows: CACHE_HOST, DB_HOST, GREETING, MEMO, NOTE, PUBLIC_TOKEN.
		{"matrix add, secret key", matrixAddKeys(0, 0, "gamma"), ".env", shownValue{"gamma", true}},
		{"matrix add, safe key", matrixAddKeys(5, 1, "ghp_third3"), ".env.prod", shownValue{"ghp_third3", false}},
		{"matrix add of a token", matrixAddKeys(2, 0, "ghp_third3"), ".env", shownValue{"ghp_third3", true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := press(customSecretsApp(t), tt.keys...)
			require.Equal(t, ModeNormal, changed.mode)
			require.Equal(t, tt.target, changed.varList.File.Name)
			require.True(t, changed.varList.File.Modified, "the keys should change %s", tt.target)
			assertShows(t, changed, tt.shown)

			saved := press(changed, "w")
			require.Contains(t, saved.statusBar.Message, "Saved "+tt.target)
			assertShows(t, saved, tt.shown)

			reset := press(press(customSecretsApp(t), tt.keys...), "r")
			require.Contains(t, reset.statusBar.Message, "Reset to saved state")
			assertShows(t, reset, customSecretsLoaded[tt.target]...)
		})
	}
}

// TestCustomSecretsConfigMasksAfterCompareSaveAndReset copies a value in
// compare, then saves or resets both files there, under a custom [secrets]
// config: back in the main view, each file masks its values as the config
// judges them.
func TestCustomSecretsConfigMasksAfterCompareSaveAndReset(t *testing.T) {
	t.Run("save", func(t *testing.T) {
		app := press(customSecretsApp(t), slices.Concat(compareProdKeys, []string{"l", "w", "q", "j"})...)

		require.Equal(t, ".env.prod", app.varList.File.Name)
		require.False(t, app.varList.File.Modified, "compare should have saved .env.prod")
		assertShows(t, app, shownValue{"alpha", true})
	})
	t.Run("reset", func(t *testing.T) {
		app := press(customSecretsApp(t), slices.Concat(compareProdKeys, []string{"l", "r", "q"})...)

		require.Contains(t, app.statusBar.Message, "Reset to saved state")
		assertShows(t, app, customSecretsLoaded[".env"]...)
		app = press(app, "j")
		require.Equal(t, ".env.prod", app.varList.File.Name)
		require.False(t, app.varList.File.Modified, "compare should have reset .env.prod")
		assertShows(t, app, customSecretsLoaded[".env.prod"]...)
	})
}

// varRowMarkers returns, for each var list row of key in view, the markers it
// shows after its value (new or modified, then duplicate, empty or
// placeholder), "" for none. The rows' values must hold no space.
func varRowMarkers(t *testing.T, view, key string) []string {
	t.Helper()
	var markers []string
	for line := range strings.SplitSeq(view, "\n") {
		panels := strings.Split(line, "│")
		if len(panels) < 2 {
			continue
		}
		// The var list is the last panel, drawn before the row's closing border.
		fields := strings.Fields(panels[len(panels)-2])
		if len(fields) >= 2 && fields[0] == key {
			markers = append(markers, strings.Join(fields[2:], ""))
		}
	}
	require.NotEmpty(t, markers, "no var list row for %s in:\n%s", key, view)
	return markers
}

// TestPlaceholderMarkerFollowsChanges edits and adds values in the var list:
// a var shows the placeholder marker exactly when its new value is a
// placeholder, before a save and after it.
func TestPlaceholderMarkerFollowsChanges(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		key           string
		marked, saved string // the row's markers after the keys, then after a save
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
			require.Equal(t, []string{placeholderGlyph}, varRowMarkers(t, stripAnsi(app.View().Content), "API_URL"))

			app = press(app, tt.keys...)
			assert.Equal(t, []string{tt.marked}, varRowMarkers(t, stripAnsi(app.View().Content), tt.key))

			app = press(app, "w")
			require.Contains(t, app.statusBar.Message, "Saved .env")
			assert.Equal(t, []string{tt.saved}, varRowMarkers(t, stripAnsi(app.View().Content), tt.key))
		})
	}
}

// TestDuplicateMarkerFollowsChanges renames or deletes one of two variables
// sharing a key: neither row shows the duplicate marker any more, before a
// save and after it.
func TestDuplicateMarkerFollowsChanges(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		marked, saved map[string][]string // each key's row markers after the keys, then after a save
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
			require.Equal(t, []string{duplicateKeyGlyph, duplicateKeyGlyph}, varRowMarkers(t, stripAnsi(app.View().Content), "A"))

			app = press(app, tt.keys...)
			for key, markers := range tt.marked {
				assert.Equal(t, markers, varRowMarkers(t, stripAnsi(app.View().Content), key), key)
			}

			app = press(app, "w")
			require.Contains(t, app.statusBar.Message, "Saved .env")
			for key, markers := range tt.saved {
				assert.Equal(t, markers, varRowMarkers(t, stripAnsi(app.View().Content), key), key)
			}
		})
	}
}
