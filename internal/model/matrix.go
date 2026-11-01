package model

import "sort"

// SortMode determines the ordering of matrix entries.
type SortMode int

const (
	SortAlpha        SortMode = iota // alphabetical by key
	SortCompleteness                 // keys with most gaps first
)

// MatrixEntry represents one row in the completeness matrix.
type MatrixEntry struct {
	Key     string
	Present []bool // one element per file, in order
}

// ComputeMatrix builds a completeness matrix from the given files.
// Returns entries sorted alphabetically.
func ComputeMatrix(files []*EnvFile) []MatrixEntry {
	present := make(map[string][]bool)
	for fi, f := range files {
		for _, v := range f.Vars {
			p, ok := present[v.Key]
			if !ok {
				p = make([]bool, len(files))
				present[v.Key] = p
			}
			p[fi] = true
		}
	}

	entries := make([]MatrixEntry, 0, len(present))
	for k, p := range present {
		entries = append(entries, MatrixEntry{Key: k, Present: p})
	}

	SortEntries(entries, SortAlpha)
	return entries
}

// SortEntries sorts matrix entries in place by the given mode.
func SortEntries(entries []MatrixEntry, mode SortMode) {
	switch mode {
	case SortAlpha:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Key < entries[j].Key
		})
	case SortCompleteness:
		sort.SliceStable(entries, func(i, j int) bool {
			ci := countPresent(entries[i].Present)
			cj := countPresent(entries[j].Present)
			if ci != cj {
				return ci < cj // fewer present = more gaps = first
			}
			return entries[i].Key < entries[j].Key
		})
	}
}

func countPresent(ps []bool) int {
	n := 0
	for _, p := range ps {
		if p {
			n++
		}
	}
	return n
}
