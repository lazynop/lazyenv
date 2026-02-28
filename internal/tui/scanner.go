package tui

import (
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanDir finds and parses all .env files in the given directory.
func ScanDir(path string, recursive bool) ([]*model.EnvFile, error) {
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

		if isEnvFile(d.Name()) {
			ef, err := parser.ParseFile(p)
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

func isEnvFile(name string) bool {
	// Exact match: .env
	if name == ".env" {
		return true
	}
	// Prefix match: .env.* (e.g. .env.local, .env.production)
	if strings.HasPrefix(name, ".env.") {
		return true
	}
	// Suffix match: *.env (e.g. app.env, prod.env)
	if strings.HasSuffix(name, ".env") {
		return true
	}
	return false
}
