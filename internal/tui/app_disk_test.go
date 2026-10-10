package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDiskWritesGoThroughChokepoint fails on any reference, outside
// app_disk.go, to a function that writes, creates, renames or deletes files:
// such a call would skip the SessionStats hook that the App methods in
// app_disk.go run right after each disk operation succeeds.
func TestDiskWritesGoThroughChokepoint(t *testing.T) {
	// Disk writes allowed outside app_disk.go, keyed "file:function pkg.Func",
	// with how many times each may occur there. Every allowance must be used
	// in full, so a stale entry fails too.
	allowed := map[string]int{
		// Best-effort cleanup of a just-created file that failed to parse.
		"app_fileinput.go:finaliseNewFile os.Remove": 1,
	}

	refs := append(sourceRefs(t, "os", "Remove", "Rename", "WriteFile"),
		sourceRefs(t, internalParserPath, "WriteFile", "WriteFileAtomic")...)
	var violations []string
	for _, ref := range refs {
		if ref.file == "app_disk.go" {
			continue
		}
		if key := ref.file + ":" + ref.owner + " " + ref.name; allowed[key] > 0 {
			allowed[key]--
			continue
		}
		violations = append(violations, ref.String())
	}

	assert.Empty(t, violations, "write, create, rename or delete .env files through the App methods "+
		"in app_disk.go, which run the matching SessionStats hook")
	for key, unused := range allowed {
		assert.Zero(t, unused, "allowed entry %q matches %d fewer references than it allows: "+
			"lower or remove it", key, unused)
	}
}
