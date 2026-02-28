package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSecretByExactMatch(t *testing.T) {
	tests := []string{"PASSWORD", "SECRET", "TOKEN", "API_KEY", "ACCESS_KEY", "PRIVATE_KEY"}
	for _, key := range tests {
		assert.True(t, IsSecret(key, ""), "%s should be secret", key)
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
		assert.Equal(t, tt.secret, IsSecret(tt.key, ""), "key=%s", tt.key)
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
		assert.Equal(t, tt.secret, IsSecret(tt.key, ""), "key=%s", tt.key)
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
		assert.Equal(t, tt.secret, IsSecret("HOSTNAME", tt.value), "value=%s", tt.value)
	}
}

func TestIsSecretByValueHeuristic(t *testing.T) {
	// 20+ chars with alphanumeric mix
	assert.True(t, IsSecret("NORMAL", "aB3cD4eF5gH6iJ7kL8mN9"))
	// Too short
	assert.False(t, IsSecret("NORMAL", "short123"))
	// Long but no mix
	assert.False(t, IsSecret("NORMAL", "aaaaaaaaaaaaaaaaaaaaa"))
}

func TestIsSecretCaseInsensitiveKey(t *testing.T) {
	assert.True(t, IsSecret("password", ""))
	assert.True(t, IsSecret("db_Password", ""))
}

func TestMaskValue(t *testing.T) {
	assert.Equal(t, "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022", MaskValue("secret123"))
	assert.Equal(t, "", MaskValue(""))
}
