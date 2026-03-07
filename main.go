package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	toml "github.com/pelletier/go-toml/v2"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
	"gitlab.com/traveltoaiur/lazyenv/internal/tui"
)

var (
	version = "0.2.0"
	commit  = "none"
	date    = "unknown"
)

var cli struct {
	Path       string           `arg:"" optional:"" default:"." help:"Directory to scan." type:"existingdir"`
	Recursive  *bool            `short:"r" help:"Scan subdirectories recursively."`
	ShowAll    *bool            `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	NoGitCheck *bool            `short:"G" name:"no-git-check" help:"Disable .gitignore check."`
	NoBackup   *bool            `short:"B" name:"no-backup" help:"Disable .bak backup before first save."`
	ShowConfig bool             `name:"show-config" help:"Show effective configuration and exit."`
	Version    kong.VersionFlag `short:"v" help:"Show version."`
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

	cfg, warnings, err := config.Load(cli.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	cfg.Dir = cli.Path
	if cli.Recursive != nil {
		cfg.Recursive = *cli.Recursive
	}
	if cli.ShowAll != nil {
		cfg.ShowAll = *cli.ShowAll
	}
	if cli.NoBackup != nil {
		cfg.NoBackup = *cli.NoBackup
	}
	if cli.NoGitCheck != nil {
		cfg.NoGitCheck = *cli.NoGitCheck
	} else if !cfg.NoGitCheck {
		if _, err := exec.LookPath("git"); err != nil {
			cfg.NoGitCheck = true
		}
	}

	cfg.Warnings = warnings

	if cli.ShowConfig {
		enc := toml.NewEncoder(os.Stdout)
		enc.SetIndentTables(true)
		if err := enc.Encode(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app := tui.NewApp(cfg)

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
