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
	unparsableNewFile := filepath.Join(t.TempDir(), ".env.new")
	require.NoError(t, os.Mkdir(unparsableNewFile, 0o755))
	app.sessionStats.RecordCreateScratch(unparsableNewFile, nil)

	_, _ = app.finaliseNewFile(unparsableNewFile, "Created .env.new")

	assert.NoDirExists(t, unparsableNewFile)
	assert.Empty(t, app.sessionStats.Summary(), "a cleaned-up new file must not be reported as created")
}
