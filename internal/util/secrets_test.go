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
	assert.True(t, IsSecret("NORMAL", "aB3cD4eF5gH6iJ7kL8mN9", defaultCfg),
		"a long, high-entropy token mixing upper, lower and digits should be flagged")
	assert.False(t, IsSecret("NORMAL", "short123", defaultCfg),
		"a value shorter than 16 characters should not be flagged")
	assert.False(t, IsSecret("NORMAL", "aaaaaaaaaaaaaaaaaaaaa", defaultCfg),
		"a long value without digits should not be flagged")
	assert.False(t, IsSecret("NORMAL", "AKIAIOSFODNN7EXAMPLE1", defaultCfg),
		"uppercase and digits stay below the entropy threshold (3.78 < 4.0)")
	assert.False(t, IsSecret("NORMAL", "a1b2c3d4e5f6a7b8c9d0e1", defaultCfg),
		"lowercase and digits stay below the entropy threshold (3.91 < 4.0)")
}

func TestIsSecretCaseInsensitiveKey(t *testing.T) {
	assert.True(t, IsSecret("password", "", defaultCfg))
	assert.True(t, IsSecret("db_Password", "", defaultCfg))
}

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
	assert.False(t, IsSecret("DATABASE_HOST", "db-prod-ap-03.internal", defaultCfg),
		"a long hostname mixing letters and digits should not be flagged as secret")
	assert.False(t, IsSecret("API_ENDPOINT", "https://api.example.com/v2", defaultCfg),
		"short URL should not be flagged as secret")
	cfg := config.SecretsConfig{SafePatterns: []string{"_ENDPOINT"}}
	assert.False(t, IsSecret("API_ENDPOINT", "https://api.example.com/v2/resources", cfg),
		"a long URL the heuristic may flag should not be flagged once a safe pattern covers it")
}

func TestIsSecretSafePatterns(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"_TOKEN"},
	}
	assert.False(t, IsSecret("AUTH_TOKEN", "", cfg), "a safe pattern should override the built-in _TOKEN suffix")
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg), "a key no safe pattern matches should stay secret")
}

func TestIsSecretSafePatternsExact(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"PASSWORD"},
	}
	assert.False(t, IsSecret("PASSWORD", "", cfg))
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg), "the built-in suffix should still match")
}

func TestIsSecretSafePatternsPrefix(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns: []string{"SECRET_"},
	}
	assert.False(t, IsSecret("SECRET_VALUE", "", cfg))
	assert.True(t, IsSecret("MY_SECRET", "", cfg), "the built-in suffix should still match")
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
	assert.False(t, IsSecret("NORMAL", "aB3cD4eF5gH6iJ7kL8mN9", cfg), "the disabled heuristic should not flag a high-entropy token")
	assert.True(t, IsSecret("DB_PASSWORD", "", cfg), "key-name detection should still work")
	assert.True(t, IsSecret("NORMAL", "sk-abc123", cfg), "value-prefix detection should still work")
}

func TestIsSecretSafePatternsOverrideExtraPatterns(t *testing.T) {
	cfg := config.SecretsConfig{
		SafePatterns:  []string{"_ENDPOINT"},
		ExtraPatterns: []string{"_ENDPOINT"},
	}
	assert.False(t, IsSecret("API_ENDPOINT", "", cfg))
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		pattern string
		match   bool
	}{
		{"suffix", "DATABASE_HOST", "_HOST", true},
		{"not a suffix", "MY_HOST_NAME", "_HOST", false},
		{"prefix", "PUBLIC_URL", "PUBLIC_", true},
		{"not a prefix", "NOT_PUBLIC_URL", "PUBLIC_", false},
		{"exact", "DATABASE_HOST", "DATABASE_HOST", true},
		{"case insensitive", "database_host", "DATABASE_HOST", true},
		{"no match", "OTHER", "DATABASE_HOST", false},
	}
	for _, tt := range tests {
		result := matchesPattern(strings.ToUpper(tt.key), tt.pattern)
		assert.Equal(t, tt.match, result, "%s: key=%s pattern=%s", tt.name, tt.key, tt.pattern)
	}
}

func TestMaskValue(t *testing.T) {
	assert.Equal(t, "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022", MaskValue("secret123"))
	assert.Equal(t, "", MaskValue(""))
}
