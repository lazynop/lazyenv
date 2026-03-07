package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ".", cfg.Dir)
	assert.Equal(t, "position", cfg.Sort)
	assert.False(t, cfg.Recursive)
	assert.False(t, cfg.ShowAll)
	assert.False(t, cfg.NoGitCheck)
	assert.False(t, cfg.NoBackup)
}

func TestDefaultLayoutValues(t *testing.T) {
	l := DefaultConfig().Layout

	assert.Equal(t, 30, l.VarListMaxKeyWidth)
	assert.Equal(t, 25, l.DiffMaxKeyWidth)
	assert.Equal(t, 20, l.MatrixKeyWidth)
	assert.Equal(t, 14, l.MatrixColWidth)
	assert.Equal(t, 10, l.VarListMinValueWidth)
	assert.Equal(t, 12, l.VarListPadding)
	assert.Equal(t, 8, l.DiffMinValueWidth)
	assert.Equal(t, 10, l.DiffPadding)
	assert.Equal(t, 2*time.Second, l.MessageTimeout)
	assert.Equal(t, 3*time.Second, l.ErrorMessageTimeout)
}

func TestDefaultFilePatterns(t *testing.T) {
	f := DefaultConfig().Files

	assert.Equal(t, []string{".env", ".env.*", "*.env"}, f.Include)
	assert.Equal(t, []string{"*.bak"}, f.Exclude)
}

func TestTomlTagsMatchExample(t *testing.T) {
	data, err := os.ReadFile("../../examples/config/full.toml")
	require.NoError(t, err)

	// Unmarshal into defaults so unset toml:"-" fields keep their values
	cfg := DefaultConfig()
	err = toml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	assert.Equal(t, "position", cfg.Sort)
	assert.False(t, cfg.Recursive)
	assert.Equal(t, 30, cfg.Layout.VarListMaxKeyWidth)
	assert.Equal(t, 25, cfg.Layout.DiffMaxKeyWidth)
	assert.Equal(t, []string{".env", ".env.*", "*.env"}, cfg.Files.Include)
	assert.Equal(t, []string{"*.bak"}, cfg.Files.Exclude)
	// toml:"-" fields should keep defaults
	assert.Equal(t, 2*time.Second, cfg.Layout.MessageTimeout)
	assert.Equal(t, 3*time.Second, cfg.Layout.ErrorMessageTimeout)
}

func TestLoadNoConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte("recursive = true\nsort = \"alphabetical\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.True(t, cfg.Recursive)
	assert.Equal(t, "alphabetical", cfg.Sort)
	assert.Equal(t, 30, cfg.Layout.VarListMaxKeyWidth)
}

func TestLoadMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), []byte("invalid toml [[["), 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadInvalidValues(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[layout]\nvar-list-max-key-width = -5\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadPartialOverride(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[layout]\nvar-list-max-key-width = 50\n\n[colors]\nprimary = \"#FF0000\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, 50, cfg.Layout.VarListMaxKeyWidth)
	assert.Equal(t, "#FF0000", cfg.Colors.Primary)
	assert.Equal(t, 25, cfg.Layout.DiffMaxKeyWidth)
}

func TestLoadUnknownKey(t *testing.T) {
	dir := t.TempDir()
	content := []byte("recusrive = true\n") // typo
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "unknown")
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadXDGConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	configDir := filepath.Join(xdgDir, "lazyenv")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "config.toml"),
		[]byte("sort = \"alphabetical\"\n"),
		0644,
	))

	projectDir := t.TempDir()
	cfg, warnings, err := Load(projectDir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "alphabetical", cfg.Sort)
}

func TestLoadProjectOverridesGlobal(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	configDir := filepath.Join(xdgDir, "lazyenv")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "config.toml"),
		[]byte("sort = \"alphabetical\"\nrecursive = true\n"),
		0644,
	))

	projectDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, ".lazyenvrc"),
		[]byte("sort = \"position\"\n"),
		0644,
	))

	cfg, warnings, err := Load(projectDir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "position", cfg.Sort, "project config wins")
	assert.False(t, cfg.Recursive, "global config should NOT be loaded when project config exists")
}

func TestLoadColorOverrides(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[colors]\nprimary = \"#FF0000\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "#FF0000", cfg.Colors.Primary)
	assert.Empty(t, cfg.Colors.Warning, "unset colors stay empty")
}

func TestLoadFilePatterns(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[files]\ninclude = [\".env\", \".secrets\"]\nexclude = [\"*.bak\", \"*.tmp\"]\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, []string{".env", ".secrets"}, cfg.Files.Include)
	assert.Equal(t, []string{"*.bak", "*.tmp"}, cfg.Files.Exclude)
}

func TestDefaultColorsEmpty(t *testing.T) {
	c := DefaultConfig().Colors

	assert.Empty(t, c.Primary, "default colors should be empty for auto-detection")
	assert.Empty(t, c.Warning)
	assert.Empty(t, c.Error)
	assert.Empty(t, c.Success)
	assert.Empty(t, c.Muted)
	assert.Empty(t, c.Fg)
	assert.Empty(t, c.Border)
	assert.Empty(t, c.CursorBg)
}
