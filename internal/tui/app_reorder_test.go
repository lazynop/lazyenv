package tui

import (
	"os"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReorderAlphabeticalWritesToDisk(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "C=3\nA=1\nB=2\n")

	// O → menu, a → alphabetical, y → confirm + write.
	app = press(app, "O")
	require.Equal(t, ModeReorderMenu, app.mode)

	app = press(app, "a")
	require.Equal(t, ModeReorderConfirm, app.mode)

	app = press(app, "y")
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "A=1\nB=2\nC=3\n", string(data))

	// In-memory file is refreshed from disk in the new order.
	assert.Equal(t, "A", app.fileList.Files[0].Vars[0].Key)
	assert.False(t, app.fileList.Files[0].Modified)
}

func TestReorderGroupedWritesToDisk(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "REDIS_PORT=1\nDB_HOST=h\nDB_PORT=p\nREDIS_HOST=r\n")

	app = press(app, "O", "g")
	require.Equal(t, ModeReorderConfirm, app.mode)
	app = press(app, "y")
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "DB_HOST=h\nDB_PORT=p\n\nREDIS_HOST=r\nREDIS_PORT=1\n", string(data))
}

func TestReorderPreservesGitWarning(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n") // helper sets GitWarning=true

	app = press(app, "O", "a", "y")

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning must survive a reorder")
}

func TestReorderCancelFromMenuLeavesDiskUntouched(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = press(app, "O", "esc")
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data), "esc in menu must not rewrite the file")
}

func TestReorderCancelFromConfirmLeavesDiskUntouched(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = press(app, "O", "a", "n")
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data), "n at confirm must not rewrite the file")
}

func TestReorderNeedsTwoVariables(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "ONLY=1\n")

	app = press(app, "O")
	assert.Equal(t, ModeNormal, app.mode, "single-var file must not open the reorder menu")
}

func TestReorderBlockedInReadOnly(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
	app.config.ReadOnly = true

	app = press(app, "O")
	assert.Equal(t, ModeNormal, app.mode, "read-only must block the reorder menu")

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data))
}

func TestReorderCreatesBackup(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = press(app, "O", "a", "y")
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath + ".bak")
	require.NoError(t, err, "a .bak backup must exist after the first reorder")
	assert.Equal(t, "B=2\nA=1\n", string(data), "backup must hold the pre-reorder content")
}

func TestReorderMenuSurvivesStaleClear(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	// A prior flash (e.g. pressing `o`) scheduled an auto-clear timer.
	app.statusBar.SetMessage("Sorted alphabetically")
	staleGen := app.statusBar.msgGen

	// Open the reorder menu before that timer fires.
	app = press(app, "O")
	require.Equal(t, ModeReorderMenu, app.mode)
	require.NotEmpty(t, app.statusBar.Message)

	// The earlier flash's clear timer fires while the menu is open.
	updated, _ := app.Update(ClearMessageMsg{gen: staleGen})
	app = updated.(App)

	assert.NotEmpty(t, app.statusBar.Message,
		"reorder menu prompt must persist despite a stale clear timer")
	assert.Equal(t, ModeReorderMenu, app.mode)
}

func TestReorderMenuNamesTheKeysTheKeyMapBinds(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n")
	app.keys.ReorderAlpha = key.NewBinding(key.WithKeys("x"))

	app = press(app, "O")

	assert.Equal(t, "Reorder on disk: [x]alphabetical  [g]grouped  [Esc]cancel", app.statusBar.Message)
}

func TestReorderConfirmNamesTheKeysTheKeyMapBinds(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n")
	app.keys.Confirm = key.NewBinding(key.WithKeys("j"))
	app.keys.Deny = key.NewBinding(key.WithKeys("x"))

	app = press(app, "O", "a")

	assert.True(t, strings.HasSuffix(app.statusBar.Message, "This rewrites the file. (j/x)"), app.statusBar.Message)
}
