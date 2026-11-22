package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/parser"
)

func assertFlash(t *testing.T, app App, kind messageKind, want string) {
	t.Helper()
	assert.Equal(t, want, app.statusBar.Message)
	assert.Equal(t, kind, app.statusBar.kind)
}

func assertFlashMatches(t *testing.T, app App, kind messageKind, pattern string) {
	t.Helper()
	assert.Regexp(t, "^"+pattern+"$", app.statusBar.Message)
	assert.Equal(t, kind, app.statusBar.kind)
}

func q(s string) string {
	return regexp.QuoteMeta(s)
}

func blockBackup(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(path+".bak", "blocker"), 0o755))
}

func backupBlockedPattern(path string) string {
	return `backup failed: renaming temp file: rename ` + q(filepath.Dir(path)) + `/\.lazyenv-\d+\.tmp ` + q(path+".bak") + `: [^:]+ - `
}

func writeFailedPattern(dir string) string {
	return `creating temp file: open ` + q(dir) + `/\.lazyenv-\d+\.tmp: no such file or directory`
}

func makeWriteOnly(t *testing.T, path string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a write-only file, so the re-read after the save cannot fail")
	}
	require.NoError(t, os.Chmod(path, 0o200))
}

func readDenied(path string) string {
	return "open " + path + ": permission denied"
}

func TestSaveStatusMessages(t *testing.T) {
	modified := func(t *testing.T) (App, string) {
		t.Helper()
		app, envPath := newTestAppWithDiskFile(t, "A=1\n")
		app.fileList.Files[0].UpdateVar(0, "2")
		return app, envPath
	}

	t.Run("success", func(t *testing.T) {
		app, _ := modified(t)
		app = press(app, "w")
		assertFlash(t, app, messageInfo, "Saved .env")
	})

	t.Run("failed backup then successful write", func(t *testing.T) {
		app, envPath := modified(t)
		blockBackup(t, envPath)
		app = press(app, "w")
		assertFlashMatches(t, app, messageInfo, backupBlockedPattern(envPath)+`Saved \.env`)
	})

	t.Run("failed write drops the backup warning", func(t *testing.T) {
		app, envPath := modified(t)
		require.NoError(t, os.RemoveAll(filepath.Dir(envPath)))
		app = press(app, "w")
		assertFlashMatches(t, app, messageAlert, `Error saving: `+writeFailedPattern(filepath.Dir(envPath)))
	})

	t.Run("failed refresh", func(t *testing.T) {
		app, envPath := modified(t)
		app.config.NoBackup = true
		makeWriteOnly(t, envPath)
		app = press(app, "w")
		assertFlash(t, app, messageAlert, "Saved but refresh failed: "+readDenied(envPath))
	})

	t.Run("failed refresh keeps the backup warning", func(t *testing.T) {
		app, envPath := modified(t)
		makeWriteOnly(t, envPath)
		app = press(app, "w")
		assertFlash(t, app, messageAlert,
			"backup failed: reading file for backup: "+readDenied(envPath)+" - Saved but refresh failed: "+readDenied(envPath))
	})
}

func TestReorderStatusMessages(t *testing.T) {
	for _, tt := range []struct {
		key, label string
	}{
		{"a", "alphabetical"},
		{"g", "grouped"},
	} {
		t.Run("success "+tt.label, func(t *testing.T) {
			app, _ := newTestAppWithDiskFile(t, "B=2\nA=1\n")
			app = press(app, "O", tt.key, "y")
			assertFlash(t, app, messageInfo, "Reordered .env ("+tt.label+")")
		})
	}

	t.Run("failed backup then successful write", func(t *testing.T) {
		app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
		blockBackup(t, envPath)
		app = press(app, "O", "a", "y")
		assertFlashMatches(t, app, messageInfo, backupBlockedPattern(envPath)+`Reordered \.env \(alphabetical\)`)
	})

	t.Run("failed write drops the backup warning", func(t *testing.T) {
		app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
		require.NoError(t, os.RemoveAll(filepath.Dir(envPath)))
		app = press(app, "O", "a", "y")
		assertFlashMatches(t, app, messageAlert, `Error saving: `+writeFailedPattern(filepath.Dir(envPath)))
	})

	t.Run("failed refresh", func(t *testing.T) {
		app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
		app.config.NoBackup = true
		makeWriteOnly(t, envPath)
		app = press(app, "O", "a", "y")
		assertFlash(t, app, messageAlert, "Reordered but refresh failed: "+readDenied(envPath))
	})

	t.Run("failed refresh keeps the backup warning", func(t *testing.T) {
		app, envPath := newTestAppWithDiskFile(t, "B=2\nA=1\n")
		makeWriteOnly(t, envPath)
		app = press(app, "O", "a", "y")
		assertFlash(t, app, messageAlert,
			"backup failed: reading file for backup: "+readDenied(envPath)+" - Reordered but refresh failed: "+readDenied(envPath))
	})
}

