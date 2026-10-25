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

// The property: each key a help section names acts in the screen the section
// describes. Pressed there, it changes what the screen shows or returns a
// command (a quit, a clipboard write), and flashes no warning: a refusal is
// not the action the help describes. A no-op reported as plain info ("No
// changes to save") would still count, so each section has a state where its
// keys have something to act on: the edited panel states hold a change for
// save, reset and peek. A key filed under a section whose screens all refuse
// or ignore it fails; one filed under a narrower section than it works in (w
// under Files panel) still acts there, and TestHelpScreenMatchesKeybindingsDoc
// catches it instead.

// helpState is a state of the app a help section describes, reached by
// pressing keys from the main view with the files panel focused on the first
// file. In a state with an input bar a printable key or Space is typed, so
// only the other keys are tried there.
type helpState struct {
	name  string
	keys  []string
	ready func(App) bool // the state's precondition, beyond its mode
	mode  AppMode
}

// Each panel has an edited state, which changes a value first so save, reset
// and peek have a change to act on, and a clean one, with the file
// unchanged: deleting and renaming a file need it so, and Both panels checks
// each of its rows there too. A state listed under two sections is one of
// these vars, the same pointer in both.
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

// helpSectionStates maps each key section of the help screen to the states
// it describes. A key acts in at least one of its section's states; in a
// section marked everyRow, each row acts in every state.
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
			return !matrixCellPresent(a.matrixView) && a.matrixView.cursorCol > 0
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
			return h.offset > 0 && h.offset+h.visibleLines() < len(h.lines)
		}},
	}},
}

// panelReady returns the precondition of a panel state: the panel named by
// focus has the focus and its cursor sits between two items, and the active
// file is modified or not as modified says. In the variables panel the cursor
// is also on a variable; when modified is set, on an edited one rather than a
// new one, so peek shows an original value ("was: ...").
func panelReady(focus Focus, modified bool) func(App) bool {
	return func(a App) bool {
		if a.focus != focus || a.varList.File == nil || a.varList.File.Modified != modified {
			return false
		}
		if focus == FocusFiles {
			return fileCursorBetween(a)
		}
		v := a.varList.SelectedVar()
		return v != nil && (!modified || (v.Modified && !v.IsNew)) && varCursorBetween(a)
	}
}

// fileCursorBetween reports whether the file list's cursor sits between two
// files, so it can move either way.
func fileCursorBetween(a App) bool {
	return a.fileList.Cursor > 0 && a.fileList.Cursor < len(a.fileList.Files)-1
}

// varCursorBetween reports whether the var list's cursor sits between two
// items, so it can move either way.
func varCursorBetween(a App) bool {
	return a.varList.Cursor > 0 && a.varList.Cursor < len(a.varList.displayItems)-1
}

// diffCursorBetweenDifferences reports whether the diff view's cursor sits on
// a difference with another before and after it, so every compare key has
// something to act on.
func diffCursorBetweenDifferences(a App) bool {
	entries, cursor := a.diffView.Entries, a.diffView.Cursor
	differs := func(e model.DiffEntry) bool { return e.Status != model.DiffEqual }
	return cursor < len(entries) && differs(entries[cursor]) &&
		slices.ContainsFunc(entries[:cursor], differs) && slices.ContainsFunc(entries[cursor+1:], differs)
}

// helpKeysFixture is written to disk for every state. It has three files, so
// the file list's cursor can sit between two, and a DB_ group in the first.
// The first two files differ on their first two variables, so the diff view's
// cursor can sit on a difference with another before and after it, and
// .env.local lacks DEBUG, so a matrix cell can be missing.
var helpKeysFixture = map[string]string{
	".env":       "DB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=secret\nDEBUG=false\n",
	".env.local": "DB_HOST=remote\nDB_PORT=5433\nAPI_KEY=other\nLOG=debug\n",
	".env.prod":  "DB_HOST=prod.example.com\nAPI_KEY=prod-secret\n",
}

func TestHelpScreenKeysActInTheirScreen(t *testing.T) {
	requireHelpStatesReached(t)
	inputBar := map[*helpState]bool{} // whether each state has an input bar, once opened
	hasInputBar := func(state *helpState) bool {
		if bar, ok := inputBar[state]; ok {
			return bar
		}
		inputBar[state] = typesText(openHelpState(t, state))
		return inputBar[state]
	}

	names := helpKeyNames(t)
	shown := map[string]bool{}
	for _, section := range helpContents(DefaultKeyMap(), false) {
		if strings.Contains(section.title, "Indicators") {
			continue // glyphs, not keys
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

// requireHelpStatesReached checks that each state in helpSectionStates is
// reached and that pressing the unbound keys F5 and, outside an input bar, x
// there does nothing. Each state is checked once; it also checks that no two
// states share a name, so a failure names one state.
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

// reachedHelpStates holds the states openHelpState has reached once, checking
// every key on the way: later opens press the same keys unchecked, since
// checking one renders the screen twice.
var reachedHelpStates = map[*helpState]bool{}

// openHelpState loads helpKeysFixture from disk into an app the way the
// program does, then presses state's keys and checks the state is reached:
// each key acts without a warning (checked on the state's first open), and
// the app ends in state's mode and meets its precondition. A flash a key sets
// is cleared before the next key, as its timeout would: it is told from a
// persistent prompt by the command it returns, the clear's tick, where every
// persistent prompt returns none.
func openHelpState(t *testing.T, state *helpState) App {
	t.Helper()
	app := loadFixtureApp(t, config.DefaultConfig(), helpKeysFixture, 120, 24)
	for _, k := range state.keys {
		gen := app.statusBar.msgGen
		var cmd tea.Cmd
		if reachedHelpStates[state] {
			var updated tea.Model
			updated, cmd = app.Update(keyMsg(k))
			app = updated.(App)
		} else {
			var acted bool
			app, cmd, acted = helpKeyActs(app, keyMsg(k))
			require.True(t, acted, "pressing %s on the way to the %s does nothing", k, state.name)
		}
		if cmd != nil && app.statusBar.msgGen != gen {
			updated, _ := app.Update(ClearMessageMsg{gen: app.statusBar.msgGen})
			app = updated.(App)
		}
	}
	require.Equal(t, state.mode, app.mode, "pressing %v does not reach the %s", state.keys, state.name)
	require.True(t, state.ready == nil || state.ready(app), "pressing %v does not reach the %s", state.keys, state.name)
	reachedHelpStates[state] = true
	return app
}

// helpKeyActs presses msg in app and returns the updated app and command,
// and whether the key acted: it changed what the app shows or returned a
// command, without flashing a warning.
func helpKeyActs(app App, msg tea.KeyPressMsg) (App, tea.Cmd, bool) {
	before := app.View().Content
	updated, cmd := app.Update(msg)
	next := updated.(App)
	if next.statusBar.Message != "" && next.statusBar.kind == messageWarning {
		return next, cmd, false
	}
	return next, cmd, cmd != nil || next.View().Content != before
}

// typesText reports whether app's mode has an input bar, where printable keys
// are typed.
func typesText(app App) bool {
	_, in := app.inputBar()
	return in != nil
}

// typedKey reports whether an input bar types the key named name as text.
func typedKey(name string) bool {
	return len([]rune(name)) == 1 || name == "space"
}

// helpKeyNames maps each key label the help writes to the key name a KeyMap
// binding uses for it, failing if two names share a label.
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

// helpRowLabels splits a help row's key column into key labels: alternatives
// are joined by " / " ("q / Ctrl+C"), the columns of a pair by spaces and its
// halves by "/" ("↑/↓  k/j"), and "/" alone is the slash key.
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
