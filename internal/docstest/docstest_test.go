package docstest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const subsectionDoc = `# Title

## First

### Rules

first rules

## Second

intro

### Rules

second rules

### Other

other text

## Third

### Last

last text

## Fourth

fourth text
`

const fencedDoc = "Run `just build` first.\n\n" +
	"```bash\n" +
	"echo `date`\n" +
	"```\n\n" +
	"Then `just run`, or `just check`.\n\n" +
	"1. In a list:\n\n" +
	"    ```bash\n" +
	"    echo `whoami`\n" +
	"    ```\n\n" +
	"   then `just test`.\n"

func TestCodeSpansSkipFencedBlocks(t *testing.T) {
	assert.Equal(t, []string{"just build", "just run", "just check", "just test"}, CodeSpans(fencedDoc))
}

func TestSubsectionReadsTheSubsectionOfTheNamedSection(t *testing.T) {
	assert.Equal(t, "\nsecond rules\n", Subsection(t, subsectionDoc, "Second", "Rules"))
}

func TestSubsectionEndsWithItsSection(t *testing.T) {
	assert.Equal(t, "\nlast text\n", Subsection(t, subsectionDoc, "Third", "Last"))
}
