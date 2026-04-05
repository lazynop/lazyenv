package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
	"github.com/lazynop/lazyenv/internal/util"
)

// AppMode represents the current application mode.
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeEditing
	ModeEditingCompare
	ModeCompareSelect
	ModeComparing
	ModeConfirmDelete
	ModeHelp
	ModeSearching
	ModeMatrix
	ModeMatrixEditing
	ModeConfigError
	ModeCreateFile
	ModeDuplicateFile
	ModeConfirmDeleteFile
	ModeRenameFile
	ModeTemplateFile
)

// Focus represents which panel has focus.
type Focus int

const (
	FocusFiles Focus = iota
	FocusVars
)

// FilesLoadedMsg is sent when directory scanning completes.
type FilesLoadedMsg struct {
	Files []*model.EnvFile
	Err   error
}

// ClearMessageMsg clears the status bar message.
type ClearMessageMsg struct{}

// ConfigWarningMsg is sent at startup if the config file has issues.
type ConfigWarningMsg struct{ Warning string }

// App is the main Bubble Tea model.
type App struct {
	config         config.Config
	configWarnings []string
	keys           KeyMap
	theme          Theme
	hasDarkBg      bool
	focus          Focus
	mode           AppMode
	width          int
	height         int
	fileWidth      int
	ready          bool

	fileList    FileListModel
	varList     VarListModel
	statusBar   StatusBarModel
	diffView    DiffViewModel
	editor      EditorModel
	searchInput textinput.Model
	matrixView  MatrixModel

	// Config error shown as blocking alert
	configError string

	// Backup state: tracks which files have been backed up this session
	backedUpPaths map[string]bool

	// Compare mode state
	compareFirstFile  *model.EnvFile
	compareEditFile   *model.EnvFile // which file is being edited in compare mode
	compareEditVarIdx int            // var index in that file

	// Create/duplicate/rename file state
	createFileInput    textinput.Model
	duplicateFileInput textinput.Model
	duplicateSource    *model.EnvFile // source file for duplication
	renameFileInput    textinput.Model
	renameSource       *model.EnvFile // file being renamed
	templateFileInput  textinput.Model
	templateSource     *model.EnvFile // source file for template
}

// NewApp creates a new App model.
func NewApp(cfg config.Config, warnings []string) App {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	cfi := textinput.New()
	cfi.Placeholder = ".env.example"
	cfi.CharLimit = 256

	dfi := textinput.New()
	dfi.CharLimit = 256

	rfi := textinput.New()
	rfi.CharLimit = 256

	tfi := textinput.New()
	tfi.CharLimit = 256

	varList := NewVarListModel(cfg.Layout)
	if cfg.Sort == "alphabetical" {
		varList.SortAlpha = true
	}

	return App{
		config:             cfg,
		configWarnings:     warnings,
		keys:               DefaultKeyMap(),
		theme:              BuildTheme(true, cfg.Colors), // default to dark, will update on BackgroundColorMsg
		hasDarkBg:          true,
		focus:              FocusFiles,
		mode:               ModeNormal,
		fileList:           NewFileListModel(),
		varList:            varList,
		statusBar:          NewStatusBarModel(),
		diffView:           NewDiffViewModel(cfg.Layout, cfg.Secrets),
		editor:             NewEditorModel(),
		searchInput:        ti,
		createFileInput:    cfi,
		duplicateFileInput: dfi,
		renameFileInput:    rfi,
		templateFileInput:  tfi,
		backedUpPaths:      make(map[string]bool),
	}
}

