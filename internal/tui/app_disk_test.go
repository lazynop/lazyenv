package tui

import "testing"

func TestDiskWritesGoThroughChokepoint(t *testing.T) {
	osWrites := []string{"CopyFS", "Create", "CreateTemp", "Link", "OpenFile", "OpenRoot",
		"Remove", "RemoveAll", "Rename", "Symlink", "Truncate", "WriteFile"}
	parserWrites := []string{"CreateBackup", "WriteFile", "WriteFileAtomic"}
	refs := append(sourceRefs(t, "os", osWrites...), sourceRefs(t, internalParserPath, parserWrites...)...)
	assertRefsAllowed(t, refs, map[string]string{
		"app_disk.go":                   "the chokepoint itself",
		"app_actions.go:backupIfNeeded": "writes a .bak backup, which the session summary does not track",
	}, "change .env files on disk only through the App methods in app_disk.go, which run the "+
		"matching SessionStats hook (add one for a new kind of operation); os.OpenFile and os.OpenRoot "+
		"count even for reads, since either can write: read a file with os.Open or os.ReadFile")
}
