package model

import (
	"slices"
	"sort"
)

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
	Path            string
	Name            string // filename (e.g. ".env.local")
	Vars            []EnvVar
	DeletedVars     []EnvVar  // vars removed since last save (for UI display)
	Lines           []RawLine // all original lines for faithful write-back
	Modified        bool
	GitWarning      bool // true if file is NOT covered by .gitignore
	TrailingNewline bool // true if the source file ended with a newline
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

// ReorderMode selects the ordering used by Reorder.
type ReorderMode int

const (
	// ReorderAlphabetical sorts variables A→Z by key.
	ReorderAlphabetical ReorderMode = iota
	// ReorderGrouped sorts by prefix group (groups alphabetical, ungrouped
	// last) and by key within each group, mirroring the visual grouping.
	ReorderGrouped
)

// reorderUnit is a variable together with the comment lines attached directly
// above it (no blank line in between). Attached comments travel with the
// variable when the file is reordered.
type reorderUnit struct {
	varIdx   int       // index into the pre-reorder ef.Vars
	varLine  RawLine   // the variable's original RawLine (Content preserved)
	comments []RawLine // attached comment lines, in original order
}

// orderedUnit pairs a unit with its group id, used to insert a blank line at
// group boundaries in grouped mode.
type orderedUnit struct {
	unit  reorderUnit
	group int
}

// Reorder rewrites the file's variables in the given order, rebuilding Vars and
// Lines so a subsequent write persists the new layout. Rules:
//   - comments directly above a variable (no blank line between) travel with it;
//   - the leading block (before the first variable) and trailing block (after
//     the last variable) are preserved verbatim;
//   - stand-alone comments inside the body float to just above the first
//     variable, so no comment is ever lost;
//   - blank lines are regenerated: one between groups in grouped mode, none in
//     alphabetical mode.
//
// Sets Modified. No-op (leaves Modified untouched) when the file has no
// variables.
func (ef *EnvFile) Reorder(mode ReorderMode) {
	firstVar, lastVar := -1, -1
	for i, l := range ef.Lines {
		if l.Type == LineVariable {
			if firstVar == -1 {
				firstVar = i
			}
			lastVar = i
		}
	}
	if firstVar == -1 {
		return // nothing to reorder
	}

	// The head ends just before the first variable's attached comment block,
	// so those comments stay with the variable rather than the file header.
	headEnd := firstVar
	for headEnd > 0 && ef.Lines[headEnd-1].Type == LineComment {
		headEnd--
	}
	head := ef.Lines[:headEnd]
	tail := ef.Lines[lastVar+1:]

	// Split the body into units (var + attached comments) and floating comments
	// (stand-alone blocks separated from any variable by a blank line).
	var units []reorderUnit
	var floating []RawLine
	var pending []RawLine
	for _, l := range ef.Lines[headEnd : lastVar+1] {
		switch l.Type {
		case LineComment:
			pending = append(pending, l)
		case LineEmpty:
			floating = append(floating, pending...)
			pending = nil
		case LineVariable:
			units = append(units, reorderUnit{
				varIdx:   l.VarIdx,
				varLine:  l,
				comments: pending,
			})
			pending = nil
		}
	}

	ordered := ef.orderUnits(units, mode)

	newVars := make([]EnvVar, 0, len(ef.Vars))
	newLines := make([]RawLine, 0, len(ef.Lines))
	newLines = append(newLines, head...)
	newLines = append(newLines, floating...)

	prevGroup := -1
	for _, o := range ordered {
		if mode == ReorderGrouped && prevGroup != -1 && o.group != prevGroup {
			newLines = append(newLines, RawLine{Type: LineEmpty, VarIdx: -1})
		}
		prevGroup = o.group
		newLines = append(newLines, o.unit.comments...)
		newIdx := len(newVars)
		newVars = append(newVars, ef.Vars[o.unit.varIdx])
		vl := o.unit.varLine
		vl.VarIdx = newIdx
		newLines = append(newLines, vl)
	}
	newLines = append(newLines, tail...)

	ef.Vars = newVars
	ef.Lines = newLines
	ef.Modified = true
}

func (ef *EnvFile) orderUnits(units []reorderUnit, mode ReorderMode) []orderedUnit {
	if mode == ReorderGrouped {
		return ef.orderGrouped(units)
	}
	return ef.orderAlphabetical(units)
}

func (ef *EnvFile) orderAlphabetical(units []reorderUnit) []orderedUnit {
	sorted := slices.Clone(units)
	sort.SliceStable(sorted, func(i, j int) bool {
		return ef.Vars[sorted[i].varIdx].Key < ef.Vars[sorted[j].varIdx].Key
	})
	out := make([]orderedUnit, len(sorted))
	for i, u := range sorted {
		out[i] = orderedUnit{unit: u}
	}
	return out
}

func (ef *EnvFile) orderGrouped(units []reorderUnit) []orderedUnit {
	unitByVar := make(map[int]reorderUnit, len(units))
	for _, u := range units {
		unitByVar[u.varIdx] = u
	}

	groups := ComputeGroups(ef.Vars)
	sort.SliceStable(groups, func(i, j int) bool {
		// Ungrouped pinned last, named groups alphabetical by prefix.
		if groups[i].IsUngrouped() != groups[j].IsUngrouped() {
			return !groups[i].IsUngrouped()
		}
		return groups[i].Prefix < groups[j].Prefix
	})

	var out []orderedUnit
	for gi, g := range groups {
		idxs := slices.Clone(g.Vars)
		sort.SliceStable(idxs, func(a, b int) bool {
			return ef.Vars[idxs[a]].Key < ef.Vars[idxs[b]].Key
		})
		for _, vi := range idxs {
			if u, ok := unitByVar[vi]; ok {
				out = append(out, orderedUnit{unit: u, group: gi})
			}
		}
	}
	return out
}
