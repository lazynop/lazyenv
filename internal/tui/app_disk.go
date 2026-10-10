package tui

import (
	"os"

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

// createFile writes data to the new file at path and records it as created
// from scratch, with no variables.
func (a App) createFile(path string, data []byte) error {
	if err := parser.WriteFileAtomic(path, data, envFilePerm); err != nil {
		return err
	}
	a.sessionStats.RecordCreateScratch(path, nil)
	return nil
}

// duplicateFile writes data, a byte copy of src, to the new file at path and
// records it as a duplicate holding src's vars.
func (a App) duplicateFile(path string, data []byte, src *model.EnvFile) error {
	if err := parser.WriteFileAtomic(path, data, envFilePerm); err != nil {
		return err
	}
	a.sessionStats.RecordCreateDuplicate(path, src.Path, src.Vars)
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
