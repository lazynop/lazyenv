package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// The property: in read-only mode no key press writes to disk, marks a file
// modified, or enters a mode that exists only to change a file. Bindings come
// from KeyMap by reflection, so a new binding is covered without touching this
// file; a mode that a key press reaches but neither table below lists fails
// the test until someone classifies it.

// mutatingModes are the AppModes that exist only to change a file, so their
// entry key must be guarded in read-only mode.
var mutatingModes = map[AppMode]string{
	ModeEditing:             "edits a value or key, or adds a variable",
	ModeEditingCompare:      "edits a value in the diff view",
	ModeConfirmDelete:       "confirms deleting a variable",
	ModeMatrixEditing:       "adds a variable from the matrix",
	ModeConfirmMatrixDelete: "confirms deleting a variable from the matrix",
	ModeCreateFile:          "names a new file",
	ModeDuplicateFile:       "names a copy of a file",
	ModeTemplateFile:        "names a template of a file",
	ModeRenameFile:          "renames a file",
	ModeConfirmDeleteFile:   "confirms deleting a file",
	ModeReorderMenu:         "picks the order of an on-disk rewrite",
	ModeReorderConfirm:      "confirms an on-disk rewrite",
}

// readOnlyStartStates are the states every binding is pressed from: Normal
// mode in each panel, and every other mode a key press reaches in read-only
// mode. ModeConfigError is absent on purpose: a config error at startup
// enters it, never a key.
var readOnlyStartStates = []struct {
	name  string
	keys  []string       // pressed from Normal mode, files panel, first file selected
	mode  AppMode        // the mode the keys must reach
	ready func(App) bool // extra precondition, nil when the mode is enough
}{
	{name: "normal-files", mode: ModeNormal},
	{name: "normal-vars", keys: []string{"l"}, mode: ModeNormal, ready: func(a App) bool {
		return a.focus == FocusVars && a.varList.SelectedVar() != nil
	}},
	{name: "normal-vars-group-header", keys: []string{"l", "g", "k"}, mode: ModeNormal, ready: func(a App) bool {
		return a.focus == FocusVars && a.varList.IsHeaderAtCursor()
	}},
	{name: "help", keys: []string{"?"}, mode: ModeHelp},
	{name: "searching", keys: []string{"l", "/"}, mode: ModeSearching},
	{name: "compare-select", keys: []string{"c"}, mode: ModeCompareSelect},
	{name: "comparing", keys: []string{"c", "j", "enter"}, mode: ModeComparing},
	{name: "matrix-present-cell", keys: []string{"m"}, mode: ModeMatrix, ready: func(a App) bool {
		return matrixCellPresent(a.matrixView)
	}},
	{name: "matrix-missing-cell", keys: []string{"m", "l"}, mode: ModeMatrix, ready: func(a App) bool {
		return !matrixCellPresent(a.matrixView)
	}},
}

// readOnlyFollowUps are pressed after each binding to walk two-step flows
// (open an input or prompt, then submit or confirm it).
var readOnlyFollowUps = []string{"enter", "y"}

// readOnlyFixture is written to disk for every case. Its first file has a
// DB_ group, so grouping shows a header to put the cursor on.
var readOnlyFixture = map[string]string{
	".env":       "# Database\nDB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=secret\n",
	".env.local": "DB_HOST=remote\nDEBUG=true\n",
	".env.prod":  "DB_HOST=prod.example.com\nAPI_KEY=prod-secret\n",
}

func TestReadOnlyNoKeyChangesFiles(t *testing.T) {
	bindings := keyMapBindings(t)
	for _, st := range readOnlyStartStates {
		t.Run(st.name, func(t *testing.T) {
			for _, b := range bindings {
				binding := b.binding
				require.NotEmpty(t, binding.Keys(), "KeyMap.%s has no keys", b.field)

				t.Run(b.field, func(t *testing.T) {
					app, dir := newReadOnlyDiskApp(t)
					for _, k := range st.keys {
						app = sendKey(app, keyMsg(k))
					}
					require.Equal(t, st.mode, app.mode, "start state %s not reached", st.name)
					require.True(t, st.ready == nil || st.ready(app), "start state %s not reached", st.name)

					for _, k := range append([]string{binding.Keys()[0]}, readOnlyFollowUps...) {
						app = sendKey(app, keyMsg(k))
						requireReadOnlyHeld(t, app, dir, k)
					}
				})
			}
		})
	}
}

// newReadOnlyDiskApp writes readOnlyFixture to a temp dir and loads it into a
// sized read-only App the way the program does: scan, then FilesLoadedMsg.
func newReadOnlyDiskApp(t *testing.T) (App, string) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.ReadOnly = true
	app := loadFixtureApp(t, cfg, readOnlyFixture, 120, 40)
	return app, app.config.Dir
}

// requireReadOnlyHeld fails the test if the last key press (pressed) broke the
// property. Commands returned by Update are not run: every disk write in this
// package happens synchronously inside Update.
func requireReadOnlyHeld(t *testing.T, app App, dir, pressed string) {
	t.Helper()
	require.Equal(t, readOnlyFixture, readDirContents(t, dir), "key %q changed files on disk", pressed)
	for _, f := range loadedFiles(app) {
		require.False(t, f.Modified, "key %q marked %s modified", pressed, f.Name)
	}
	if why, ok := mutatingModes[app.mode]; ok {
		require.Failf(t, "entered a mutating mode", "key %q entered AppMode %d, which %s", pressed, app.mode, why)
	}
	for _, st := range readOnlyStartStates {
		if st.mode == app.mode {
			return
		}
	}
	require.Failf(t, "reached an unclassified mode",
		"key %q entered AppMode %d: add it to readOnlyStartStates, or to mutatingModes and guard its entry key",
		pressed, app.mode)
}

// readDirContents maps every entry in dir to its content ("" for a directory).
func readDirContents(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	got := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			got[e.Name()+"/"] = ""
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		got[e.Name()] = string(data)
	}
	return got
}

// matrixCellPresent reports whether the variable under the matrix cursor
// exists in the file under it: delete needs a present cell, add a missing one.
func matrixCellPresent(m MatrixModel) bool {
	return m.entries[m.cursorRow].Present[m.cursorCol]
}

// loadedFiles returns every EnvFile the app holds in memory.
func loadedFiles(app App) []*model.EnvFile {
	files := append([]*model.EnvFile{}, app.fileList.Files...)
	for _, f := range []*model.EnvFile{app.varList.File, app.diffView.FileA, app.diffView.FileB} {
		if f != nil {
			files = append(files, f)
		}
	}
	return files
}
