package tui

import "testing"

// TestDiskWritesGoThroughChokepoint fails on any reference, outside
// app_disk.go, to an os or internal/parser function that creates, writes,
// truncates, renames, links or deletes files: such a call would skip the
// SessionStats hook that the App methods in app_disk.go run right after each
// disk operation succeeds. Functions that leave file content and existence
// alone (Mkdir, Chmod, ...) are out of scope: the summary compares variables.
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
