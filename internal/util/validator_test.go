package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPlaceholderExactMatch(t *testing.T) {
	placeholders := []string{
		"changeme", "todo", "fixme", "xxx",
		"replace_me", "change_me", "your_value_here", "insert_here",
	}
	for _, p := range placeholders {
		assert.True(t, IsPlaceholderValue(p), "%q should be placeholder", p)
	}
}

func TestIsPlaceholderCaseInsensitive(t *testing.T) {
	assert.True(t, IsPlaceholderValue("CHANGEME"))
	assert.True(t, IsPlaceholderValue("Todo"))
	assert.True(t, IsPlaceholderValue("FIXME"))
}

func TestIsPlaceholderYourXHerePattern(t *testing.T) {
	assert.True(t, IsPlaceholderValue("your_api_key_here"))
	assert.True(t, IsPlaceholderValue("your_password_here"))
	assert.True(t, IsPlaceholderValue("your-api-key-here"))
	assert.True(t, IsPlaceholderValue("your-token-here"))
}

func TestIsNotPlaceholder(t *testing.T) {
	notPlaceholders := []string{
		"real_value", "localhost", "8080", "production",
		"my_api_key", "", "  ",
	}
	for _, v := range notPlaceholders {
		assert.False(t, IsPlaceholderValue(v), "%q should not be placeholder", v)
	}
}
