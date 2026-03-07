package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gitlab.com/traveltoaiur/lazyenv/internal/config"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
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
	ready          bool

	fileList    FileListModel
	varList     VarListModel
	statusBar   StatusBarModel
	diffView    DiffViewModel
	editor      EditorModel
	searchInput textinput.Model
	matrixView  MatrixModel

	// Backup state: tracks which files have been backed up this session
	backedUpPaths map[string]bool

	// Compare mode state
	compareFirstFile  *model.EnvFile
	compareEditFile   *model.EnvFile // which file is being edited in compare mode
	compareEditVarIdx int            // var index in that file
}

// NewApp creates a new App model.
func NewApp(cfg config.Config, warnings []string) App {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	varList := NewVarListModel(cfg.Layout)
	if cfg.Sort == "alphabetical" {
		varList.SortAlpha = true
	}

	return App{
		config:         cfg,
		configWarnings: warnings,
		keys:           DefaultKeyMap(),
		theme:          BuildTheme(true, cfg.Colors), // default to dark, will update on BackgroundColorMsg
		hasDarkBg:      true,
		focus:          FocusFiles,
		mode:           ModeNormal,
		fileList:       NewFileListModel(),
		varList:        varList,
		statusBar:      NewStatusBarModel(),
		diffView:       NewDiffViewModel(cfg.Layout),
		editor:         NewEditorModel(),
		searchInput:    ti,
		backedUpPaths:  make(map[string]bool),
	}
}

func (a App) Init() tea.Cmd {
	noGitCheck := a.config.NoGitCheck
	cmds := []tea.Cmd{
		func() tea.Msg {
			files, err := ScanDir(a.config.Dir, a.config.Recursive, a.config.Files)
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
		a.statusBar.SetMessage("Config: " + msg.Warning)
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)

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

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
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
	}

	// Normal mode
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

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

	case key.Matches(msg, a.keys.Left):
		a.focus = FocusFiles
		a.fileList.Focused = true
		a.varList.Focused = false

	case key.Matches(msg, a.keys.Right):
		a.focus = FocusVars
		a.fileList.Focused = false
		a.varList.Focused = true

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

	case key.Matches(msg, a.keys.Edit):
		if a.focus == FocusVars {
			v := a.varList.SelectedVar()
			if v != nil {
				idx := a.varList.SelectedVarIndex()
				a.editor.StartEdit(v, idx)
				a.mode = ModeEditing
				return a, a.editor.input.Focus()
			}
		}

	case key.Matches(msg, a.keys.Add):
		if a.focus == FocusVars && a.varList.File != nil {
			a.editor.StartAdd()
			a.mode = ModeEditing
			return a, a.editor.input.Focus()
		}

	case key.Matches(msg, a.keys.Delete):
		if a.focus == FocusVars {
			v := a.varList.SelectedVar()
			if v != nil {
				a.mode = ModeConfirmDelete
			}
		}

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
		a.matrixView = NewMatrixModel(a.fileList.Files, a.config.Layout)
		a.matrixView.Width = a.width
		a.matrixView.Height = a.height - 1
		a.mode = ModeMatrix

	case key.Matches(msg, a.keys.YankValue):
		if a.focus == FocusVars {
			v := a.varList.SelectedVar()
			if v != nil {
				a.statusBar.SetMessage(fmt.Sprintf("Copied %s value to clipboard", v.Key))
				return a, tea.Batch(
					tea.SetClipboard(v.Value),
					clearMessageAfter(a.config.Layout.MessageTimeout),
				)
			}
		}

	case key.Matches(msg, a.keys.YankLine):
		if a.focus == FocusVars {
			v := a.varList.SelectedVar()
			if v != nil {
				line := v.Key + "=" + v.Value
				a.statusBar.SetMessage(fmt.Sprintf("Copied %s to clipboard", v.Key+"=..."))
				return a, tea.Batch(
					tea.SetClipboard(line),
					clearMessageAfter(a.config.Layout.MessageTimeout),
				)
			}
		}

	case key.Matches(msg, a.keys.Peek):
		if a.focus == FocusVars {
			a.varList.Peeking = !a.varList.Peeking
		}

	case key.Matches(msg, a.keys.Help):
		a.mode = ModeHelp
	}

	return a, nil
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
			a.varList.File.AddVar(a.editor.addKey, result.Value)
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

