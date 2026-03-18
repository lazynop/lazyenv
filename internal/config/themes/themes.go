package themes

import "sort"

// Colors holds a complete set of semantic colors for a theme.
type Colors struct {
	Primary  string
	Warning  string
	Error    string
	Success  string
	Muted    string
	Fg       string
	Bg       string
	Border   string
	CursorBg string
	Modified string
	Added    string
	Deleted  string
}

// registry maps theme names to their color palettes.
var registry = map[string]Colors{
	"ayu-dark":               ayuDark,
	"ayu-light":              ayuLight,
	"ayu-mirage":             ayuMirage,
	"catppuccin-frappe":      catppuccinFrappe,
	"catppuccin-latte":       catppuccinLatte,
	"catppuccin-macchiato":   catppuccinMacchiato,
	"catppuccin-mocha":       catppuccinMocha,
	"cobalt2":                cobalt2,
	"cyberpunk":              cyberpunk,
	"dawnfox":                dawnfox,
	"default-dark":           defaultDark,
	"default-light":          defaultLight,
	"dracula":                dracula,
	"everforest":             everforest,
	"github-dark":            githubDark,
	"github-light":           githubLight,
	"gruvbox-dark":           gruvboxDark,
	"gruvbox-light":          gruvboxLight,
	"horizon":                horizon,
	"iceberg":                iceberg,
	"kanagawa":               kanagawa,
	"material-darker":        materialDarker,
	"material-ocean":         materialOcean,
	"monokai":                monokai,
	"monokai-pro":            monokaiPro,
	"nightfox":               nightfox,
	"nord":                   nord,
	"one-dark":               oneDark,
	"one-light":              oneLight,
	"oxocarbon":              oxocarbon,
	"palenight":              palenight,
	"poimandres":             poimandres,
	"rose-pine":              rosePine,
	"rose-pine-dawn":         rosePineDawn,
	"rose-pine-moon":         rosePineMoon,
	"snazzy":                 snazzy,
	"solarized-dark":         solarizedDark,
	"solarized-light":        solarizedLight,
	"synthwave-84":           synthwave84,
	"tokyo-night":            tokyoNight,
	"tokyo-night-moon":       tokyoNightMoon,
	"tokyo-night-storm":      tokyoNightStorm,
	"vesper":                 vesper,
	"zenburn":                zenburn,
}

// sortedNames is computed once at init time.
var sortedNames []string

func init() {
	sortedNames = make([]string, 0, len(registry))
	for name := range registry {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
}

// Lookup returns the Colors for a named theme.
func Lookup(name string) (Colors, bool) {
	t, ok := registry[name]
	return t, ok
}

// Names returns all available theme names sorted.
func Names() []string {
	out := make([]string, len(sortedNames))
	copy(out, sortedNames)
	return out
}
