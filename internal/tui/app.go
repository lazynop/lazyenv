package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

// envFilePerm is the permission mode for newly created .env files
// (owner-writable, world-readable), matching parser.WriteFile.
const envFilePerm os.FileMode = 0o644

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
	ModeConfirmMatrixDelete
	ModeReorderMenu
	ModeReorderConfirm
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

	// File-operation state (create/duplicate/rename/template). The active
	// AppMode distinguishes the operation; the input and source are shared.
	fileInput    textinput.Model
	fileOpSource *model.EnvFile // source for duplicate/rename/template; nil for create

	// Reorder state: the mode chosen in the menu, applied on confirm.
	reorderMode model.ReorderMode

	// Session statistics: nil when disabled (e.g. read-only or session-summary=false).
	sessionStats *SessionStats
}

// NewApp creates a new App model.
func NewApp(cfg config.Config, warnings []string) App {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	fi := textinput.New()
	fi.CharLimit = 256

	varList := NewVarListModel(cfg.Layout)
	if cfg.Sort == "alphabetical" {
		varList.SortAlpha = true
	}
	varList.Grouping = cfg.Group

	sb := NewStatusBarModel()
	sb.ReadOnly = cfg.ReadOnly

	var stats *SessionStats
	if cfg.SessionSummary && !cfg.ReadOnly {
		stats = NewSessionStats()
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
		statusBar:      sb,
		diffView:       NewDiffViewModel(cfg.Layout, cfg.Secrets),
		editor:         NewEditorModel(),
		searchInput:    ti,
		fileInput:      fi,
		backedUpPaths:  make(map[string]bool),
		sessionStats:   stats,
	}
}

