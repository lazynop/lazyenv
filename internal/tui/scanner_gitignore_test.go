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
		scan      string // the directory scanned, relative, from the fixture's root; empty scans the root by its absolute path
		recursive bool
		notARepo  bool
		warned    map[string]bool // each .env file's GitWarning, by its path in the fixture
	}{
		{
			// Each pattern holds a slash, so it matches only the path from
			// the root: read from services, services/.env would be another.
			name:    "relative scan of a subdirectory",
			fixture: map[string]string{"services/.env": "FOO=bar\n", "services/.env.local": "LOCAL=1\n", ".gitignore": "services/.env\n"},
			scan:    "services",
			warned:  map[string]bool{"services/.env": false, "services/.env.local": true},
		},
		{
			// The first file found sits in a, and b/.env lies outside it.
			name:      "relative recursive scan from a root without .env",
			fixture:   map[string]string{"a/.env": "A=1\n", "b/.env": "B=1\n", "c/.env": "C=1\n", ".gitignore": "/a/.env\n/b/.env\n"},
			scan:      ".",
			recursive: true,
			warned:    map[string]bool{"a/.env": false, "b/.env": false, "c/.env": true},
		},
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
			// git reads a path that starts with ':' as pathspec magic: it
			// would match :x.env as x.env, and fail on :!y.env's exclude.
			name:    "names git reads as pathspec magic",
			fixture: map[string]string{":x.env": "X=1\n", ":!y.env": "Y=1\n", ".env.local": "LOCAL=1\n", ".gitignore": ":x.env\n:!y.env\n"},
			warned:  map[string]bool{":x.env": false, ":!y.env": false, ".env.local": true},
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
			scan := dir
			if tt.scan != "" {
				t.Chdir(dir)
				scan = tt.scan
			}
			files, err := ScanDir(scan, tt.recursive, config.DefaultConfig().Files, config.SecretsConfig{})
			require.NoError(t, err)

			CheckGitIgnore(files)

			warned := map[string]bool{}
			for _, f := range files {
				rel := f.Path
				if filepath.IsAbs(rel) {
					rel, err = filepath.Rel(dir, rel)
					require.NoError(t, err)
				}
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
