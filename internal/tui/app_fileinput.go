package tui

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// activeFileInput returns a pointer to the textinput.Model the user is
// editing in the current file-operation mode, or nil for any other mode.
func (a *App) activeFileInput() *textinput.Model {
	switch a.mode {
	case ModeCreateFile, ModeDuplicateFile, ModeRenameFile, ModeTemplateFile:
		return &a.fileInput
	}
	return nil
}

// confirmActiveFileInput dispatches to the confirm-fn matching the current
// file-operation mode. Returns the app unchanged for any other mode.
func (a App) confirmActiveFileInput() (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeCreateFile:
		return a.confirmCreateFile()
	case ModeDuplicateFile:
		return a.confirmDuplicateFile()
	case ModeRenameFile:
		return a.confirmRenameFile()
	case ModeTemplateFile:
		return a.confirmTemplateFile()
	}
	return a, nil
}

// handleFileInputKey handles text input for all file-operation modes
// (create, duplicate, rename, template). They share the same interaction:
// Escape cancels, Enter confirms, other keys go to the text input.
func (a App) handleFileInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.fileOpSource = nil
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		return a.confirmActiveFileInput()
	default:
		if input := a.activeFileInput(); input != nil {
			var cmd tea.Cmd
			*input, cmd = input.Update(msg)
			return a, cmd
		}
		return a, nil
	}
}

// insertBeforeExt inserts a suffix before .env for names ending with .env,
// or appends it for names starting with .env (e.g. ".env.local").
// "demo.env" + "copy" → "demo.copy.env", ".env" + "copy" → ".env.copy"
func insertBeforeExt(name, suffix string) string {
	if strings.HasSuffix(name, ".env") && !strings.HasPrefix(name, ".env") {
		return name[:len(name)-4] + "." + suffix + ".env"
	}
	return name + "." + suffix
}

func duplicateName(name string) string { return insertBeforeExt(name, "copy") }
func templateName(name string) string  { return insertBeforeExt(name, "example") }

// validateNewPath checks that name is valid and the target path doesn't already exist.
// Returns the full path on success, or an error message on failure.
func (a App) validateNewPath(name, dir string) (string, string) {
	if strings.ContainsAny(name, "/\\") {
		return "", "Invalid name: must not contain path separators"
	}
	if !isEnvFile(name, a.config.Files) {
		return "", "Invalid name: must match .env file patterns"
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", "File already exists: " + name
	}
	return path, ""
}

// finaliseNewFile parses, registers, and selects a newly created file.
func (a App) finaliseNewFile(path string, successMsg string) (tea.Model, tea.Cmd) {
	ef, err := parser.ParseFile(path, a.config.Secrets)
	if err != nil {
		_ = os.Remove(path) // best-effort cleanup of the half-created file
		return a, a.flashError("Error parsing new file: " + err.Error())
	}

	if !a.config.NoGitCheck {
		CheckGitIgnore([]*model.EnvFile{ef})
	}

	a.fileList.Files = append(a.fileList.Files, ef)
	a.fileList.SetCursor(len(a.fileList.Files) - 1)
	a.varList.SetFile(ef)

	return a, a.flashMessage(successMsg)
}
