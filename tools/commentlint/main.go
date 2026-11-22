// Command commentlint ratchets the number of banned comment lines in each Go
// file of the lazyenv module down to zero.
//
// Allowed: directives (//go:build, //exhaustive:enforce, …), the package doc
// comment, and doc comments on identifiers godoc shows (exported names, and the
// fields and methods of exported types) outside _test.go files. Every other
// comment line is banned.
// Each file may hold at most the banned lines its baseline entry records; the
// check also fails when a file holds fewer, until -update locks in the drop.
//
// -list prints each banned comment group as path:line: first line (+more lines).
//
// Usage: commentlint [-update | -list] [-baseline FILE] [ROOT]
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	update := flag.Bool("update", false, "rewrite the baseline with the current counts")
	list := flag.Bool("list", false, "print every banned comment instead of checking the baseline")
	baseline := flag.String("baseline", "tools/commentlint/baseline.txt", "baseline file, relative to ROOT")
	flag.Parse()
	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	var err error
	if *list {
		err = listTree(root, os.Stdout)
	} else {
		err = run(root, filepath.Join(root, *baseline), *update)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, baselinePath string, update bool) error {
	found, err := scanTree(root)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for file, groups := range found {
		counts[file] = sumLines(groups)
	}
	if update {
		return writeBaseline(baselinePath, counts)
	}
	base, err := readBaseline(baselinePath)
	if err != nil {
		return err
	}
	problems := compare(counts, base)
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "\n"))
}

func listTree(root string, w io.Writer) error {
	found, err := scanTree(root)
	if err != nil {
		return err
	}
	for _, file := range sortedKeys(found) {
		for _, g := range found[file] {
			more := ""
			if g.lines > 1 {
				more = fmt.Sprintf(" (+%d more lines)", g.lines-1)
			}
			fmt.Fprintf(w, "%s:%d: %s%s\n", file, g.line, g.first, more)
		}
	}
	return nil
}

type banned struct {
	line, lines int
	first       string
}

func sumLines(groups []banned) int {
	n := 0
	for _, g := range groups {
		n += g.lines
	}
	return n
}

func scanTree(root string) (map[string][]banned, error) {
	found := map[string][]banned{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDir(root, path, d.Name())
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		groups, err := bannedComments(path, src)
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			rel, _ := filepath.Rel(root, path)
			found[filepath.ToSlash(rel)] = groups
		}
		return nil
	})
	return found, err
}

func skipDir(root, path, name string) error {
	if path == root {
		return nil
	}
	if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
		return filepath.SkipDir
	}
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return filepath.SkipDir
	}
	return nil
}

var (
	generated = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)
	directive = regexp.MustCompile(`^//([a-z0-9]+:[a-z0-9]|line |extern |export )`)
)

func isDirective(text string) bool {
	return directive.MatchString(text) || generated.MatchString(text)
}

func bannedComments(path string, src []byte) ([]banned, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	allowed := map[*ast.CommentGroup]bool{}
	if !strings.HasSuffix(path, "_test.go") {
		allowed = godocGroups(f)
	}
	var groups []banned
	for _, g := range f.Comments {
		if allowed[g] {
			continue
		}
		b := banned{}
		for _, c := range g.List {
			if isDirective(c.Text) {
				continue
			}
			if b.lines == 0 {
				b.line = fset.Position(c.Pos()).Line
				b.first, _, _ = strings.Cut(c.Text, "\n")
			}
			b.lines += fset.Position(c.End()).Line - fset.Position(c.Pos()).Line + 1
		}
		if b.lines > 0 {
			groups = append(groups, b)
		}
	}
	return groups, nil
}

func godocGroups(f *ast.File) map[*ast.CommentGroup]bool {
	groups := map[*ast.CommentGroup]bool{}
	keep := func(gs ...*ast.CommentGroup) {
		for _, g := range gs {
			if g != nil {
				groups[g] = true
			}
		}
	}
	keep(f.Doc)
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() && exportedReceiver(d.Recv) {
				keep(d.Doc)
			}
		case *ast.GenDecl:
			if genDeclDocs(d, keep) {
				keep(d.Doc)
			}
		}
	}
	return groups
}

func genDeclDocs(d *ast.GenDecl, keep func(...*ast.CommentGroup)) bool {
	exported := false
	for _, s := range d.Specs {
		switch s := s.(type) {
		case *ast.TypeSpec:
			if s.Name.IsExported() {
				exported = true
				keep(s.Doc, s.Comment)
				memberDocs(s.Type, keep)
			}
		case *ast.ValueSpec:
			if anyExported(s.Names) {
				exported = true
				keep(s.Doc, s.Comment)
			}
		}
	}
	return exported
}

func memberDocs(t ast.Expr, keep func(...*ast.CommentGroup)) {
	var fields *ast.FieldList
	switch t := t.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	default:
		return
	}
	for _, field := range fields.List {
		if anyExported(field.Names) || (len(field.Names) == 0 && embedsExported(field.Type)) {
			keep(field.Doc, field.Comment)
			memberDocs(field.Type, keep)
		}
	}
}

func anyExported(names []*ast.Ident) bool {
	for _, n := range names {
		if n.IsExported() {
			return true
		}
	}
	return false
}

func embedsExported(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return t.IsExported()
	case *ast.StarExpr:
		return embedsExported(t.X)
	case *ast.SelectorExpr:
		return t.Sel.IsExported()
	case *ast.IndexExpr:
		return embedsExported(t.X)
	case *ast.IndexListExpr:
		return embedsExported(t.X)
	}
	return false
}

func exportedReceiver(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) == 0 {
		return true
	}
	return embedsExported(recv.List[0].Type)
}

func readBaseline(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}
	defer f.Close()
	base := map[string]int{}
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		countText, file, ok := strings.Cut(text, " ")
		n, err := strconv.Atoi(countText)
		if !ok || err != nil {
			return nil, fmt.Errorf("%s:%d: want \"<count> <path>\", got %q", path, line, text)
		}
		base[file] = n
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}
	return base, nil
}

func writeBaseline(path string, counts map[string]int) error {
	var b strings.Builder
	b.WriteString("# Banned comment lines allowed per file; only ever lower these.\n")
	b.WriteString("# Regenerate with: just comment-baseline\n")
	for _, file := range sortedKeys(counts) {
		fmt.Fprintf(&b, "%d %s\n", counts[file], file)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing baseline: %w", err)
	}
	return nil
}

func compare(counts, base map[string]int) []string {
	files := map[string]int{}
	for f := range counts {
		files[f] = 0
	}
	for f := range base {
		files[f] = 0
	}
	var problems []string
	for _, file := range sortedKeys(files) {
		got, want := counts[file], base[file]
		switch {
		case got > want:
			problems = append(problems, fmt.Sprintf("%s: %d banned comment lines, baseline allows %d: delete the new comments; code moved here brings its count (CODING_STANDARDS.md, Comments)", file, got, want))
		case got < want:
			problems = append(problems, fmt.Sprintf("%s: %d banned comment lines, baseline allows %d: run `just comment-baseline` to lock in the drop", file, got, want))
		}
	}
	return problems
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
