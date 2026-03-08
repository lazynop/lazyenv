package config

import "github.com/lazynop/lazyenv/internal/config/themes"

// LookupTheme returns the ColorConfig for a named theme.
func LookupTheme(name string) (ColorConfig, bool) {
	t, ok := themes.Lookup(name)
	if !ok {
		return ColorConfig{}, false
	}
	return ColorConfig{
		Primary:  t.Primary,
		Warning:  t.Warning,
		Error:    t.Error,
		Success:  t.Success,
		Muted:    t.Muted,
		Fg:       t.Fg,
		Bg:       t.Bg,
		Border:   t.Border,
		CursorBg: t.CursorBg,
		Modified: t.Modified,
		Added:    t.Added,
		Deleted:  t.Deleted,
	}, true
}

// ThemeNames returns all available theme names sorted.
func ThemeNames() []string {
	return themes.Names()
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
	if colors.Bg != "" {
		base.Bg = colors.Bg
	}
	if colors.Border != "" {
		base.Border = colors.Border
	}
	if colors.CursorBg != "" {
		base.CursorBg = colors.CursorBg
	}
	if colors.Modified != "" {
		base.Modified = colors.Modified
	}
	if colors.Added != "" {
		base.Added = colors.Added
	}
	if colors.Deleted != "" {
		base.Deleted = colors.Deleted
	}
	return base
}