// SessionSummary returns the formatted session summary (empty string if disabled
// or nothing to report).
func (a App) SessionSummary() string {
	return a.sessionStats.Format()
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
		a.diffView.Width = a.width - compareMarginWidth
		a.diffView.Height = a.height - compareBottomChromeHeight
		a.matrixView.Width = a.width
		a.matrixView.Height = a.height - statusBarHeight
		a.statusBar.Width = a.width
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
		for _, f := range msg.Files {
			a.sessionStats.RecordInitialLoad(f.Path, f.Vars)
		}
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
	case ModeConfirmMatrixDelete:
		return a.handleConfirmMatrixDeleteKey(msg)
	case ModeReorderMenu, ModeReorderConfirm:
		return a.handleReorderKey(msg)
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

	// Reorder-on-disk (O): a global mutation handled here so the global-action
	// dispatcher stays within the project's cyclomatic-complexity limit.
	if key.Matches(msg, a.keys.Reorder) {
		return a.startReorder()
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
	// Block mutating file actions in read-only mode
	if a.config.ReadOnly && key.Matches(msg, a.keys.CreateFile, a.keys.DuplicateFile, a.keys.DeleteFile, a.keys.RenameFile, a.keys.TemplateFile) {
		return a, a.readOnlyFlash(), true
	}

	switch {
	case key.Matches(msg, a.keys.CreateFile):
		a.fileInput.SetValue("")
		a.fileInput.Placeholder = ".env.example"
		a.mode = ModeCreateFile
		return a, a.fileInput.Focus(), true
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
		a.fileOpSource = f
		a.fileInput.SetValue(duplicateName(f.Name))
		a.mode = ModeDuplicateFile
		return a, a.fileInput.Focus(), true

	case key.Matches(msg, a.keys.DeleteFile):
		if f.Modified {
			return a, a.flashMessage("Save or reset changes before deleting"), true
		}
		a.mode = ModeConfirmDeleteFile
		a.statusBar.SetMessage("Delete " + f.Name + "? (y/n)")
		return a, nil, true

	case key.Matches(msg, a.keys.RenameFile):
		if f.Modified {
			return a, a.flashMessage("Save or reset changes before renaming"), true
		}
		a.fileOpSource = f
		a.fileInput.SetValue(f.Name)
		a.mode = ModeRenameFile
		return a, a.fileInput.Focus(), true

	case key.Matches(msg, a.keys.TemplateFile):
		a.fileOpSource = f
		a.fileInput.SetValue(templateName(f.Name))
		a.mode = ModeTemplateFile
		return a, a.fileInput.Focus(), true
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
		if cmd := a.readOnlyFlash(); cmd != nil {
			return a, cmd
		}
		return a.handleSave()

	case key.Matches(msg, a.keys.Reset):
		return a.handleReset()

	case key.Matches(msg, a.keys.ToggleSecret):
		a.varList.ShowSecrets = !a.varList.ShowSecrets
		if a.varList.ShowSecrets {
			return a, a.flashMessage("Secrets revealed")
		}
		return a, a.flashMessage("Secrets hidden")

	case key.Matches(msg, a.keys.ToggleSort):
		return a, a.toggleSortFlash()

	case key.Matches(msg, a.keys.ToggleGroup):
		return a, a.toggleGroupingFlash()

	case key.Matches(msg, a.keys.Search):
		a.mode = ModeSearching
		a.searchInput.SetValue("")
		return a, a.searchInput.Focus()

	case key.Matches(msg, a.keys.Matrix):
		if len(a.fileList.Files) < 2 {
			return a, a.flashMessage("Need at least 2 files for matrix")
		}
		a.matrixView = NewMatrixModel(a.fileList.Files, a.config.Layout, a.config.Secrets)
		a.matrixView.Width = a.width
		a.matrixView.Height = a.height - statusBarHeight
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
		return a.handleNormalEnter(), true

	case key.Matches(msg, a.keys.ToggleCollapse) && a.focus == FocusVars && a.varList.IsHeaderAtCursor():
		a.varList.ToggleCollapseAtCursor()
		return a, true
	}

	return a, false
}

func (a App) handleNormalEnter() App {
	switch {
	case a.focus == FocusFiles:
		a.fileList.Select()
		if f := a.fileList.SelectedFile(); f != nil {
			a.varList.SetFile(f)
		}
		a.focus = FocusVars
		a.fileList.Focused = false
		a.varList.Focused = true
	case a.varList.IsHeaderAtCursor():
		a.varList.ToggleCollapseAtCursor()
	}
	return a
}

func (a App) handleNormalVarAction(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if a.focus != FocusVars {
		return a, nil, false
	}

	// Mutating actions (edit, add, delete) — blocked in read-only mode
	if key.Matches(msg, a.keys.Edit, a.keys.EditKey, a.keys.Add, a.keys.Delete) {
		return a.handleNormalVarEdit(msg)
	}

	switch {
	case key.Matches(msg, a.keys.YankValue):
		v := a.varList.SelectedVar()
		if v != nil {
			return a, tea.Batch(
				tea.SetClipboard(v.Value),
				a.flashMessage(fmt.Sprintf("Copied %s value to clipboard", v.Key)),
			), true
		}

	case key.Matches(msg, a.keys.YankLine):
		v := a.varList.SelectedVar()
		if v != nil {
			line := v.Key + "=" + v.Value
			return a, tea.Batch(
				tea.SetClipboard(line),
				a.flashMessage(fmt.Sprintf("Copied %s to clipboard", v.Key+"=...")),
			), true
		}

	case key.Matches(msg, a.keys.Peek):
		a.varList.Peeking = !a.varList.Peeking
		return a, nil, true
	}

	return a, nil, false
}

func (a App) handleNormalVarEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd, true
	}
	if a.varList.IsHeaderAtCursor() {
		return a, a.flashMessage("No variable selected"), true
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

	case key.Matches(msg, a.keys.EditKey):
		v := a.varList.SelectedVar()
		if v != nil {
			idx := a.varList.SelectedVarIndex()
			a.editor.StartEditKey(v, idx)
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
	}

	return a, nil, false
}

func (a App) handleConfirmDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Confirm):
		a.mode = ModeNormal
		idx := a.varList.SelectedVarIndex()
		if idx >= 0 {
			name := a.varList.File.Vars[idx].Key
			a.varList.File.DeleteVar(idx)
			a.varList.Refresh()
			return a, a.flashMessage("Deleted " + name)
		}
		return a, nil
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
	case ModeMatrix, ModeConfirmMatrixDelete:
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
		return bar.Render("  New file: " + a.fileInput.View())
	case ModeDuplicateFile:
		return bar.Render("  Duplicate as: " + a.fileInput.View())
	case ModeRenameFile:
		return bar.Render("  Rename: " + a.fileInput.View())
	case ModeTemplateFile:
		return bar.Render("  Template as: " + a.fileInput.View())
	}
	return ""
}

func (a *App) updateLayout() {
	a.fileWidth = max(a.width/4, config.FileListMinWidth)
	if a.config.Layout.FileListWidth > 0 {
		a.fileWidth = a.config.Layout.FileListWidth
	}
	varWidth := a.width - a.fileWidth

	panelHeight := a.height - bottomChromeHeight

	a.fileList.Width = a.fileWidth
	a.fileList.Height = panelHeight
	a.fileList.Focused = a.focus == FocusFiles

	a.varList.Width = varWidth
	a.varList.Height = panelHeight
	a.varList.Focused = a.focus == FocusVars

	a.statusBar.Width = a.width
}
