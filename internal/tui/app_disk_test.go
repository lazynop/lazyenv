package tui

import "testing"

// TestDiskWritesGoThroughChokepoint fails on any reference, outside
// app_disk.go, to a function that writes, creates, renames or deletes files:
// such a call would skip the SessionStats hook that the App methods in
// app_disk.go run right after each disk operation succeeds.
func TestDiskWritesGoThroughChokepoint(t *testing.T) {
	refs := append(sourceRefs(t, "os", "Remove", "Rename", "WriteFile"),
		sourceRefs(t, internalParserPath, "WriteFile", "WriteFileAtomic")...)
	assertRefsAllowed(t, refs, map[string]string{"app_disk.go": "the chokepoint itself"},
		"write, create, rename or delete .env files through the App methods in app_disk.go, "+
			"which run the matching SessionStats hook")
}
