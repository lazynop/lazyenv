package tui

import (
	"fmt"
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
	kind        createKind
	source      string
	initialVars map[string]string
}

// SessionStats tracks disk-level changes during a lazyenv session.
// All maps are keyed by absolute file path.
type SessionStats struct {
	initial map[string]map[string]string // path → key→value at session start
	final   map[string]map[string]string // path → key→value after latest save; nil value = deleted
	renames map[string]string            // currentPath → originalPath (chain collapsed)
	created map[string]createOrigin      // newPath → origin info
}

// NewSessionStats returns an empty stats tracker.
func NewSessionStats() *SessionStats {
	return &SessionStats{
		initial: map[string]map[string]string{},
		final:   map[string]map[string]string{},
		renames: map[string]string{},
		created: map[string]createOrigin{},
	}
}

// snapshot builds a key→value map from vars using shell semantics: later
// occurrences of a duplicate key overwrite earlier ones, matching what a shell
// would see when sourcing the file.
func snapshot(vars []model.EnvVar) map[string]string {
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Key] = v.Value
	}
	return m
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
	// initialVars is only consulted for createDuplicate (diff against post-create edits).
	// Storing it for scratch/template would be dead weight.
	if kind == createDuplicate {
		origin.initialVars = snap
	}
	s.created[path] = origin
	s.final[path] = snap
}

func pluralVars(n int) string {
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
// rename-back-to-origin (a→b→a leaves no trace).
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
	if f, ok := s.final[oldPath]; ok {
		s.final[newPath] = f
		delete(s.final, oldPath)
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

func (s *SessionStats) formatLiveRow(path string, content map[string]string) (string, bool) {
	if content == nil {
		return "", false
	}
	if co, ok := s.created[path]; ok {
		return s.formatCreatedRow(path, co, content), true
	}
	if origin, ok := s.renames[path]; ok {
		return fmt.Sprintf("%s (renamed from %s) — %s", path, origin, diffCounts(s.initial[origin], content)), true
	}
	base, ok := s.initial[path]
	if !ok {
		return "", false
	}
	a, c, d := diff(base, content)
	if a == 0 && c == 0 && d == 0 {
		return "", false
	}
	return fmt.Sprintf("%s — %s", path, formatCounts(a, c, d)), true
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

func (s *SessionStats) formatCreatedRow(path string, co createOrigin, content map[string]string) string {
	switch co.kind {
	case createScratch:
		return fmt.Sprintf("%s — new file (%s)", path, pluralVars(len(content)))
	case createTemplate:
		return fmt.Sprintf("%s — from template %s (%s)", path, co.source, pluralVars(len(content)))
	case createDuplicate:
		a, c, d := diff(co.initialVars, content)
		if a == 0 && c == 0 && d == 0 {
			return fmt.Sprintf("%s — duplicated from %s (%s)", path, co.source, pluralVars(len(content)))
		}
		return fmt.Sprintf("%s — duplicated from %s, %s", path, co.source, formatCounts(a, c, d))
	}
	return ""
}

func formatCounts(added, changed, deleted int) string {
	return fmt.Sprintf("%d added, %d changed, %d deleted", added, changed, deleted)
}

func diffCounts(base, target map[string]string) string {
	a, c, d := diff(base, target)
	return formatCounts(a, c, d)
}

// Format returns the stdout-ready summary block (with trailing newline), or ""
// when there is nothing to report.
func (s *SessionStats) Format() string {
	lines := s.Summary()
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Session summary:\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
