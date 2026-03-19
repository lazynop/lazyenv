package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

var defaultCfg = config.SecretsConfig{}

func TestIsSecretByExactMatch(t *testing.T) {
	tests := []string{"PASSWORD", "SECRET", "TOKEN", "API_KEY", "ACCESS_KEY", "PRIVATE_KEY"}
	for _, key := range tests {
		assert.True(t, IsSecret(key, "", defaultCfg), "%s should be secret", key)
	}
}

func TestIsSecretBySuffix(t *testing.T) {
	tests := []struct {
		key    string
		secret bool
	}{
		{"DB_PASSWORD", true},
		{"MY_SECRET", true},
		{"AUTH_TOKEN", true},
		{"SESSION_KEY", true},
		{"ADMIN_PASS", true},
		{"MY_API_KEY", true},
		{"APP_AUTH", true},
		{"AWS_CREDENTIAL", true},
		{"SSH_PRIVATE", true},
		{"HOSTNAME", false},
		{"PORT", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.secret, IsSecret(tt.key, "", defaultCfg), "key=%s", tt.key)
	}
}

func TestIsSecretByPrefix(t *testing.T) {
	tests := []struct {
		key    string
		secret bool
	}{
		{"SECRET_VALUE", true},
		{"TOKEN_DATA", true},
		{"AUTH_HEADER", true},
		{"PRIVATE_IP", true},
		{"PUBLIC_URL", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.secret, IsSecret(tt.key, "", defaultCfg), "key=%s", tt.key)
	}
}

func TestIsSecretByValuePrefix(t *testing.T) {
	tests := []struct {
		value  string
		secret bool
	}{
		{"sk-abc123", true},
		{"pk-abc123", true},
		{"ghp_xxxx", true},
		{"gho_xxxx", true},
		{"Bearer eyJ", true},
		{"hello", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.secret, IsSecret("HOSTNAME", tt.value, defaultCfg), "value=%s", tt.value)
	}
}

func TestIsSecretByValueHeuristic(t *testing.T) {
	// High-entropy token with upper+lower+digit — should be flagged
	assert.True(t, IsSecret("NORMAL", "aB3cD4eF5gH6iJ7kL8mN9", defaultCfg))
	// Too short
	assert.False(t, IsSecret("NORMAL", "short123", defaultCfg))
	// Long but no alphanumeric mix
	assert.False(t, IsSecret("NORMAL", "aaaaaaaaaaaaaaaaaaaaa", defaultCfg))
	// Uppercase+digit — below entropy threshold (3.78 < 4.0)
	assert.False(t, IsSecret("NORMAL", "AKIAIOSFODNN7EXAMPLE1", defaultCfg))
	// Lowercase+digit — below entropy threshold (3.91 < 4.0)
	assert.False(t, IsSecret("NORMAL", "a1b2c3d4e5f6a7b8c9d0e1", defaultCfg))
}

func TestIsSecretCaseInsensitiveKey(t *testing.T) {
	assert.True(t, IsSecret("password", "", defaultCfg))
	assert.True(t, IsSecret("db_Password", "", defaultCfg))
}

// --- New tests for entropy and config ---

func TestShannonEntropy(t *testing.T) {
	tests := []struct {
		name  string
		value string
		minE  float64
		maxE  float64
	}{
		{"empty", "", 0, 0},
		{"single char repeated", "aaaaaaaaaa", 0, 0.01},
		{"hostname", "db-prod-ap-03.internal", 3.0, 4.0},
		{"url", "https://api.example.com/v2", 3.0, 4.0},
		{"random token", "aB3cD4eF5gH6iJ7kL8mN9", 4.0, 5.0},
		{"aws-like key", "AKIAIOSFODNN7EXAMPLE1", 3.0, 5.0},
	}
	for _, tt := range tests {
		e := ShannonEntropy(tt.value)
		assert.GreaterOrEqual(t, e, tt.minE, "%s: entropy %.2f < min %.2f", tt.name, e, tt.minE)
		assert.LessOrEqual(t, e, tt.maxE, "%s: entropy %.2f > max %.2f", tt.name, e, tt.maxE)
	}
}

