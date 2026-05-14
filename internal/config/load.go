package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// hexColorRE matches a hex color literal with an explicit '#' prefix.
// Length is checked separately so we can pin valid digit counts (3/4/6/8).
var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]+$`)

// isValidColor reports whether s is a valid color string for lipgloss:
// either a hex literal (#RGB, #RGBA, #RRGGBB, #RRGGBBAA) or an ANSI 256
// numeric (0-255). Empty is valid and means "fall back to theme/default".
func isValidColor(s string) bool {
	if s == "" {
		return true
	}
	if hexColorRE.MatchString(s) {
		n := len(s) - 1
		return n == 3 || n == 4 || n == 6 || n == 8
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 255 {
		return true
	}
	return false
}

// LoadResult holds the result of loading a config file.
type LoadResult struct {
	Config   Config
	Path     string   // path of the config file used, empty if none found
	Warnings []string // validation warnings
}

// Load searches for config files and returns the merged configuration.
// If configPath is non-empty, it takes highest priority.
// A malformed or invalid config produces warnings and falls back to defaults.
func Load(projectDir, configPath string) (Config, []string) {
	r := LoadFull(projectDir, configPath)
	return r.Config, r.Warnings
}

// LoadFull searches for config files and returns detailed results.
func LoadFull(projectDir, configPath string) LoadResult {
	cfg := DefaultConfig()

	path := findConfigFile(projectDir, configPath)
	if path == "" {
		return LoadResult{Config: cfg}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{Config: cfg, Path: path, Warnings: []string{fmt.Sprintf("cannot read %s: %v", path, err)}}
	}

	merged, warns := merge(cfg, data)
	return LoadResult{Config: merged, Path: path, Warnings: warns}
}

// ConfigSearchPaths returns all paths searched for config files, in priority order.
// If configPath is non-empty, it is prepended as the highest priority.
func ConfigSearchPaths(projectDir, configPath string) []string {
	var paths []string
	if configPath != "" {
		paths = append(paths, configPath)
	}
	paths = append(paths, filepath.Join(projectDir, ".lazyenvrc"))
	if xdg, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(xdg, "lazyenv", "config.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".lazyenvrc"))
	}
	return paths
}

// findConfigFile returns the first config file found, or "".
func findConfigFile(projectDir, configPath string) string {
	for _, p := range ConfigSearchPaths(projectDir, configPath) {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func merge(defaults Config, rawData []byte) (Config, []string) {
	result := defaults
	dec := toml.NewDecoder(bytes.NewReader(rawData))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&result); err != nil {
		return defaults, []string{fmt.Sprintf("config error: %v", err)}
	}

	result.Dir = defaults.Dir

	if warns := validate(result); len(warns) > 0 {
		return defaults, warns
	}

	result.Secrets = normalizeSecrets(result.Secrets)
	result.Colors = resolveColors(result.Theme, result.Colors)
	if result.NoThemeBg {
		result.Colors.Bg = ""
	}
	return result, nil
}

// normalizeSecrets uppercases patterns so matchesPattern can skip ToUpper per call.
func normalizeSecrets(s SecretsConfig) SecretsConfig {
	s.SafePatterns = upperAll(s.SafePatterns)
	s.ExtraPatterns = upperAll(s.ExtraPatterns)
	return s
}

func upperAll(ss []string) []string {
	if ss == nil {
		return nil
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToUpper(s)
	}
	return out
}

func validate(cfg Config) []string {
	var warns []string

	if cfg.Theme != "" {
		if _, ok := LookupTheme(cfg.Theme); !ok {
			warns = append(warns, fmt.Sprintf("unknown theme %q (see --list-themes)", cfg.Theme))
		}
	}

	if cfg.Sort != "position" && cfg.Sort != "alphabetical" {
		warns = append(warns, fmt.Sprintf("invalid sort value: %q", cfg.Sort))
	}

	l := cfg.Layout
	for _, v := range []struct {
		name string
		val  int
	}{
		{"var-list-max-key-width", l.VarListMaxKeyWidth},
		{"diff-max-key-width", l.DiffMaxKeyWidth},
		{"matrix-key-width", l.MatrixKeyWidth},
		{"matrix-col-width", l.MatrixColWidth},
		{"var-list-min-value-width", l.VarListMinValueWidth},
		{"var-list-padding", l.VarListPadding},
		{"diff-min-value-width", l.DiffMinValueWidth},
		{"diff-padding", l.DiffPadding},
	} {
		if v.val <= 0 {
			warns = append(warns, fmt.Sprintf("invalid %s: %d (must be > 0)", v.name, v.val))
		}
	}

	if l.FileListWidth != 0 && l.FileListWidth < FileListMinWidth {
		warns = append(warns, fmt.Sprintf("invalid file-list-width: %d (must be 0 or >= %d)", l.FileListWidth, FileListMinWidth))
	}

	if l.MouseScrollLines < 0 {
		warns = append(warns, fmt.Sprintf("invalid mouse-scroll-lines: %d (must be >= 0)", l.MouseScrollLines))
	}

	if slices.Contains(cfg.Secrets.SafePatterns, "") {
		warns = append(warns, "empty string in secrets.safe_patterns")
	}
	if slices.Contains(cfg.Secrets.ExtraPatterns, "") {
		warns = append(warns, "empty string in secrets.extra_patterns")
	}

	c := cfg.Colors
	for _, v := range []struct {
		name, val string
	}{
		{"primary", c.Primary},
		{"warning", c.Warning},
		{"error", c.Error},
		{"success", c.Success},
		{"muted", c.Muted},
		{"fg", c.Fg},
		{"bg", c.Bg},
		{"border", c.Border},
		{"cursor-bg", c.CursorBg},
		{"modified", c.Modified},
		{"added", c.Added},
		{"deleted", c.Deleted},
	} {
		if !isValidColor(v.val) {
			warns = append(warns, fmt.Sprintf("invalid colors.%s: %q (expected hex like #RRGGBB or ANSI 0-255)", v.name, v.val))
		}
	}

	return warns
}
