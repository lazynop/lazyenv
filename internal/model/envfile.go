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
	// OriginalIsSecret is IsSecret as it was for OriginalValue, so the
	// original stays masked after the var stops being a secret.
	OriginalIsSecret bool
	// revival is set while the var holds a key it revived (see reviveDeleted).
	revival *revival
}

// revival is a variable deleted this session whose key another variable took,
// with what that variable had before taking it, so renaming or deleting the
// variable can undo it.
type revival struct {
	deleted          EnvVar
	index            int // where deleted sat in DeletedVars
	originalValue    string
	originalIsSecret bool
	isNew            bool
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

// UpdateVar updates the value of a variable by index. isSecret is whether the
// variable is a secret with its new value: the caller runs the detection,
// which needs the secrets config.
func (ef *EnvFile) UpdateVar(idx int, newValue string, isSecret bool) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	ef.markModified(idx)
	ef.Vars[idx].Value = newValue
	ef.Vars[idx].IsEmpty = newValue == ""
	ef.Vars[idx].IsSecret = isSecret
}

// RenameVar updates the key of a variable by index. isSecret is whether the
// variable is a secret with its new key, as for UpdateVar. Renaming a variable
// onto a key deleted this session revives the key (see reviveDeleted), and
// renaming it off that key lists the key as deleted again (see undoRevival).
func (ef *EnvFile) RenameVar(idx int, newKey string, isSecret bool) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	ef.markModified(idx)
	ef.undoRevival(idx)
	ef.Vars[idx].Key = newKey
	ef.Vars[idx].IsSecret = isSecret
	ef.reviveDeleted(idx)
}

// markModified flags the variable at idx and the file as changed, saving the
// variable's value before its first change as its OriginalValue.
func (ef *EnvFile) markModified(idx int) {
	v := &ef.Vars[idx]
	v.OriginalValue, v.OriginalIsSecret = v.original()
	v.Modified = true
	ef.Modified = true
}

// original returns the value v had before its first change this session,
// and whether that value was a secret.
func (v EnvVar) original() (string, bool) {
	if v.Modified {
		return v.OriginalValue, v.OriginalIsSecret
	}
	return v.Value, v.IsSecret
}

// reviveDeleted makes the variable at idx, whose key was just set, take the
// place of a variable deleted this session under that key. The deleted one
// leaves DeletedVars. The variable takes its original value and secret status,
// which peek shows, and is no longer new, as the deleted one was not: DeleteVar
// lists no new variable.
func (ef *EnvFile) reviveDeleted(idx int) {
	v := &ef.Vars[idx]
	i := slices.IndexFunc(ef.DeletedVars, func(d EnvVar) bool { return d.Key == v.Key })
	if i < 0 {
		return
	}
	d := ef.DeletedVars[i]
	v.revival = &revival{deleted: d, index: i, originalValue: v.OriginalValue, originalIsSecret: v.OriginalIsSecret, isNew: v.IsNew}
	v.OriginalValue, v.OriginalIsSecret = d.original()
	v.IsNew = false
	ef.DeletedVars = slices.Delete(ef.DeletedVars, i, i+1)
}

// undoRevival reverses reviveDeleted for the variable at idx, before a rename
// or a delete takes it off the key it revived: the deleted variable is listed
// again where it was, and the variable gets back its original and whether it
// is new.
func (ef *EnvFile) undoRevival(idx int) {
	v := &ef.Vars[idx]
	r := v.revival
	if r == nil {
		return
	}
	ef.DeletedVars = slices.Insert(ef.DeletedVars, min(r.index, len(ef.DeletedVars)), r.deleted)
	v.OriginalValue, v.OriginalIsSecret, v.IsNew = r.originalValue, r.originalIsSecret, r.isNew
	v.revival = nil
}

// AddVar appends a new variable to the file. A key deleted this session is
// revived (see reviveDeleted).
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
	ef.reviveDeleted(varIdx)
	ef.Modified = true
}

// DeleteVar removes a variable by index. Deleting a variable that revived a
// key lists the key's deleted variable again (see undoRevival).
func (ef *EnvFile) DeleteVar(idx int) {
	if idx < 0 || idx >= len(ef.Vars) {
		return
	}
	ef.undoRevival(idx)
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

// LastIndexOfKey returns the index in Vars of the last variable with the
// given key (shell semantics), or -1 when there is none.
func (ef *EnvFile) LastIndexOfKey(key string) int {
	for i := len(ef.Vars) - 1; i >= 0; i-- {
		if ef.Vars[i].Key == key {
			return i
		}
	}
	return -1
}

// VarByKey returns the last variable with the given key (shell semantics).
func (ef *EnvFile) VarByKey(key string) *EnvVar {
	if i := ef.LastIndexOfKey(key); i >= 0 {
		return &ef.Vars[i]
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
