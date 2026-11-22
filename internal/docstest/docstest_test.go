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

func TestSubsectionReadsTheSubsectionOfTheNamedSection(t *testing.T) {
	assert.Equal(t, "\nsecond rules\n", Subsection(t, subsectionDoc, "Second", "Rules"))
}

func TestSubsectionEndsWithItsSection(t *testing.T) {
	assert.Equal(t, "\nlast text\n", Subsection(t, subsectionDoc, "Third", "Last"))
}
