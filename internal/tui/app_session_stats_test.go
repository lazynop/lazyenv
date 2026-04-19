package tui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func TestApp_SessionStats_InitDisabledByReadOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ReadOnly = true
	app := NewApp(cfg, nil)
	assert.Nil(t, app.sessionStats)
}

func TestApp_SessionStats_InitDisabledBySetting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SessionSummary = false
	app := NewApp(cfg, nil)
	assert.Nil(t, app.sessionStats)
}

func TestApp_SessionStats_InitEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg, nil)
	assert.NotNil(t, app.sessionStats)
}

func TestApp_SessionStats_InitialLoadRecorded(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg, nil)
	files := []*model.EnvFile{
		{Path: "/p/.env", Vars: []model.EnvVar{{Key: "FOO", Value: "1"}}},
	}
	out, _ := app.Update(FilesLoadedMsg{Files: files})
	got := out.(App)
	assert.NotNil(t, got.sessionStats)
	assert.Empty(t, got.sessionStats.Summary()) // no changes yet
}

// setupStatsFixture bootstraps a tempdir with a single .env file loaded into
// an App with SessionStats enabled. Returns the post-load app, the parsed
// EnvFile, the tempdir and the file path.
func setupStatsFixture(t *testing.T, name, body string) (App, *model.EnvFile, string, string) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/" + name
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	app := NewApp(cfg, nil)

	ef, err := parser.ParseFile(path, cfg.Secrets)
	require.NoError(t, err)
	out, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	return out.(App), ef, dir, path
}

func TestApp_SessionStats_CreateScratch(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	app := NewApp(cfg, nil)
	out, _ := app.Update(FilesLoadedMsg{Files: nil})
	app = out.(App)

	app.createFileInput.SetValue(".env.new")
	app.mode = ModeCreateFile
	m, _ := app.confirmCreateFile()
	app = m.(App)

	assert.Equal(t, []string{
		dir + "/.env.new — new file (0 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Duplicate(t *testing.T) {
	app, ef, dir, srcPath := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")

	app.duplicateSource = ef
	app.duplicateFileInput.SetValue(".env.copy")
	app.mode = ModeDuplicateFile
	out, _ := app.confirmDuplicateFile()
	app = out.(App)

	assert.Equal(t, []string{
		dir + "/.env.copy — duplicated from " + srcPath + " (2 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Template(t *testing.T) {
	app, ef, dir, srcPath := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")

	app.templateSource = ef
	app.templateFileInput.SetValue(".env.example")
	app.mode = ModeTemplateFile
	out, _ := app.confirmTemplateFile()
	app = out.(App)

	assert.Equal(t, []string{
		dir + "/.env.example — from template " + srcPath + " (2 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Rename(t *testing.T) {
	app, ef, dir, oldPath := setupStatsFixture(t, ".env.local", "FOO=1\n")

	app.renameSource = ef
	app.renameFileInput.SetValue(".env.dev")
	app.mode = ModeRenameFile
	out, _ := app.confirmRenameFile()
	app = out.(App)

	newPath := dir + "/.env.dev"
	// Without save, final for the new path isn't populated yet.
	assert.Empty(t, app.sessionStats.Summary())

	app.varList.File.AddVar("BAR", "2", false)
	app, _ = app.handleSave()

	assert.Equal(t, []string{
		newPath + " (renamed from " + oldPath + ") — 1 added, 0 changed, 0 deleted",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	localPath := dir + "/.env.local"
	stagingPath := dir + "/.env.staging"
	require.NoError(t, os.WriteFile(localPath, []byte("FOO=1\nBAR=2\n"), 0644))
	require.NoError(t, os.WriteFile(stagingPath, []byte("X=a\nY=b\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	cfg.NoGitCheck = true
	app := NewApp(cfg, nil)

	localEf, err := parser.ParseFile(localPath, cfg.Secrets)
	require.NoError(t, err)
	stagingEf, err := parser.ParseFile(stagingPath, cfg.Secrets)
	require.NoError(t, err)
	out, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{localEf, stagingEf}})
	app = out.(App)

	// 1. Edit .env.local (change FOO 1 → 99) and save.
	app.varList.File.UpdateVar(0, "99")
	app, _ = app.handleSave()

	// 2. Duplicate .env.staging → .env.backup.
	app.duplicateSource = stagingEf
	app.duplicateFileInput.SetValue(".env.backup")
	app.mode = ModeDuplicateFile
	m, _ := app.confirmDuplicateFile()
	app = m.(App)

	// 3. Create .env.new from scratch, add NEW=val, save.
	app.createFileInput.SetValue(".env.new")
	app.mode = ModeCreateFile
	m, _ = app.confirmCreateFile()
	app = m.(App)
	app.varList.File.AddVar("NEW", "val", false)
	app, _ = app.handleSave()

	// 4. Rename .env.local → .env.dev.
	var localRef *model.EnvFile
	for _, f := range app.fileList.Files {
		if f.Path == localPath {
			localRef = f
			break
		}
	}
	require.NotNil(t, localRef)
	app.renameSource = localRef
	app.renameFileInput.SetValue(".env.dev")
	app.mode = ModeRenameFile
	m, _ = app.confirmRenameFile()
	app = m.(App)

	// 5. Delete .env.staging.
	for i, f := range app.fileList.Files {
		if f.Path == stagingPath {
			app.fileList.SetCursor(i)
			app.fileList.Selected = i
			break
		}
	}
	app.mode = ModeConfirmDeleteFile
	m, _ = app.handleConfirmDeleteFileKey(tea.KeyPressMsg{Text: "y"})
	app = m.(App)

	want := "Session summary:\n" +
		"  " + dir + "/.env.backup — duplicated from " + stagingPath + " (2 variables)\n" +
		"  " + dir + "/.env.dev (renamed from " + localPath + ") — 0 added, 1 changed, 0 deleted\n" +
		"  " + dir + "/.env.new — new file (1 variable)\n" +
		"  " + stagingPath + " — deleted\n"
	assert.Equal(t, want, app.SessionSummary())
}

func TestApp_SessionStats_Delete(t *testing.T) {
	app, _, _, path := setupStatsFixture(t, ".env", "FOO=1\n")

	app.mode = ModeConfirmDeleteFile
	out, _ := app.handleConfirmDeleteFileKey(tea.KeyPressMsg{Text: "y"})
	app = out.(App)

	assert.Equal(t, []string{path + " — deleted"}, app.sessionStats.Summary())
}

func TestApp_SessionStats_HandleSave(t *testing.T) {
	app, _, _, path := setupStatsFixture(t, ".env", "FOO=1\n")

	app.varList.File.UpdateVar(0, "99")
	app, _ = app.handleSave()

	assert.Equal(t, []string{path + " — 0 added, 1 changed, 0 deleted"}, app.sessionStats.Summary())
}
