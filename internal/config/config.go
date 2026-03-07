package config

import "time"

// Config holds all application configuration.
type Config struct {
	Dir        string
	Recursive  bool
	ShowAll    bool
	NoGitCheck bool
	NoBackup   bool
	Sort       string // "position" | "alphabetical"

	Layout LayoutConfig
	Colors ColorConfig
	Files  FileConfig
}

// FileConfig holds file detection patterns.
type FileConfig struct {
	Include []string // glob patterns to include (e.g. ".env", ".env.*", "*.env")
	Exclude []string // glob patterns to exclude (e.g. "*.bak")
}

// LayoutConfig holds layout/sizing constants used by TUI components.
type LayoutConfig struct {
	VarListMaxKeyWidth   int
	DiffMaxKeyWidth      int
	MatrixKeyWidth       int
	MatrixColWidth       int
	VarListMinValueWidth int
	VarListPadding       int
	DiffMinValueWidth    int
	DiffPadding          int
	MessageTimeout       time.Duration
	ErrorMessageTimeout  time.Duration
}

// ColorConfig holds semantic color overrides (hex strings).
// Empty string means "use auto-detected dark/light default".
type ColorConfig struct {
	Primary  string
	Warning  string
	Error    string
	Success  string
	Muted    string
	Fg       string
	Border   string
	CursorBg string
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
