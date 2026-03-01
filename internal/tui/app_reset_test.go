package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
)

func TestResetPreservesGitWarning(t *testing.T) {
	// Create a real .env file on disk so parser.ParseFile works during reset.
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("FOO=bar\n"), 0644))

	f, err := parser.ParseFile(envPath)
	require.NoError(t, err)

	// Simulate CheckGitIgnore having flagged this file.
	f.GitWarning = true

	// Simulate an edit so Modified=true (reset requires it).
	f.UpdateVar(0, "changed")
	require.True(t, f.Modified)

	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.fileList.Selected = 0
	app.varList.SetFile(f)
	app.focus = FocusVars

	// Press 'r' to reset.
	updated, _ := app.Update(tea.KeyPressMsg{Text: "r"})
	app = updated.(App)

	resetFile := app.fileList.Files[0]
	assert.False(t, resetFile.Modified, "Modified should be cleared after reset")
	assert.True(t, resetFile.GitWarning, "GitWarning must survive reset")
	assert.Equal(t, "bar", resetFile.Vars[0].Value, "Value should be restored from disk")
}
