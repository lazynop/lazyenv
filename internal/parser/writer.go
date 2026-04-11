package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lazynop/lazyenv/internal/model"
)

// NormalizeForWrite adjusts quote styles for variables whose current Value
// cannot be safely serialized in their current style. Specifically, a
// QuoteSingle variable whose value contains a ' character is downgraded to
// QuoteDouble, because POSIX shell single quotes have no escape mechanism and
// re-parsing such a value would silently truncate it.
//
// Returns the keys of the variables whose quote style was changed, in file
// order. Callers (typically TUI save handlers) use this to surface a flash
// message explaining the silent switch to the user.
//
// Idempotent: a second call on the same file is a no-op.
func NormalizeForWrite(ef *model.EnvFile) []string {
	var changed []string
	for i := range ef.Vars {
		v := &ef.Vars[i]
		if v.QuoteStyle == model.QuoteSingle && strings.ContainsRune(v.Value, '\'') {
			v.QuoteStyle = model.QuoteDouble
			v.Modified = true
			changed = append(changed, v.Key)
		}
	}
	return changed
}

// Marshal serializes an EnvFile back to bytes.
// Unmodified lines are emitted as-is for round-trip fidelity.
// Modified lines are reconstructed from EnvVar metadata.
func Marshal(ef *model.EnvFile) []byte {
	var b strings.Builder

	for i, line := range ef.Lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if line.Type != model.LineVariable || line.VarIdx < 0 || line.VarIdx >= len(ef.Vars) {
			b.WriteString(line.Content)
			continue
		}

		v := ef.Vars[line.VarIdx]
		if !v.Modified {
			b.WriteString(line.Content)
			continue
		}

		// Reconstruct the line from metadata
		b.WriteString(reconstructLine(&v))
	}

	b.WriteByte('\n')
	return []byte(b.String())
}

func reconstructLine(v *model.EnvVar) string {
	var b strings.Builder

	if v.HasExport {
		b.WriteString("export ")
	}

	b.WriteString(v.Key)
	b.WriteByte('=')

	switch v.QuoteStyle {
	case model.QuoteDouble:
		b.WriteByte('"')
		b.WriteString(escapeDouble(v.Value))
		b.WriteByte('"')
	case model.QuoteSingle:
		b.WriteByte('\'')
		b.WriteString(v.Value)
		b.WriteByte('\'')
	default:
		b.WriteString(v.Value)
	}

	if v.Comment != "" {
		b.WriteString(" # ")
		b.WriteString(v.Comment)
	}

	return b.String()
}

func escapeDouble(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// WriteFile writes an EnvFile to disk atomically (temp file + rename).
func WriteFile(ef *model.EnvFile) error {
	if ef.Path == "" {
		return fmt.Errorf("no file path set")
	}

	data := Marshal(ef)

	dir := filepath.Dir(ef.Path)
	tmp, err := os.CreateTemp(dir, ".lazyenv-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Get original file permissions
	mode := os.FileMode(0644)
	if info, err := os.Stat(ef.Path); err == nil {
		mode = info.Mode()
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpName, mode); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpName, ef.Path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// CreateBackup creates a .bak copy of the file at the given path.
// It preserves the original file's permissions.
func CreateBackup(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file for backup: %w", err)
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
	}
	return os.WriteFile(path+".bak", src, mode)
}
