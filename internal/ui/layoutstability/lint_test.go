package layoutstability

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A line of text that is inserted into a page while something loads, saves or applies, and
// removed when it is done, moves everything below it twice. Progress belongs in the state of
// the controls (disabled, busy), in a toast, or in an overlay; never in the flow of a page.
//
// This test walks the UI sources and rejects `if <loading condition> { ui.Text(...) }` where the
// text is simply added to the flow. Placeholders that replace their content (an if with an else or
// a return) and texts inside popups, dialogs and menus are not the problem and are not flagged.

var (
	stateWord = regexp.MustCompile(`(?i)(loading|saving|busy|applying|checking|syncing|preparing|updating|downloading|installing|verifying|refreshing|connecting)`)
	negated   = regexp.MustCompile(`(?i)!\s*[\w\.\*\(\)]*(loading|saving|busy|applying|checking|syncing|preparing|updating|downloading|installing|verifying|refreshing|connecting)`)
	// "Saved" / "Copied" acknowledgements arrive and clear by themselves: the same jump.
	noticeCond = regexp.MustCompile(`(?i)\.\w*(notice|lastaction)\w*\s*!=\s*""`)
	overlay    = regexp.MustCompile(`(?i)(popup|dialog|modal|overlay|popover|menu|toast|tooltip|floating|sheet|drawer|picker|attach)`)
)

func code(fset *token.FileSet, n ast.Node) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func hasReturn(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			found = true
		}
		return !found
	})
	return found
}

func TestNoLoadingTextIsInsertedIntoTheFlow(t *testing.T) {
	root := ".." // internal/ui
	var violations []string
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)
			s, ok := n.(*ast.IfStmt)
			if !ok || s.Else != nil || len(s.Body.List) != 1 || hasReturn(s.Body) {
				return true
			}
			cond := code(fset, s.Cond)
			if !(stateWord.MatchString(cond) && !negated.MatchString(cond)) && !noticeCond.MatchString(cond) {
				return true
			}
			es, ok := s.Body.List[0].(*ast.ExprStmt)
			if !ok {
				return true
			}
			call, ok := es.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			if !startsWithUIText(call) {
				return true
			}
			if chainHas(call, "Absolute", "Attach", "AttachTo") {
				return true // floating: it takes no space in the flow
			}
			// an else-if arm is part of a chain of alternatives, not an addition
			if len(stack) >= 2 {
				if parent, ok := stack[len(stack)-2].(*ast.IfStmt); ok && parent.Else == ast.Stmt(s) {
					return true
				}
			}
			for i := len(stack) - 2; i >= 0; i-- {
				switch a := stack[i].(type) {
				case *ast.CallExpr:
					switch fn := a.Fun.(type) {
					case *ast.SelectorExpr:
						if overlay.MatchString(fn.Sel.Name) {
							return true
						}
					case *ast.Ident:
						if overlay.MatchString(fn.Name) {
							return true
						}
					}
				case *ast.FuncDecl:
					if overlay.MatchString(a.Name.Name) {
						return true
					}
				}
			}
			p := fset.Position(s.Pos())
			violations = append(violations, filepath.ToSlash(p.Filename)+":"+itoa(p.Line)+": if "+cond)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("loading/saving text inserted into the page flow (it makes the UI jump; show nothing, a toast, or an overlay):\n%s", strings.Join(violations, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

// startsWithUIText reports whether call is ui.Text(...) with any chain of methods after it.
func startsWithUIText(call *ast.CallExpr) bool {
	for {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			return id.Name == "ui" && sel.Sel.Name == "Text"
		}
		next, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		call = next
	}
}

// chainHas reports whether the method chain of call names one of methods.
func chainHas(call *ast.CallExpr, methods ...string) bool {
	for {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		for _, m := range methods {
			if sel.Sel.Name == m {
				return true
			}
		}
		next, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		call = next
	}
}
