package config

import "sort"

// themes maps theme names to their color palettes.
var themes = map[string]ColorConfig{
	"catppuccin-latte": themeCatppuccinLatte,
	"catppuccin-mocha": themeCatppuccinMocha,
	"cyberpunk":        themeCyberpunk,
	"dracula":          themeDracula,
	"everforest":       themeEverforest,
	"gruvbox-dark":     themeGruvboxDark,
	"gruvbox-light":    themeGruvboxLight,
	"kanagawa":         themeKanagawa,
	"monokai-pro":      themeMonokaiPro,
	"nord":             themeNord,
	"one-dark":         themeOneDark,
	"rose-pine":        themeRosePine,
	"solarized-dark":   themeSolarizedDark,
	"solarized-light":  themeSolarizedLight,
	"tokyo-night":      themeTokyoNight,
}

// LookupTheme returns the ColorConfig for a named theme.
func LookupTheme(name string) (ColorConfig, bool) {
	t, ok := themes[name]
	return t, ok
}

// ThemeNames returns all available theme names sorted.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveColors merges theme defaults with explicit color overrides.
// Explicit (non-empty) fields in colors take priority over the theme.
func resolveColors(theme string, colors ColorConfig) ColorConfig {
	base, ok := LookupTheme(theme)
	if !ok {
		return colors
	}
	if colors.Primary != "" {
		base.Primary = colors.Primary
	}
	if colors.Warning != "" {
		base.Warning = colors.Warning
	}
	if colors.Error != "" {
		base.Error = colors.Error
	}
	if colors.Success != "" {
		base.Success = colors.Success
	}
	if colors.Muted != "" {
		base.Muted = colors.Muted
	}
	if colors.Fg != "" {
		base.Fg = colors.Fg
	}
	if colors.Border != "" {
		base.Border = colors.Border
	}
	if colors.CursorBg != "" {
		base.CursorBg = colors.CursorBg
	}
	return base
}
