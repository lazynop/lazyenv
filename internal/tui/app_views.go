package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	configErrorHint    = "Press any key to exit"
	configErrorCutHint = "Press any key to exit · see lazyenv --check-config"
)

func (a App) viewConfigError() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ColorError).
		Padding(1, 3).
		Align(lipgloss.Center)

	// The path and the errors wrap to the columns the box's frame leaves, so
	// a long one never pushes the box past the terminal.
	room := max(a.width-box.GetHorizontalFrameSize(), 1)
	var path []string
	if a.configPath != "" {
		path = strings.Split(ansi.Wrap(a.configPath, room, ""), "\n")
	}
	hint := configErrorHint

	// When the box is taller than the terminal it drops its vertical padding,
	// then the blank rows, then the errors that do not fit, and points to
	// --check-config, which lists them all.
	blankRows := 2
	rowsBesideMsg := func() int { return box.GetVerticalFrameSize() + 1 + len(path) + blankRows + 1 }
	msg := wrapEach(a.configLines, room)
	msgRows := len(slices.Concat(msg...))
	if rowsBesideMsg()+msgRows > a.height {
		box = box.Padding(0, 3)
	}
	if rowsBesideMsg()+msgRows > a.height {
		blankRows = 0
	}
	var cut bool
	if msg, cut = cutToRows(msg, max(a.height-rowsBesideMsg(), 1), room); cut {
		hint = configErrorCutHint
	}

	lines := []string{a.theme.GitWarning.Render("Configuration Error")}
	for _, p := range path {
		lines = append(lines, a.theme.MutedItem.Render(p))
	}
	if blankRows > 0 {
		lines = append(lines, "")
	}
	for _, m := range slices.Concat(msg...) {
		lines = append(lines, a.theme.NormalItem.Render(m))
	}
	if blankRows > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, a.theme.MutedItem.Render(hint))

	alert := box.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, alert)
}

// wrapEach wraps each text to width columns and returns the rows of each.
func wrapEach(texts []string, width int) [][]string {
	wrapped := make([][]string, len(texts))
	for i, text := range texts {
		wrapped[i] = strings.Split(ansi.Wrap(text, width, ""), "\n")
	}
	return wrapped
}

// cutToRows keeps the leading texts whose rows fit in rows, and says on one
// more row how many it left out. When not even the first one fits, it keeps
// the rows of the first that fit, the last one ending in an ellipsis. It
// reports whether it cut anything.
func cutToRows(texts [][]string, rows, width int) ([][]string, bool) {
	if len(slices.Concat(texts...)) <= rows {
		return texts, false
	}
	more := func(n int) []string { return []string{fmt.Sprintf("… and %d more", n)} }
	used := 1 // the "… and N more" row
	kept := 0
	for kept < len(texts) && used+len(texts[kept]) <= rows {
		used += len(texts[kept])
		kept++
	}
	if kept > 0 {
		return append(slices.Clone(texts[:kept]), more(len(texts)-kept)), true
	}
	room := rows
	if len(texts) > 1 {
		room = max(rows-1, 1)
	}
	first := slices.Clone(texts[0][:room])
	first[room-1] = ansi.Truncate(first[room-1], width-1, "") + "…"
	cut := [][]string{first}
	if len(texts) > 1 && rows > 1 {
		cut = append(cut, more(len(texts)-1))
	}
	return cut, true
}
