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
	version = "0.3.1"
	commit  = "none"
	date    = "unknown"
)

var cli struct {
	Path          string           `arg:"" optional:"" default:"." help:"Directory to scan." type:"existingdir"`
	Recursive     *bool            `short:"r" help:"Scan subdirectories recursively."`
	ShowAll       *bool            `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	NoGitCheck    *bool            `short:"G" name:"no-git-check" help:"Disable .gitignore check."`
	NoBackup      *bool            `short:"B" name:"no-backup" help:"Disable .bak backup before first save."`
	NoThemeBg     *bool            `name:"no-theme-bg" help:"Disable theme background color."`
	Sort          *string          `short:"s" name:"sort" help:"Sort order: position or alphabetical." enum:"position,alphabetical"`
	FileListWidth *int             `name:"file-list-width" help:"Width of the file list panel (0=auto)."`
	ShowConfig    bool             `name:"show-config" help:"Show effective configuration and exit."`
	ListThemes    bool             `name:"list-themes" help:"List available built-in themes and exit."`
	Version       kong.VersionFlag `short:"v" help:"Show version."`
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
	if cli.NoThemeBg != nil {
		cfg.NoThemeBg = *cli.NoThemeBg
		if cfg.NoThemeBg {
			cfg.Colors.Bg = ""
		}
	}
	if cli.FileListWidth != nil {
		cfg.Layout.FileListWidth = *cli.FileListWidth
	}
	if cli.Sort != nil {
		cfg.Sort = *cli.Sort
	}
	if cli.NoGitCheck != nil {
		cfg.NoGitCheck = *cli.NoGitCheck
	} else if !cfg.NoGitCheck {
		if _, err := exec.LookPath("git"); err != nil {
			cfg.NoGitCheck = true
		}
	}
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

	cfg, warnings, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	cfg.Dir = cli.Path
	applyCLIOverrides(&cfg)

	if cli.ListThemes {
		for _, name := range config.ThemeNames() {
			fmt.Println(name)
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
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
