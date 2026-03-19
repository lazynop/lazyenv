package model

// DiffStatus represents the type of difference between two env files.
type DiffStatus int

const (
	DiffEqual   DiffStatus = iota // same key and value
	DiffChanged                   // same key, different value
	DiffAdded                     // only in file A (left)
	DiffRemoved                   // only in file B (right)
)

// DiffEntry represents a single difference between two files.
type DiffEntry struct {
	Key      string
	Status   DiffStatus
	ValueA   string // value in file A
	ValueB   string // value in file B
	IsSecret bool   // true if the key is secret in either file
}

// ComputeDiff compares two env files key-by-key.
// Uses last occurrence for duplicates (shell semantics).
// Returns entries ordered: equal/changed first (by A's order), then added (A only), then removed (B only).
func ComputeDiff(a, b *EnvFile) []DiffEntry {
	// Build maps using last occurrence (shell semantics)
	bMap := make(map[string]string)
	bSecret := make(map[string]bool)
	for _, v := range b.Vars {
		bMap[v.Key] = v.Value
		bSecret[v.Key] = bSecret[v.Key] || v.IsSecret
	}
	aMap := make(map[string]string)
	aSecret := make(map[string]bool)
	for _, v := range a.Vars {
		aMap[v.Key] = v.Value
		aSecret[v.Key] = aSecret[v.Key] || v.IsSecret
	}

	seen := make(map[string]bool)
	var entries []DiffEntry

	// Walk A's vars in order (using last occurrence per key)
	for i := len(a.Vars) - 1; i >= 0; i-- {
		key := a.Vars[i].Key
		if seen[key] {
			continue
		}
		seen[key] = true
	}

	// Reset and walk in forward order for output
	seen = make(map[string]bool)
	for _, v := range a.Vars {
		if seen[v.Key] {
			continue
		}
		seen[v.Key] = true
		valA := aMap[v.Key]
		secret := aSecret[v.Key] || bSecret[v.Key]
		if valB, ok := bMap[v.Key]; ok {
			if valA == valB {
				entries = append(entries, DiffEntry{
					Key:      v.Key,
					Status:   DiffEqual,
					ValueA:   valA,
					ValueB:   valB,
					IsSecret: secret,
				})
			} else {
				entries = append(entries, DiffEntry{
					Key:      v.Key,
					Status:   DiffChanged,
					ValueA:   valA,
					ValueB:   valB,
					IsSecret: secret,
				})
			}
		} else {
			entries = append(entries, DiffEntry{
				Key:      v.Key,
				Status:   DiffAdded,
				ValueA:   valA,
				IsSecret: secret,
			})
		}
	}

	// Keys only in B (removed from A's perspective)
	seenB := make(map[string]bool)
	for _, v := range b.Vars {
		if seenB[v.Key] {
			continue
		}
		seenB[v.Key] = true
		if _, ok := aMap[v.Key]; !ok {
			entries = append(entries, DiffEntry{
				Key:      v.Key,
				Status:   DiffRemoved,
				ValueB:   bMap[v.Key],
				IsSecret: bSecret[v.Key],
			})
		}
	}

	return entries
}
