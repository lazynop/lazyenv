package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupAndSaveBacksUpOnceThenWrites(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "A=1\n")
	f := app.varList.File
	app.sessionStats.RecordInitialLoad(envPath, f.Vars)

	f.UpdateVar(0, "2")
	warn, err := app.backupAndSave(f)
	require.NoError(t, err)
	assert.Empty(t, warn)

	f.UpdateVar(0, "3")
	warn, err = app.backupAndSave(f)
	require.NoError(t, err)
	assert.Empty(t, warn)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "A=3\n", string(data))
	backup, err := os.ReadFile(envPath + ".bak")
	require.NoError(t, err)
	assert.Equal(t, "A=1\n", string(backup), "the second save must not back up again")
	assert.Equal(t, []string{envPath + " — 0 added, 1 changed, 0 deleted"}, app.sessionStats.Summary())
}

func TestBackupAndSaveReturnsTheBackupWarningAndStillWrites(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "A=1\n")
	f := app.varList.File
	require.NoError(t, os.MkdirAll(filepath.Join(envPath+".bak", "blocker"), 0o755))

	f.UpdateVar(0, "2")
	warn, err := app.backupAndSave(f)
	require.NoError(t, err)
	assert.Regexp(t, `^backup failed: .+ - $`, warn)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "A=2\n", string(data))

	f.UpdateVar(0, "3")
	warn, err = app.backupAndSave(f)
	require.NoError(t, err)
	assert.Empty(t, warn, "a failed backup must not be attempted again")
}

func TestBackupAndSaveReturnsTheWriteError(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "A=1\n")
	f := app.varList.File
	app.sessionStats.RecordInitialLoad(envPath, f.Vars)
	require.NoError(t, os.RemoveAll(filepath.Dir(envPath)))

	f.UpdateVar(0, "2")
	warn, err := app.backupAndSave(f)
	require.Error(t, err)
	assert.Regexp(t, `^backup failed: .+ - $`, warn)
	assert.Empty(t, app.sessionStats.Summary(), "a failed write must not be recorded")
}

func TestBackupAndSaveSkipsTheBackupWithNoBackup(t *testing.T) {
	app, envPath := newTestAppWithDiskFile(t, "A=1\n")
	app.config.NoBackup = true
	f := app.varList.File

	f.UpdateVar(0, "2")
	warn, err := app.backupAndSave(f)
	require.NoError(t, err)
	assert.Empty(t, warn)

	assert.NoFileExists(t, envPath+".bak")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "A=2\n", string(data))
}
