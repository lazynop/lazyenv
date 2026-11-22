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
	const contentRows = 5
	const height = panelBorderHeight + panelTitleHeight + contentRows

	cases := []struct {
		name               string
		lastContentRowItem string
		render             func() string
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
			dv := NewDiffViewModel(config.DefaultConfig().Layout)
			dv.Width, dv.Height = 80, height
			dv.SetFiles(makeTestFile(".env", keys...), makeTestFile(".env.local", keys...))
			return dv.View(theme)
		}},
	}

	for _, tc := range cases {
		rows := strings.Split(stripAnsi(tc.render()), "\n")
		assert.Len(t, rows, height, "%s: panel height", tc.name)
		assert.Contains(t, rows[len(rows)-2], tc.lastContentRowItem,
			"%s: the row above the bottom border should hold content, not stay blank", tc.name)
	}
}
