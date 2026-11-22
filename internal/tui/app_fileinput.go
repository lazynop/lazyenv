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

func (a *App) activeFileInput() *textinput.Model {
	if _, ok := fileInputLabels[a.mode]; ok {
		return &a.fileInput
	}
	return nil
}

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

func insertBeforeExt(name, suffix string) string {
	if strings.HasSuffix(name, ".env") && !strings.HasPrefix(name, ".env") {
		return name[:len(name)-4] + "." + suffix + ".env"
	}
	return name + "." + suffix
}

func duplicateName(name string) string { return insertBeforeExt(name, "copy") }
func templateName(name string) string  { return insertBeforeExt(name, "example") }

func (a *App) validateNewPath(name, dir string) (string, tea.Cmd) {
	if strings.ContainsAny(name, "/\\") {
		return "", a.flashWarning("Invalid name: must not contain path separators")
	}
	if !isEnvFile(name, a.config.Files) {
		return "", a.flashWarning("Invalid name: must match .env file patterns")
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", a.flashWarning("File already exists: " + name)
	}
	return path, nil
}

func (a App) finaliseNewFile(path string, successMsg string) (tea.Model, tea.Cmd) {
	ef, err := parser.ParseFile(path, a.config.Secrets)
	if err != nil {
		_ = a.deleteFile(path)
		cmd := a.flashError("Error parsing new file: " + err.Error())
		return a, cmd
	}

	if !a.config.NoGitCheck {
		CheckGitIgnore([]*model.EnvFile{ef})
	}

	a.fileList.Files = append(a.fileList.Files, ef)
	a.fileList.MoveCursorTo(len(a.fileList.Files) - 1)
	a.selectCursorFile()

	cmd := a.flashMessage(successMsg)
	return a, cmd
}
