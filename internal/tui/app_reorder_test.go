package tui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReorderAlphabeticalWritesToDisk(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "C=3\nA=1\nB=2\n")

	// O → menu, a → alphabetical, y → confirm + write.
	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	require.Equal(t, ModeReorderMenu, app.mode)

	app = sendKey(app, tea.KeyPressMsg{Text: "a"})
	require.Equal(t, ModeReorderConfirm, app.mode)

	app = sendKey(app, tea.KeyPressMsg{Text: "y"})
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

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	app = sendKey(app, tea.KeyPressMsg{Text: "g"})
	require.Equal(t, ModeReorderConfirm, app.mode)
	app = sendKey(app, tea.KeyPressMsg{Text: "y"})
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "DB_HOST=h\nDB_PORT=p\n\nREDIS_HOST=r\nREDIS_PORT=1\n", string(data))
}

func TestReorderPreservesGitWarning(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n") // helper sets GitWarning=true

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	app = sendKey(app, tea.KeyPressMsg{Text: "a"})
	app = sendKey(app, tea.KeyPressMsg{Text: "y"})

	assert.True(t, app.fileList.Files[0].GitWarning, "GitWarning must survive a reorder")
}

func TestReorderCancelFromMenuLeavesDiskUntouched(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	app = sendKey(app, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data), "esc in menu must not rewrite the file")
}

func TestReorderCancelFromConfirmLeavesDiskUntouched(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	app = sendKey(app, tea.KeyPressMsg{Text: "a"})
	app = sendKey(app, tea.KeyPressMsg{Text: "n"})
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data), "n at confirm must not rewrite the file")
}

func TestReorderNeedsTwoVariables(t *testing.T) {
	app, _ := newTestAppWithDiskFile(t, "ONLY=1\n")

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	assert.Equal(t, ModeNormal, app.mode, "single-var file must not open the reorder menu")
}

func TestReorderBlockedInReadOnly(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
	app.config.ReadOnly = true

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	assert.Equal(t, ModeNormal, app.mode, "read-only must block the reorder menu")

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "B=2\nA=1\n", string(data))
}

func TestReorderCreatesBackup(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")

	app = sendKey(app, tea.KeyPressMsg{Text: "O"})
	app = sendKey(app, tea.KeyPressMsg{Text: "a"})
	app = sendKey(app, tea.KeyPressMsg{Text: "y"})
	require.Equal(t, ModeNormal, app.mode)

	data, err := os.ReadFile(envPath + ".bak")
	require.NoError(t, err, "a .bak backup must exist after the first reorder")
	assert.Equal(t, "B=2\nA=1\n", string(data), "backup must hold the pre-reorder content")
}

// sendKey feeds one key press to the app and returns the updated model.
func sendKey(app App, msg tea.KeyPressMsg) App {
	updated, _ := app.Update(msg)
	return updated.(App)
}