func (a App) Init() tea.Cmd {
	noGitCheck := a.config.NoGitCheck
	cmds := []tea.Cmd{
		func() tea.Msg {
			files, err := ScanDir(a.config.Dir, a.config.Recursive, a.config.Files, a.config.Secrets)
			if err == nil && !noGitCheck {
				CheckGitIgnore(files)
			}
			return FilesLoadedMsg{Files: files, Err: err}
		},
		tea.RequestBackgroundColor,
	}
	if len(a.configWarnings) > 0 {
		warning := strings.Join(a.configWarnings, "; ")
		cmds = append(cmds, func() tea.Msg {
			return ConfigWarningMsg{Warning: warning}
		})
	}
	return tea.Batch(cmds...)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.updateLayout()
		a.diffView.Width = a.width - 2
		a.diffView.Height = a.height - 4
		a.matrixView.Width = a.width
		a.matrixView.Height = a.height - 1
		return a, nil

	case tea.BackgroundColorMsg:
		a.hasDarkBg = msg.IsDark()
		a.theme = BuildTheme(a.hasDarkBg, a.config.Colors)
		return a, nil

	case ConfigWarningMsg:
		a.configError = msg.Warning
		a.mode = ModeConfigError
		return a, nil

	case FilesLoadedMsg:
		if msg.Err != nil {
			a.statusBar.SetMessage("Error: " + msg.Err.Error())
			return a, nil
		}
		a.fileList.SetFiles(msg.Files)
		if len(msg.Files) > 0 {
			a.varList.SetFile(msg.Files[0])
			a.varList.ShowSecrets = a.config.ShowAll
		}
		return a, nil

	case ClearMessageMsg:
		a.statusBar.ClearMessage()
		return a, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			return a.handleMouseClick(msg)
		}
		return a, nil

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg), nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeNormal, ModeSearching:
		return a.handleNormalMouseClick(msg), nil
	case ModeComparing, ModeEditingCompare:
		return a.handleCompareMouseClick(msg), nil
	case ModeCompareSelect:
		return a.handleCompareSelectMouseClick(msg), nil
	case ModeMatrix:
		return a.handleMatrixMouseClick(msg), nil
	case ModeHelp:
		a.mode = ModeNormal
		return a, nil
	}
	return a, nil
}

func (a App) handleNormalMouseClick(msg tea.MouseClickMsg) App {
	if msg.X < a.fileWidth {
		// Click on file list panel
		a.focus = FocusFiles
		a.fileList.Focused = true
		a.varList.Focused = false
		index := msg.Y - 2 + a.fileList.Offset // Y=0 border, Y=1 title, Y=2 first item
		if index >= 0 {
			prev := a.fileList.SelectedFile()
			a.fileList.SetCursor(index)
			if f := a.fileList.SelectedFile(); f != nil && f != prev {
				a.varList.SetFile(f)
			}
		}
	} else {
		// Click on var list panel
		a.focus = FocusVars
		a.fileList.Focused = false
		a.varList.Focused = true
		index := msg.Y - 2 + a.varList.Offset
		if index >= 0 && index < a.varList.DisplayCount() {
			a.varList.SetCursor(index)
		}
	}
	return a
}

func (a App) handleCompareMouseClick(msg tea.MouseClickMsg) App {
	index := msg.Y - 3 + a.diffView.Offset // Y=0 border, Y=1 title, Y=2 header, Y=3 first entry
	if index >= 0 {
		a.diffView.SetCursor(index)
	}
	return a
}

func (a App) handleCompareSelectMouseClick(msg tea.MouseClickMsg) App {
	index := msg.Y - 2 + a.fileList.Offset
	if index >= 0 {
		// Only move cursor, don't update Selected (compare target, not active file)
		a.fileList.Cursor = max(0, min(index, len(a.fileList.Files)-1))
		visible := a.fileList.Height - 2
		if visible > 0 && a.fileList.Cursor < a.fileList.Offset {
			a.fileList.Offset = a.fileList.Cursor
		}
		if visible > 0 && a.fileList.Cursor >= a.fileList.Offset+visible {
			a.fileList.Offset = a.fileList.Cursor - visible + 1
		}
	}
	return a
}

func (a App) handleMatrixMouseClick(msg tea.MouseClickMsg) App {
	row := msg.Y - 2 + a.matrixView.offsetRow // Y=0 header, Y=1 separator, Y=2 first row
	if row < 0 {
		return a
	}
	keyWidth := a.matrixView.layout.MatrixKeyWidth
	colWidth := a.matrixView.layout.MatrixColWidth
	if msg.X >= keyWidth {
		col := (msg.X-keyWidth)/colWidth + a.matrixView.offsetCol
		a.matrixView.SetCursor(row, col)
	} else {
		a.matrixView.SetCursor(row, a.matrixView.cursorCol)
	}
	return a
}

func (a App) handleMouseWheel(msg tea.MouseWheelMsg) App {
	up := msg.Button == tea.MouseWheelUp
	switch a.mode {
	case ModeNormal, ModeSearching:
		if msg.X < a.fileWidth {
			a = a.scrollFileList(up)
		} else {
			a = a.scrollVarList(up)
		}
	case ModeComparing, ModeEditingCompare:
		a = a.scrollDiffView(up)
	case ModeCompareSelect:
		a = a.scrollFileList(up)
	case ModeMatrix, ModeMatrixEditing:
		a = a.scrollMatrix(up)
	}
	return a
}

