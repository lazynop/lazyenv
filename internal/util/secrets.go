package util

import (
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/lazynop/lazyenv/internal/config"
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

// EntropyThreshold is the minimum Shannon entropy (bits/char) for a value
// to be flagged by the heuristic. Real tokens typically score 4.0–5.0,
// while hostnames and URLs score 3.0–4.0.
const EntropyThreshold = 4.0

const tokenMinLength = 16

// IsSecret returns true if the key or value looks like a secret.
func IsSecret(key, value string, cfg config.SecretsConfig) bool {
	upper := strings.ToUpper(key)

	for _, p := range cfg.SafePatterns {
		if matchesPattern(upper, p) {
			return false
		}
	}

	for _, p := range cfg.ExtraPatterns {
		if matchesPattern(upper, p) {
			return true
		}
	}

	if isSecretKeyName(upper) {
		return true
	}

	if hasSecretValuePrefix(value) {
		return true
	}

	if cfg.ValueHeuristicEnabled() && looksLikeToken(value) {
		return true
	}

	return false
}

func isSecretKeyName(upperKey string) bool {
	if slices.Contains(secretExact, upperKey) {
		return true
	}
	for _, suffix := range secretSuffixes {
		if strings.HasSuffix(upperKey, suffix) {
			return true
		}
	}
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(upperKey, prefix) {
			return true
		}
	}
	return false
}

func hasSecretValuePrefix(value string) bool {
	for _, prefix := range secretValuePrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func looksLikeToken(value string) bool {
	return len(value) >= tokenMinLength && hasAlphanumericMix(value) && ShannonEntropy(value) >= EntropyThreshold
}

// ShannonEntropy returns the Shannon entropy in bits per byte.
// Operates on raw bytes, suitable for ASCII-dominated .env values.
func ShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	var freq [256]int
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}
	n := float64(len(s))
	var entropy float64
	for _, count := range freq {
		if count == 0 {
			continue
		}
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}

func matchesPattern(upperKey, upperPattern string) bool {
	if strings.HasPrefix(upperPattern, "_") {
		return strings.HasSuffix(upperKey, upperPattern)
	}
	if strings.HasSuffix(upperPattern, "_") {
		return strings.HasPrefix(upperKey, upperPattern)
	}
	return upperKey == upperPattern
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
	return hasDigit && (hasUpper || hasLower)
}

// MaskValue returns a masked representation of a value.
func MaskValue(value string) string {
	if value == "" {
		return ""
	}
	return "••••••••"
}
