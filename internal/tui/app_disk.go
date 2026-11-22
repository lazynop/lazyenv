package tui

import (
	"os"
	"path/filepath"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

const envFilePerm os.FileMode = 0o644

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

func (a App) duplicateFile(path string, srcOnDisk []byte, srcPath string) error {
	if err := parser.WriteFileAtomic(path, srcOnDisk, envFilePerm); err != nil {
		return err
	}
	copied := parser.ParseBytes(filepath.Base(path), srcOnDisk, config.SecretsConfig{})
	a.sessionStats.RecordCreateDuplicate(path, srcPath, copied.Vars)
	return nil
}

func (a App) templateFile(path string, strippedSrc []byte, src *model.EnvFile) error {
	if err := parser.WriteFileAtomic(path, strippedSrc, envFilePerm); err != nil {
		return err
	}
	keysOnly := make([]model.EnvVar, 0, len(src.Vars))
	for _, v := range src.Vars {
		keysOnly = append(keysOnly, model.EnvVar{Key: v.Key})
	}
	a.sessionStats.RecordCreateTemplate(path, src.Path, keysOnly)
	return nil
}

func (a App) renameFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	a.sessionStats.RecordRename(oldPath, newPath)
	return nil
}

func (a App) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	a.sessionStats.RecordDelete(path)
	return nil
}
