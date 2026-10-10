package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinaliseNewFileCleanupCancelsCreate(t *testing.T) {
	app := newTestApp(nil)
	// A directory where the new file should be: parsing it fails, so
	// finaliseNewFile cleans it up.
	path := filepath.Join(t.TempDir(), ".env.new")
	require.NoError(t, os.Mkdir(path, 0o755))
	app.sessionStats.RecordCreateScratch(path, nil)

	_, _ = app.finaliseNewFile(path, "Created .env.new")

	assert.NoDirExists(t, path)
	assert.Empty(t, app.sessionStats.Summary(), "a cleaned-up new file must not be reported as created")
}
