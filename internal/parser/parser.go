package parser

import (
	"os"
	"strings"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

const utf8BOM = "\uFEFF"

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
		Name:    name,
		Secrets: secrets,
	}

	content := string(data)
	content = strings.TrimPrefix(content, utf8BOM)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	ef.TrailingNewline = content == "" || strings.HasSuffix(content, "\n")

	var lines []string
	if content != "" {
		lines = strings.Split(content, "\n")
	}

	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}

	i := 0
	for i < len(lines) {
		line := lines[i]
		lineNum := i + 1

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineEmpty,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineComment,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		key, value, comment, quoteStyle, hasExport, rawContent, consumed := parseLine(lines, i)
		if key == "" {
			ef.Lines = append(ef.Lines, model.RawLine{
				Type:    model.LineComment,
				Content: line,
				VarIdx:  -1,
			})
			i++
			continue
		}

		v := model.EnvVar{
			Key:        key,
			Value:      value,
			Comment:    comment,
			LineNum:    lineNum,
			QuoteStyle: quoteStyle,
			HasExport:  hasExport,
		}

		varIdx := len(ef.Vars)
		ef.Vars = append(ef.Vars, v)
		ef.Lines = append(ef.Lines, model.RawLine{
			Type:    model.LineVariable,
			Content: rawContent,
			VarIdx:  varIdx,
		})

		i += consumed
	}

	ef.DeriveFlags()
	return ef
}

func parseLine(lines []string, i int) (key, value, comment string, quote model.QuoteStyle, hasExport bool, rawContent string, consumed int) {
	line := lines[i]
	trimmed := strings.TrimSpace(line)

	working := trimmed
	if strings.HasPrefix(working, "export ") {
		hasExport = true
		working = strings.TrimSpace(working[7:])
	} else if strings.HasPrefix(working, "export\t") {
		hasExport = true
		working = strings.TrimSpace(working[7:])
	}

	before, after, ok := strings.Cut(working, "=")
	if !ok {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	key = strings.TrimSpace(before)
	if key == "" || !IsValidKey(key) {
		return "", "", "", model.QuoteNone, false, "", 0
	}

	rest := after

	if len(rest) > 0 && rest[0] == '"' {
		value, comment, rawContent, consumed = parseDoubleQuoted(lines, i, rest)
		return key, value, comment, model.QuoteDouble, hasExport, rawContent, consumed
	}
	if len(rest) > 0 && rest[0] == '\'' {
		value, comment, rawContent, consumed = parseSingleQuoted(lines, i, rest)
		return key, value, comment, model.QuoteSingle, hasExport, rawContent, consumed
	}

	value, comment = parseUnquoted(rest)
	return key, value, comment, model.QuoteNone, hasExport, line, 1
}

func parseDoubleQuoted(lines []string, startIdx int, rest string) (string, string, string, int) {
	return parseQuoted(lines, startIdx, rest, findClosingDoubleQuote, processEscapes)
}

func parseSingleQuoted(lines []string, startIdx int, rest string) (string, string, string, int) {
	return parseQuoted(lines, startIdx, rest, findClosingSingleQuote, identityTransform)
}

func parseQuoted(
	lines []string,
	startIdx int,
	fromOpeningQuote string,
	findClose func(string) int,
	transform func(string) string,
) (string, string, string, int) {
	content := fromOpeningQuote[1:]
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

func findClosingDoubleQuote(content string) int {
	for j := 0; j < len(content); j++ {
		if content[j] == '\\' && j+1 < len(content) {
			j++
			continue
		}
		if content[j] == '"' {
			return j
		}
	}
	return -1
}

func findClosingSingleQuote(content string) int {
	return strings.Index(content, "'")
}

func identityTransform(s string) string { return s }

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
