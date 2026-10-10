package config

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	exampleTableRe = regexp.MustCompile(`^\[([\w.-]+)\]\s*(?:#.*)?$`)
	exampleKeyRe   = regexp.MustCompile(`^#?\s*([\w-]+)\s*=`)
)

// TestExampleDocumentsEveryKey checks that examples/config/full.toml, the
// reference users copy from, documents every key Config reads: each table as
// a [table] header, each field as a "key = ..." line under its table
// (commented out counts).
func TestExampleDocumentsEveryKey(t *testing.T) {
	data, err := os.ReadFile("../../examples/config/full.toml")
	require.NoError(t, err)

	checkKeysDocumented(t, exampleKeys(string(data)), "", reflect.TypeFor[Config]())
}

// checkKeysDocumented runs one subtest per toml key of typ, found under table,
// recursing into nested tables.
func checkKeysDocumented(t *testing.T, documented map[string]bool, table string, typ reflect.Type) {
	t.Helper()
	for field := range typ.Fields() {
		if !field.IsExported() {
			continue
		}
		key, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if key == "-" {
			continue
		}
		// go-toml still decodes an untagged exported field, under its Go name,
		// so it would be a config key this test cannot name.
		if key == "" {
			t.Errorf(`%s.%s has no toml tag: add a kebab-case one, or toml:"-" if it is not a config key`, typ.Name(), field.Name)
			continue
		}

		path := joinKey(table, key)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			t.Run(path, func(t *testing.T) {
				assert.True(t, documented[path], "examples/config/full.toml has no [%s] table header", path)
			})
			checkKeysDocumented(t, documented, path, fieldType)
			continue
		}

		where := "at the top level, before any table"
		if table != "" {
			where = "under [" + table + "]"
		}
		t.Run(path, func(t *testing.T) {
			assert.True(t, documented[path],
				"examples/config/full.toml has no `%s = ...` line %s (commented out counts)", key, where)
		})
	}
}

// exampleKeys returns the key paths a TOML example documents: each [table]
// header as "table", each "key = ..." line, commented out or not, as "key" at
// the top level or "table.key" below a header.
func exampleKeys(src string) map[string]bool {
	keys := map[string]bool{}
	table := ""
	for line := range strings.Lines(src) {
		line = strings.TrimSpace(line)
		if m := exampleTableRe.FindStringSubmatch(line); m != nil {
			table = m[1]
			keys[table] = true
			continue
		}
		if m := exampleKeyRe.FindStringSubmatch(line); m != nil {
			keys[joinKey(table, m[1])] = true
		}
	}
	return keys
}

// joinKey returns the dotted path of key inside table ("" is the top level).
func joinKey(table, key string) string {
	if table == "" {
		return key
	}
	return table + "." + key
}
