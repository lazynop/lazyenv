package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestYankValueCopiesValue(t *testing.T) {
	f := makeTestFile(".env", "DB_HOST", "SECRET")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Copied DB_HOST value")
	assert.NotNil(t, cmd, "should return a command (SetClipboard + clearMessage)")
}

func TestYankLineCopiesKeyValue(t *testing.T) {
	f := makeTestFile(".env", "API_KEY")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "Y"})
	app = updated.(App)

	assert.Contains(t, app.statusBar.Message, "Copied API_KEY=")
	assert.NotNil(t, cmd)
}

func TestYankNoOpOnFilePanel(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusFiles

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Empty(t, app.statusBar.Message, "yank should not fire on file panel")
	assert.Nil(t, cmd)
}

func TestYankNoOpWithNoVar(t *testing.T) {
	f := makeTestFile(".env") // no variables
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars

	updated, cmd := app.Update(tea.KeyPressMsg{Text: "y"})
	app = updated.(App)

	assert.Empty(t, app.statusBar.Message)
	assert.Nil(t, cmd)
}
