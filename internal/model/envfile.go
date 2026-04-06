package model

// LineType categorizes a raw line in an env file.
type LineType int

const (
	LineVariable LineType = iota
	LineComment
	LineEmpty
)

// QuoteStyle tracks the original quoting for round-trip fidelity.
type QuoteStyle int

const (
	QuoteNone QuoteStyle = iota
	QuoteSingle
	QuoteDouble
)

// EnvVar represents a single environment variable.
type EnvVar struct {
	Key           string
	Value         string
	Comment       string // inline comment (after value)
	LineNum       int
	QuoteStyle    QuoteStyle
	HasExport     bool
	IsSecret      bool
	IsEmpty       bool
	IsPlaceholder bool
	IsDuplicate   bool
	Modified      bool
	IsNew         bool
	OriginalValue string // value before first edit (empty for new vars)
}

// EnvFile represents a parsed .env file.
type EnvFile struct {
	Path        string
	Name        string // filename (e.g. ".env.local")
	Vars        []EnvVar
	DeletedVars []EnvVar  // vars removed since last save (for UI display)
	Lines       []RawLine // all original lines for faithful write-back
	Modified    bool
	GitWarning  bool // true if file is NOT covered by .gitignore
}

// RawLine preserves original content for round-trip writing.
type RawLine struct {
	Type    LineType
	Content string // original line text
	VarIdx  int    // index into Vars if Type == LineVariable, else -1
}

// UpdateVar updates the value of a variable by index.
func (ef *EnvFile) UpdateVar(idx int, newValue string) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	if !ef.Vars[idx].Modified {
		ef.Vars[idx].OriginalValue = ef.Vars[idx].Value
	}
	ef.Vars[idx].Value = newValue
	ef.Vars[idx].Modified = true
	ef.Vars[idx].IsEmpty = newValue == ""
	ef.Modified = true
}

// RenameVar updates the key of a variable by index.
func (ef *EnvFile) RenameVar(idx int, newKey string) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	ef.Vars[idx].Key = newKey
	ef.Vars[idx].Modified = true
	ef.Modified = true
}

// AddVar appends a new variable to the file.
func (ef *EnvFile) AddVar(key, value string, isSecret bool) {
	v := EnvVar{
		Key:        key,
		Value:      value,
		LineNum:    len(ef.Lines) + 1,
		QuoteStyle: QuoteNone,
		IsSecret:   isSecret,
		IsEmpty:    value == "",
		Modified:   true,
		IsNew:      true,
	}
	varIdx := len(ef.Vars)
	ef.Vars = append(ef.Vars, v)
	ef.Lines = append(ef.Lines, RawLine{
		Type:    LineVariable,
		Content: key + "=" + value,
		VarIdx:  varIdx,
	})
	// Remove from DeletedVars if re-adding a previously deleted key,
	// and preserve the original value for peek.
	for i, d := range ef.DeletedVars {
		if d.Key == key {
			last := &ef.Vars[varIdx]
			last.OriginalValue = d.Value
			last.IsNew = false // was deleted then re-added: treat as modified
			ef.DeletedVars = append(ef.DeletedVars[:i], ef.DeletedVars[i+1:]...)
			break
		}
	}
	ef.Modified = true
}

// DeleteVar removes a variable by index.
func (ef *EnvFile) DeleteVar(idx int) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	// Track deleted var for UI display (skip if it was newly added this session).
	if !ef.Vars[idx].IsNew {
		ef.DeletedVars = append(ef.DeletedVars, ef.Vars[idx])
	}
	// Find and remove the corresponding RawLine
	for i, line := range ef.Lines {
		if line.Type == LineVariable && line.VarIdx == idx {
			ef.Lines = append(ef.Lines[:i], ef.Lines[i+1:]...)
			break
		}
	}
	// Fix VarIdx references for lines after the deleted variable
	for i := range ef.Lines {
		if ef.Lines[i].Type == LineVariable && ef.Lines[i].VarIdx > idx {
			ef.Lines[i].VarIdx--
		}
	}
	ef.Vars = append(ef.Vars[:idx], ef.Vars[idx+1:]...)
	ef.Modified = true
}

// VarByKey returns the last variable with the given key (shell semantics).
func (ef *EnvFile) VarByKey(key string) *EnvVar {
	for i := len(ef.Vars) - 1; i >= 0; i-- {
		if ef.Vars[i].Key == key {
			return &ef.Vars[i]
		}
	}
	return nil
}
