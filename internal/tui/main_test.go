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

const eastAsianEnv = "RUNEWIDTH_EASTASIAN"

func TestMain(m *testing.M) {
	if _, set := os.LookupEnv(eastAsianEnv); set {
		os.Exit(rerunWithout(eastAsianEnv))
	}
	runewidth.DefaultCondition.EastAsianWidth = false
	m.Run()
}

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