func (a App) scrollFileList(up bool) App {
	prev := a.fileList.SelectedFile()
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.fileList.MoveUp()
		} else {
			a.fileList.MoveDown()
		}
	}
	a.fileList.Select()
	if f := a.fileList.SelectedFile(); f != nil && f != prev {
		a.varList.SetFile(f)
	}
	return a
}

func (a App) scrollVarList(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.varList.MoveUp()
		} else {
			a.varList.MoveDown()
		}
	}
	return a
}

func (a App) scrollDiffView(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.diffView.MoveUp()
		} else {
			a.diffView.MoveDown()
		}
	}
	return a
}

func (a App) scrollMatrix(up bool) App {
	for range a.config.Layout.MouseScrollLines {
		if up {
			a.matrixView.MoveUp()
		} else {
			a.matrixView.MoveDown()
		}
	}
	return a
}

func (a App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Mode-specific handling
	switch a.mode {
	case ModeEditing:
		return a.handleEditingKey(msg)
	case ModeEditingCompare:
		return a.handleEditingCompareKey(msg)
	case ModeConfirmDelete:
		return a.handleConfirmDeleteKey(msg)
	case ModeHelp:
		return a.handleHelpKey(msg)
	case ModeSearching:
		return a.handleSearchKey(msg)
	case ModeComparing:
		return a.handleComparingKey(msg)
	case ModeCompareSelect:
		return a.handleCompareSelectKey(msg)
	case ModeMatrix:
		return a.handleMatrixKey(msg)
	case ModeMatrixEditing:
		return a.handleMatrixEditingKey(msg)
	case ModeCreateFile, ModeDuplicateFile, ModeRenameFile, ModeTemplateFile:
		return a.handleFileInputKey(msg)
	case ModeConfirmDeleteFile:
		return a.handleConfirmDeleteFileKey(msg)
	case ModeConfigError:
		return a, tea.Quit
	}

	// Normal mode
	return a.handleNormalKey(msg)
}

func (a App) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit
	case key.Matches(msg, a.keys.Help):
		a.mode = ModeHelp
		return a, nil
	}

	// Navigation keys (Up/Down/Left/Right/Enter)
	if updated, handled := a.handleNormalNavigation(msg); handled {
		return updated, nil
	}

	// Var-focused actions (Edit/Add/Delete/Yank/Peek)
	if m, cmd, handled := a.handleNormalVarAction(msg); handled {
		return m, cmd
	}

	// File-focused actions (Create/Duplicate/Delete/Rename)
	if a.focus == FocusFiles {
		if m, cmd, handled := a.handleNormalFileAction(msg); handled {
			return m, cmd
		}
	}

	// Global actions (Compare/Save/Reset/Toggle/Search/Matrix)
	return a.handleNormalGlobalAction(msg)
}

func (a App) handleNormalFileAction(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, a.keys.CreateFile):
		a.createFileInput.SetValue("")
		a.mode = ModeCreateFile
		return a, a.createFileInput.Focus(), true
	}

	f := a.fileList.SelectedFile()
	if f == nil {
		f = a.fileList.CursorFile()
	}
	if f == nil {
		return a, nil, false
	}

	switch {
	case key.Matches(msg, a.keys.DuplicateFile):
		a.duplicateSource = f
		a.duplicateFileInput.SetValue(duplicateName(f.Name))
		a.mode = ModeDuplicateFile
		return a, a.duplicateFileInput.Focus(), true

	case key.Matches(msg, a.keys.DeleteFile):
		if f.Modified {
			a.statusBar.SetMessage("Save or reset changes before deleting")
			return a, clearMessageAfter(a.config.Layout.MessageTimeout), true
		}
		a.mode = ModeConfirmDeleteFile
		a.statusBar.SetMessage("Delete " + f.Name + "? (y/n)")
		return a, nil, true

	case key.Matches(msg, a.keys.RenameFile):
		if f.Modified {
			a.statusBar.SetMessage("Save or reset changes before renaming")
			return a, clearMessageAfter(a.config.Layout.MessageTimeout), true
		}
		a.renameSource = f
		a.renameFileInput.SetValue(f.Name)
		a.mode = ModeRenameFile
		return a, a.renameFileInput.Focus(), true

	case key.Matches(msg, a.keys.TemplateFile):
		a.templateSource = f
		a.templateFileInput.SetValue(templateName(f.Name))
		a.mode = ModeTemplateFile
		return a, a.templateFileInput.Focus(), true
	}

	return a, nil, false
}

