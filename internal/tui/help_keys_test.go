package tui

import (
	"maps"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

type helpState struct {
	name  string
	keys  []string
	ready func(App) bool
	mode  AppMode
}

var (
	cleanFilesPanelState = &helpState{name: "files panel, file unchanged", keys: []string{"j"},
		mode: ModeNormal, ready: panelReady(FocusFiles, false)}
	filesPanelState = &helpState{name: "files panel, value edited", keys: []string{"j", "l", "e", "x", "enter", "h"},
		mode: ModeNormal, ready: panelReady(FocusFiles, true)}
	cleanVarsPanelState = &helpState{name: "variables panel, file unchanged", keys: []string{"l", "j"},
		mode: ModeNormal, ready: panelReady(FocusVars, false)}
	varsPanelState = &helpState{name: "variables panel, value edited", keys: []string{"l", "j", "e", "x", "enter"},
		mode: ModeNormal, ready: panelReady(FocusVars, true)}
	groupHeaderState = &helpState{name: "group header", keys: []string{"l", "g", "k"}, mode: ModeNormal, ready: func(a App) bool {
		return a.focus == FocusVars && a.varList.IsHeaderAtCursor()
	}}
)

var helpSectionStates = map[string]struct {
	states   []*helpState
	everyRow bool
}{
	"Files panel":     {states: []*helpState{filesPanelState, cleanFilesPanelState}},
	"Variables panel": {states: []*helpState{varsPanelState, groupHeaderState}},
	"Both panels": {states: []*helpState{
		filesPanelState, cleanFilesPanelState, varsPanelState, cleanVarsPanelState,
	}, everyRow: true},
	"Compare: pick the second file": {states: []*helpState{
		{name: "compare picker", keys: []string{"c", "j"}, mode: ModeCompareSelect, ready: func(a App) bool {
			return a.fileList.CursorFile() != a.compareFirstFile && fileCursorBetween(a)
		}},
	}},
	"Compare view": {states: []*helpState{
		{name: "diff view", keys: []string{"c", "j", "enter", "j", "e", "x", "enter"}, mode: ModeComparing, ready: func(a App) bool {
			return a.diffView.FileA != nil && a.diffView.FileA.Modified && diffCursorBetweenDifferences(a)
		}},
	}},
	"Matrix": {states: []*helpState{
		{name: "matrix present cell", keys: []string{"m", "j"}, mode: ModeMatrix, ready: func(a App) bool {
			return matrixCellPresent(a.matrixView) && a.matrixView.rows.Cursor > 0
		}},
		{name: "matrix missing cell", keys: []string{"m", "j", "j", "j", "l"}, mode: ModeMatrix, ready: func(a App) bool {
			return !matrixCellPresent(a.matrixView) && a.matrixView.cols.Cursor > 0
		}},
	}},
	"Prompts": {states: []*helpState{
		{name: "edit bar", keys: []string{"l", "e"}, mode: ModeEditing},
		{name: "search bar", keys: []string{"/"}, mode: ModeSearching},
		{name: "delete confirmation", keys: []string{"l", "d"}, mode: ModeConfirmDelete},
		{name: "reorder menu", keys: []string{"O"}, mode: ModeReorderMenu},
	}},
	"This help": {states: []*helpState{
		{name: "help scrolled a page", keys: []string{"?", "pgdown"}, mode: ModeHelp, ready: func(a App) bool {
			h := a.helpView
			return h.offset > 0 && h.offset+h.visibleLines() < len(h.wrappedLines)
		}},
	}},
}

func panelReady(focus Focus, modified bool) func(App) bool {
	return func(a App) bool {
		if a.focus != focus || a.varList.File == nil || a.varList.File.Modified != modified {
			return false
		}
		if focus == FocusFiles {
			return fileCursorBetween(a)
		}
		v := a.varList.SelectedVar()
		return v != nil && (!modified || peekShowsAnOriginal(v)) && varCursorBetween(a)
	}
}

func peekShowsAnOriginal(v *model.EnvVar) bool {
	return v.Modified && !v.IsNew
}

func fileCursorBetween(a App) bool {
	return a.fileList.Cursor > 0 && a.fileList.Cursor < len(a.fileList.Files)-1
}

func varCursorBetween(a App) bool {
	return a.varList.Cursor > 0 && a.varList.Cursor < len(a.varList.displayItems)-1
}

func diffCursorBetweenDifferences(a App) bool {
	entries, cursor := a.diffView.Entries, a.diffView.Cursor
	differs := func(e model.DiffEntry) bool { return e.Status != model.DiffEqual }
	return cursor < len(entries) && differs(entries[cursor]) &&
		slices.ContainsFunc(entries[:cursor], differs) && slices.ContainsFunc(entries[cursor+1:], differs)
}

var helpKeysFixture = map[string]string{
	".env":       "DB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=secret\nDEBUG=false\n",
	".env.local": "DB_HOST=remote\nDB_PORT=5433\nAPI_KEY=other\nLOG=debug\n",
	".env.prod":  "DB_HOST=prod.example.com\nAPI_KEY=prod-secret\n",
}

func TestHelpScreenKeysActInTheirScreen(t *testing.T) {
	requireHelpStatesReached(t)
	stateHasInputBar := map[*helpState]bool{}
	hasInputBar := func(state *helpState) bool {
		if bar, ok := stateHasInputBar[state]; ok {
			return bar
		}
		stateHasInputBar[state] = typesText(openHelpState(t, state))
		return stateHasInputBar[state]
	}

	names := helpKeyNames(t)
	shown := map[string]bool{}
	for _, section := range helpContents(DefaultKeyMap(), false) {
		listsGlyphsNotKeys := strings.Contains(section.title, "Indicators")
		if listsGlyphsNotKeys {
			continue
		}
		shown[section.title] = true
		mapping, ok := helpSectionStates[section.title]
		if !assert.True(t, ok, "help section %q maps to no states in helpSectionStates", section.title) {
			continue
		}
		for _, row := range section.rows {
			labels := helpRowLabels(row.keys)
			keyNames := make([]string, len(labels))
			msgs := make([]tea.KeyPressMsg, len(labels))
			for i, label := range labels {
				name, ok := names[label]
				require.True(t, ok, "no KeyMap binding has a key written %q (help section %s)", label, section.title)
				keyNames[i] = name
				require.NotPanics(t, func() { msgs[i] = keyMsg(name) }, "keyMsg cannot press %q (help section %s)", name, section.title)
			}
			acted := map[string]bool{}
			for _, state := range mapping.states {
				rowActed := false
				bar := hasInputBar(state)
				for i, label := range labels {
					if bar && typedKey(keyNames[i]) {
						continue
					}
					if _, _, ok := helpKeyActs(openHelpState(t, state), msgs[i]); ok {
						acted[label] = true
						rowActed = true
					}
				}
				if mapping.everyRow {
					assert.True(t, rowActed, "no key of %q (help section %s) acts in the %s", row.keys, section.title, state.name)
				}
			}
			for _, label := range labels {
				assert.True(t, acted[label], "%s (help section %s) acts in none of its states", label, section.title)
			}
		}
	}
	for _, title := range slices.Sorted(maps.Keys(helpSectionStates)) {
		assert.True(t, shown[title], "helpSectionStates maps %q, which the help screen has no key section for", title)
	}
}

func requireHelpStatesReached(t *testing.T) {
	t.Helper()
	bound := slices.Collect(maps.Values(helpKeyNames(t)))
	for _, k := range []string{"f5", "x"} {
		require.NotContains(t, bound, k, "a KeyMap binding uses %s, which this check presses as an unbound key: pick another", k)
	}
	type checkedState struct {
		section string
		state   *helpState
	}
	checked := map[string]checkedState{}
	for _, title := range slices.Sorted(maps.Keys(helpSectionStates)) {
		for _, state := range helpSectionStates[title].states {
			if first, ok := checked[state.name]; ok {
				clash := "help sections " + first.section + " and " + title + " map"
				if first.section == title {
					clash = "help section " + title + " maps"
				}
				require.Same(t, first.state, state, "%s different states named %q", clash, state.name)
				continue
			}
			checked[state.name] = checkedState{title, state}
			unbound := []string{"f5"}
			if !typesText(openHelpState(t, state)) {
				unbound = append(unbound, "x")
			}
			for _, k := range unbound {
				_, _, acted := helpKeyActs(openHelpState(t, state), keyMsg(k))
				require.False(t, acted, "the unbound key %s acts in the %s", k, state.name)
			}
		}
	}
}

var statesWithKeysChecked = map[*helpState]bool{}

func openHelpState(t *testing.T, state *helpState) App {
	t.Helper()
	app := loadFixtureApp(t, config.DefaultConfig(), helpKeysFixture, 120, 24)
	for _, k := range state.keys {
		gen := app.statusBar.msgGen
		var cmd tea.Cmd
		if statesWithKeysChecked[state] {
			var updated tea.Model
			updated, cmd = app.Update(keyMsg(k))
			app = updated.(App)
		} else {
			var acted bool
			app, cmd, acted = helpKeyActs(app, keyMsg(k))
			require.True(t, acted, "pressing %s on the way to the %s does nothing", k, state.name)
		}
		flashScheduledItsClear := cmd != nil && app.statusBar.msgGen != gen
		if flashScheduledItsClear {
			updated, _ := app.Update(ClearMessageMsg{gen: app.statusBar.msgGen})
			app = updated.(App)
		}
	}
	require.Equal(t, state.mode, app.mode, "pressing %v does not reach the %s", state.keys, state.name)
	require.True(t, state.ready == nil || state.ready(app), "pressing %v does not reach the %s", state.keys, state.name)
	statesWithKeysChecked[state] = true
	return app
}

func helpKeyActs(app App, msg tea.KeyPressMsg) (App, tea.Cmd, bool) {
	before := app.View().Content
	updated, cmd := app.Update(msg)
	next := updated.(App)
	if next.statusBar.Message != "" && next.statusBar.kind == messageWarning {
		return next, cmd, false
	}
	return next, cmd, cmd != nil || next.View().Content != before
}

func typesText(app App) bool {
	_, in := app.inputBar()
	return in != nil
}

func typedKey(name string) bool {
	return len([]rune(name)) == 1 || name == "space"
}

func helpKeyNames(t *testing.T) map[string]string {
	t.Helper()
	names := map[string]string{}
	for _, b := range keyMapBindings(t) {
		for _, k := range b.binding.Keys() {
			label := KeyLabel(k)
			require.True(t, names[label] == "" || names[label] == k, "keys %q and %q are both written %q", names[label], k, label)
			names[label] = k
		}
	}
	return names
}

func helpRowLabels(column string) []string {
	var labels []string
	for _, alternative := range strings.Split(column, " / ") {
		for _, part := range strings.Fields(alternative) {
			if part == "/" {
				labels = append(labels, part)
				continue
			}
			for label := range strings.SplitSeq(part, "/") {
				if label != "" {
					labels = append(labels, label)
				}
			}
		}
	}
	return labels
}
