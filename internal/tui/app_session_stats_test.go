package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

func TestApp_SessionStats_InitDisabledByReadOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ReadOnly = true
	app := NewApp(cfg, StartupMessages{})
	assert.Nil(t, app.sessionStats)
}

func TestApp_SessionStats_InitDisabledBySetting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SessionSummary = false
	app := NewApp(cfg, StartupMessages{})
	assert.Nil(t, app.sessionStats)
}

func TestApp_SessionStats_InitEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg, StartupMessages{})
	assert.NotNil(t, app.sessionStats)
}

func TestApp_SessionStats_InitialLoadRecorded(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg, StartupMessages{})
	files := []*model.EnvFile{
		{Path: "/p/.env", Vars: []model.EnvVar{{Key: "FOO", Value: "1"}}},
	}
	out, _ := app.Update(FilesLoadedMsg{Files: files})
	got := out.(App)
	assert.NotNil(t, got.sessionStats)
	assert.Empty(t, got.sessionStats.Summary(), "loading files should record no change")
}

func setupStatsFixture(t *testing.T, name, body string) (app App, ef *model.EnvFile, dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = dir + "/" + name
	ef = writeAndParse(t, path, body)

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	app = NewApp(cfg, StartupMessages{})

	out, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	return out.(App), ef, dir, path
}

func TestApp_SessionStats_CreateScratch(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	app := NewApp(cfg, StartupMessages{})
	out, _ := app.Update(FilesLoadedMsg{Files: nil})
	app = out.(App)

	app.fileInput.SetValue(".env.new")
	app.mode = ModeCreateFile
	m, _ := app.confirmCreateFile()
	app = m.(App)

	assert.Equal(t, []string{
		dir + "/.env.new — new file (0 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Duplicate(t *testing.T) {
	app, ef, dir, srcPath := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")

	app.fileOpSource = ef
	app.fileInput.SetValue(".env.copy")
	app.mode = ModeDuplicateFile
	out, _ := app.confirmDuplicateFile()
	app = out.(App)

	assert.Equal(t, []string{
		dir + "/.env.copy — duplicated from " + srcPath + " (2 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_DuplicateLeavesOutTheSourcesUnsavedEdits(t *testing.T) {
	app, ef, dir, srcPath := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")
	ef.UpdateVar(0, "99")
	ef.AddVar("BAZ", "3")

	app.fileOpSource = ef
	app.fileInput.SetValue(".env.copy")
	app.mode = ModeDuplicateFile
	out, _ := app.confirmDuplicateFile()
	app = out.(App)

	assert.Equal(t, []string{
		dir + "/.env.copy — duplicated from " + srcPath + " (2 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Template(t *testing.T) {
	app, ef, dir, srcPath := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")

	app.fileOpSource = ef
	app.fileInput.SetValue(".env.example")
	app.mode = ModeTemplateFile
	out, _ := app.confirmTemplateFile()
	app = out.(App)

	assert.Equal(t, []string{
		dir + "/.env.example — from template " + srcPath + " (2 variables)",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_RenameDiffsTheNewPathAgainstTheOriginBaseline(t *testing.T) {
	app, ef, dir, oldPath := setupStatsFixture(t, ".env.local", "FOO=1\n")

	app.fileOpSource = ef
	app.fileInput.SetValue(".env.dev")
	app.mode = ModeRenameFile
	out, _ := app.confirmRenameFile()
	app = out.(App)

	newPath := dir + "/.env.dev"
	assert.Equal(t, []string{
		newPath + " (renamed from " + oldPath + ") — 0 added, 0 changed, 0 deleted",
	}, app.sessionStats.Summary(), "a rename alone should emit a row, seeded from the initial snapshot")

	app.varList.File.AddVar("BAR", "2")
	app, _ = app.handleSave()

	assert.Equal(t, []string{
		newPath + " (renamed from " + oldPath + ") — 1 added, 0 changed, 0 deleted",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	localPath := dir + "/.env.local"
	stagingPath := dir + "/.env.staging"
	localEf := writeAndParse(t, localPath, "FOO=1\nBAR=2\n")
	stagingEf := writeAndParse(t, stagingPath, "X=a\nY=b\n")

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	cfg.NoGitCheck = true
	app := NewApp(cfg, StartupMessages{})

	out, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{localEf, stagingEf}})
	app = out.(App)

	app.varList.File.UpdateVar(0, "99")
	app, _ = app.handleSave()

	app.fileOpSource = stagingEf
	app.fileInput.SetValue(".env.backup")
	app.mode = ModeDuplicateFile
	m, _ := app.confirmDuplicateFile()
	app = m.(App)

	app.fileInput.SetValue(".env.new")
	app.mode = ModeCreateFile
	m, _ = app.confirmCreateFile()
	app = m.(App)
	app.varList.File.AddVar("NEW", "val")
	app, _ = app.handleSave()

	var localRef *model.EnvFile
	for _, f := range app.fileList.Files {
		if f.Path == localPath {
			localRef = f
			break
		}
	}
	require.NotNil(t, localRef)
	app.fileOpSource = localRef
	app.fileInput.SetValue(".env.dev")
	app.mode = ModeRenameFile
	m, _ = app.confirmRenameFile()
	app = m.(App)

	for i, f := range app.fileList.Files {
		if f.Path == stagingPath {
			app.fileList.MoveCursorTo(i)
			app.fileList.Selected = i
			break
		}
	}
	app.mode = ModeConfirmDeleteFile
	m, _ = app.handleConfirmDeleteFileKey(keyMsg("y"))
	app = m.(App)

	want := "Session summary:\n" +
		"  " + dir + "/.env.backup — duplicated from " + stagingPath + " (2 variables)\n" +
		"  " + dir + "/.env.dev (renamed from " + localPath + ") — 0 added, 1 changed, 0 deleted\n" +
		"  " + dir + "/.env.new — new file (1 variable)\n" +
		"  " + stagingPath + " — deleted\n"
	assert.Equal(t, want, app.SessionSummary())
}

func TestApp_SessionStats_SummaryFlattensFileNames(t *testing.T) {
	app, _, dir, _ := setupStatsFixture(t, ".env\n\x1b]0;x\x07", "FOO=1\n")

	app.varList.File.UpdateVar(0, "2")
	app, _ = app.handleSave()

	assert.Equal(t, "Session summary:\n"+
		"  "+dir+"/.env↵␛]0;x␇ — 0 added, 1 changed, 0 deleted\n", app.SessionSummary())
}

func TestApp_SessionStats_Delete(t *testing.T) {
	app, _, _, path := setupStatsFixture(t, ".env", "FOO=1\n")

	app.mode = ModeConfirmDeleteFile
	out, _ := app.handleConfirmDeleteFileKey(keyMsg("y"))
	app = out.(App)

	assert.Equal(t, []string{path + " — deleted"}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Reorder(t *testing.T) {
	app, _, _, path := setupStatsFixture(t, ".env", "FOO=1\nBAR=2\n")

	app = press(app, "O", "a", "y")

	assert.Equal(t, []string{path + " — reordered"}, app.sessionStats.Summary())
}

func TestApp_SessionStats_HandleSave(t *testing.T) {
	app, _, _, path := setupStatsFixture(t, ".env", "FOO=1\n")

	app.varList.File.UpdateVar(0, "99")
	app, _ = app.handleSave()

	assert.Equal(t, []string{path + " — 0 added, 1 changed, 0 deleted"}, app.sessionStats.Summary())
}
