package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	toml "github.com/pelletier/go-toml/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/tui"
)

var (
	version = "0.7.1"
	commit  = "none"
	date    = "unknown"
)

var cli struct {
	Path           string           `arg:"" optional:"" default:"." help:"Directory to scan." type:"existingdir"`
	Recursive      *bool            `short:"r" help:"Scan subdirectories recursively."`
	ShowAll        *bool            `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	NoGitCheck     *bool            `short:"G" name:"no-git-check" help:"Disable .gitignore check."`
	NoBackup       *bool            `short:"B" name:"no-backup" help:"Disable .bak backup before first save."`
	Theme          *string          `name:"theme" help:"Built-in theme preset (see --list-themes). Overrides theme setting in config file."`
	NoThemeBg      *bool            `name:"no-theme-bg" help:"Disable theme background color."`
	NoMouse        *bool            `name:"no-mouse" help:"Disable mouse support."`
	ReadOnly       *bool            `name:"read-only" help:"Disable all write operations."`
	SessionSummary *bool            `name:"session-summary" negatable:"" help:"Print a session summary on exit (default on). Use --no-session-summary to disable."`
	Sort           *string          `short:"s" name:"sort" help:"Sort order: position or alphabetical." enum:"position,alphabetical"`
	Group          *bool            `short:"g" name:"group" negatable:"" help:"Start with prefix grouping enabled (default off). Use --no-group to disable."`
	FileListWidth  *int             `name:"file-list-width" help:"Width of the file list panel (0=auto)."`
	Config         string           `short:"c" name:"config" help:"Path to configuration file." type:"existingfile"`
	CheckConfig    bool             `name:"check-config" help:"Validate configuration file and exit."`
	ShowConfig     bool             `name:"show-config" help:"Show effective configuration and exit."`
	ListThemes     bool             `name:"list-themes" help:"List available built-in themes and exit."`
	Themes         bool             `name:"themes" help:"Interactive theme preview."`
	Version        kong.VersionFlag `short:"v" help:"Show version."`
}

func applyCLIOverrides(cfg *config.Config) {
	if cli.Recursive != nil {
		cfg.Recursive = *cli.Recursive
	}
	if cli.ShowAll != nil {
		cfg.ShowAll = *cli.ShowAll
	}
	if cli.NoBackup != nil {
		cfg.NoBackup = *cli.NoBackup
	}
	if cli.Theme != nil {
		cfg.Theme = *cli.Theme
		if cfg.Theme != "" {
			if _, ok := config.LookupTheme(cfg.Theme); !ok {
				fmt.Fprintf(os.Stderr, "Warning: unknown theme %q (see --list-themes)\n", cfg.Theme)
			}
		}
	}
	if cli.NoThemeBg != nil {
		cfg.NoThemeBg = *cli.NoThemeBg
	}
	if cli.NoMouse != nil {
		cfg.NoMouse = *cli.NoMouse
	}
	if cli.ReadOnly != nil {
		cfg.ReadOnly = *cli.ReadOnly
	}
	applySessionSummaryOverride(cfg)
	applyFileListWidthOverride(cfg)
	if cli.Sort != nil {
		cfg.Sort = *cli.Sort
	}
	if cli.Group != nil {
		cfg.Group = *cli.Group
	}
	if cli.NoGitCheck != nil {
		cfg.NoGitCheck = *cli.NoGitCheck
	} else if !cfg.NoGitCheck {
		if _, err := exec.LookPath("git"); err != nil {
			cfg.NoGitCheck = true
		}
	}
}

func applyFileListWidthOverride(cfg *config.Config) {
	if cli.FileListWidth == nil {
		return
	}
	v := *cli.FileListWidth
	if v != 0 && v < config.FileListMinWidth {
		fmt.Fprintf(os.Stderr, "Warning: --file-list-width %d is below minimum %d, using %d\n", v, config.FileListMinWidth, config.FileListMinWidth)
		v = config.FileListMinWidth
	}
	cfg.Layout.FileListWidth = v
}

// applySessionSummaryOverride resolves the final SessionSummary value: CLI flag
// wins over config file, and --read-only forces it off.
func applySessionSummaryOverride(cfg *config.Config) {
	if cli.SessionSummary != nil {
		cfg.SessionSummary = *cli.SessionSummary
	}
	if cfg.ReadOnly {
		cfg.SessionSummary = false
	}
}

func checkConfig() {
	r := config.LoadFull(".", cli.Config)
	paths := config.ConfigSearchPaths(".", cli.Config)

	fmt.Println("Search paths (highest priority first):")
	for _, p := range paths {
		if p == r.Path {
			fmt.Printf("  ✓ %s\n", p)
		} else {
			fmt.Printf("  · %s\n", p)
		}
	}
	fmt.Println()

	if r.Path == "" {
		fmt.Println("No configuration file found (defaults will be used).")
		return
	}

	if len(r.Warnings) == 0 {
		fmt.Printf("Config OK: %s\n", r.Path)
		return
	}

	fmt.Fprintf(os.Stderr, "Config errors in %s:\n", r.Path)
	for _, w := range r.Warnings {
		fmt.Fprintf(os.Stderr, "  ✗ %s\n", w)
	}
	os.Exit(1)
}

func main() {
	kong.Parse(&cli,
		kong.Name("lazyenv"),
		kong.Description("TUI for managing .env files."),
		kong.Vars{"version": fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)},
	)

	if cli.Path == "" {
		cli.Path = "."
	}

	if cli.CheckConfig {
		checkConfig()
		return
	}

	cfg, warnings := config.Load(".", cli.Config)

	cfg.Dir = cli.Path
	applyCLIOverrides(&cfg)
	config.FinalizeColors(&cfg)

	if cli.ListThemes {
		for _, name := range config.ThemeNames() {
			fmt.Println(name)
		}
		return
	}

	if cli.Themes {
		selected, err := tui.RunThemePreview(cfg.NoMouse)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if selected != "" {
			fmt.Printf("To use this theme, add to your config file:\n\n  theme = %q\n", selected)
		}
		return
	}

	if cli.ShowConfig {
		enc := toml.NewEncoder(os.Stdout)
		enc.SetIndentTables(true)
		if err := enc.Encode(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app := tui.NewApp(cfg, warnings)

	p := tea.NewProgram(app)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if finalApp, ok := finalModel.(tui.App); ok {
		if summary := finalApp.SessionSummary(); summary != "" {
			fmt.Print(summary)
		}
	}
}
