package config

import (
	"fmt"
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
	assert.Equal(t, 0, l.FileListWidth)
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
	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte("recursive = true\nsort = \"alphabetical\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.True(t, cfg.Recursive)
	assert.Equal(t, "alphabetical", cfg.Sort)
	assert.Equal(t, 30, cfg.Layout.VarListMaxKeyWidth)
}

func TestLoadMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), []byte("invalid toml [[["), 0644))

	cfg, warnings := Load(dir, "")
	assert.Len(t, warnings, 1)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadInvalidValues(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[layout]\nvar-list-max-key-width = -5\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Len(t, warnings, 1)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoadPartialOverride(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[layout]\nvar-list-max-key-width = 50\n\n[colors]\nprimary = \"#FF0000\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, 50, cfg.Layout.VarListMaxKeyWidth)
	assert.Equal(t, "#FF0000", cfg.Colors.Primary)
	assert.Equal(t, 25, cfg.Layout.DiffMaxKeyWidth)
}

func TestLoadFileListWidth(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[layout]\nfile-list-width = 40\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, 40, cfg.Layout.FileListWidth)
}

func TestLoadFileListWidthTooSmallWarns(t *testing.T) {
	for _, val := range []int{-1, 5, 19} {
		dir := t.TempDir()
		content := fmt.Appendf(nil, "[layout]\nfile-list-width = %d\n", val)
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

		cfg, warnings := Load(dir, "")
		assert.Len(t, warnings, 1, "val=%d", val)
		assert.Contains(t, warnings[0], "file-list-width")
		assert.Equal(t, DefaultConfig(), cfg)
	}
}

func TestLoadUnknownKey(t *testing.T) {
	dir := t.TempDir()
	content := []byte("recusrive = true\n") // typo
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "config error")
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
	cfg, warnings := Load(projectDir, "")
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

	cfg, warnings := Load(projectDir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "position", cfg.Sort, "project config wins")
	assert.False(t, cfg.Recursive, "global config should NOT be loaded when project config exists")
}

func TestLoadColorOverrides(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[colors]\nprimary = \"#FF0000\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "#FF0000", cfg.Colors.Primary)
	assert.Empty(t, cfg.Colors.Warning, "unset colors stay empty")
}

