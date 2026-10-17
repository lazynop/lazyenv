package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// eastAsianEnv makes go-runewidth and x/ansi (the width oracle of lipgloss)
// count East Asian ambiguous runes (•, ≠, ─, …) as two columns.
const eastAsianEnv = "RUNEWIDTH_EASTASIAN"

// TestMain counts East Asian ambiguous runes as one column whatever the
// machine running the tests says, so every rendered screen, and the tests that
// measure one, come out the same everywhere. go-runewidth also reads the
// setting from the locale, into DefaultCondition, which is pinned here; x/ansi
// reads eastAsianEnv at init into unexported state, so with the variable set
// the tests rerun in a process without it.
func TestMain(m *testing.M) {
	if _, set := os.LookupEnv(eastAsianEnv); set {
		os.Exit(rerunWithout(eastAsianEnv))
	}
	runewidth.DefaultCondition.EastAsianWidth = false
	m.Run()
}

// rerunWithout runs this test binary again, with the same arguments and
// without the environment variable env, and returns its exit code.
func rerunWithout(env string) int {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, env+"=") })
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "rerunning the tests without "+env+":", err)
		return 1
	}
	return 0
}