func (a App) handleNormalGlobalAction(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Compare):
		if len(a.fileList.Files) >= 2 {
			f := a.fileList.SelectedFile()
			if f == nil && a.fileList.CursorFile() != nil {
				f = a.fileList.CursorFile()
			}
			a.compareFirstFile = f
			a.mode = ModeCompareSelect
			a.focus = FocusFiles
			a.fileList.Focused = true
			a.varList.Focused = false
			a.statusBar.SetMessage("Select second file to compare...")
		}

	case key.Matches(msg, a.keys.Save):
		return a.handleSave()

	case key.Matches(msg, a.keys.Reset):
		return a.handleReset()

	case key.Matches(msg, a.keys.ToggleSecret):
		a.varList.ShowSecrets = !a.varList.ShowSecrets
		if a.varList.ShowSecrets {
			a.statusBar.SetMessage("Secrets revealed")
		} else {
			a.statusBar.SetMessage("Secrets hidden")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)

	case key.Matches(msg, a.keys.ToggleSort):
		a.varList.ToggleSort()
		if a.varList.SortAlpha {
			a.statusBar.SetMessage("Sorted alphabetically")
		} else {
			a.statusBar.SetMessage("Sorted by position")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)

	case key.Matches(msg, a.keys.Search):
		a.mode = ModeSearching
		a.searchInput.SetValue("")
		return a, a.searchInput.Focus()

	case key.Matches(msg, a.keys.Matrix):
		if len(a.fileList.Files) < 2 {
			a.statusBar.SetMessage("Need at least 2 files for matrix")
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
		a.matrixView = NewMatrixModel(a.fileList.Files, a.config.Layout, a.config.Secrets)
		a.matrixView.Width = a.width
		a.matrixView.Height = a.height - 1
		a.mode = ModeMatrix
	}

	return a, nil
}

func (a App) handleNormalNavigation(msg tea.KeyPressMsg) (App, bool) {
	switch {
	case key.Matches(msg, a.keys.Up):
		if a.focus == FocusFiles {
			a.fileList.MoveUp()
			a.fileList.Select()
			if f := a.fileList.SelectedFile(); f != nil {
				a.varList.SetFile(f)
			}
		} else {
			a.varList.MoveUp()
		}
		return a, true

	case key.Matches(msg, a.keys.Down):
		if a.focus == FocusFiles {
			a.fileList.MoveDown()
			a.fileList.Select()
			if f := a.fileList.SelectedFile(); f != nil {
				a.varList.SetFile(f)
			}
		} else {
			a.varList.MoveDown()
		}
		return a, true

	case key.Matches(msg, a.keys.Left):
		a.focus = FocusFiles
		a.fileList.Focused = true
		a.varList.Focused = false
		return a, true

	case key.Matches(msg, a.keys.Right):
		a.focus = FocusVars
		a.fileList.Focused = false
		a.varList.Focused = true
		return a, true

	case key.Matches(msg, a.keys.Enter):
		if a.focus == FocusFiles {
			a.fileList.Select()
			f := a.fileList.SelectedFile()
			if f != nil {
				a.varList.SetFile(f)
			}
			a.focus = FocusVars
			a.fileList.Focused = false
			a.varList.Focused = true
		}
		return a, true
	}

	return a, false
}

func (a App) handleNormalVarAction(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if a.focus != FocusVars {
		return a, nil, false
	}

	switch {
	case key.Matches(msg, a.keys.Edit):
		v := a.varList.SelectedVar()
		if v != nil {
			idx := a.varList.SelectedVarIndex()
			a.editor.StartEdit(v, idx)
			a.mode = ModeEditing
			return a, a.editor.input.Focus(), true
		}

	case key.Matches(msg, a.keys.Add):
		if a.varList.File != nil {
			a.editor.StartAdd()
			a.mode = ModeEditing
			return a, a.editor.input.Focus(), true
		}

	case key.Matches(msg, a.keys.Delete):
		v := a.varList.SelectedVar()
		if v != nil {
			a.mode = ModeConfirmDelete
			return a, nil, true
		}

	case key.Matches(msg, a.keys.YankValue):
		v := a.varList.SelectedVar()
		if v != nil {
			a.statusBar.SetMessage(fmt.Sprintf("Copied %s value to clipboard", v.Key))
			return a, tea.Batch(
				tea.SetClipboard(v.Value),
				clearMessageAfter(a.config.Layout.MessageTimeout),
			), true
		}

	case key.Matches(msg, a.keys.YankLine):
		v := a.varList.SelectedVar()
		if v != nil {
			line := v.Key + "=" + v.Value
			a.statusBar.SetMessage(fmt.Sprintf("Copied %s to clipboard", v.Key+"=..."))
			return a, tea.Batch(
				tea.SetClipboard(line),
				clearMessageAfter(a.config.Layout.MessageTimeout),
			), true
		}

	case key.Matches(msg, a.keys.Peek):
		a.varList.Peeking = !a.varList.Peeking
		return a, nil, true
	}

	return a, nil, false
}

