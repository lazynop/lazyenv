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
	Present []bool   // one element per file, in order
	Values  []string // value in each file (empty string if absent)
}

// ComputeMatrix builds a completeness matrix from the given files.
// Returns entries sorted alphabetically and file names in input order.
// For duplicate keys within a file, the last occurrence wins (shell semantics).
func ComputeMatrix(files []*EnvFile) ([]MatrixEntry, []string) {
	if len(files) == 0 {
		return nil, nil
	}

	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}

	// Collect unique keys and last-occurrence values per file.
	type fileVal struct {
		value   string
		present bool
	}
	keyData := make(map[string][]fileVal)

	for fi, f := range files {
		for _, v := range f.Vars {
			fv, ok := keyData[v.Key]
			if !ok {
				fv = make([]fileVal, len(files))
				keyData[v.Key] = fv
			}
			fv[fi] = fileVal{value: v.Value, present: true}
		}
	}

	entries := make([]MatrixEntry, 0, len(keyData))
	for k, fvs := range keyData {
		e := MatrixEntry{
			Key:     k,
			Present: make([]bool, len(files)),
			Values:  make([]string, len(files)),
		}
		for i, fv := range fvs {
			e.Present[i] = fv.present
			e.Values[i] = fv.value
		}
		entries = append(entries, e)
	}

	SortEntries(entries, SortAlpha)
	return entries, names
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
