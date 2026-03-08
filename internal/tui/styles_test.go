package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestBuildThemeDefaults(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})
	assert.NotNil(t, theme.ColorPrimary)
	assert.NotNil(t, theme.ColorFg)
	assert.NotNil(t, theme.ColorBorder)
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
	assert.NotNil(t, theme.ColorBorder)
}

func TestBuildThemeLightMode(t *testing.T) {
	theme := BuildTheme(false, config.ColorConfig{})
	assert.NotNil(t, theme.ColorPrimary)
}
