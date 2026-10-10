package tui

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazyenv/internal/model"
)

func (a App) confirmTemplateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.fileOpSource
	a.fileOpSource = nil

	name := strings.TrimSpace(a.fileInput.Value())
	if name == "" || src == nil {
		return a, nil
	}

	destPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		return a, a.flashMessage(errMsg)
	}

	var b strings.Builder
	for i, line := range src.Lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if line.Type != model.LineVariable || line.VarIdx < 0 || line.VarIdx >= len(src.Vars) {
			b.WriteString(line.Content)
			continue
		}
		v := src.Vars[line.VarIdx]
		if v.HasExport {
			b.WriteString("export ")
		}
		b.WriteString(v.Key)
		b.WriteByte('=')
	}
	b.WriteByte('\n')

	if err := a.templateFile(destPath, []byte(b.String()), src); err != nil {
		return a, a.flashError("Error creating file: " + err.Error())
	}

	return a.finaliseNewFile(destPath, "Template from "+src.Name+" → "+name)
}
