package main

import (
	"flag"
	"fmt"
	"gitlab.com/traveltoaiur/lazyenv/internal/tui"
	"os"

	tea "charm.land/bubbletea/v2"
)

var version = "0.1.0"

func main() {
	recursive := flag.Bool("r", false, "Scan subdirectories recursively")
	showAll := flag.Bool("a", false, "Show secrets in cleartext at startup")
	showVersion := flag.Bool("v", false, "Show version")
	showHelp := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("lazyenv %s\n", version)
		os.Exit(0)
	}

	// Determine directory to scan
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// Validate directory
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", dir)
		os.Exit(1)
	}

	app := tui.NewApp(tui.AppConfig{
		Dir:       dir,
		Recursive: *recursive,
		ShowAll:   *showAll,
	})

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: lazyenv [path] [flags]")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println(" path    Directory to scan (default: current directory)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println(" -r    Scan subdirectories recursively")
	fmt.Println(" -a    Show secrets in cleartext at startup")
	fmt.Println(" -v    Show version")
	fmt.Println(" -h    Show help")
}
