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

// snapshot builds a key→value map from []EnvVar using shell semantics
// (last occurrence wins for duplicate keys).
func snapshot(vars []model.EnvVar) map[string]string {
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Key] = v.Value
	}
	return m
}

// RecordInitialLoad snapshots a file's state at session start.
// No-op on subsequent calls with the same path.
func (s *SessionStats) RecordInitialLoad(path string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	if _, ok := s.initial[path]; ok {
		return
	}
	s.initial[path] = snapshot(vars)
}

// RecordSave snapshots a file's state after a successful write to disk.
func (s *SessionStats) RecordSave(path string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	s.final[path] = snapshot(vars)
}

// RecordCreateScratch registers a file created from scratch (empty body).
func (s *SessionStats) RecordCreateScratch(path string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	snap := snapshot(vars)
	s.created[path] = createOrigin{kind: createScratch, initialVars: snap}
	s.final[path] = snap
}

// RecordCreateTemplate registers a file created as a keys-only template of src.
func (s *SessionStats) RecordCreateTemplate(path, source string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	snap := snapshot(vars)
	s.created[path] = createOrigin{kind: createTemplate, source: source, initialVars: snap}
	s.final[path] = snap
}

// RecordCreateDuplicate registers a byte-for-byte copy of source.
func (s *SessionStats) RecordCreateDuplicate(path, source string, vars []model.EnvVar) {
	if s == nil {
		return
	}
	snap := snapshot(vars)
	s.created[path] = createOrigin{kind: createDuplicate, source: source, initialVars: snap}
	s.final[path] = snap
}

func pluralVars(n int) string {
	if n == 1 {
		return "1 variable"
	}
	return fmt.Sprintf("%d variables", n)
}

// RecordDelete registers a file deletion. If the file was created in this
// session, the create is cancelled (net-zero).
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

// RecordRename registers a file rename. Handles chain collapse and
// rename-back-to-origin (no rename recorded in that case).
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

// diff returns (added, changed, deleted) counts between two snapshots.
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
// Returns nil when there is nothing to report.
func (s *SessionStats) Summary() []string {
	if s == nil {
		return nil
	}

	type row struct {
		key, line string
	}
	var rows []row

	for path, content := range s.final {
		if content == nil {
			continue
		}
		if co, ok := s.created[path]; ok {
			switch co.kind {
			case createScratch:
				rows = append(rows, row{path, fmt.Sprintf("%s — new file (%s)", path, pluralVars(len(content)))})
			case createTemplate:
				rows = append(rows, row{path, fmt.Sprintf("%s — from template %s (%s)", path, co.source, pluralVars(len(content)))})
			case createDuplicate:
				a, c, d := diff(co.initialVars, content)
				if a == 0 && c == 0 && d == 0 {
					rows = append(rows, row{path, fmt.Sprintf("%s — duplicated from %s (%s)", path, co.source, pluralVars(len(content)))})
				} else {
					rows = append(rows, row{path, fmt.Sprintf("%s — duplicated from %s, %d added, %d changed, %d deleted", path, co.source, a, c, d)})
				}
			}
			continue
		}
		if origin, ok := s.renames[path]; ok {
			a, c, d := diff(s.initial[origin], content)
			rows = append(rows, row{path, fmt.Sprintf("%s (renamed from %s) — %d added, %d changed, %d deleted", path, origin, a, c, d)})
			continue
		}
		base, ok := s.initial[path]
		if !ok {
			continue
		}
		a, c, d := diff(base, content)
		if a == 0 && c == 0 && d == 0 {
			continue
		}
		rows = append(rows, row{path, fmt.Sprintf("%s — %d added, %d changed, %d deleted", path, a, c, d)})
	}

	// Pass 2: deletions.
	for path, content := range s.final {
		if content != nil {
			continue
		}
		target := path
		if origin, ok := s.renames[path]; ok {
			target = origin
		}
		if _, hadInitial := s.initial[target]; !hadInitial {
			continue
		}
		rows = append(rows, row{target, fmt.Sprintf("%s — deleted", target)})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.line
	}
	return out
}

// Format returns the full stdout-ready summary block (including trailing newline),
// or "" when there is nothing to report.
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
