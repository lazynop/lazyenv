package tui

import (
	"fmt"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/parser"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

// AppConfig holds startup configuration.
type AppConfig struct {
	Dir       string
	Recursive bool
	ShowAll   bool
}

// App is the main Bubble Tea model.
type App struct {
	config    AppConfig
	keys      KeyMap
	theme     Theme
	hasDarkBg bool
	focus     Focus
	mode      AppMode
	width     int
	height    int
	ready     bool

	fileList    FileListModel
	varList     VarListModel
	statusBar   StatusBarModel
	diffView    DiffViewModel
	editor      EditorModel
	searchInput textinput.Model

	// Compare mode state
	compareFirstFile  *model.EnvFile
	compareEditFile   *model.EnvFile // which file is being edited in compare mode
	compareEditVarIdx int            // var index in that file
}

// NewApp creates a new App model.
func NewApp(config AppConfig) App {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100

	return App{
		config:      config,
		keys:        DefaultKeyMap(),
		theme:       BuildTheme(true), // default to dark, will update on BackgroundColorMsg
		hasDarkBg:   true,
		focus:       FocusFiles,
		mode:        ModeNormal,
		fileList:    NewFileListModel(),
		varList:     NewVarListModel(),
		statusBar:   NewStatusBarModel(),
		diffView:    NewDiffViewModel(),
		editor:      NewEditorModel(),
		searchInput: ti,
	}
}

func (a App) Init() tea.Cmd {
	return func() tea.Msg {
		files, err := ScanDir(a.config.Dir, a.config.Recursive)
		return FilesLoadedMsg{Files: files, Err: err}
	}
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
		return a, nil

	case tea.BackgroundColorMsg:
		a.hasDarkBg = msg.IsDark()
		a.theme = BuildTheme(a.hasDarkBg)
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

	case key.Matches(msg, a.keys.ToggleSecret):
		a.varList.ShowSecrets = !a.varList.ShowSecrets
		if a.varList.ShowSecrets {
			a.statusBar.SetMessage("Secrets revealed")
		} else {
			a.statusBar.SetMessage("Secrets hidden")
		}
		return a, clearMessageAfter(2 * time.Second)

	case key.Matches(msg, a.keys.ToggleSort):
		a.varList.ToggleSort()
		if a.varList.SortAlpha {
			a.statusBar.SetMessage("Sorted alphabetically")
		} else {
			a.statusBar.SetMessage("Sorted by position")
		}
		return a, clearMessageAfter(2 * time.Second)

	case key.Matches(msg, a.keys.Search):
		a.mode = ModeSearching
		a.searchInput.SetValue("")
		return a, a.searchInput.Focus()

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
		return a, clearMessageAfter(2 * time.Second)
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
		return a, clearMessageAfter(2 * time.Second)
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
	case key.Matches(msg, a.keys.Up):
		a.diffView.MoveUp()
	case key.Matches(msg, a.keys.Down):
		a.diffView.MoveDown()
	case key.Matches(msg, a.keys.Right):
		if k := a.diffView.CopyToRight(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileB.Name)
			return a, clearMessageAfter(2 * time.Second)
		}
	case key.Matches(msg, a.keys.Left):
		if k := a.diffView.CopyToLeft(); k != "" {
			a.statusBar.SetMessage(k + " → " + a.diffView.FileA.Name)
			return a, clearMessageAfter(2 * time.Second)
		}
	case key.Matches(msg, a.keys.Filter):
		a.diffView.ToggleFilter()
		if a.diffView.HideEqual {
			a.statusBar.SetMessage("Showing differences only")
		} else {
			a.statusBar.SetMessage("Showing all entries")
		}
		return a, clearMessageAfter(2 * time.Second)
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
				}
				if f.Path == a.diffView.FileB.Path {
					a.fileList.Files[i] = a.diffView.FileB
				}
			}
			a.statusBar.SetMessage("Reset to saved state")
		}
		return a, clearMessageAfter(2 * time.Second)
	}
	return a, nil
}

func (a App) handleCompareSave() (App, tea.Cmd) {
	saved := []string{}
	for _, f := range []*model.EnvFile{a.diffView.FileA, a.diffView.FileB} {
		if f != nil && f.Modified {
			if err := parser.WriteFile(f); err != nil {
				a.statusBar.SetMessage("Error saving " + f.Name + ": " + err.Error())
				return a, clearMessageAfter(3 * time.Second)
			}
			// Re-parse to refresh RawLines
			refreshed, err := parser.ParseFile(f.Path)
			if err == nil {
				for i, existing := range a.fileList.Files {
					if existing.Path == f.Path {
						a.fileList.Files[i] = refreshed
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
		a.statusBar.SetMessage("Saved " + strings.Join(saved, ", "))
	}
	// Recompute diff after save
	a.diffView.allEntries = model.ComputeDiff(a.diffView.FileA, a.diffView.FileB)
	a.diffView.recompute()
	return a, clearMessageAfter(2 * time.Second)
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
		return a, clearMessageAfter(2 * time.Second)
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
		return a, clearMessageAfter(2 * time.Second)
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

func (a App) handleSave() (App, tea.Cmd) {
	f := a.varList.File
	if f == nil {
		a.statusBar.SetMessage("No file selected")
		return a, clearMessageAfter(2 * time.Second)
	}
	if !f.Modified {
		a.statusBar.SetMessage("No changes to save")
		return a, clearMessageAfter(2 * time.Second)
	}

	if err := parser.WriteFile(f); err != nil {
		a.statusBar.SetMessage("Error saving: " + err.Error())
		return a, clearMessageAfter(3 * time.Second)
	}

	// Re-parse to refresh RawLines
	refreshed, err := parser.ParseFile(f.Path)
	if err != nil {
		a.statusBar.SetMessage("Saved but refresh failed: " + err.Error())
		return a, clearMessageAfter(3 * time.Second)
	}

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

	a.statusBar.SetMessage("Saved " + f.Name)
	return a, clearMessageAfter(2 * time.Second)
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
	fileWidth := a.width / 4
	if fileWidth < 20 {
		fileWidth = 20
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
    w              Save changes
    c              Compare two files (diff view)
    /              Search variables
    o              Toggle sort (position / alphabetical)
    Ctrl+S         Toggle secret masking

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
