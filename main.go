package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"

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
	Recursive  bool             `short:"r" help:"Scan subdirectories recursively."`
	ShowAll    bool             `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	NoGitCheck bool             `short:"G" name:"no-git-check" help:"Disable .gitignore check."`
	NoBackup   bool             `short:"B" name:"no-backup" help:"Disable .bak backup before first save."`
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

	noGitCheck := cli.NoGitCheck
	if !noGitCheck {
		if _, err := exec.LookPath("git"); err != nil {
			noGitCheck = true
		}
	}

	cfg := config.DefaultConfig()
	cfg.Dir = cli.Path
	cfg.Recursive = cli.Recursive
	cfg.ShowAll = cli.ShowAll
	cfg.NoGitCheck = noGitCheck
	cfg.NoBackup = cli.NoBackup

	app := tui.NewApp(cfg)

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