func (a App) handleEditingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		result := a.editor.Finish()
		a.mode = ModeNormal
		if result.Cancelled {
			return a, nil
		}
		if result.IsAdd {
			if result.AddStep == addStepKey {
				// Just got the key, now need value
				a.editor.StartAddValue(result.Value)
				a.mode = ModeEditing
				return a, a.editor.input.Focus()
			}
			// Got the value, add the variable
			a.varList.File.AddVar(a.editor.addKey, result.Value, util.IsSecret(a.editor.addKey, result.Value, a.config.Secrets))
			a.varList.Refresh()
			a.statusBar.SetMessage("Added " + a.editor.addKey)
		} else {
			a.varList.File.UpdateVar(result.VarIndex, result.Value)
			a.varList.Refresh()
			a.statusBar.SetMessage("Modified " + a.varList.File.Vars[result.VarIndex].Key)
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

func (a App) handleConfirmDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		idx := a.varList.SelectedVarIndex()
		if idx >= 0 {
			name := a.varList.File.Vars[idx].Key
			a.varList.File.DeleteVar(idx)
			a.varList.Refresh()
			a.statusBar.SetMessage("Deleted " + name)
		}
		a.mode = ModeNormal
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
	}
	return a, nil
}

func (a App) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, a.keys.Escape) || key.Matches(msg, a.keys.Help) || key.Matches(msg, a.keys.Quit) {
		a.mode = ModeNormal
	}
	return a, nil
}

func (a App) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.varList.SetSearch("")
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		a.mode = ModeNormal
		return a, nil
	default:
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		a.varList.SetSearch(a.searchInput.Value())
		return a, cmd
	}
}

// handleFileInputKey handles text input for all file-operation modes
// (create, duplicate, rename, template). They share the same interaction:
// Escape cancels, Enter confirms, other keys go to the text input.
func (a App) handleFileInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.duplicateSource = nil
		a.renameSource = nil
		a.templateSource = nil
		return a, nil
	case key.Matches(msg, a.keys.Enter):
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
	default:
		var cmd tea.Cmd
		switch a.mode {
		case ModeCreateFile:
			a.createFileInput, cmd = a.createFileInput.Update(msg)
		case ModeDuplicateFile:
			a.duplicateFileInput, cmd = a.duplicateFileInput.Update(msg)
		case ModeRenameFile:
			a.renameFileInput, cmd = a.renameFileInput.Update(msg)
		case ModeTemplateFile:
			a.templateFileInput, cmd = a.templateFileInput.Update(msg)
		}
		return a, cmd
	}
}

