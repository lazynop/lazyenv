package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

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
	name                  string
	keysFromTheFilesPanel []string
	mode                  AppMode
	ready                 func(App) bool
}{
	{name: "normal-files", mode: ModeNormal},
	{name: "normal-vars", keysFromTheFilesPanel: []string{"l"}, mode: ModeNormal, ready: func(a App) bool {
		return a.focus == FocusVars && a.varList.SelectedVar() != nil
	}},
	{name: "normal-vars-group-header", keysFromTheFilesPanel: []string{"l", "g", "k"}, mode: ModeNormal, ready: func(a App) bool {
		return a.focus == FocusVars && a.varList.IsHeaderAtCursor()
	}},
	{name: "help", keysFromTheFilesPanel: []string{"?"}, mode: ModeHelp},
	{name: "searching", keysFromTheFilesPanel: []string{"l", "/"}, mode: ModeSearching},
	{name: "compare-select", keysFromTheFilesPanel: []string{"c"}, mode: ModeCompareSelect},
	{name: "comparing", keysFromTheFilesPanel: []string{"c", "j", "enter"}, mode: ModeComparing},
	{name: "matrix-present-cell-to-delete", keysFromTheFilesPanel: []string{"m"}, mode: ModeMatrix, ready: func(a App) bool {
		return matrixCellPresent(a.matrixView)
	}},
	{name: "matrix-missing-cell-to-add", keysFromTheFilesPanel: []string{"m", "l"}, mode: ModeMatrix, ready: func(a App) bool {
		return !matrixCellPresent(a.matrixView)
	}},
}

var submitThenConfirmKeys = []string{"enter", "y"}

const envWithAGroupHeader = "# Database\nDB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=secret\n"

var readOnlyFixture = map[string]string{
	".env":       envWithAGroupHeader,
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
					for _, k := range st.keysFromTheFilesPanel {
						app = press(app, k)
					}
					require.Equal(t, st.mode, app.mode, "start state %s not reached", st.name)
					require.True(t, st.ready == nil || st.ready(app), "start state %s not reached", st.name)

					for _, k := range append([]string{binding.Keys()[0]}, submitThenConfirmKeys...) {
						app = press(app, k)
						requireReadOnlyHeld(t, app, dir, k)
					}
				})
			}
		})
	}
}

func newReadOnlyDiskApp(t *testing.T) (App, string) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.ReadOnly = true
	app := loadFixtureApp(t, cfg, readOnlyFixture, 120, 40)
	return app, app.config.Dir
}

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

func matrixCellPresent(m MatrixModel) bool {
	cell, _ := m.CursorCell()
	return cell.Present
}

func loadedFiles(app App) []*model.EnvFile {
	files := append([]*model.EnvFile{}, app.fileList.Files...)
	for _, f := range []*model.EnvFile{app.varList.File, app.diffView.FileA, app.diffView.FileB} {
		if f != nil {
			files = append(files, f)
		}
	}
	return files
}