func newCompareAppOnDisk(t *testing.T) (app App, pathA, pathB string) {
	t.Helper()
	pathA = filepath.Join(t.TempDir(), ".env")
	pathB = filepath.Join(t.TempDir(), ".env.prod")
	require.NoError(t, os.WriteFile(pathA, []byte("FOO=a\n"), 0o644))
	require.NoError(t, os.WriteFile(pathB, []byte("FOO=b\n"), 0o644))

	fA, err := parser.ParseFile(pathA, config.SecretsConfig{})
	require.NoError(t, err)
	fB, err := parser.ParseFile(pathB, config.SecretsConfig{})
	require.NoError(t, err)

	app = enterCompareMode(t, fA, fB)
	require.Same(t, fA, app.diffView.FileA)
	require.Same(t, fB, app.diffView.FileB)
	return app, pathA, pathB
}

func modifyBothCompared(app App) {
	app.diffView.FileA.UpdateVar(0, "a2")
	app.diffView.FileB.UpdateVar(0, "b2")
}

func TestCompareSaveStatusMessages(t *testing.T) {
	t.Run("success on both files", func(t *testing.T) {
		app, _, _ := newCompareAppOnDisk(t)
		modifyBothCompared(app)
		app = press(app, "w")
		assertFlash(t, app, messageInfo, "Saved .env, .env.prod")
	})

	t.Run("success on the second file only", func(t *testing.T) {
		app, _, _ := newCompareAppOnDisk(t)
		app.diffView.FileB.UpdateVar(0, "b2")
		app = press(app, "w")
		assertFlash(t, app, messageInfo, "Saved .env.prod")
	})

	t.Run("failed backup of the second file then successful writes", func(t *testing.T) {
		app, _, pathB := newCompareAppOnDisk(t)
		blockBackup(t, pathB)
		modifyBothCompared(app)
		app = press(app, "w")
		assertFlashMatches(t, app, messageInfo, backupBlockedPattern(pathB)+`Saved \.env, \.env\.prod`)
	})

	t.Run("failed backup of both files then successful writes", func(t *testing.T) {
		app, pathA, pathB := newCompareAppOnDisk(t)
		blockBackup(t, pathA)
		blockBackup(t, pathB)
		modifyBothCompared(app)
		app = press(app, "w")
		assertFlashMatches(t, app, messageInfo,
			backupBlockedPattern(pathA)+backupBlockedPattern(pathB)+`Saved \.env, \.env\.prod`)
	})

	t.Run("failed write of the second file reports only its error", func(t *testing.T) {
		app, pathA, pathB := newCompareAppOnDisk(t)
		require.NoError(t, os.RemoveAll(filepath.Dir(pathB)))
		modifyBothCompared(app)
		app = press(app, "w")
		assertFlashMatches(t, app, messageAlert, `Error saving \.env\.prod: `+writeFailedPattern(filepath.Dir(pathB)))

		data, err := os.ReadFile(pathA)
		require.NoError(t, err)
		assert.Equal(t, "FOO=a2\n", string(data), "the first file is written before the second fails")
	})

	t.Run("failed refresh is not reported", func(t *testing.T) {
		app, pathA, _ := newCompareAppOnDisk(t)
		app.config.NoBackup = true
		makeWriteOnly(t, pathA)
		app.diffView.FileA.UpdateVar(0, "a2")
		app = press(app, "w")
		assertFlash(t, app, messageInfo, "Saved .env")
	})

	t.Run("failed refresh keeps the backup warning", func(t *testing.T) {
		app, pathA, _ := newCompareAppOnDisk(t)
		makeWriteOnly(t, pathA)
		app.diffView.FileA.UpdateVar(0, "a2")
		app = press(app, "w")
		assertFlash(t, app, messageInfo, "backup failed: reading file for backup: "+readDenied(pathA)+" - Saved .env")
	})
}

func TestCompareSaveBacksUpEachFile(t *testing.T) {
	app, pathA, pathB := newCompareAppOnDisk(t)
	modifyBothCompared(app)

	app = press(app, "w")
	assertFlash(t, app, messageInfo, "Saved .env, .env.prod")

	for path, original := range map[string]string{pathA: "FOO=a\n", pathB: "FOO=b\n"} {
		backup, err := os.ReadFile(path + ".bak")
		require.NoError(t, err)
		assert.Equal(t, original, string(backup), "the backup of %s", path)
	}
}
