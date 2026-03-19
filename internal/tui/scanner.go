package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// ScanDir finds and parses all .env files in the given directory.
func ScanDir(path string, recursive bool, fileCfg config.FileConfig, secrets config.SecretsConfig) ([]*model.EnvFile, error) {
	var files []*model.EnvFile

	walkFn := func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}
		if d.IsDir() {
			if !recursive && p != path {
				return filepath.SkipDir
			}
			// Skip hidden directories (except root)
			if p != path && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			// Skip common non-useful directories
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if isEnvFile(d.Name(), fileCfg) {
			ef, err := parser.ParseFile(p, secrets)
			if err != nil {
				return nil // skip unparseable files
			}
			files = append(files, ef)
		}
		return nil
	}

	if err := filepath.WalkDir(path, walkFn); err != nil {
		return nil, err
	}

	// Sort: files in root first, then by name
	sort.Slice(files, func(i, j int) bool {
		depthI := strings.Count(files[i].Path, string(filepath.Separator))
		depthJ := strings.Count(files[j].Path, string(filepath.Separator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func isEnvFile(name string, cfg config.FileConfig) bool {
	// Check exclude patterns first
	for _, pat := range cfg.Exclude {
		if matched, _ := filepath.Match(pat, name); matched {
			return false
		}
	}
	// Check include patterns
	for _, pat := range cfg.Include {
		if matched, _ := filepath.Match(pat, name); matched {
			return true
		}
	}
	return false
}

// CheckGitIgnore marks files NOT covered by .gitignore with GitWarning = true.
// Silently does nothing if not in a git repo or git is unavailable.
func CheckGitIgnore(files []*model.EnvFile) {
	if len(files) == 0 {
		return
	}

	// Check if we're in a git repo first
	dir := filepath.Dir(files[0].Path)
	if !isGitRepo(dir) {
		return
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	args := append([]string{"check-ignore"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, _ := cmd.Output()

	ignored := make(map[string]bool)
	for line := range strings.SplitSeq(string(output), "\n") {
		if line := strings.TrimSpace(line); line != "" {
			ignored[line] = true
		}
	}

	for _, f := range files {
		if !ignored[f.Path] {
			f.GitWarning = true
		}
	}
}

// isGitRepo checks if the given directory is inside a git repository.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}
