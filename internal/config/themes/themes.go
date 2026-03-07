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
	"catppuccin-latte": catppuccinLatte,
	"catppuccin-mocha": catppuccinMocha,
	"cyberpunk":        cyberpunk,
	"default-dark":     defaultDark,
	"default-light":    defaultLight,
	"dracula":          dracula,
	"everforest":       everforest,
	"gruvbox-dark":     gruvboxDark,
	"gruvbox-light":    gruvboxLight,
	"kanagawa":         kanagawa,
	"monokai-pro":      monokaiPro,
	"nord":             nord,
	"one-dark":         oneDark,
	"rose-pine":        rosePine,
	"solarized-dark":   solarizedDark,
	"solarized-light":  solarizedLight,
	"tokyo-night":      tokyoNight,
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
