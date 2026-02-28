package util

import (
	"strings"
	"unicode"
)

var secretSuffixes = []string{
	"_KEY", "_SECRET", "_TOKEN", "_PASSWORD", "_PASS",
	"_API_KEY", "_AUTH", "_CREDENTIAL", "_PRIVATE",
}

var secretPrefixes = []string{
	"SECRET_", "TOKEN_", "AUTH_", "PRIVATE_",
}

var secretExact = []string{
	"PASSWORD", "SECRET", "TOKEN", "API_KEY",
	"ACCESS_KEY", "PRIVATE_KEY",
}

var secretValuePrefixes = []string{
	"sk-", "pk-", "ghp_", "gho_", "Bearer ",
}

// IsSecret returns true if the key or value looks like a secret.
func IsSecret(key, value string) bool {
	upper := strings.ToUpper(key)

	for _, exact := range secretExact {
		if upper == exact {
			return true
		}
	}
	for _, suffix := range secretSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}

	// Value heuristic
	if looksLikeToken(value) {
		return true
	}

	return false
}

func looksLikeToken(value string) bool {
	for _, prefix := range secretValuePrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	if len(value) >= 20 && hasAlphanumericMix(value) {
		return true
	}
	return false
}

func hasAlphanumericMix(s string) bool {
	var hasUpper, hasLower, hasDigit bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	// Must have at least digit + (upper or lower)
	return hasDigit && (hasUpper || hasLower)
}

// MaskValue returns a masked representation of a value.
func MaskValue(value string) string {
	if value == "" {
		return ""
	}
	return "••••••••"
}
