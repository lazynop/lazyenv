package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ".", cfg.Dir)
	assert.Equal(t, "position", cfg.Sort)
	assert.False(t, cfg.Recursive)
	assert.False(t, cfg.ShowAll)
	assert.False(t, cfg.NoGitCheck)
	assert.False(t, cfg.NoBackup)
}

func TestDefaultLayoutValues(t *testing.T) {
	l := DefaultConfig().Layout

	assert.Equal(t, 30, l.VarListMaxKeyWidth)
	assert.Equal(t, 25, l.DiffMaxKeyWidth)
	assert.Equal(t, 20, l.MatrixKeyWidth)
	assert.Equal(t, 14, l.MatrixColWidth)
	assert.Equal(t, 10, l.VarListMinValueWidth)
	assert.Equal(t, 12, l.VarListPadding)
	assert.Equal(t, 8, l.DiffMinValueWidth)
	assert.Equal(t, 10, l.DiffPadding)
	assert.Equal(t, 2*time.Second, l.MessageTimeout)
	assert.Equal(t, 3*time.Second, l.ErrorMessageTimeout)
}

func TestDefaultFilePatterns(t *testing.T) {
	f := DefaultConfig().Files

	assert.Equal(t, []string{".env", ".env.*", "*.env"}, f.Include)
	assert.Equal(t, []string{"*.bak"}, f.Exclude)
}

func TestDefaultColorsEmpty(t *testing.T) {
	c := DefaultConfig().Colors

	assert.Empty(t, c.Primary, "default colors should be empty for auto-detection")
	assert.Empty(t, c.Warning)
	assert.Empty(t, c.Error)
	assert.Empty(t, c.Success)
	assert.Empty(t, c.Muted)
	assert.Empty(t, c.Fg)
	assert.Empty(t, c.Border)
	assert.Empty(t, c.CursorBg)
}
