package errcopy_test

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// updateCarriers rewrites testdata/carriers_baseline.txt with the current
// counts: `go test ./internal/errcopy -run CarriersRatchet -update`.
var updateCarriers = flag.Bool("update-carriers", false, "rewrite the carriers ratchet baseline")

// ratchetDirs are the layers whose view state must speak classified
// failures (internal/state.Fault), never a raw .Error() string. The scan is
// a ratchet: every known violation is counted, new ones fail the test, and
// shrinking a count without -update also fails so the baseline locks each
// improvement in.
var ratchetDirs = []string{"../store", "../app", "../ui/sidebar"}

// carrierViolation is one assignment of a failure's own text into a field:
// x.Err = err.Error(), Err: err.Error(), or a fmt string built from an
// error. These are the shapes a view-state/snapshot struct uses to carry a
// failure, and they are exactly what docs/error-copy.md forbids there.
type carrierViolation struct {
	file string
	line int
	rule string // "rawerr" for .Error(), "fmt" for Sprintf/Errorf/Sprint of an error
	text string
}

// errCarrierInspect flags assignment and composite-literal values built
// from an error's own text.
func errCarrierInspect(file *ast.File, fset *token.FileSet, name string, out *[]carrierViolation) {
	flagValue := func(e ast.Expr, rule string) {
		pos := fset.Position(e.Pos())
		var b strings.Builder
		_ = format.Node(&b, fset, e)
		*out = append(*out, carrierViolation{
			file: name, line: pos.Line, rule: rule,
			text: strings.TrimSpace(b.String()),
		})
	}
	scan := func(e ast.Expr) {
		var rule string
		ast.Inspect(e, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch {
			case sel.Sel.Name == "Error" && len(call.Args) == 0:
				rule = "rawerr"
			case isFmtTextCall(sel) && mentionsError(call.Args):
				rule = "fmt"
			}
			return true
		})
		if rule != "" {
			flagValue(e, rule)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, rhs := range x.Rhs {
				scan(rhs)
			}
		case *ast.KeyValueExpr:
			scan(x.Value)
		case *ast.ValueSpec:
			for _, v := range x.Values {
				scan(v)
			}
		}
		return true
	})
}

func isFmtTextCall(sel *ast.SelectorExpr) bool {
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "fmt" {
		return false
	}
	switch sel.Sel.Name {
	case "Sprint", "Sprintf", "Sprintln", "Errorf":
		return true
	}
	return false
}

// mentionsError reports whether an argument list builds text from an error:
// an identifier that names an error variable or a nested .Error() call.
func mentionsError(args []ast.Expr) bool {
	found := false
	for _, arg := range args {
		ast.Inspect(arg, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				l := strings.ToLower(x.Name)
				if l == "err" || strings.HasSuffix(l, "err") || strings.HasPrefix(l, "err") {
					found = true
					return false
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Error" && len(x.Args) == 0 {
					found = true
					return false
				}
			}
			return true
		})
	}
	return found
}

func collectCarrierViolations(t *testing.T) []carrierViolation {
	t.Helper()
	var out []carrierViolation
	for _, dir := range ratchetDirs {
		root := filepath.Clean(dir)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			errCarrierInspect(file, fset, filepath.ToSlash(path), &out)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestCarriersRatchet(t *testing.T) {
	violations := collectCarrierViolations(t)
	counts := map[string]int{}
	for _, v := range violations {
		counts[v.file+"\t"+v.rule]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	baselinePath := filepath.Join("testdata", "carriers_baseline.txt")
	if *updateCarriers {
		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "%s\t%d\n", k, counts[k])
		}
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(baselinePath, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s with %d entries", baselinePath, len(keys))
		return
	}

	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v (run with -update-carriers to create it)", err)
	}
	baseline := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		tab := strings.LastIndex(line, "\t")
		if tab < 0 {
			t.Fatalf("bad baseline line %q", line)
		}
		var n int
		if _, err := fmt.Sscanf(line[tab+1:], "%d", &n); err != nil {
			t.Fatalf("bad baseline line %q: %v", line, err)
		}
		baseline[line[:tab]] = n
	}

	failed := false
	for _, k := range keys {
		base, ok := baseline[k]
		switch {
		case !ok:
			t.Errorf("new raw-error carrier in %s (count %d): view state must carry a classified Fault, not the failure's own text", k, counts[k])
			failed = true
		case counts[k] > base:
			t.Errorf("%s grew: %d > baseline %d — view state must carry a classified Fault, not the failure's own text", k, counts[k], base)
			failed = true
		case counts[k] < base:
			t.Errorf("%s shrank: %d < baseline %d — run `go test ./internal/errcopy -run CarriersRatchet -update-carriers` to lock the improvement in", k, counts[k], base)
			failed = true
		}
	}
	for k := range baseline {
		if _, ok := counts[k]; !ok {
			t.Errorf("%s is clean now — run `go test ./internal/errcopy -run CarriersRatchet -update-carriers` to lock the improvement in", k)
			failed = true
		}
	}
	if failed {
		for _, v := range violations {
			t.Logf("  %s:%d [%s] %s", v.file, v.line, v.rule, v.text)
		}
	}
}
