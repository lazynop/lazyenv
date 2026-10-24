package tui

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-b", "main", dir)
	require.NoError(t, cmd.Run())
}

// TestCheckGitIgnore scans each fixture, in a git repo unless notARepo, and
// checks which .env files CheckGitIgnore marks as not covered by .gitignore.
func TestCheckGitIgnore(t *testing.T) {
	tests := []struct {
		name      string
		fixture   map[string]string
		recursive bool
		notARepo  bool
		warned    map[string]bool // each .env file's GitWarning, by its path in the fixture
	}{
		{
			name:    "marks uncovered files",
			fixture: map[string]string{".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".gitignore": ".env.local\n"},
			warned:  map[string]bool{".env": true, ".env.local": false},
		},
		{
			name:    "all ignored",
			fixture: map[string]string{".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".gitignore": ".env\n.env.local\n"},
			warned:  map[string]bool{".env": false, ".env.local": false},
		},
		{
			name:     "not a git repo does nothing",
			fixture:  map[string]string{".env": "FOO=bar\n"},
			notARepo: true,
			warned:   map[string]bool{".env": false},
		},
		{
			name: "wildcard",
			fixture: map[string]string{
				".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".env.production": "PROD=1\n", ".gitignore": ".env*\n",
			},
			warned: map[string]bool{".env": false, ".env.local": false, ".env.production": false},
		},
		{
			name:      "subdirectory",
			fixture:   map[string]string{".env": "FOO=bar\n", "sub/.env": "SUB=1\n", ".gitignore": ".env\n"},
			recursive: true,
			warned:    map[string]bool{".env": false, "sub/.env": false},
		},
		{
			// One path per line, git would quote a path holding a control
			// character, or a non-ASCII byte while core.quotePath is on (the
			// default).
			name:    "names git quotes",
			fixture: map[string]string{".env.café": "FOO=bar\n", ".env.\nx": "FOO=bar\n", ".gitignore": ".env*\n"},
			warned:  map[string]bool{".env.café": false, ".env.\nx": false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupScanDir(t, tt.fixture)
			if !tt.notARepo {
				initGitRepo(t, dir)
			}
			files, err := ScanDir(dir, tt.recursive, config.DefaultConfig().Files, config.SecretsConfig{})
			require.NoError(t, err)

			CheckGitIgnore(files)

			warned := map[string]bool{}
			for _, f := range files {
				rel, err := filepath.Rel(dir, f.Path)
				require.NoError(t, err)
				warned[rel] = f.GitWarning
			}
			assert.Equal(t, tt.warned, warned)
		})
	}
}

func TestCheckGitIgnoreEmptyFiles(t *testing.T) {
	CheckGitIgnore(nil)
	// Should not panic
}
