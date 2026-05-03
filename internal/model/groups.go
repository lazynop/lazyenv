package model

import "strings"

// VarGroup is a prefix-based grouping of variables in an EnvFile.
// Vars are referenced by their index into EnvFile.Vars.
type VarGroup struct {
	// Prefix is the substring before the first '_' in the variable's Key.
	// Empty string identifies the trailing "Ungrouped" bucket, which holds
	// vars with no '_', a leading '_', or a unique prefix that didn't reach
	// the grouping threshold.
	Prefix string
	// Vars are the indices into EnvFile.Vars belonging to this group, in
	// the order they appear in the file.
	Vars []int
}

// IsUngrouped reports whether the group is the trailing "Ungrouped" bucket.
func (g VarGroup) IsUngrouped() bool {
	return g.Prefix == ""
}

// prefixOf returns the substring before the first '_' in key. Returns ""
// if key has no '_' or starts with '_' (in which case the var is considered
// ungroupable).
func prefixOf(key string) string {
	if i := strings.IndexByte(key, '_'); i > 0 {
		return key[:i]
	}
	return ""
}

// ComputeGroups returns the groups for a list of variables.
//
// Rules:
//   - prefix = substring before the first '_' in Key
//   - vars without '_' or starting with '_' get prefix = ""
//   - a group is emitted only if at least 2 vars share the same non-empty
//     prefix (case-sensitive)
//   - vars with a unique prefix or empty prefix go into the trailing
//     "Ungrouped" group (Prefix == "")
//   - groups are ordered by the position of their first variable in vars,
//     except the Ungrouped group which is always last
//   - the Ungrouped group is emitted only if it contains at least one var
//   - returns nil for an empty input
func ComputeGroups(vars []EnvVar) []VarGroup {
	if len(vars) == 0 {
		return nil
	}

	// First pass: count occurrences of each non-empty prefix and record
	// its encounter order, so we can preserve file-order for groups.
	counts := make(map[string]int)
	order := make([]string, 0)
	for _, v := range vars {
		p := prefixOf(v.Key)
		if p == "" {
			continue
		}
		if _, seen := counts[p]; !seen {
			order = append(order, p)
		}
		counts[p]++
	}

	// Build the named groups in file order, keeping only those that pass
	// the threshold of 2.
	var groups []VarGroup
	groupIdx := make(map[string]int, len(order))
	for _, p := range order {
		if counts[p] >= 2 {
			groupIdx[p] = len(groups)
			groups = append(groups, VarGroup{Prefix: p})
		}
	}

	// Second pass: distribute vars into either a named group or Ungrouped.
	var ungrouped VarGroup
	for i, v := range vars {
		p := prefixOf(v.Key)
		if gi, ok := groupIdx[p]; ok {
			groups[gi].Vars = append(groups[gi].Vars, i)
			continue
		}
		ungrouped.Vars = append(ungrouped.Vars, i)
	}

	if len(ungrouped.Vars) > 0 {
		groups = append(groups, ungrouped)
	}
	return groups
}