func TestEntropyFalsePositiveFix(t *testing.T) {
	// The original bug: hostname flagged as secret due to length+alphanumeric mix
	assert.False(t, IsSecret("DATABASE_HOST", "db-prod-ap-03.internal", defaultCfg),
		"hostname should not be flagged as secret")
	// Short URLs are not flagged
	assert.False(t, IsSecret("API_ENDPOINT", "https://api.example.com/v2", defaultCfg),
		"short URL should not be flagged as secret")
	// Very long URLs with high entropy may be flagged — users can use safe_patterns
	cfg := config.SecretsConfig{SafePatterns: []string{"_ENDPOINT"}}
	assert.False(t, IsSecret("API_ENDPOINT", "https://api.example.com/v2/resources", cfg),
		"URL with safe_pattern should not be flagged")
}

func TestIsSecretSafePatterns(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"_TOKEN"}, // override built-in suffix
	}
	// AUTH_TOKEN matches built-in _TOKEN suffix, but safe_patterns overrides
	assert.False(t, IsSecret("AUTH_TOKEN", "", cfg))
	// DB_PASSWORD is not in safe_patterns, still secret
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg))
}

func TestIsSecretSafePatternsExact(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"PASSWORD"}, // exact match
	}
	assert.False(t, IsSecret("PASSWORD", "", cfg))
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg)) // suffix still matches built-in
}

func TestIsSecretSafePatternsPrefix(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"SECRET_"}, // prefix match
	}
	assert.False(t, IsSecret("SECRET_VALUE", "", cfg))
	assert.True(t, IsSecret("MY_SECRET", "", cfg)) // suffix still matches built-in
}

func TestIsSecretExtraPatterns(t *testing.T) {
	cfg := config.SecretsConfig{
		ExtraPatterns: []string{"_ENDPOINT", "INTERNAL_"},
	}
	assert.True(t, IsSecret("API_ENDPOINT", "", cfg))
	assert.True(t, IsSecret("INTERNAL_URL", "", cfg))
	assert.False(t, IsSecret("PUBLIC_URL", "", cfg))
}

func TestIsSecretExtraPatternExact(t *testing.T) {
	cfg := config.SecretsConfig{
		ExtraPatterns: []string{"MY_SPECIAL_VAR"},
	}
	assert.True(t, IsSecret("MY_SPECIAL_VAR", "", cfg))
	assert.False(t, IsSecret("OTHER_VAR", "", cfg))
}

func TestIsSecretValueHeuristicDisabled(t *testing.T) {
	f := false
	cfg := config.SecretsConfig{ValueHeuristic: &f}
	// High-entropy token should NOT be flagged when heuristic is disabled
	assert.False(t, IsSecret("NORMAL", "aB3cD4eF5gH6iJ7kL8mN9", cfg))
	// But key-name detection still works
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg))
	// And value prefix detection still works
	assert.True(t, IsSecret("NORMAL", "sk-abc123", cfg))
}

func TestIsSecretSafePatternsOverrideExtraPatterns(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns:  []string{"_ENDPOINT"},
		ExtraPatterns: []string{"_ENDPOINT"},
	}
	// Safe takes precedence over extra
	assert.False(t, IsSecret("API_ENDPOINT", "", cfg))
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		key     string
		pattern string
		match   bool
	}{
		{"DATABASE_HOST", "_HOST", true},         // suffix
		{"MY_HOST_NAME", "_HOST", false},         // not a suffix
		{"PUBLIC_URL", "PUBLIC_", true},          // prefix
		{"NOT_PUBLIC_URL", "PUBLIC_", false},     // not a prefix
		{"DATABASE_HOST", "DATABASE_HOST", true}, // exact
		{"database_host", "DATABASE_HOST", true}, // case insensitive
		{"OTHER", "DATABASE_HOST", false},        // no match
	}
	for _, tt := range tests {
		result := matchesPattern(strings.ToUpper(tt.key), tt.pattern)
		assert.Equal(t, tt.match, result, "key=%s pattern=%s", tt.key, tt.pattern)
	}
}

func TestMaskValue(t *testing.T) {
	assert.Equal(t, "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022", MaskValue("secret123"))
	assert.Equal(t, "", MaskValue(""))
}
