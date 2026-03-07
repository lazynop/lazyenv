package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// Load searches for config files and returns the merged configuration.
// Search order: ./.lazyenvrc → ~/.config/lazyenv/config.toml → ~/.lazyenvrc
// Returns (config, warnings, error). Warnings are non-fatal issues.
// A malformed config file produces a warning and falls back to defaults.
func Load(projectDir string) (Config, []string, error) {
	cfg := DefaultConfig()

	path := findConfigFile(projectDir)
	if path == "" {
		return cfg, nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, []string{fmt.Sprintf("cannot read %s: %v", path, err)}, nil
	}

	merged, warns := merge(cfg, data)
	return merged, warns, nil
}

// findConfigFile returns the first config file found, or "".
func findConfigFile(projectDir string) string {
	if p := filepath.Join(projectDir, ".lazyenvrc"); fileExists(p) {
		return p
	}
	if xdg, err := os.UserConfigDir(); err == nil {
		if p := filepath.Join(xdg, "lazyenv", "config.toml"); fileExists(p) {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, ".lazyenvrc"); fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

	return result, nil
}

func validate(cfg Config) []string {
	var warns []string

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

	return warns
}
