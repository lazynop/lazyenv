package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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

// ClearMessageMsg clears the status bar message, but only if it is still the
// message that scheduled the clear (matched by generation). This prevents a
// stale auto-clear timer from wiping a newer message or a persistent prompt.
type ClearMessageMsg struct{ gen int }

// StartupNoticeMsg is sent at startup when an input was ignored or adjusted
// but the run can go on (e.g. an unknown --theme).
type StartupNoticeMsg struct{ Notice string }

// StartupMessages are the problems found while building the config, before
// the TUI starts.
type StartupMessages struct {
	// ConfigErrors are the reasons the config file was rejected and the
	// defaults are in use. The App shows them on a blocking screen and quits
	// on the next key.
	ConfigErrors []string
	// Notices report input that was ignored or adjusted. The App flashes them
	// in the status bar and the run goes on.
	Notices []string
}

// App is the main Bubble Tea model.
type App struct {
	config    config.Config
	keys      KeyMap
	theme     Theme
	hasDarkBg bool
	focus     Focus
	mode      AppMode
	width     int
	height    int
	fileWidth int
	ready     bool

	fileList    FileListModel
	varList     VarListModel
	statusBar   StatusBarModel
	diffView    DiffViewModel
	helpView    HelpViewModel
	editor      EditorModel
	searchInput textinput.Model
	matrixView  MatrixModel

	// Text of the blocking config error screen (ModeConfigError).
	configError string

	// Startup notice, flashed once Init runs; empty when there is none.
	startupNotice string

	// Backup state: tracks which files have been backed up this session
	backedUpPaths map[string]bool

	// Compare mode state
	compareFirstFile *model.EnvFile
	compareEditFile  *model.EnvFile // which file is being edited in compare mode

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
func NewApp(cfg config.Config, startup StartupMessages) App {
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

	app := App{
		config:        cfg,
		keys:          DefaultKeyMap(),
		theme:         BuildTheme(true, cfg.Colors), // default to dark, will update on BackgroundColorMsg
		hasDarkBg:     true,
		mode:          ModeNormal,
		fileList:      NewFileListModel(),
		varList:       varList,
		statusBar:     sb,
		diffView:      NewDiffViewModel(cfg.Layout, cfg.Secrets),
		matrixView:    NewMatrixModel(nil, cfg.Layout, cfg.Secrets),
		editor:        NewEditorModel(),
		searchInput:   ti,
		fileInput:     fi,
		backedUpPaths: make(map[string]bool),
		sessionStats:  stats,
	}
	app.setFocus(FocusFiles)
	if len(startup.ConfigErrors) > 0 {
		// The run stops on this screen, so the notices go on it too.
		app.configError = strings.Join(slices.Concat(startup.ConfigErrors, startup.Notices), "; ")
		app.mode = ModeConfigError
	} else {
		app.startupNotice = strings.Join(startup.Notices, "; ")
	}
	return app
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
	if notice := a.startupNotice; notice != "" {
		cmds = append(cmds, func() tea.Msg {
			return StartupNoticeMsg{Notice: notice}
		})
	}
	return tea.Batch(cmds...)
}

// Update handles msg, then lays the views out for the terminal size and fits
// the text input of the current mode's input bar to it, so View draws the
// state Update leaves. Input arriving while the terminal is below the minimum
// size goes to tooSmallInput instead.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, cmd := tooSmallInput(a.width, a.height, msg); handled {
		return a, cmd
	}

	updated, cmd := a.update(msg)
	if app, ok := updated.(App); ok {
		app.updateLayout()
		app.fitInputBar()
		return app, cmd
	}
	return updated, cmd
}

