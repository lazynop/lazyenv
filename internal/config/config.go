package config

import "time"

// Config holds all application configuration.
type Config struct {
	Dir        string `toml:"-"` // CLI-only, not in config file
	Recursive  bool   `toml:"recursive"`
	ShowAll    bool   `toml:"show-secrets"`
	NoGitCheck bool   `toml:"no-git-check"`
	NoBackup   bool   `toml:"no-backup"`
	NoThemeBg  bool   `toml:"no-theme-bg"`
	Sort       string `toml:"sort"`            // "position" | "alphabetical"
	Theme      string `toml:"theme,omitempty"` // built-in theme name (e.g. "dracula", "nord")

	Layout LayoutConfig `toml:"layout"`
	Colors ColorConfig  `toml:"colors"`
	Files  FileConfig   `toml:"files"`
}

// FileConfig holds file detection patterns.
type FileConfig struct {
	Include []string `toml:"include"` // glob patterns to include (e.g. ".env", ".env.*", "*.env")
	Exclude []string `toml:"exclude"` // glob patterns to exclude (e.g. "*.bak")
}

// LayoutConfig holds layout/sizing constants used by TUI components.
type LayoutConfig struct {
	VarListMaxKeyWidth   int `toml:"var-list-max-key-width"`
	DiffMaxKeyWidth      int `toml:"diff-max-key-width"`
	MatrixKeyWidth       int `toml:"matrix-key-width"`
	MatrixColWidth       int `toml:"matrix-col-width"`
	VarListMinValueWidth int `toml:"var-list-min-value-width"`
	VarListPadding       int `toml:"var-list-padding"`
	DiffMinValueWidth    int `toml:"diff-min-value-width"`
	DiffPadding          int `toml:"diff-padding"`

	// Internal constants, not exposed in config file.
	MessageTimeout      time.Duration `toml:"-"`
	ErrorMessageTimeout time.Duration `toml:"-"`
}

// ColorConfig holds semantic color overrides (hex strings).
// Empty string means "use auto-detected dark/light default".
type ColorConfig struct {
	Primary  string `toml:"primary,omitempty"`
	Warning  string `toml:"warning,omitempty"`
	Error    string `toml:"error,omitempty"`
	Success  string `toml:"success,omitempty"`
	Muted    string `toml:"muted,omitempty"`
	Fg       string `toml:"fg,omitempty"`
	Bg       string `toml:"bg,omitempty"`
	Border   string `toml:"border,omitempty"`
	CursorBg string `toml:"cursor-bg,omitempty"`
	Modified string `toml:"modified,omitempty"`
	Added    string `toml:"added,omitempty"`
	Deleted  string `toml:"deleted,omitempty"`
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		Dir:  ".",
		Sort: "position",
		Layout: LayoutConfig{
			VarListMaxKeyWidth:   30,
			DiffMaxKeyWidth:      25,
			MatrixKeyWidth:       20,
			MatrixColWidth:       14,
			VarListMinValueWidth: 10,
			VarListPadding:       12,
			DiffMinValueWidth:    8,
			DiffPadding:          10,
			MessageTimeout:       2 * time.Second,
			ErrorMessageTimeout:  3 * time.Second,
		},
		// Colors: all empty = use theme auto-detection
		Files: FileConfig{
			Include: []string{".env", ".env.*", "*.env"},
			Exclude: []string{"*.bak"},
		},
	}
}
