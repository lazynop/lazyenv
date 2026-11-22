package config

import "github.com/lazynop/lazyenv/internal/config/themes"

// LookupTheme returns the ColorConfig for a named theme.
func LookupTheme(name string) (ColorConfig, bool) {
	t, ok := themes.Lookup(name)
	if !ok {
		return ColorConfig{}, false
	}
	return ColorConfig(t), true
}

// ThemeNames returns all available theme names sorted.
func ThemeNames() []string {
	return themes.Names()
}

// FinalizeColors resolves cfg.Colors against cfg.Theme and masks Bg if
// cfg.NoThemeBg is set. Call this AFTER any CLI overrides — calling it
// earlier is destructive because resolveColors collapses the
// theme + explicit-override + NoThemeBg precedence into a single
// ColorConfig from which the original Bg cannot be recovered.
func FinalizeColors(cfg *Config) {
	cfg.Colors = resolveColors(cfg.Theme, cfg.Colors)
	if cfg.NoThemeBg {
		cfg.Colors.Bg = ""
	}
}

func resolveColors(theme string, colors ColorConfig) ColorConfig {
	base, ok := LookupTheme(theme)
	if !ok {
		return colors
	}
	return OverlayColors(base, colors)
}

// OverlayColors returns base with every non-empty field of overrides
// copied over it.
func OverlayColors(base, overrides ColorConfig) ColorConfig {
	if overrides.Primary != "" {
		base.Primary = overrides.Primary
	}
	if overrides.Warning != "" {
		base.Warning = overrides.Warning
	}
	if overrides.Error != "" {
		base.Error = overrides.Error
	}
	if overrides.Success != "" {
		base.Success = overrides.Success
	}
	if overrides.Muted != "" {
		base.Muted = overrides.Muted
	}
	if overrides.Fg != "" {
		base.Fg = overrides.Fg
	}
	if overrides.Bg != "" {
		base.Bg = overrides.Bg
	}
	if overrides.Border != "" {
		base.Border = overrides.Border
	}
	if overrides.CursorBg != "" {
		base.CursorBg = overrides.CursorBg
	}
	if overrides.Modified != "" {
		base.Modified = overrides.Modified
	}
	if overrides.Added != "" {
		base.Added = overrides.Added
	}
	if overrides.Deleted != "" {
		base.Deleted = overrides.Deleted
	}
	return base
}
