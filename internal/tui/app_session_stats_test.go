package tui

import (
	"os"
	"testing"

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