func (a App) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		return a, nil

	case tea.BackgroundColorMsg:
		a.hasDarkBg = msg.IsDark()
		a.theme = BuildTheme(a.hasDarkBg, a.config.Colors)
		return a, nil

	case StartupNoticeMsg:
		return a, a.flashError(msg.Notice)

	case FilesLoadedMsg:
		if msg.Err != nil {
			a.statusBar.SetAlert("Error: " + msg.Err.Error())
			return a, nil
		}
		a.fileList.SetFiles(msg.Files)
		for _, f := range msg.Files {
			a.sessionStats.RecordInitialLoad(f.Path, f.Vars)
		}
		if len(msg.Files) > 0 {
			a.varList.SetFile(msg.Files[0])
			a.varList.SetShowSecrets(a.config.ShowAll)
		}
		return a, nil

	case ClearMessageMsg:
		if msg.gen == a.statusBar.msgGen {
			a.statusBar.ClearMessage()
		}
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
		a.helpView.ScrollToTop()
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
			return a, a.flashWarning("Save or reset changes before deleting"), true
		}
		a.mode = ModeConfirmDeleteFile
		a.statusBar.SetAlert("Delete " + f.Name + "? (y/n)")
		return a, nil, true

	case key.Matches(msg, a.keys.RenameFile):
		if f.Modified {
			return a, a.flashWarning("Save or reset changes before renaming"), true
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
			a.setFocus(FocusFiles)
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
		a.varList.SetShowSecrets(!a.varList.ShowSecrets)
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
			return a, a.flashWarning("Need at least 2 files for matrix")
		}
		a.matrixView = NewMatrixModel(a.fileList.Files, a.config.Layout, a.config.Secrets)
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
		a.setFocus(FocusFiles)
		return a, true

	case key.Matches(msg, a.keys.Right):
		a.setFocus(FocusVars)
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
		a.setFocus(FocusVars)
	case a.varList.IsHeaderAtCursor():
		a.varList.ToggleCollapseAtCursor()
	}
	return a
}

// setFocus moves the focus to the files or vars panel and tells each panel
// whether it holds it.
func (a *App) setFocus(f Focus) {
	a.focus = f
	a.fileList.Focused = f == FocusFiles
	a.varList.SetFocused(f == FocusVars)
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
		a.varList.TogglePeek()
		return a, nil, true
	}

	return a, nil, false
}

