package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
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

	// Config error shown as blocking alert
	configError string

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

	// Global actions (Compare/Save/Reset/Toggle/Search/Matrix)
	return a.handleNormalGlobalAction(msg)
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
		a.matrixView = NewMatrixModel(a.fileList.Files, a.config.Layout)
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
	view.BackgroundColor = a.theme.ColorBg
	return view
}

func (a *App) updateLayout() {
	fileWidth := max(a.width/4, 20)
	if a.config.Layout.FileListWidth > 0 {
		fileWidth = a.config.Layout.FileListWidth
	}
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
