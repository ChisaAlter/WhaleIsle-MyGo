// Package uipurity scans the view layer for work that does not belong in the
// build path. The ratchet test compares per-file counts against a locked
// baseline so existing debt cannot grow and every improvement is recorded.
package uipurity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The rules counted per file.
const (
	// RuleImport counts imports of packages that do I/O or touch the OS: a
	// build path must read prepared snapshots, never disk, network or Win32.
	RuleImport = "import"
	// RuleSnapshot counts zero-argument X.Snapshot() calls: taking a fresh
	// deep copy inside a build path is per-frame work.
	RuleSnapshot = "snapshot"
	// RuleGo counts go statements: background work belongs to an owning
	// module with a lifecycle, not a goroutine spawned from view code.
	RuleGo = "go"
	// RuleTimer counts time.AfterFunc/NewTicker/NewTimer/Tick calls.
	RuleTimer = "timer"
)

// ruleOrder fixes the baseline's rule sort order.
var ruleOrder = []string{RuleGo, RuleImport, RuleSnapshot, RuleTimer}

var deniedImports = map[string]bool{
	"os":                       true,
	"os/exec":                  true,
	"syscall":                  true,
	"golang.org/x/sys/windows": true,
	"net/http":                 true,
	"io/ioutil":                true,
}

var timerCalls = map[string]bool{
	"AfterFunc": true,
	"NewTicker": true,
	"NewTimer":  true,
	"Tick":      true,
}

// Counts maps a rule name to the number of matches in one file.
type Counts map[string]int

func (c Counts) add(rule string) { c[rule] = c[rule] + 1 }

// countFile parses one Go source file and counts the ratcheted rules.
func countFile(path string) (Counts, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	return countAST(f), nil
}

// countSource is the in-memory form used by unit tests: no filesystem, same
// rules as countFile.
func countSource(name, src string) (Counts, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	return countAST(f), nil
}

func countAST(f *ast.File) Counts {
	out := Counts{}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if deniedImports[path] {
			out.add(RuleImport)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GoStmt:
			out.add(RuleGo)
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			if sel.Sel.Name == "Snapshot" && len(n.Args) == 0 {
				out.add(RuleSnapshot)
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "time" && timerCalls[sel.Sel.Name] {
				out.add(RuleTimer)
			}
		}
		return true
	})
	return out
}

// Entry is one baseline line: a file, a rule, the allowed count.
type Entry struct {
	File  string
	Rule  string
	Count int
}

// Scan walks dir (usually internal/ui) and returns one entry per file+rule
// with a non-zero count, in baseline order. Test files are skipped: tests may
// touch what production code may not.
func Scan(dir string) ([]Entry, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []Entry
	for _, path := range files {
		counts, err := countFile(path)
		if err != nil {
			return nil, err
		}
		rel := filepath.ToSlash(path)
		for _, rule := range ruleOrder {
			if counts[rule] > 0 {
				out = append(out, Entry{File: rel, Rule: rule, Count: counts[rule]})
			}
		}
	}
	return out, nil
}

// Format serializes entries as sorted `file<TAB>rule<TAB>count` lines.
func Format(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.File)
		b.WriteByte('\t')
		b.WriteString(e.Rule)
		b.WriteByte('\t')
		b.WriteString(strconv.Itoa(e.Count))
		b.WriteByte('\n')
	}
	return b.String()
}

// Parse reads a baseline back into entries keyed by file+rule.
func Parse(text string) (map[string]Entry, error) {
	out := map[string]Entry{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			return nil, &ParseError{Line: i + 1, Text: line}
		}
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, &ParseError{Line: i + 1, Text: line}
		}
		e := Entry{File: parts[0], Rule: parts[1], Count: n}
		out[e.File+"\x00"+e.Rule] = e
	}
	return out, nil
}

// ParseError marks a malformed baseline line.
type ParseError struct {
	Line int
	Text string
}

func (e *ParseError) Error() string {
	return "uipurity baseline line " + strconv.Itoa(e.Line) + ": malformed " + strconv.Quote(e.Text)
}