func (a App) handleComparingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit), key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.compareFirstFile = nil
		// Restore cursor to the selected file so file list and var panel stay in sync.
		a.fileList.Cursor = a.fileList.Selected
	case key.Matches(msg, a.keys.Up):
		a.diffView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.diffView.MoveDown()
	case key.Matches(msg, a.keys.Right):
		if k := a.diffView.CopyToRight(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileB.Name)
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
	case key.Matches(msg, a.keys.Left):
		if k := a.diffView.CopyToLeft(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileA.Name)
			return a, clearMessageAfter(a.config.Layout.MessageTimeout)
		}
	case key.Matches(msg, a.keys.Filter):
		a.diffView.ToggleFilter()
		if a.diffView.HideEqual {
			a.statusBar.SetMessage("Showing differences only")
		} else {
			a.statusBar.SetMessage("Showing all entries")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	case key.Matches(msg, a.keys.Save):
		return a.handleCompareSave()
	case key.Matches(msg, a.keys.Edit): // e = edit left
		return a.startCompareEdit(a.diffView.FileA)
	case msg.String() == "E": // E = edit right
		return a.startCompareEdit(a.diffView.FileB)
	case msg.String() == "r":
		if errMsg := a.diffView.Reset(); errMsg != "" {
			a.statusBar.SetMessage(errMsg)
		} else {
			// Update file references in the main list too
			for i, f := range a.fileList.Files {
				if f.Path == a.diffView.FileA.Path {
					a.fileList.Files[i] = a.diffView.FileA
					if a.fileList.Selected == i {
						a.varList.SetFile(a.diffView.FileA)
					}
				}
				if f.Path == a.diffView.FileB.Path {
					a.fileList.Files[i] = a.diffView.FileB
					if a.fileList.Selected == i {
						a.varList.SetFile(a.diffView.FileB)
					}
				}
			}
			a.statusBar.SetMessage("Reset to saved state")
		}
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	return a, nil
}

func (a App) handleCompareSave() (App, tea.Cmd) {
	saved := []string{}
	var warn strings.Builder
	for _, f := range []*model.EnvFile{a.diffView.FileA, a.diffView.FileB} {
		if f != nil && f.Modified {
			warn.WriteString(a.backupIfNeeded(f.Path))
			if err := parser.WriteFile(f); err != nil {
				a.statusBar.SetMessage("Error saving " + f.Name + ": " + err.Error())
				return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
			}
			// Re-parse to refresh RawLines
			refreshed, err := parser.ParseFile(f.Path)
			if err == nil {
				refreshed.GitWarning = f.GitWarning
				for i, existing := range a.fileList.Files {
					if existing.Path == f.Path {
						a.fileList.Files[i] = refreshed
						if a.fileList.Selected == i {
							a.varList.SetFile(refreshed)
						}
						break
					}
				}
				if f == a.diffView.FileA {
					a.diffView.FileA = refreshed
				} else {
					a.diffView.FileB = refreshed
				}
			}
			saved = append(saved, f.Name)
		}
	}
	if len(saved) == 0 {
		a.statusBar.SetMessage("No changes to save")
	} else {
		a.statusBar.SetMessage(warn.String() + "Saved " + strings.Join(saved, ", "))
	}
	// Recompute diff after save
	a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
	a.diffView.recompute()
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) startCompareEdit(file *model.EnvFile) (tea.Model, tea.Cmd) {
	if a.diffView.Cursor < 0 || a.diffView.Cursor >= len(a.diffView.Entries) {
		return a, nil
	}
	e := a.diffView.Entries[a.diffView.Cursor]

	// Find the var index in the target file
	varIdx := -1
	for i := len(file.Vars) - 1; i >= 0; i-- {
		if file.Vars[i].Key == e.Key {
			varIdx = i
			break
		}
	}

	if varIdx < 0 {
		// Key doesn't exist in this file
		a.statusBar.SetMessage(e.Key + " not in " + file.Name)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	a.compareEditFile = file
	a.compareEditVarIdx = varIdx
	a.editor.StartEdit(&file.Vars[varIdx], varIdx)
	a.editor.label = fmt.Sprintf("Edit %s in %s: ", e.Key, file.Name)
	a.mode = ModeEditingCompare
	return a, a.editor.input.Focus()
}

func (a App) handleEditingCompareKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeComparing
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		result := a.editor.Finish()
		a.compareEditFile.UpdateVar(a.compareEditVarIdx, result.Value)
		// Recompute diff
		a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
		a.diffView.recompute()
		a.mode = ModeComparing
		a.statusBar.SetMessage("Modified " + a.compareEditFile.Vars[a.compareEditVarIdx].Key)
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	default:
		var cmd tea.Cmd
		a.editor.input, cmd = a.editor.input.Update(msg)
		return a, cmd
	}
}

func (a App) handleCompareSelectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.compareFirstFile = nil
		a.statusBar.ClearMessage()
		// Restore cursor to the selected file so file list and var panel stay in sync.
		a.fileList.Cursor = a.fileList.Selected
	case key.Matches(msg, a.keys.Up):
		a.fileList.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.fileList.MoveDown()
	case key.Matches(msg, a.keys.Enter):
		second := a.fileList.CursorFile()
		if second != nil && a.compareFirstFile != nil && second != a.compareFirstFile {
			a.diffView.SetFiles(a.compareFirstFile, second)
			a.diffView.Width = a.width - 2
			a.diffView.Height = a.height - 4
			a.mode = ModeComparing
			a.statusBar.ClearMessage()
		}
	}
	return a, nil
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

