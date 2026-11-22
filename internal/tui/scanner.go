package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

var skippedDirNames = []string{"node_modules", "vendor", ".git"}

// ScanDir finds and parses all .env files in the given directory.
func ScanDir(path string, recursive bool, fileCfg config.FileConfig, secrets config.SecretsConfig) ([]*model.EnvFile, error) {
	var files []*model.EnvFile

	walkFn := func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if !recursive && p != path {
				return filepath.SkipDir
			}
			if p != path && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if slices.Contains(skippedDirNames, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if isEnvFile(d.Name(), fileCfg) {
			ef, err := parser.ParseFile(p, secrets)
			if err != nil {
				return nil
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
	for _, pat := range cfg.Exclude {
		if matched, _ := filepath.Match(pat, name); matched {
			return false
		}
	}
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

	dir := filepath.Dir(files[0].Path)
	if !isGitRepo(dir) {
		return
	}

	paths := make([]string, len(files))
	for i, f := range files {
		rel, err := filepath.Rel(dir, f.Path)
		if err != nil {
			return
		}
		paths[i] = "./" + rel
	}

	const nul = "\x00"
	cmd := exec.Command("git", "check-ignore", "-z", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(strings.Join(paths, nul))
	output, _ := cmd.Output()

	ignored := make(map[string]bool)
	for path := range strings.SplitSeq(string(output), nul) {
		if path != "" {
			ignored[path] = true
		}
	}

	for i, f := range files {
		if !ignored[paths[i]] {
			f.GitWarning = true
		}
	}
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}
