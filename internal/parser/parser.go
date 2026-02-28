package parser

import (
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"gitlab.com/traveltoaiur/lazyenv/internal/util"
	"os"
	"strings"
)

// ParseFile reads and parses an .env file from disk.
func ParseFile(path string) (*model.EnvFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}
	ef := ParseBytes(name, data)
	ef.Path = path
	return ef, nil
}

// ParseBytes parses .env content from raw bytes.
func ParseBytes(name string, data []byte) *model.EnvFile {
	ef := &model.EnvFile{
		Name: name,
	}

	content := string(data)
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

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
			IsSecret:      util.IsSecret(key, value),
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
			for j := 0; j < varIdx; j++ {
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
	eqIdx := strings.Index(working, "=")
	if eqIdx < 0 {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	key := strings.TrimSpace(working[:eqIdx])
	if key == "" || !isValidKey(key) {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	rest := working[eqIdx+1:]

	// Determine quoting
	if len(rest) > 0 && rest[0] == '"' {
		// Double-quoted value — may span multiple lines
		value, comment, rawContent, consumed := parseDoubleQuoted(lines, i, rest)
		return key, value, comment, model.QuoteDouble, hasExport, rawContent, consumed
	}
	if len(rest) > 0 && rest[0] == '\'' {
		// Single-quoted value
		value, comment := parseSingleQuoted(rest)
		return key, value, comment, model.QuoteSingle, hasExport, line, 1
	}

	// Unquoted value
	value, comment := parseUnquoted(rest)
	return key, value, comment, model.QuoteNone, hasExport, line, 1
}

func parseDoubleQuoted(lines []string, startIdx int, rest string) (string, string, string, int) {
	// rest starts with "
	content := rest[1:] // skip opening quote

	var valueParts []string
	rawLines := []string{lines[startIdx]}
	consumed := 1

	for {
		// Scan for closing unescaped quote
		j := 0
		for j < len(content) {
			if content[j] == '\\' && j+1 < len(content) {
				j += 2 // skip escape sequence
				continue
			}
			if content[j] == '"' {
				// Found closing quote
				valueParts = append(valueParts, content[:j])
				remainder := strings.TrimSpace(content[j+1:])
				value := processEscapes(strings.Join(valueParts, "\n"))

				// Check for inline comment after closing quote
				comment := ""
				if strings.HasPrefix(remainder, "#") {
					comment = remainder[1:]
					if len(comment) > 0 && comment[0] == ' ' {
						comment = comment[1:]
					}
				}

				return value, comment, strings.Join(rawLines, "\n"), consumed
			}
			j++
		}

		// No closing quote on this line — multiline
		valueParts = append(valueParts, content)
		consumed++
		if startIdx+consumed-1 >= len(lines) {
			// Unterminated quote — return what we have
			return processEscapes(strings.Join(valueParts, "\n")), "", strings.Join(rawLines, "\n"), consumed
		}
		nextLine := lines[startIdx+consumed-1]
		rawLines = append(rawLines, nextLine)
		content = nextLine
	}
}

func parseSingleQuoted(rest string) (string, string) {
	// rest starts with '
	content := rest[1:]
	// Find closing single quote (no escape processing in single quotes)
	endIdx := strings.Index(content, "'")
	if endIdx < 0 {
		// Unterminated — take everything
		return content, ""
	}
	value := content[:endIdx]
	remainder := strings.TrimSpace(content[endIdx+1:])

	comment := ""
	if strings.HasPrefix(remainder, "#") {
		comment = remainder[1:]
		if len(comment) > 0 && comment[0] == ' ' {
			comment = comment[1:]
		}
	}
	return value, comment
}

func parseUnquoted(rest string) (string, string) {
	// For unquoted values, inline comments are preceded by whitespace + #
	value := rest
	comment := ""

	// Find first occurrence of whitespace followed by #
	for j := 0; j < len(value); j++ {
		if value[j] == ' ' || value[j] == '\t' {
			remaining := value[j:]
			trimRemaining := strings.TrimLeft(remaining, " \t")
			if strings.HasPrefix(trimRemaining, "#") {
				value = value[:j]
				comment = trimRemaining[1:]
				if len(comment) > 0 && comment[0] == ' ' {
					comment = comment[1:]
				}
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

func isValidKey(key string) bool {
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