func TestLoadFilePatterns(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[files]\ninclude = [\".env\", \".secrets\"]\nexclude = [\"*.bak\", \"*.tmp\"]\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, []string{".env", ".secrets"}, cfg.Files.Include)
	assert.Equal(t, []string{"*.bak", "*.tmp"}, cfg.Files.Exclude)
}

func TestLoadThemePreset(t *testing.T) {
	dir := t.TempDir()
	content := []byte("theme = \"dracula\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "dracula", cfg.Theme)
	assert.Equal(t, "#BD93F9", cfg.Colors.Primary)
	assert.Equal(t, "#FF5555", cfg.Colors.Error)
	assert.Equal(t, "#282A36", cfg.Colors.Bg)
}

func TestLoadThemeNoThemeBg(t *testing.T) {
	dir := t.TempDir()
	content := []byte("theme = \"dracula\"\nno-theme-bg = true\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "#BD93F9", cfg.Colors.Primary, "other colors still resolved")
	assert.Empty(t, cfg.Colors.Bg, "bg cleared by no-theme-bg")
}

func TestLoadBgColorOverride(t *testing.T) {
	dir := t.TempDir()
	content := []byte("[colors]\nbg = \"#111111\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "#111111", cfg.Colors.Bg)
}

func TestLoadThemeWithOverride(t *testing.T) {
	dir := t.TempDir()
	content := []byte("theme = \"nord\"\n\n[colors]\nprimary = \"#FF0000\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Empty(t, warnings)
	assert.Equal(t, "#FF0000", cfg.Colors.Primary, "explicit override wins")
	assert.Equal(t, "#EBCB8B", cfg.Colors.Warning, "rest comes from theme")
}

func TestLoadUnknownTheme(t *testing.T) {
	dir := t.TempDir()
	content := []byte("theme = \"nonexistent\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), content, 0644))

	cfg, warnings := Load(dir, "")
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "unknown theme")
	assert.Contains(t, warnings[0], "--list-themes")
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestThemeNames(t *testing.T) {
	names := ThemeNames()
	assert.Len(t, names, 44)
	assert.Contains(t, names, "dracula")
	assert.Contains(t, names, "catppuccin-mocha")
	assert.Contains(t, names, "nord")
	assert.Contains(t, names, "tokyo-night")
	assert.Contains(t, names, "rose-pine")
	assert.Contains(t, names, "cyberpunk")
	assert.Contains(t, names, "github-dark")
	assert.Contains(t, names, "ayu-mirage")
	assert.Contains(t, names, "synthwave-84")
	assert.Contains(t, names, "vesper")
	// verify sorted
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i-1] < names[i], "themes should be sorted")
	}
}

func TestAllThemesHaveAllColors(t *testing.T) {
	for _, name := range ThemeNames() {
		colors, ok := LookupTheme(name)
		assert.True(t, ok, "theme %s should exist", name)
		assert.NotEmpty(t, colors.Primary, "%s: primary", name)
		assert.NotEmpty(t, colors.Warning, "%s: warning", name)
		assert.NotEmpty(t, colors.Error, "%s: error", name)
		assert.NotEmpty(t, colors.Success, "%s: success", name)
		assert.NotEmpty(t, colors.Muted, "%s: muted", name)
		assert.NotEmpty(t, colors.Fg, "%s: fg", name)
		if name != "default-dark" && name != "default-light" {
			assert.NotEmpty(t, colors.Bg, "%s: bg", name)
		}
		assert.NotEmpty(t, colors.Border, "%s: border", name)
		assert.NotEmpty(t, colors.CursorBg, "%s: cursor-bg", name)
		assert.NotEmpty(t, colors.Modified, "%s: modified", name)
		assert.NotEmpty(t, colors.Added, "%s: added", name)
		assert.NotEmpty(t, colors.Deleted, "%s: deleted", name)
	}
}

func TestLookupTheme(t *testing.T) {
	colors, ok := LookupTheme("dracula")
	assert.True(t, ok)
	assert.Equal(t, "#BD93F9", colors.Primary)

	_, ok = LookupTheme("nonexistent")
	assert.False(t, ok)
}

func TestConfigSearchPathsDefault(t *testing.T) {
	paths := ConfigSearchPaths("/project", "")
	assert.Equal(t, "/project/.lazyenvrc", paths[0])
	assert.GreaterOrEqual(t, len(paths), 2, "should include at least project and one global path")
}

func TestConfigSearchPathsWithCustom(t *testing.T) {
	paths := ConfigSearchPaths("/project", "/custom/config.toml")
	assert.Equal(t, "/custom/config.toml", paths[0], "custom path should be first")
	assert.Equal(t, "/project/.lazyenvrc", paths[1], "project path should be second")
}

func TestLoadFullReturnsPath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), []byte("recursive = true\n"), 0644))

	r := LoadFull(dir, "")
	assert.Equal(t, filepath.Join(dir, ".lazyenvrc"), r.Path)
	assert.Empty(t, r.Warnings)
	assert.True(t, r.Config.Recursive)
}

func TestLoadFullNoConfig(t *testing.T) {
	dir := t.TempDir()
	r := LoadFull(dir, "")
	assert.Empty(t, r.Path)
	assert.Empty(t, r.Warnings)
	assert.Equal(t, DefaultConfig(), r.Config)
}

func TestLoadFullCustomPath(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom.toml")
	require.NoError(t, os.WriteFile(customPath, []byte("sort = \"alphabetical\"\n"), 0644))

	r := LoadFull(t.TempDir(), customPath)
	assert.Equal(t, customPath, r.Path)
	assert.Empty(t, r.Warnings)
	assert.Equal(t, "alphabetical", r.Config.Sort)
}

func TestLoadFullCustomPathOverridesProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lazyenvrc"), []byte("recursive = true\n"), 0644))

	customPath := filepath.Join(dir, "custom.toml")
	require.NoError(t, os.WriteFile(customPath, []byte("sort = \"alphabetical\"\n"), 0644))

	r := LoadFull(dir, customPath)
	assert.Equal(t, customPath, r.Path, "custom path should win over project .lazyenvrc")
	assert.Equal(t, "alphabetical", r.Config.Sort)
	assert.False(t, r.Config.Recursive, "project config should not be loaded")
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
	assert.Empty(t, c.Modified)
	assert.Empty(t, c.Added)
	assert.Empty(t, c.Deleted)
}
