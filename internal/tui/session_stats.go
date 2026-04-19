package tui

import (
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