// backupIfNeeded creates a .bak backup of the file before the first save of
// the session. It is a no-op if --no-backup was set or the file was already
// backed up. Returns a warning message (empty on success or skip).
func (a App) backupIfNeeded(path string) string {
	if a.config.NoBackup || a.backedUpPaths[path] {
		return ""
	}
	if err := parser.CreateBackup(path); err != nil {
		a.backedUpPaths[path] = true // don't retry on every save
		return "backup failed: " + err.Error() + " - "
	}
	a.backedUpPaths[path] = true
	return ""
}

func (a App) handleSave() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.statusBar.SetMessage("No file selected")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	if !f.Modified {
		a.statusBar.SetMessage("No changes to save")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	warn := a.backupIfNeeded(f.Path)

	if err := parser.WriteFile(f); err != nil {
		a.statusBar.SetMessage("Error saving: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
	}

	// Re-parse to refresh RawLines
	refreshed, err := parser.ParseFile(f.Path)
	if err != nil {
		a.statusBar.SetMessage(warn + "Saved but refresh failed: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
	}
	refreshed.GitWarning = f.GitWarning

	// Replace file in the list
	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = refreshed
			if a.fileList.Selected == i {
				a.varList.SetFile(refreshed)
			}
			break
		}
	}

	a.statusBar.SetMessage(warn + "Saved " + f.Name)
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) handleReset() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.statusBar.SetMessage("No file selected")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}
	if !f.Modified {
		a.statusBar.SetMessage("No changes to reset")
		return a, clearMessageAfter(a.config.Layout.MessageTimeout)
	}

	refreshed, err := parser.ParseFile(f.Path)
	if err != nil {
		a.statusBar.SetMessage("Error reloading: " + err.Error())
		return a, clearMessageAfter(a.config.Layout.ErrorMessageTimeout)
	}
	refreshed.GitWarning = f.GitWarning

	for i, existing := range a.fileList.Files {
		if existing.Path == f.Path {
			a.fileList.Files[i] = refreshed
			if a.fileList.Selected == i {
				a.varList.SetFile(refreshed)
			}
			break
		}
	}

	a.statusBar.SetMessage("Reset to saved state")
	return a, clearMessageAfter(a.config.Layout.MessageTimeout)
}

func (a App) View() tea.View {
	if !a.ready {
		return tea.NewView("Loading...")
	}

	var content string

	switch a.mode {
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

		// Search bar
		if a.mode == ModeSearching {
			searchBar := a.theme.StatusBar.Width(a.width).Render("  / " + a.searchInput.View())
			content = lipgloss.JoinVertical(lipgloss.Left, panels, searchBar, statusBarContent)
		} else if a.mode == ModeEditing {
			editorBar := a.theme.StatusBar.Width(a.width).Render("  " + a.editor.View())
			content = lipgloss.JoinVertical(lipgloss.Left, panels, editorBar, statusBarContent)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, panels, statusBarContent)
		}
	}

	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (a *App) updateLayout() {
	fileWidth := max(a.width/4, 20)
	varWidth := a.width - fileWidth

	panelHeight := a.height - 3 // space for status bar

	a.fileList.Width = fileWidth
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

func (a App) viewHelp() string {
	helpText := `
  lazyenv — TUI for managing .env files

  Navigation
    ↑/↓  j/k      Navigate items
    ←/→  h/l      Switch panels (files / variables)
    Enter          Select file

  Actions
    e              Edit variable value
    a              Add new variable
    d              Delete variable (with confirmation)
    y              Copy value to clipboard
    Y              Copy KEY=value to clipboard
    p              Peek original value (toggle)
    w              Save changes
    r              Reset file (discard changes)
    c              Compare two files (diff view)
    m              Completeness matrix (multi-file)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    Ctrl+S         Toggle secret masking

  File Indicators
    ●              Selected file
    *              Modified (unsaved changes)
    !              Not covered by .gitignore

  Variable Indicators
    +              Newly added variable
    *              Modified variable
    -              Deleted variable (until save)
    D              Duplicate key
    ○              Empty value
    …              Placeholder value

  General
    ?              Show/hide this help
    q / Ctrl+C     Quit
    Esc            Back / cancel
`

	style := lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Padding(1, 2).
		Foreground(a.theme.ColorFg)

	footer := a.theme.MutedItem.Render("\n  Press Esc or ? to close")

	return style.Render(helpText + footer)
}
