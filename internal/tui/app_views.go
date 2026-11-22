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
	configErrorCutHint = configErrorHint + " · see lazyenv --check-config"

	configPathMaxRows = 2
)

func (a App) viewConfigError() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ColorError).
		Padding(1, 3).
		Align(lipgloss.Center)

	room := max(a.width-box.GetHorizontalFrameSize(), 1)
	var path []string
	if a.configPath != "" {
		path = wrapRows(a.configPath, room)
	}
	if len(path) > configPathMaxRows {
		const ellipsis, wideCharRowSlack = "…", 1
		keep := configPathMaxRows*room - ansi.StringWidth(ellipsis) - wideCharRowSlack
		tail := ansi.TruncateLeft(a.configPath, ansi.StringWidth(a.configPath)-keep, ellipsis)
		path = strings.Split(ansi.Hardwrap(tail, room, false), "\n")
	}
	hint := configErrorHint

	blankRows := 2
	rowsBesideMsg := func() int { return box.GetVerticalFrameSize() + 1 + len(path) + blankRows + 1 }
	msg := wrapEach(a.configLines, room)
	msgRows := len(slices.Concat(msg...))
	if rowsBesideMsg()+msgRows > a.height {
		box = box.PaddingTop(0).PaddingBottom(0)
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

func wrapEach(texts []string, width int) [][]string {
	wrapped := make([][]string, len(texts))
	for i, text := range texts {
		wrapped[i] = wrapRows(text, width)
	}
	return wrapped
}

func cutToRows(texts [][]string, rows, width int) ([][]string, bool) {
	if len(slices.Concat(texts...)) <= rows {
		return texts, false
	}
	more := func(n int) []string { return []string{fmt.Sprintf("… and %d more", n)} }
	used := len(more(0))
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
