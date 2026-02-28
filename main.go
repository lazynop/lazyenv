package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	tea "charm.land/bubbletea/v2"
	"gitlab.com/traveltoaiur/lazyenv/internal/tui"
)

var version = "0.1.1"

var cli struct {
	Path      string           `arg:"" optional:"" default:"." help:"Directory to scan." type:"path"`
	Recursive bool             `short:"r" help:"Scan subdirectories recursively."`
	ShowAll   bool             `short:"a" name:"show-all" help:"Show secrets in cleartext at startup."`
	Version   kong.VersionFlag `short:"v" help:"Show version."`
}

func main() {
	kong.Parse(&cli,
		kong.Name("lazyenv"),
		kong.Description("TUI for managing .env files."),
		kong.Vars{"version": version},
	)

	// Validate directory
	info, err := os.Stat(cli.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", cli.Path)
		os.Exit(1)
	}

	app := tui.NewApp(tui.AppConfig{
		Dir:       cli.Path,
		Recursive: cli.Recursive,
		ShowAll:   cli.ShowAll,
	})

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
