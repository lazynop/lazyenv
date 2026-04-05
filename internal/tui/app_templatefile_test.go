package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func TestTemplateFile_EntersModeOnT(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "T"})
	app = updated.(App)

	assert.Equal(t, ModeTemplateFile, app.mode)
	assert.Equal(t, ".env.example", app.templateFileInput.Value())
}

func TestTemplateFile_DotEnvSuffix(t *testing.T) {
	f := makeTestFile("demo.env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "T"})
	app = updated.(App)

	assert.Equal(t, "demo.example.env", app.templateFileInput.Value())
}

func TestTemplateFile_NoFileNoOp(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusFiles

	updated, _ := app.Update(tea.KeyPressMsg{Text: "T"})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
}

func TestTemplateFile_OnlyFromFocusFiles(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(tea.KeyPressMsg{Text: "T"})
	app = updated.(App)

	assert.NotEqual(t, ModeTemplateFile, app.mode)
}

func TestTemplateFile_EscapeCancels(t *testing.T) {
	app := newTestApp(nil)
	app.mode = ModeTemplateFile

	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Nil(t, app.templateSource)
}

func TestTemplateFile_Success(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("# Database config\nDB_HOST=localhost\nDB_PASS=secret123\n"), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = src
	app.templateFileInput.SetValue(".env.example")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "Template from")

	// File exists on disk with keys but no values
	data, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	require.NoError(t, err)
	assert.Equal(t, "# Database config\nDB_HOST=\nDB_PASS=\n", string(data))

	// File is in the list
	assert.Len(t, app.fileList.Files, 2)
}

func TestTemplateFile_PreservesExportPrefix(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(srcPath, []byte("export API_KEY=abc123\n"), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = src
	app.templateFileInput.SetValue(".env.example")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	data, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	require.NoError(t, err)
	assert.Equal(t, "export API_KEY=\n", string(data))
}

func TestTemplateFile_PreservesCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, ".env")
	content := "# App settings\nAPP_NAME=myapp\n\n# Database\nDB_HOST=localhost\nDB_PORT=5432\n"
	require.NoError(t, os.WriteFile(srcPath, []byte(content), 0644))

	src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = src
	app.templateFileInput.SetValue(".env.example")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	data, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	require.NoError(t, err)
	assert.Equal(t, "# App settings\nAPP_NAME=\n\n# Database\nDB_HOST=\nDB_PORT=\n", string(data))
}

func TestTemplateFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.example"), []byte(""), 0644))

	src, err := parser.ParseFile(filepath.Join(dir, ".env"), config.SecretsConfig{})
	require.NoError(t, err)

	app := newTestApp([]*model.EnvFile{src})
	app.config.Dir = dir
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = src
	app.templateFileInput.SetValue(".env.example")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "already exists")
}

func TestTemplateFile_InvalidPattern(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = f
	app.templateFileInput.SetValue("config.yaml")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	assert.Contains(t, app.statusBar.Message, "must match")
}

func TestTemplateFile_EmptyName(t *testing.T) {
	app := newTestApp(nil)
	app.config.Dir = t.TempDir()
	app.config.NoGitCheck = true
	app.mode = ModeTemplateFile
	app.templateSource = makeTestFile(".env", "FOO")
	app.templateFileInput.SetValue("")

	result, _ := app.confirmTemplateFile()
	app = result.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
}