func (a App) confirmCreateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal

	name := strings.TrimSpace(a.createFileInput.Value())
	if name == "" {
		return a, nil
	}

	fullPath, errMsg := a.validateNewPath(name, a.config.Dir)
	if errMsg != "" {
		a.statusBar.SetMessage(errMsg)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	if err := os.WriteFile(fullPath, nil, 0644); err != nil {
		a.statusBar.SetMessage("Error creating file: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	return a.finaliseNewFile(fullPath, "Created "+name)
}

func (a App) confirmDuplicateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.duplicateSource
	a.duplicateSource = nil

	name := strings.TrimSpace(a.duplicateFileInput.Value())
	if name == "" || src == nil {
		return a, nil
	}

	// Place beside the source so it appears in the same watched directory
	destPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		a.statusBar.SetMessage(errMsg)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	// Copy raw bytes to preserve comments, formatting, and round-trip fidelity
	data, err := os.ReadFile(src.Path)
	if err != nil {
		a.statusBar.SetMessage("Error reading source: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		a.statusBar.SetMessage("Error creating file: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	return a.finaliseNewFile(destPath, "Duplicated "+src.Name+" → "+name)
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
		os.Remove(path)
		a.statusBar.SetMessage("Error parsing new file: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	if !a.config.NoGitCheck {
		CheckGitIgnore([]*model.EnvFile{ef})
	}

	a.fileList.Files = append(a.fileList.Files, ef)
	a.fileList.SetCursor(len(a.fileList.Files) - 1)
	a.varList.SetFile(ef)

	a.statusBar.SetMessage(successMsg)
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) handleConfirmDeleteFileKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		f := a.fileList.SelectedFile()
		if f == nil {
			f = a.fileList.CursorFile()
		}
		a.mode = ModeNormal
		if f == nil {
			return a, nil
		}

		if err := os.Remove(f.Path); err != nil {
			a.statusBar.SetMessage("Error deleting file: " + err.Error())
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}

		// Remove from file list
		idx := -1
		for i, ef := range a.fileList.Files {
			if ef.Path == f.Path {
				idx = i
				break
			}
		}
		if idx >= 0 {
			a.fileList.Files = append(a.fileList.Files[:idx], a.fileList.Files[idx+1:]...)
		}

		if len(a.fileList.Files) == 0 {
			a.fileList.Cursor = 0
			a.fileList.Selected = 0
			a.varList.SetFile(nil)
		} else {
			if a.fileList.Cursor >= len(a.fileList.Files) {
				a.fileList.Cursor = len(a.fileList.Files) - 1
			}
			a.fileList.Selected = a.fileList.Cursor
			a.varList.SetFile(a.fileList.Files[a.fileList.Cursor])
		}

		a.statusBar.SetMessage("Deleted " + f.Name)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)

	case key.Matches(msg, a.keys.Deny), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.statusBar.ClearMessage()
	}
	return a, nil
}

