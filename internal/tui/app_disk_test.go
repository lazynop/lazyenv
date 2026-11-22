package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

func TestDiskWritesGoThroughChokepoint(t *testing.T) {
	osWrites := []string{"CopyFS", "Create", "CreateTemp", "Link", "OpenFile", "OpenRoot",
		"Remove", "RemoveAll", "Rename", "Symlink", "Truncate", "WriteFile"}
	parserWrites := []string{"CreateBackup", "WriteFile", "WriteFileAtomic"}
	refs := append(sourceRefs(t, "os", osWrites...), sourceRefs(t, internalParserPath, parserWrites...)...)
	assertRefsMatchAllowList(t, refs, allowList{
		inFile("app_disk.go"):                      "the chokepoint itself",
		inFunc("app_actions.go", "backupIfNeeded"): "writes a .bak backup, which the session summary does not track",
	}, "change .env files on disk only through the App methods in app_disk.go, which run the "+
		"matching SessionStats hook (add one for a new kind of operation); os.OpenFile and os.OpenRoot "+
		"count even for reads, since either can write: read a file with os.Open or os.ReadFile")
}

func TestCreatedFilesGetNewFileModeNotTheSourceMode(t *testing.T) {
	for action, openKey := range map[string]string{"create": "N", "duplicate": "C", "template": "T"} {
		t.Run(action, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, ".env")
			require.NoError(t, os.WriteFile(srcPath, []byte("FOO=bar\n"), 0o640))
			require.NoError(t, os.Chmod(srcPath, 0o640))
			src, err := parser.ParseFile(srcPath, config.SecretsConfig{})
			require.NoError(t, err)

			app := newTestApp([]*model.EnvFile{src})
			app.config.Dir = dir
			app.config.NoGitCheck = true
			app.focus = FocusFiles

			submitFileName(app, openKey, ".env.new")

			info, err := os.Stat(filepath.Join(dir, ".env.new"))
			require.NoError(t, err)
			assert.Equal(t, parser.NewFileMode, info.Mode().Perm())
		})
	}
}
