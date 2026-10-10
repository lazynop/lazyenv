package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestColorFieldsWiredEverywhere walks ColorConfig by reflection, so a new
// color field is covered without editing this test: every built-in theme must
// set it, an explicit [colors] override must win over the theme, and
// validate() must reject an invalid value for it.
func TestColorFieldsWiredEverywhere(t *testing.T) {
	// The default-* themes mirror BuildTheme's built-in defaults, which paint
	// no background: their bg stays empty so the terminal's own shows through.
	terminalNativeBg := map[string]bool{"default-dark": true, "default-light": true}
	for name := range terminalNativeBg {
		require.Contains(t, ThemeNames(), name, "stale terminal-native bg exception")
	}

	const overrideTheme, override = "dracula", "#123456"

	typ := reflect.TypeFor[ColorConfig]()
	for i := range typ.NumField() {
		field := typ.Field(i)
		key, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		require.NotEmpty(t, key, "ColorConfig.%s has no toml key", field.Name)

		t.Run(key, func(t *testing.T) {
			var bad []string
			for _, name := range ThemeNames() {
				colors, ok := LookupTheme(name)
				require.True(t, ok, "theme %s should exist", name)
				val := reflect.ValueOf(colors).Field(i).String()
				if key == "bg" && terminalNativeBg[name] {
					if val != "" {
						bad = append(bad, fmt.Sprintf("%s=%q (must stay empty: terminal native)", name, val))
					}
					continue
				}
				if val == "" || !isValidColor(val) {
					bad = append(bad, fmt.Sprintf("%s=%q", name, val))
				}
			}
			assert.Empty(t, bad, "LookupTheme() must return a valid colors.%s for every built-in theme (values in internal/config/themes/*.go)", key)

			base, ok := LookupTheme(overrideTheme)
			require.True(t, ok)
			var explicit ColorConfig
			reflect.ValueOf(&explicit).Elem().Field(i).SetString(override)
			want := base
			reflect.ValueOf(&want).Elem().Field(i).SetString(override)
			assert.Equal(t, want, resolveColors(overrideTheme, explicit),
				"resolveColors() must let an explicit colors.%s win over the theme and keep every other theme color (theme.go)", key)

			cfg := DefaultConfig()
			reflect.ValueOf(&cfg.Colors).Elem().Field(i).SetString("not-a-color")
			warns := validate(cfg)
			require.Len(t, warns, 1, "validate() must reject an invalid colors.%s, and only that key (color list in load.go)", key)
			assert.Contains(t, warns[0], "colors."+key+":", "validate() reports the invalid color under the wrong key (color list in load.go)")
		})
	}
}
