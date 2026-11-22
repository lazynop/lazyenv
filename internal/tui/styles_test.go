package tui

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/config/themes"
)

func TestBuildThemeDefaults(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})
	assert.NotNil(t, theme.ColorPrimary)
	assert.NotNil(t, theme.ColorFg)
}

func TestBuildThemeWithColorOverrides(t *testing.T) {
	colors := config.ColorConfig{
		Primary:  "#FF0000",
		Warning:  "#00FF00",
		Error:    "#0000FF",
		Success:  "#FFFF00",
		Muted:    "#888888",
		Fg:       "#FFFFFF",
		Border:   "#333333",
		CursorBg: "#444444",
	}
	theme := BuildTheme(true, colors)
	assert.NotNil(t, theme.ColorPrimary)
	assert.NotNil(t, theme.ColorFg)
}

func TestBuildThemeLightMode(t *testing.T) {
	theme := BuildTheme(false, config.ColorConfig{})
	assert.NotNil(t, theme.ColorPrimary)
}

var terminalPalettes = []struct {
	isDark  bool
	palette string
}{
	{isDark: true, palette: "default-dark"},
	{isDark: false, palette: "default-light"},
}

func lookupPalette(t *testing.T, name string) config.ColorConfig {
	t.Helper()
	palette, ok := themes.Lookup(name)
	require.True(t, ok, "theme %s should exist", name)
	return config.ColorConfig(palette)
}

func TestBuildThemeWithoutThemeMatchesTheDefaultPalette(t *testing.T) {
	for _, tc := range terminalPalettes {
		t.Run(tc.palette, func(t *testing.T) {
			want := BuildTheme(tc.isDark, lookupPalette(t, tc.palette))
			got := BuildTheme(tc.isDark, config.ColorConfig{})
			assert.Equal(t, want, got)
			assert.Nil(t, got.ColorBg)
		})
	}
}

func TestBuildThemeOverrideWinsOverTheDefaultPalette(t *testing.T) {
	const override = "#123456"
	typ := reflect.TypeFor[config.ColorConfig]()
	for _, tc := range terminalPalettes {
		for i := range typ.NumField() {
			key, _, _ := strings.Cut(typ.Field(i).Tag.Get("toml"), ",")
			t.Run(tc.palette+"/"+key, func(t *testing.T) {
				var colors config.ColorConfig
				reflect.ValueOf(&colors).Elem().Field(i).SetString(override)
				palette := lookupPalette(t, tc.palette)
				reflect.ValueOf(&palette).Elem().Field(i).SetString(override)

				got := BuildTheme(tc.isDark, colors)
				assert.NotEqual(t, BuildTheme(tc.isDark, config.ColorConfig{}), got)
				assert.Equal(t, BuildTheme(tc.isDark, palette), got)
			})
		}
	}
}

func TestBuildThemeUsesTheOverriddenColors(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{Primary: "#123456", Bg: "#654321"})
	assert.Equal(t, lipgloss.Color("#123456"), theme.ColorPrimary)
	assert.Equal(t, lipgloss.Color("#654321"), theme.ColorBg)
	assert.Equal(t, lipgloss.Color(lookupPalette(t, "default-dark").Fg), theme.ColorFg)
}