func (a App) confirmRenameFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.renameSource
	a.renameSource = nil

	name := strings.TrimSpace(a.renameFileInput.Value())
	if name == "" || src == nil || name == src.Name {
		return a, nil
	}

	newPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		a.statusBar.SetMessage(errMsg)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	if err := os.Rename(src.Path, newPath); err != nil {
		a.statusBar.SetMessage("Error renaming file: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	if a.backedUpPaths[src.Path] {
		delete(a.backedUpPaths, src.Path)
		a.backedUpPaths[newPath] = true
	}

	oldName := src.Name
	src.Path = newPath
	src.Name = name

	// Re-check gitignore — the new name may have different coverage
	if !a.config.NoGitCheck {
		src.GitWarning = false
		CheckGitIgnore([]*model.EnvFile{src})
	}

	a.varList.SetFile(src)

	a.statusBar.SetMessage("Renamed " + oldName + " → " + name)
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) confirmTemplateFile() (tea.Model, tea.Cmd) {
	a.mode = ModeNormal
	src := a.templateSource
	a.templateSource = nil

	name := strings.TrimSpace(a.templateFileInput.Value())
	if name == "" || src == nil {
		return a, nil
	}

	destPath, errMsg := a.validateNewPath(name, filepath.Dir(src.Path))
	if errMsg != "" {
		a.statusBar.SetMessage(errMsg)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
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

	if err := os.WriteFile(destPath, []byte(b.String()), 0644); err != nil {
		a.statusBar.SetMessage("Error creating file: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	return a.finaliseNewFile(destPath, "Template from "+src.Name+" → "+name)
}

func (a App) handleMatrixKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
	case key.Matches(msg, a.keys.Up):
		a.matrixView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.matrixView.MoveDown()
	case key.Matches(msg, a.keys.Left):
		a.matrixView.MoveLeft()
	case key.Matches(msg, a.keys.Right):
		a.matrixView.MoveRight()
	case key.Matches(msg, a.keys.ToggleSort):
		a.matrixView.ToggleSort()
		if a.matrixView.sortMode == model.SortCompleteness {
			a.statusBar.SetMessage("Sorted by completeness")
		} else {
			a.statusBar.SetMessage("Sorted alphabetically")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	case key.Matches(msg, a.keys.Add):
		cmd := a.matrixView.StartEdit()
		if a.matrixView.editing {
			a.mode = ModeMatrixEditing
			return a, cmd
		}
		if a.matrixView.message != "" {
			a.statusBar.SetMessage(a.matrixView.message)
			a.matrixView.message = ""
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
	}
	return a, nil
}

func (a App) handleMatrixEditingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.matrixView.CancelEdit()
		a.mode = ModeMatrix
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		a.matrixView.ConfirmEdit()
		a.mode = ModeMatrix
		a.statusBar.SetMessage("Variable added")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	default:
		var cmd tea.Cmd
		a.matrixView.editor, cmd = a.matrixView.editor.Update(msg)
		return a, cmd
	}
}

func (a App) View() tea.View {
	if !a.ready {
		return tea.NewView("Loading...")
	}

	var content string

	switch a.mode {
	case ModeConfigError:
		content = a.viewConfigError()
	case ModeHelp:
		content = a.viewHelp()
	case ModeComparing:
		diffContent := a.diffView.View(a.theme)
		statusBarContent := a.statusBar.View(a.theme, a.mode, a.focus, "", 0, a.diffView.Stats)
		content = lipgloss.JoinVertical(lipgloss.Left, diffContent, statusBarContent)
	case ModeEditingCompare:
		diffContent := a.diffView.View(a.theme)
		editorBar := a.theme.StatusBar.Width(a.width).Render("  " + a.editor.View())
		statusBarContent := a.statusBar.View(a.theme, ModeEditing, a.focus, "", 0)
		content = lipgloss.JoinVertical(lipgloss.Left, diffContent, editorBar, statusBarContent)
	case ModeMatrix:
		matrixContent := a.matrixView.View(a.theme)
		statusBarContent := a.statusBar.View(a.theme, a.mode, a.focus, "", 0)
		content = lipgloss.JoinVertical(lipgloss.Left, matrixContent, statusBarContent)
	case ModeMatrixEditing:
		matrixContent := a.matrixView.View(a.theme)
		prompt := fmt.Sprintf("  Add %s to %s: ", a.matrixView.editKey, a.matrixView.fileNames[a.matrixView.editFile])
		editorBar := a.theme.StatusBar.Width(a.width).Render(prompt + a.matrixView.editor.View())
		statusBarContent := a.statusBar.View(a.theme, ModeEditing, a.focus, "", 0)
		content = lipgloss.JoinVertical(lipgloss.Left, matrixContent, editorBar, statusBarContent)
	default:
		a.updateLayout()

		// Two-panel layout
		filePanel := a.fileList.View(a.theme)
		varPanel := a.varList.View(a.theme)

		panels := lipgloss.JoinHorizontal(lipgloss.Top, filePanel, varPanel)

		// Status bar
		fileName := ""
		varCount := 0
		if f := a.varList.File; f != nil {
			fileName = f.Name
			varCount = len(f.Vars)
		}

		statusBarContent := a.statusBar.View(a.theme, a.mode, a.focus, fileName, varCount)

		if inputBar := a.inputBarView(); inputBar != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, panels, inputBar, statusBarContent)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, panels, statusBarContent)
		}
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.BackgroundColor = a.theme.ColorBg
	if !a.config.NoMouse {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
}

func (a App) inputBarView() string {
	bar := a.theme.StatusBar.Width(a.width)
	switch a.mode {
	case ModeSearching:
		return bar.Render("  / " + a.searchInput.View())
	case ModeEditing:
		return bar.Render("  " + a.editor.View())
	case ModeCreateFile:
		return bar.Render("  New file: " + a.createFileInput.View())
	case ModeDuplicateFile:
		return bar.Render("  Duplicate as: " + a.duplicateFileInput.View())
	case ModeRenameFile:
		return bar.Render("  Rename: " + a.renameFileInput.View())
	case ModeTemplateFile:
		return bar.Render("  Template as: " + a.templateFileInput.View())
	}
	return ""
}

func (a *App) updateLayout() {
	a.fileWidth = max(a.width/4, config.FileListMinWidth)
	if a.config.Layout.FileListWidth > 0 {
		a.fileWidth = a.config.Layout.FileListWidth
	}
	varWidth := a.width - a.fileWidth

	panelHeight := a.height - 3 // space for status bar

	a.fileList.Width = a.fileWidth
	a.fileList.Height = panelHeight
	a.fileList.Focused = a.focus == FocusFiles

	a.varList.Width = varWidth
	a.varList.Height = panelHeight
	a.varList.Focused = a.focus == FocusVars

	a.statusBar.Width = a.width
}

func clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearMessageMsg{}
	})
}
