package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	"gitlab.com/traveltoaiur/lazyenv/internal/tui"
)

var version = "0.1.3"

var cli struct {
	Path       string           `arg:"" optional:"" default:"." help:"Directory to scan." type:"existingdir"`
	Recursive  bool             `short:"r" help:"Scan subdirectories recursively."`
	ShowAll    bool             `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	NoGitCheck bool             `short:"G" name:"no-git-check" help:"Disable .gitignore check."`
	Version    kong.VersionFlag `short:"v" help:"Show version."`
}

func main() {
	kong.Parse(&cli,
		kong.Name("lazyenv"),
		kong.Description("TUI for managing .env files."),
		kong.Vars{"version": version},
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

	app := tui.NewApp(tui.AppConfig{
		Dir:        cli.Path,
		Recursive:  cli.Recursive,
		ShowAll:    cli.ShowAll,
		NoGitCheck: noGitCheck,
	})

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
