package parser

import (
	"os"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/util"
)

// basename returns the last path component of a file path, supporting both
// '/' and '\' as separators so the behavior is identical on Linux and Windows.
func basename(path string) string {
	if idx := strings.LastIndexAny(path, `/\`); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// ParseFile reads and parses an .env file from disk.
func ParseFile(path string, secrets config.SecretsConfig) (*model.EnvFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ef := ParseBytes(basename(path), data, secrets)
	ef.Path = path
	return ef, nil
}

// ParseBytes parses .env content from raw bytes.
func ParseBytes(name string, data []byte, secrets config.SecretsConfig) *model.EnvFile {
	ef := &model.EnvFile{
		Name: name,
	}

	content := string(data)
	// Strip UTF-8 BOM if present (common with files saved by Windows editors).
	content = strings.TrimPrefix(content, "\uFEFF")
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	// Track whether the source ended with a newline so Marshal can preserve
	// the round-trip exactly (no silent injection of \n on files without one).
	ef.TrailingNewline = strings.HasSuffix(content, "\n")

	lines := strings.Split(content, "\n")

	// If file ends with newline, Split produces a trailing empty string — remove it
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}

	keySeen := make(map[string]bool)

	i := 0
	for i < len(lines) {
		line := lines[i]
		lineNum := i + 1

		trimmed := strings.TrimSpace(line)

		// Empty line
		if trimmed == "" {
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineEmpty,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		// Comment line
		if strings.HasPrefix(trimmed, "#") {
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineComment,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		// Try to parse as KEY=VALUE
		key, value, comment, quoteStyle, hasExport, rawContent, consumed := parseLine(lines, i)
		if key == "" {
			// Not a valid variable line — treat as comment
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineComment,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		isDuplicate := keySeen[key]
		keySeen[key] = true

		v := model.EnvVar{
			Key:           key,
			Value:         value,
			Comment:       comment,
			LineNum:       lineNum,
			QuoteStyle:    quoteStyle,
			HasExport:     hasExport,
			IsSecret:      util.IsSecret(key, value, secrets),
			IsEmpty:       value == "",
			IsPlaceholder: util.IsPlaceholderValue(value),
			IsDuplicate:   isDuplicate,
		}

		varIdx := len(ef.Vars)
		ef.Vars = append(ef.Vars, v)
		ef.Lines = append(ef.Lines, model.RawLine{
			Type:    model.LineVariable,
			Content: rawContent,
			VarIdx:  varIdx,
		})

		// Mark earlier occurrences as duplicate too
		if isDuplicate {
			for j := range varIdx {
				if ef.Vars[j].Key == key {
					ef.Vars[j].IsDuplicate = true
				}
			}
		}

		i += consumed
	}

	return ef
}

// parseLine attempts to parse a variable from lines starting at index i.
// Returns key, value, inline comment, quote style, hasExport, raw content, lines consumed.
func parseLine(lines []string, i int) (string, string, string, model.QuoteStyle, bool, string, int) {
	line := lines[i]
	trimmed := strings.TrimSpace(line)

	// Check for export prefix
	hasExport := false
	working := trimmed
	if strings.HasPrefix(working, "export ") {
		hasExport = true
		working = strings.TrimSpace(working[7:])
	} else if strings.HasPrefix(working, "export\t") {
		hasExport = true
		working = strings.TrimSpace(working[7:])
	}

	// Find the = sign
	before, after, ok := strings.Cut(working, "=")
	if !ok {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	key := strings.TrimSpace(before)
	if key == "" || !IsValidKey(key) {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	rest := after

	// Determine quoting
	if len(rest) > 0 && rest[0] == '"' {
		// Double-quoted value — may span multiple lines
		value, comment, rawContent, consumed := parseDoubleQuoted(lines, i, rest)
		return key, value, comment, model.QuoteDouble, hasExport, rawContent, consumed
	}
	if len(rest) > 0 && rest[0] == '\'' {
		// Single-quoted value — may span multiple lines
		value, comment, rawContent, consumed := parseSingleQuoted(lines, i, rest)
		return key, value, comment, model.QuoteSingle, hasExport, rawContent, consumed
	}

	// Unquoted value
	value, comment := parseUnquoted(rest)
	return key, value, comment, model.QuoteNone, hasExport, line, 1
}

func parseDoubleQuoted(lines []string, startIdx int, rest string) (string, string, string, int) {
	return parseQuoted(lines, startIdx, rest, findClosingDoubleQuote, processEscapes)
}

func parseSingleQuoted(lines []string, startIdx int, rest string) (string, string, string, int) {
	return parseQuoted(lines, startIdx, rest, findClosingSingleQuote, identityTransform)
}

// parseQuoted reads a (possibly multi-line) quoted value starting at lines[startIdx].
// rest is the substring after the `=` sign, including the opening quote char.
// findClose locates the closing quote on a given content slice (or -1), and
// transform maps the raw joined value into the final form (e.g. escape processing).
func parseQuoted(
	lines []string,
	startIdx int,
	rest string,
	findClose func(string) int,
	transform func(string) string,
) (string, string, string, int) {
	content := rest[1:] // skip opening quote
	var valueParts []string
	rawLines := []string{lines[startIdx]}
	consumed := 1

	for {
		if idx := findClose(content); idx >= 0 {
			valueParts = append(valueParts, content[:idx])
			remainder := strings.TrimSpace(content[idx+1:])
			value := transform(strings.Join(valueParts, "\n"))
			return value, extractInlineComment(remainder), strings.Join(rawLines, "\n"), consumed
		}

		// No closing quote on this line — consume the next one.
		valueParts = append(valueParts, content)
		consumed++
		if startIdx+consumed-1 >= len(lines) {
			return transform(strings.Join(valueParts, "\n")), "", strings.Join(rawLines, "\n"), consumed
		}
		nextLine := lines[startIdx+consumed-1]
		rawLines = append(rawLines, nextLine)
		content = nextLine
	}
}

// findClosingDoubleQuote returns the index of the first unescaped `"` in content,
// or -1 if none is present.
func findClosingDoubleQuote(content string) int {
	for j := 0; j < len(content); j++ {
		if content[j] == '\\' && j+1 < len(content) {
			j++ // skip the next char, loop increments past it
			continue
		}
		if content[j] == '"' {
			return j
		}
	}
	return -1
}

// findClosingSingleQuote returns the index of the first `'` in content, or -1.
// Single-quoted values have no escape mechanism in shell semantics.
func findClosingSingleQuote(content string) int {
	return strings.Index(content, "'")
}

func identityTransform(s string) string { return s }

// extractInlineComment returns the comment text from a remainder string that
// starts with `#`, trimming at most one leading space. Returns empty string
// when there is no `#` at the start of remainder.
func extractInlineComment(remainder string) string {
	if !strings.HasPrefix(remainder, "#") {
		return ""
	}
	comment := remainder[1:]
	if len(comment) > 0 && comment[0] == ' ' {
		return comment[1:]
	}
	return comment
}

func parseUnquoted(rest string) (string, string) {
	// Inline comments on unquoted values are delimited by whitespace + `#`.
	value := rest
	comment := ""

	for j := 0; j < len(value); j++ {
		if value[j] == ' ' || value[j] == '\t' {
			trimRemaining := strings.TrimLeft(value[j:], " \t")
			if strings.HasPrefix(trimRemaining, "#") {
				value = value[:j]
				comment = extractInlineComment(trimRemaining)
				break
			}
		}
	}

	value = strings.TrimRight(value, " \t")
	return value, comment
}

func processEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// IsValidKey checks whether a string is a valid .env variable key.
func IsValidKey(key string) bool {
	for i, r := range key {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		if r == '.' || r == '-' {
			continue
		}
		return false
	}
	return len(key) > 0
}
