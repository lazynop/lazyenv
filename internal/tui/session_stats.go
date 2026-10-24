package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/lazynop/lazyenv/internal/model"
)

type createKind int

const (
	createScratch createKind = iota
	createDuplicate
	createTemplate
)

type createOrigin struct {
	kind     createKind
	source   string
	baseline *fileSnapshot
}

// SessionStats tracks disk-level changes during a lazyenv session.
// All maps are keyed by absolute file path.
type SessionStats struct {
	initial map[string]*fileSnapshot // path → vars at session start
	final   map[string]*fileSnapshot // path → vars after latest save; nil value = deleted
	renames map[string]string        // currentPath → originalPath (chain collapsed)
	created map[string]createOrigin  // newPath → origin info
}

// NewSessionStats returns an empty stats tracker.
func NewSessionStats() *SessionStats {
	return &SessionStats{
		initial: map[string]*fileSnapshot{},
		final:   map[string]*fileSnapshot{},
		renames: map[string]string{},
		created: map[string]createOrigin{},
	}
}

// fileSnapshot is a file's variables as a shell sees them when sourcing it:
// the last occurrence of a duplicate key wins, both for the value and for the
// key's position in order.
type fileSnapshot struct {
	values map[string]string // key → value
	order  []string          // keys in file order
}

func snapshot(vars []model.EnvVar) *fileSnapshot {
	values := make(map[string]string, len(vars))
	var order []string
	for i := len(vars) - 1; i >= 0; i-- {
		if _, seen := values[vars[i].Key]; !seen {
			values[vars[i].Key] = vars[i].Value
			order = append(order, vars[i].Key)
		}
	}
	slices.Reverse(order)
	return &fileSnapshot{values: values, order: order}
}

// RecordInitialLoad is idempotent: only the first call for a given path wins,
// so the baseline always reflects the true session-start state.
func (s *SessionStats) RecordInitialLoad(path string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	if _, ok := s.initial[path]; ok {
		return
	}
	s.initial[path] = snapshot(vars)
}

func (s *SessionStats) RecordSave(path string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	s.final[path] = snapshot(vars)
}

func (s *SessionStats) RecordCreateScratch(path string, vars []model.EnvVar) {
	s.recordCreate(path, createScratch, "", vars)
}

func (s *SessionStats) RecordCreateTemplate(path, source string, vars []model.EnvVar) {
	s.recordCreate(path, createTemplate, source, vars)
}

func (s *SessionStats) RecordCreateDuplicate(path, source string, vars []model.EnvVar) {
	s.recordCreate(path, createDuplicate, source, vars)
}

func (s *SessionStats) recordCreate(path string, kind createKind, source string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	snap := snapshot(vars)
	origin := createOrigin{kind: kind, source: source}
	// baseline is only consulted for createDuplicate (diff against post-create edits).
	// Storing it for scratch/template would be dead weight.
	if kind == createDuplicate {
		origin.baseline = snap
	}
	s.created[path] = origin
	s.final[path] = snap
}

// pluralVars words how many variables f holds.
func pluralVars(f *fileSnapshot) string {
	n := len(f.values)
	if n == 1 {
		return "1 variable"
	}
	return fmt.Sprintf("%d variables", n)
}

// RecordDelete cancels a same-session create (net-zero) rather than recording
// a delete for paths that didn't exist at session start.
func (s *SessionStats) RecordDelete(path string) {
	if s == nil {
		return
	}
	if _, wasCreated := s.created[path]; wasCreated {
		delete(s.created, path)
		delete(s.final, path)
		return
	}
	s.final[path] = nil
}

// RecordRename collapses chains (a→b→c is recorded as c from a) and detects
// rename-back-to-origin (a→b→a leaves no trace). The renamed path's final
// state is seeded from the previous final (if any) or from the origin's
// initial snapshot, so the rename is reported even without a subsequent save.
func (s *SessionStats) RecordRename(oldPath, newPath string) {
	if s == nil {
		return
	}
	origin := oldPath
	if prev, ok := s.renames[oldPath]; ok {
		origin = prev
		delete(s.renames, oldPath)
	}
	if origin != newPath {
		s.renames[newPath] = origin
	}
	switch {
	case s.final[oldPath] != nil:
		s.final[newPath] = s.final[oldPath]
		delete(s.final, oldPath)
	case s.initial[origin] != nil:
		// No prior save on oldPath: on-disk content at newPath equals the
		// baseline at origin (os.Rename moves bytes unchanged).
		s.final[newPath] = s.initial[origin]
	}
	if co, ok := s.created[oldPath]; ok {
		s.created[newPath] = co
		delete(s.created, oldPath)
	}
}

