package tui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

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
	dir := t.TempDir()
	srcPath := dir + "/.env"
	assert.NoError(t, os.WriteFile(srcPath, []byte("FOO=1\nBAR=2\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	app := NewApp(cfg, nil)
	ef, _ := parser.ParseFile(srcPath, cfg.Secrets)
	m, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	app = m.(App)

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
	dir := t.TempDir()
	srcPath := dir + "/.env"
	assert.NoError(t, os.WriteFile(srcPath, []byte("FOO=1\nBAR=2\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	app := NewApp(cfg, nil)
	ef, _ := parser.ParseFile(srcPath, cfg.Secrets)
	m, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	app = m.(App)

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
	dir := t.TempDir()
	oldPath := dir + "/.env.local"
	assert.NoError(t, os.WriteFile(oldPath, []byte("FOO=1\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	app := NewApp(cfg, nil)
	ef, _ := parser.ParseFile(oldPath, cfg.Secrets)
	m, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	app = m.(App)

	app.renameSource = ef
	app.renameFileInput.SetValue(".env.dev")
	app.mode = ModeRenameFile
	out, _ := app.confirmRenameFile()
	app = out.(App)

	newPath := dir + "/.env.dev"
	// After rename without save, final for the new path is not set yet → no output.
	assert.Empty(t, app.sessionStats.Summary())

	// Force a save to populate `final` under the new path.
	app.varList.File.AddVar("BAR", "2", false)
	app, _ = app.handleSave()

	assert.Equal(t, []string{
		newPath + " (renamed from " + oldPath + ") — 1 added, 0 changed, 0 deleted",
	}, app.sessionStats.Summary())
}

func TestApp_SessionStats_Delete(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	assert.NoError(t, os.WriteFile(path, []byte("FOO=1\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	app := NewApp(cfg, nil)
	ef, _ := parser.ParseFile(path, cfg.Secrets)
	m, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	app = m.(App)

	app.mode = ModeConfirmDeleteFile
	out, _ := app.handleConfirmDeleteFileKey(tea.KeyPressMsg{Text: "y"})
	app = out.(App)

	assert.Equal(t, []string{path + " — deleted"}, app.sessionStats.Summary())
}

func TestApp_SessionStats_HandleSave(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	assert.NoError(t, os.WriteFile(path, []byte("FOO=1\n"), 0644))

	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.NoBackup = true
	app := NewApp(cfg, nil)

	ef, err := parser.ParseFile(path, cfg.Secrets)
	assert.NoError(t, err)
	out, _ := app.Update(FilesLoadedMsg{Files: []*model.EnvFile{ef}})
	app = out.(App)

	// Mutate in memory, then save.
	app.varList.File.UpdateVar(0, "99")
	app, _ = app.handleSave()

	assert.Equal(t, []string{path + " — 0 added, 1 changed, 0 deleted"}, app.sessionStats.Summary())
}
