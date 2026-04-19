package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
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
