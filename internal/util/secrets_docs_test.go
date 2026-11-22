package util

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/docstest"
)

var valueEntropyRuleRe = regexp.MustCompile(`^Values ≥(\d+) chars .* entropy ≥(\d+(?:\.\d+)?) bits/char`)

func TestConfigurationDocumentsTheBuiltinSecretRules(t *testing.T) {
	patternsByRule := map[string]string{}
	for _, row := range docstest.TableRows(t, docstest.ReadFile(t, "docs/configuration.md"), "Patterns") {
		patternsByRule[row[0]] = row[1]
	}
	require.Equal(t, []string{"Exact match", "Prefix match", "Suffix match", "Value entropy", "Value prefix"}, slices.Sorted(maps.Keys(patternsByRule)),
		"docs/configuration.md § Built-in detection rules has other rules than the ones this test checks")

	keyPatterns := map[string][]string{"Exact match": secretExact, "Suffix match": secretSuffixes, "Prefix match": secretPrefixes}
	for _, rule := range slices.Sorted(maps.Keys(keyPatterns)) {
		assert.ElementsMatch(t, keyPatterns[rule], docstest.CodeSpans(patternsByRule[rule]),
			"docs/configuration.md lists other %s patterns than IsSecret checks", rule)
	}
	valuePrefixes, _, _ := strings.Cut(patternsByRule["Value prefix"], "(")
	assert.ElementsMatch(t, secretValuePrefixes, docstest.CodeSpans(valuePrefixes),
		"docs/configuration.md lists other value prefixes than secretValuePrefixes")

	m := valueEntropyRuleRe.FindStringSubmatch(patternsByRule["Value entropy"])
	require.NotNil(t, m, "docs/configuration.md should state the minimum length and the entropy threshold of the value heuristic")
	assert.Equal(t, strconv.Itoa(tokenMinLength), m[1], "docs/configuration.md states the wrong minimum length of a token-like value")
	threshold, err := strconv.ParseFloat(m[2], 64)
	require.NoError(t, err)
	assert.Equal(t, EntropyThreshold, threshold, "docs/configuration.md states the wrong entropy threshold")
}