func diff(base, target map[string]string) (added, changed, deleted int) {
	for k, vt := range target {
		vb, ok := base[k]
		if !ok {
			added++
			continue
		}
		if vb != vt {
			changed++
		}
	}
	for k := range base {
		if _, ok := target[k]; !ok {
			deleted++
		}
	}
	return
}

// Summary returns one line per file with disk-level changes, alphabetically sorted.
// nil when there is nothing to report.
func (s *SessionStats) Summary() []string {
	if s == nil {
		return nil
	}

	type row struct {
		key, line string
	}
	var rows []row

	for path, content := range s.final {
		if line, ok := s.formatLiveRow(path, content); ok {
			rows = append(rows, row{path, line})
		}
	}
	for path, content := range s.final {
		if content != nil {
			continue
		}
		target, line, ok := s.formatDeletedRow(path)
		if ok {
			rows = append(rows, row{target, line})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.line
	}
	return out
}

func (s *SessionStats) formatLiveRow(path string, content *fileSnapshot) (string, bool) {
	if content == nil {
		return "", false
	}
	if co, ok := s.created[path]; ok {
		return s.formatCreatedRow(path, co, content), true
	}
	if origin, ok := s.renames[path]; ok {
		desc, _ := describeChanges(s.initial[origin], content)
		return fmt.Sprintf("%s (renamed from %s) — %s", path, origin, desc), true
	}
	base, ok := s.initial[path]
	if !ok {
		return "", false
	}
	desc, changed := describeChanges(base, content)
	if !changed {
		return "", false
	}
	return fmt.Sprintf("%s — %s", path, desc), true
}

func (s *SessionStats) formatDeletedRow(path string) (string, string, bool) {
	target := path
	if origin, ok := s.renames[path]; ok {
		target = origin
	}
	if _, hadInitial := s.initial[target]; !hadInitial {
		return "", "", false
	}
	return target, fmt.Sprintf("%s — deleted", target), true
}

func (s *SessionStats) formatCreatedRow(path string, co createOrigin, content *fileSnapshot) string {
	switch co.kind {
	case createScratch:
		return fmt.Sprintf("%s — new file (%s)", path, pluralVars(content))
	case createTemplate:
		return fmt.Sprintf("%s — from template %s (%s)", path, co.source, pluralVars(content))
	case createDuplicate:
		desc, changed := describeChanges(co.baseline, content)
		if !changed {
			return fmt.Sprintf("%s — duplicated from %s (%s)", path, co.source, pluralVars(content))
		}
		return fmt.Sprintf("%s — duplicated from %s, %s", path, co.source, desc)
	}
	return ""
}

func formatCounts(added, changed, deleted int) string {
	return fmt.Sprintf("%d added, %d changed, %d deleted", added, changed, deleted)
}

// describeChanges words how target differs from base, and reports whether it
// differs at all; with no difference the words are the three zero counts. A
// nil base is a file with no variables.
func describeChanges(base, target *fileSnapshot) (string, bool) {
	if base == nil {
		base = snapshot(nil)
	}
	a, c, d := diff(base.values, target.values)
	edited := a+c+d > 0
	moved := reordered(base, target)
	switch {
	case moved && !edited:
		return "reordered", true
	case moved:
		return formatCounts(a, c, d) + ", reordered", true
	}
	return formatCounts(a, c, d), edited
}

// reordered reports whether the keys base and target share appear in a
// different order: adding or deleting a key moves no other key.
func reordered(base, target *fileSnapshot) bool {
	return !slices.Equal(sharedKeys(base.order, target.values), sharedKeys(target.order, base.values))
}

// sharedKeys returns the keys of order that other also holds, in order.
func sharedKeys(order []string, other map[string]string) []string {
	var shared []string
	for _, k := range order {
		if _, ok := other[k]; ok {
			shared = append(shared, k)
		}
	}
	return shared
}

// Format returns the stdout-ready summary block (with trailing newline), or ""
// when there is nothing to report. Each line is flattened, since the paths it
// names can hold any control character.
func (s *SessionStats) Format() string {
	lines := s.Summary()
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Session summary:\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(flattenText(line))
		b.WriteByte('\n')
	}
	return b.String()
}
