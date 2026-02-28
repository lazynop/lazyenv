package util

import (
	"slices"
	"strings"
)

var placeholderExact = []string{
	"changeme", "todo", "fixme", "xxx", "replace_me",
	"change_me", "your_value_here", "insert_here",
}

// IsPlaceholderValue returns true if the value looks like a placeholder.
func IsPlaceholderValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	if slices.Contains(placeholderExact, lower) {
		return true
	}
	// "your_*_here" pattern
	if strings.HasPrefix(lower, "your_") && strings.HasSuffix(lower, "_here") {
		return true
	}
	// "your-*-here" pattern
	if strings.HasPrefix(lower, "your-") && strings.HasSuffix(lower, "-here") {
		return true
	}
	return false
}
