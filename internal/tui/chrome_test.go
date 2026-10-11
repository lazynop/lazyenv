package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestBorderedPanelsFillEveryInnerRow(t *testing.T) {
	keys := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	theme := BuildTheme(true, config.ColorConfig{})
	const height = 8 // top border + title + 5 content rows + bottom border

	cases := []struct {
		name     string
		lastItem string // the 5th item, which belongs on the last inner row
		render   func() string
	}{
		{"var list", "val_E", func() string {
			vl := NewVarListModel(config.DefaultConfig().Layout)
			vl.SetFile(makeTestFile(".env", keys...))
			vl.Width, vl.Height = 40, height
			return vl.View(theme)
		}},
		{"file list", ".env.e", func() string {
			var fl FileListModel
			for _, k := range keys {
				fl.Files = append(fl.Files, makeTestFile(".env."+strings.ToLower(k), "A"))
			}
			fl.Width, fl.Height = 40, height
			return fl.View(theme)
		}},
		{"compare", "val_E", func() string {
			dv := NewDiffViewModel(config.DefaultConfig().Layout, config.SecretsConfig{})
			dv.Width, dv.Height = 80, height
			dv.SetFiles(makeTestFile(".env", keys...), makeTestFile(".env.local", keys...))
			return dv.View(theme)
		}},
	}

	for _, tc := range cases {
		rows := strings.Split(stripAnsi(tc.render()), "\n")
		assert.Len(t, rows, height, "%s: panel height", tc.name)
		assert.Contains(t, rows[len(rows)-2], tc.lastItem,
			"%s: the row above the bottom border should hold content, not stay blank", tc.name)
	}
}