func (a App) handleNormalVarEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if cmd := a.readOnlyFlash(); cmd != nil {
		return a, cmd, true
	}
	if a.varList.IsHeaderAtCursor() {
		return a, a.flashWarning("No variable selected"), true
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
	switch {
	case key.Matches(msg, a.keys.Escape, a.keys.Help, a.keys.Quit):
		a.mode = ModeNormal
	case key.Matches(msg, a.keys.Up):
		a.helpView.ScrollUp()
	case key.Matches(msg, a.keys.Down):
		a.helpView.ScrollDown()
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
	if terminalTooSmall(a.width, a.height) {
		content = tooSmallView(a.width, a.height, a.theme.PanelTitle, a.theme.MutedItem)
	} else {
		content = a.screen()
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.BackgroundColor = a.theme.ColorBg
	if !a.config.NoMouse {
		view.MouseMode = tea.MouseModeCellMotion
	}
	return view
}

// screen renders the current mode's screen.
func (a App) screen() string {
	switch a.mode {
	case ModeConfigError:
		return a.viewConfigError()
	case ModeHelp:
		return a.helpView.View(a.theme)
	case ModeComparing:
		return a.withBottom(a.diffView.View(a.theme), a.statusBar.View(a.theme, a.mode, a.focus, "", 0, a.diffView.Stats))
	case ModeEditingCompare:
		return a.withBottom(a.diffView.View(a.theme), a.statusBar.View(a.theme, ModeEditing, a.focus, "", 0))
	case ModeMatrix, ModeConfirmMatrixDelete:
		return a.withBottom(a.matrixView.View(a.theme), a.statusBar.View(a.theme, a.mode, a.focus, "", 0))
	case ModeMatrixEditing:
		return a.withBottom(a.matrixView.View(a.theme), a.statusBar.View(a.theme, ModeEditing, a.focus, "", 0))
	default:
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

		return a.withBottom(panels, a.statusBar.View(a.theme, a.mode, a.focus, fileName, varCount))
	}
}

// withBottom stacks body over the current mode's input bar, if any, and
// status, the status bar, in the bottomChromeHeight rows every view reserves
// below its body: status is cut to the rows the input bar leaves.
func (a App) withBottom(body, status string) string {
	parts := []string{body}
	statusRows := bottomChromeHeight
	if bar := a.inputBarView(); bar != "" {
		parts = append(parts, bar)
		statusRows -= lipgloss.Height(bar)
	}
	if rows := strings.Split(status, "\n"); len(rows) > statusRows {
		status = strings.Join(rows[:statusRows], "\n")
	}
	return lipgloss.JoinVertical(lipgloss.Left, append(parts, status)...)
}

// inputBar returns the label and the text input of the current mode's input
// bar, or a nil input in a mode without one.
func (a *App) inputBar() (string, *textinput.Model) {
	switch a.mode {
	case ModeSearching:
		return "  / ", &a.searchInput
	case ModeEditing, ModeEditingCompare:
		return "  " + a.editor.label, &a.editor.input
	case ModeMatrixEditing:
		return a.matrixView.EditBar()
	}
	if label, ok := fileInputLabels[a.mode]; ok {
		return label, a.activeFileInput()
	}
	return "", nil
}

// fileInputLabels labels the input bar of each file-operation mode, the modes
// activeFileInput serves.
var fileInputLabels = map[AppMode]string{
	ModeCreateFile:    "  New file: ",
	ModeDuplicateFile: "  Duplicate as: ",
	ModeRenameFile:    "  Rename: ",
	ModeTemplateFile:  "  Template as: ",
}

// minInputWidth is the fewest columns an input bar keeps for the value of its
// text input: a longer label is cut to leave them.
const minInputWidth = 10

// layoutInputBar returns label as an input bar one terminal row wide draws it,
// cut when it would leave in fewer than minInputWidth columns, and the width
// left for in's value.
func (a App) layoutInputBar(label string, in *textinput.Model) (string, int) {
	room := a.inputBarRoom()
	inputChrome := lipgloss.Width(in.Prompt) + inputCursorWidth
	label = truncate(label, max(room-inputChrome-minInputWidth, 0))
	return label, max(room-lipgloss.Width(label)-inputChrome, 1)
}

// fitInputBar gives the current mode's text input the width its bar leaves
// it, so the input scrolls its value horizontally, keeping the cursor in view,
// instead of wrapping the bar.
func (a *App) fitInputBar() {
	label, in := a.inputBar()
	if in == nil {
		return
	}
	_, width := a.layoutInputBar(label, in)
	if in.Width() == width {
		return
	}
	in.SetWidth(width)
	// textinput fits the scrolled part of its value to its width only when
	// the value or the cursor moves: moving to the end and back does it.
	pos := in.Position()
	in.CursorEnd()
	in.SetCursor(pos)
}

// inputBarView returns the current mode's input bar, or "" in a mode without
// one.
func (a App) inputBarView() string {
	label, in := a.inputBar()
	if in == nil {
		return ""
	}
	label, _ = a.layoutInputBar(label, in)
	// textinput measures runes with go-runewidth, which counts some emoji
	// (those with a variation selector) one column narrower than drawn: cut
	// what overflows so the bar keeps its one row.
	return a.theme.StatusBar.Width(a.width).Render(ansi.Truncate(label+in.View(), a.inputBarRoom(), ""))
}

// inputBarRoom is the columns inside an input bar as wide as the terminal.
func (a App) inputBarRoom() int {
	return a.width - a.theme.StatusBar.GetHorizontalFrameSize()
}

func (a *App) updateLayout() {
	a.fileWidth = max(a.width/4, config.FileListMinWidth)
	if w := a.config.Layout.FileListWidth; w > 0 {
		a.fileWidth = max(w, config.FileListMinWidth)
	}
	varWidth := a.width - a.fileWidth

	panelHeight := a.height - bottomChromeHeight

	// Each view scrolls its cursor back into its new size.
	a.fileList.Width = a.fileWidth
	a.fileList.Height = panelHeight
	a.fileList.ensureCursorVisible()

	a.varList.Width = varWidth
	a.varList.Height = panelHeight
	a.varList.ensureCursorVisible()

	a.diffView.Width = a.width - compareMarginWidth
	a.diffView.Height = panelHeight
	a.diffView.ensureVisible()

	a.matrixView.Width = a.width
	a.matrixView.Height = panelHeight
	a.matrixView.ensureVisible()
	a.matrixView.ensureHorizontalVisible()

	a.helpView.Width = a.width
	a.helpView.Height = a.height
	a.helpView.ensureVisible()

	a.statusBar.Width = a.width
}
