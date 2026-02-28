package parser

import (
	"bytes"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestBasicKeyValue(t *testing.T) {
	input := "FOO=bar\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Key != "FOO" || ef.Vars[0].Value != "bar" {
		t.Errorf("got %q=%q", ef.Vars[0].Key, ef.Vars[0].Value)
	}
	if ef.Vars[0].QuoteStyle != model.QuoteNone {
		t.Errorf("expected QuoteNone, got %d", ef.Vars[0].QuoteStyle)
	}
}

func TestDoubleQuotedValue(t *testing.T) {
	input := `FOO="hello world"` + "\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Value != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", ef.Vars[0].Value)
	}
	if ef.Vars[0].QuoteStyle != model.QuoteDouble {
		t.Errorf("expected QuoteDouble")
	}
}

func TestSingleQuotedValue(t *testing.T) {
	input := `FOO='hello world'` + "\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Value != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", ef.Vars[0].Value)
	}
	if ef.Vars[0].QuoteStyle != model.QuoteSingle {
		t.Errorf("expected QuoteSingle")
	}
}

func TestMultilineDoubleQuoted(t *testing.T) {
	input := "MULTI=\"line1\nline2\nline3\"\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	expected := "line1\nline2\nline3"
	if ef.Vars[0].Value != expected {
		t.Errorf("expected %q, got %q", expected, ef.Vars[0].Value)
	}
}

func TestExportPrefix(t *testing.T) {
	input := "export NODE_ENV=production\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Key != "NODE_ENV" || ef.Vars[0].Value != "production" {
		t.Errorf("got %q=%q", ef.Vars[0].Key, ef.Vars[0].Value)
	}
	if !ef.Vars[0].HasExport {
		t.Error("expected HasExport=true")
	}
}

func TestEmptyValue(t *testing.T) {
	input := "EMPTY=\nALSO_EMPTY=\"\"\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Value != "" || !ef.Vars[0].IsEmpty {
		t.Errorf("first var: value=%q, isEmpty=%v", ef.Vars[0].Value, ef.Vars[0].IsEmpty)
	}
	if ef.Vars[1].Value != "" || !ef.Vars[1].IsEmpty {
		t.Errorf("second var: value=%q, isEmpty=%v", ef.Vars[1].Value, ef.Vars[1].IsEmpty)
	}
}

func TestComments(t *testing.T) {
	input := "# This is a comment\nFOO=bar\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if len(ef.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(ef.Lines))
	}
	if ef.Lines[0].Type != model.LineComment {
		t.Errorf("expected first line to be comment")
	}
}

func TestInlineComment(t *testing.T) {
	input := "FOO=bar # this is inline\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Vars[0].Value != "bar" {
		t.Errorf("expected value %q, got %q", "bar", ef.Vars[0].Value)
	}
	if ef.Vars[0].Comment != "this is inline" {
		t.Errorf("expected comment %q, got %q", "this is inline", ef.Vars[0].Comment)
	}
}

func TestBlankLines(t *testing.T) {
	input := "FOO=bar\n\nBAZ=qux\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(ef.Vars))
	}
	if len(ef.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(ef.Lines))
	}
	if ef.Lines[1].Type != model.LineEmpty {
		t.Errorf("expected middle line to be empty")
	}
}

func TestEscapeSequences(t *testing.T) {
	input := `FOO="hello \"world\" \\path\nnewline"` + "\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	expected := "hello \"world\" \\path\nnewline"
	if ef.Vars[0].Value != expected {
		t.Errorf("expected %q, got %q", expected, ef.Vars[0].Value)
	}
}

func TestDuplicateDetection(t *testing.T) {
	input := "FOO=first\nBAR=baz\nFOO=second\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 3 {
		t.Fatalf("expected 3 vars, got %d", len(ef.Vars))
	}
	if !ef.Vars[0].IsDuplicate {
		t.Error("first FOO should be marked duplicate")
	}
	if ef.Vars[1].IsDuplicate {
		t.Error("BAR should not be duplicate")
	}
	if !ef.Vars[2].IsDuplicate {
		t.Error("second FOO should be marked duplicate")
	}
}

func TestInvalidLinesTreatedAsComments(t *testing.T) {
	input := "not a valid line\nFOO=bar\n"
	ef := ParseBytes(".env", []byte(input))
	if len(ef.Vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(ef.Vars))
	}
	if ef.Lines[0].Type != model.LineComment {
		t.Errorf("invalid line should be treated as comment, got %d", ef.Lines[0].Type)
	}
}

