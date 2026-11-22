package tui

import "testing"

func TestDiskWritesGoThroughChokepoint(t *testing.T) {
	// OpenFile and OpenRoot count whatever they are used for, since either can
	// write: read a file with os.Open or os.ReadFile.
	osWrites := []string{"CopyFS", "Create", "CreateTemp", "Link", "OpenFile", "OpenRoot",
		"Remove", "RemoveAll", "Rename", "Symlink", "Truncate", "WriteFile"}
	parserWrites := []string{"CreateBackup", "WriteFile", "WriteFileAtomic"}
	refs := append(sourceRefs(t, "os", osWrites...), sourceRefs(t, internalParserPath, parserWrites...)...)
	assertRefsAllowed(t, refs, map[string]string{
		"app_disk.go":                   "the chokepoint itself",
		"app_actions.go:backupIfNeeded": "writes a .bak backup, which the session summary does not track",
	}, "change .env files on disk only through the App methods in app_disk.go, which run the "+
		"matching SessionStats hook (add one for a new kind of operation)")
}
