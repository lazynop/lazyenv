package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	toml "github.com/pelletier/go-toml/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/tui"
)

var (
	version = "dev"
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

// applyCLIOverrides applies the CLI flags to cfg and returns notices about
// flags it ignored or adjusted; none of them stops the run.
func applyCLIOverrides(cfg *config.Config) []string {
	var notices []string
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
				notices = append(notices, fmt.Sprintf("unknown theme %q (see --list-themes)", cfg.Theme))
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
	return notices
}

// applyFileListWidthOverride sets the file list width from the flag; the TUI
// raises a value below config.FileListMinWidth to it.
func applyFileListWidthOverride(cfg *config.Config) {
	if cli.FileListWidth != nil {
		cfg.Layout.FileListWidth = *cli.FileListWidth
	}
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

// checkConfig lists the config search paths and validates the file found, and
// returns the exit status: 1 when the file was rejected.
func checkConfig(stdout, stderr io.Writer, projectDir, configPath string) int {
	r := config.Load(projectDir, configPath)
	paths := config.ConfigSearchPaths(projectDir, configPath)

	fmt.Fprintln(stdout, "Search paths (highest priority first):")
	for _, p := range paths {
		if p == r.Path {
			fmt.Fprintf(stdout, "  ✓ %s\n", p)
		} else {
			fmt.Fprintf(stdout, "  · %s\n", p)
		}
	}
	fmt.Fprintln(stdout)

	if r.Path == "" {
		fmt.Fprintln(stdout, "No configuration file found (defaults will be used).")
		return 0
	}

	if len(r.Errors) == 0 {
		fmt.Fprintf(stdout, "Config OK: %s\n", r.Path)
		return 0
	}

	writeConfigErrors(stderr, r.Path, r.Errors)
	return 1
}

// writeConfigErrors lists the reasons the config file at path was rejected.
func writeConfigErrors(w io.Writer, path string, errs []string) {
	fmt.Fprintf(w, "Config errors in %s:\n", path)
	for _, e := range errs {
		fmt.Fprintf(w, "  ✗ %s\n", e)
	}
}

// showConfig writes cfg to stdout as TOML, after reporting on stderr why the
// config file at configPath was rejected and the notices from the CLI flags,
// and returns the exit status: 1 when the file was rejected.
func showConfig(stdout, stderr io.Writer, cfg config.Config, configPath string, startup tui.StartupMessages) int {
	rejected := len(startup.ConfigErrors) > 0
	if rejected {
		writeConfigErrors(stderr, configPath, startup.ConfigErrors)
		fmt.Fprintln(stderr, "Showing the defaults with the command-line flags applied.")
	}
	for _, n := range startup.Notices {
		fmt.Fprintf(stderr, "Notice: %s\n", n)
	}
	enc := toml.NewEncoder(stdout)
	enc.SetIndentTables(true)
	if err := enc.Encode(cfg); err != nil {
		fmt.Fprintf(stderr, "Error encoding config: %v\n", err)
		return 1
	}
	if rejected {
		return 1
	}
	return 0
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
		os.Exit(checkConfig(os.Stdout, os.Stderr, ".", cli.Config))
	}

	loaded := config.Load(".", cli.Config)
	cfg := loaded.Config

	cfg.Dir = cli.Path
	startup := tui.StartupMessages{ConfigErrors: loaded.Errors, Notices: applyCLIOverrides(&cfg)}
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
		os.Exit(showConfig(os.Stdout, os.Stderr, cfg, loaded.Path, startup))
	}

	app := tui.NewApp(cfg, startup)

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
