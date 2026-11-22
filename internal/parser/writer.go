package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lazynop/lazyenv/internal/model"
)

// NormalizeForWrite upgrades QuoteStyle for any var whose current Value
// cannot round-trip safely in its current style. For QuoteSingle that is an
// embedded `'` (shell has no escape inside '...'). For QuoteNone: embedded
// `\n`/`\r`/`\t` (line separators break the KEY=VALUE format), a leading
// quote char (the re-parse would treat the value as quoted and swallow
// following lines), and anything parseUnquoted would not read back verbatim
// (inline-comment truncation, trailing-whitespace trim). All upgrade to
// QuoteDouble because escapeDouble can serialize every such value.
//
// An upgraded var is also marked Modified, so Marshal rebuilds its line in the
// new style instead of re-emitting the stale raw line.
//
// Production callers should use WriteFile, which invokes this automatically.
// NormalizeForWrite remains exported for direct unit testing. Idempotent.
func NormalizeForWrite(ef *model.EnvFile) {
	for i := range ef.Vars {
		v := &ef.Vars[i]
		if needsQuoteUpgrade(v) {
			v.QuoteStyle = model.QuoteDouble
			v.Modified = true
		}
	}
}

func needsQuoteUpgrade(v *model.EnvVar) bool {
	switch v.QuoteStyle {
	case model.QuoteSingle:
		return strings.ContainsRune(v.Value, '\'')
	case model.QuoteNone:
		if strings.ContainsAny(v.Value, "\n\r\t") {
			return true
		}
		if len(v.Value) > 0 && (v.Value[0] == '"' || v.Value[0] == '\'') {
			return true
		}
		parsed, comment := parseUnquoted(v.Value)
		return parsed != v.Value || comment != ""
	}
	return false
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

		b.WriteString(reconstructLine(&v))
	}

	if ef.TrailingNewline && len(ef.Lines) > 0 {
		b.WriteByte('\n')
	}
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
// It normalizes quote styles for round-trip safety before serializing,
// which may mutate ef.Vars (QuoteStyle and Modified fields) in place.
func WriteFile(ef *model.EnvFile) error {
	if ef.Path == "" {
		return fmt.Errorf("no file path set")
	}

	NormalizeForWrite(ef)
	data := Marshal(ef)
	return WriteFileAtomic(ef.Path, data, existingFileMode(ef.Path))
}

const defaultFileMode = os.FileMode(0644)

func existingFileMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode()
	}
	return defaultFileMode
}

// WriteFileAtomic writes data to path atomically: it writes to a temp file in
// the same directory, fsyncs, applies perm, then renames over path. The fsync
// is essential — without it a crash between the rename and the filesystem
// journal flush can leave a zero-byte file at path on ext4 and similar
// filesystems.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lazyenv-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
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
	return WriteFileAtomic(path+".bak", src, existingFileMode(path))
}
