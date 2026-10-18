package tui

import (
	"os"
	"path/filepath"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// This file is the only place in the package that writes, creates, renames or
// deletes .env files (enforced by TestDiskWritesGoThroughChokepoint). Each
// method performs one disk operation and, once it succeeds, records it in
// SessionStats before returning, so callers can neither forget the hook nor
// run fallible post-processing (re-parse, CheckGitIgnore) ahead of it.

// envFilePerm is the permission mode for newly created .env files
// (owner-writable, world-readable), matching parser.WriteFile.
const envFilePerm os.FileMode = 0o644

// saveFile writes f to disk and records the vars just written.
func (a App) saveFile(f *model.EnvFile) error {
	if err := parser.WriteFile(f); err != nil {
		return err
	}
	a.sessionStats.RecordSave(f.Path, f.Vars)
	return nil
}

// createFile writes an empty .env file at path and records it as created from
// scratch, with no variables. The file holds a single newline so it is
// POSIX-conventional: with an empty file ParseBytes would set
// TrailingNewline=false and the first save would emit a file without a
// trailing newline.
func (a App) createFile(path string) error {
	if err := parser.WriteFileAtomic(path, []byte("\n"), envFilePerm); err != nil {
		return err
	}
	a.sessionStats.RecordCreateScratch(path, nil)
	return nil
}

// duplicateFile writes data, a byte copy of the file at srcPath, to the new
// file at path and records it as a duplicate holding the vars parsed from
// data: the source's unsaved edits are not in the copy.
func (a App) duplicateFile(path string, data []byte, srcPath string) error {
	if err := parser.WriteFileAtomic(path, data, envFilePerm); err != nil {
		return err
	}
	// The stats read only keys and values, so no secrets config is needed.
	copied := parser.ParseBytes(filepath.Base(path), data, config.SecretsConfig{})
	a.sessionStats.RecordCreateDuplicate(path, srcPath, copied.Vars)
	return nil
}

// templateFile writes data, src with every value stripped, to the new file at
// path and records it as a template holding src's keys with empty values.
func (a App) templateFile(path string, data []byte, src *model.EnvFile) error {
	if err := parser.WriteFileAtomic(path, data, envFilePerm); err != nil {
		return err
	}
	keysOnly := make([]model.EnvVar, 0, len(src.Vars))
	for _, v := range src.Vars {
		keysOnly = append(keysOnly, model.EnvVar{Key: v.Key})
	}
	a.sessionStats.RecordCreateTemplate(path, src.Path, keysOnly)
	return nil
}

// renameFile moves the file at oldPath to newPath and records the rename.
func (a App) renameFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	a.sessionStats.RecordRename(oldPath, newPath)
	return nil
}

// deleteFile removes the file at path and records the deletion.
func (a App) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	a.sessionStats.RecordDelete(path)
	return nil
}
