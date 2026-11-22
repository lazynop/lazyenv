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

func TestCheckGitIgnore(t *testing.T) {
	tests := []struct {
		name             string
		fixture          map[string]string
		relativeScanDir  string
		recursive        bool
		notARepo         bool
		gitWarningByPath map[string]bool
	}{
		{
			name:             "relative scan of a subdirectory, against root-anchored patterns",
			fixture:          map[string]string{"services/.env": "FOO=bar\n", "services/.env.local": "LOCAL=1\n", ".gitignore": "services/.env\n"},
			relativeScanDir:  "services",
			gitWarningByPath: map[string]bool{"services/.env": false, "services/.env.local": true},
		},
		{
			name:             "relative recursive scan from a root without .env, with a match outside the first file's directory",
			fixture:          map[string]string{"a/.env": "A=1\n", "b/.env": "B=1\n", "c/.env": "C=1\n", ".gitignore": "/a/.env\n/b/.env\n"},
			relativeScanDir:  ".",
			recursive:        true,
			gitWarningByPath: map[string]bool{"a/.env": false, "b/.env": false, "c/.env": true},
		},
		{
			name:             "marks uncovered files",
			fixture:          map[string]string{".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".gitignore": ".env.local\n"},
			gitWarningByPath: map[string]bool{".env": true, ".env.local": false},
		},
		{
			name:             "all ignored",
			fixture:          map[string]string{".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".gitignore": ".env\n.env.local\n"},
			gitWarningByPath: map[string]bool{".env": false, ".env.local": false},
		},
		{
			name:             "not a git repo does nothing",
			fixture:          map[string]string{".env": "FOO=bar\n"},
			notARepo:         true,
			gitWarningByPath: map[string]bool{".env": false},
		},
		{
			name: "wildcard",
			fixture: map[string]string{
				".env": "FOO=bar\n", ".env.local": "LOCAL=1\n", ".env.production": "PROD=1\n", ".gitignore": ".env*\n",
			},
			gitWarningByPath: map[string]bool{".env": false, ".env.local": false, ".env.production": false},
		},
		{
			name:             "subdirectory",
			fixture:          map[string]string{".env": "FOO=bar\n", "sub/.env": "SUB=1\n", ".gitignore": ".env\n"},
			recursive:        true,
			gitWarningByPath: map[string]bool{".env": false, "sub/.env": false},
		},
		{
			name:             "names git reads as pathspec magic: :x.env as x.env, :!y.env as an exclude",
			fixture:          map[string]string{":x.env": "X=1\n", ":!y.env": "Y=1\n", ".env.local": "LOCAL=1\n", ".gitignore": ":x.env\n:!y.env\n"},
			gitWarningByPath: map[string]bool{":x.env": false, ":!y.env": false, ".env.local": true},
		},
		{
			name:             "names git quotes: a control character, or non-ASCII while core.quotePath is on",
			fixture:          map[string]string{".env.café": "FOO=bar\n", ".env.\nx": "FOO=bar\n", ".gitignore": ".env*\n"},
			gitWarningByPath: map[string]bool{".env.café": false, ".env.\nx": false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupScanDir(t, tt.fixture)
			if !tt.notARepo {
				initGitRepo(t, dir)
			}
			scan := dir
			if tt.relativeScanDir != "" {
				t.Chdir(dir)
				scan = tt.relativeScanDir
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
			assert.Equal(t, tt.gitWarningByPath, warned)
		})
	}
}

func TestCheckGitIgnoreOnNoFilesDoesNotPanic(t *testing.T) {
	CheckGitIgnore(nil)
}