func TestRoundTripFidelity(t *testing.T) {
	input := `# Database config
DB_HOST=localhost              # primary host
DB_PORT=5432
DB_PASSWORD="super secret"

# API settings
export API_KEY='sk-12345'
EMPTY=
QUOTED_EMPTY=""

# Multiline
MULTI="line1
line2
line3"
`
	ef := ParseBytes(".env", []byte(input))
	output := Marshal(ef)
	if !bytes.Equal([]byte(input), output) {
		t.Errorf("round-trip mismatch:\n--- input ---\n%s\n--- output ---\n%s", input, string(output))
	}
}

func TestModifiedVarWriteBack(t *testing.T) {
	input := "FOO=old\nBAR=keep\n"
	ef := ParseBytes(".env", []byte(input))
	ef.UpdateVar(0, "new")

	output := string(Marshal(ef))
	if output != "FOO=new\nBAR=keep\n" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestModifiedQuotedVarWriteBack(t *testing.T) {
	input := `FOO="old value"` + "\n"
	ef := ParseBytes(".env", []byte(input))
	ef.UpdateVar(0, "new value")

	output := string(Marshal(ef))
	expected := `FOO="new value"` + "\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestAddVar(t *testing.T) {
	input := "FOO=bar\n"
	ef := ParseBytes(".env", []byte(input))
	ef.AddVar("NEW", "val")

	if len(ef.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(ef.Vars))
	}
	output := string(Marshal(ef))
	expected := "FOO=bar\nNEW=val\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestDeleteVar(t *testing.T) {
	input := "FOO=bar\nBAZ=qux\nQUX=end\n"
	ef := ParseBytes(".env", []byte(input))
	ef.DeleteVar(1) // delete BAZ

	if len(ef.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(ef.Vars))
	}
	output := string(Marshal(ef))
	expected := "FOO=bar\nQUX=end\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestSecretDetection(t *testing.T) {
	input := "DB_PASSWORD=secret123\nAPI_KEY=sk-12345\nNORMAL=hello\n"
	ef := ParseBytes(".env", []byte(input))
	if !ef.Vars[0].IsSecret {
		t.Error("DB_PASSWORD should be secret")
	}
	if !ef.Vars[1].IsSecret {
		t.Error("API_KEY should be secret")
	}
	if ef.Vars[2].IsSecret {
		t.Error("NORMAL should not be secret")
	}
}

func TestPlaceholderDetection(t *testing.T) {
	input := "A=changeme\nB=TODO\nC=your_api_key_here\nD=real_value\n"
	ef := ParseBytes(".env", []byte(input))
	if !ef.Vars[0].IsPlaceholder {
		t.Error("changeme should be placeholder")
	}
	if !ef.Vars[1].IsPlaceholder {
		t.Error("TODO should be placeholder")
	}
	if !ef.Vars[2].IsPlaceholder {
		t.Error("your_api_key_here should be placeholder")
	}
	if ef.Vars[3].IsPlaceholder {
		t.Error("real_value should not be placeholder")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	// Write initial file
	os.WriteFile(path, []byte("FOO=bar\n"), 0644)

	ef, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ef.UpdateVar(0, "baz")
	if err := WriteFile(ef); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "FOO=baz\n" {
		t.Errorf("expected %q, got %q", "FOO=baz\n", string(data))
	}
}

func TestExportPrefixWriteBack(t *testing.T) {
	input := "export FOO=bar\n"
	ef := ParseBytes(".env", []byte(input))
	ef.UpdateVar(0, "baz")

	output := string(Marshal(ef))
	expected := "export FOO=baz\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestInlineCommentWriteBack(t *testing.T) {
	input := "FOO=bar # comment\n"
	ef := ParseBytes(".env", []byte(input))
	ef.UpdateVar(0, "baz")

	output := string(Marshal(ef))
	expected := "FOO=baz # comment\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestVarByKey(t *testing.T) {
	input := "FOO=first\nFOO=second\n"
	ef := ParseBytes(".env", []byte(input))
	v := ef.VarByKey("FOO")
	if v == nil || v.Value != "second" {
		t.Error("VarByKey should return last occurrence")
	}
	v = ef.VarByKey("MISSING")
	if v != nil {
		t.Error("VarByKey should return nil for missing key")
	}
}
