package toastpolicy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

// acknowledgementTypes are the only toast kinds the product still uses. Both
// report something that already worked, so losing one costs the reader nothing
// but a confirmation; "error" and "warning" would put a failure back in the
// floating layer.
var acknowledgementTypes = map[string]bool{"success": true, "info": true}

// lit reports the unquoted value of a string literal, or ok false for anything
// that is not one: a caller that computes the text chooses it in one place, and
// that place is checked instead.
func lit(e ast.Expr) (value string, isLit bool) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}

// toastType is the toast's Type field inside an AddToast literal.
func toastType(call *ast.CallExpr) (ast.Expr, bool) {
	for _, arg := range call.Args {
		composite, ok := arg.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range composite.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Type" {
				return kv.Value, true
			}
		}
	}
	return nil, false
}

// violations lists the toast sites in one file's source that put a failure into
// the floating layer. rel is the file's repository-relative path, used only to
// report where each site is.
func violations(rel, src string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil
	}
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		where := rel + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
		switch calleeName(call) {
		case "Toast", "ToastAction":
			found = append(found, where+": "+calleeName(call)+" is the untyped floating bar; "+
				"an acknowledgement goes through widgets.Notice and a failure through widgets.Alert")
		case "AddToast":
			value, ok := toastType(call)
			if !ok {
				found = append(found, where+": AddToast with no Type; "+
					"a toast without a kind is the floating bar a failure hides in — "+
					"acknowledgements set Type \"success\" or \"info\", and a failure is drawn inline instead")
				return true
			}
			text, isLit := lit(value)
			if !isLit {
				found = append(found, where+": AddToast with a computed Type; "+
					"a toast whose kind is decided at run time can carry \"error\" — draw the failure inline instead")
			} else if !acknowledgementTypes[text] {
				found = append(found, where+": "+strconv.Quote(text)+" toast; "+
					"only \"success\" and \"info\" acknowledgements stay in the toast layer, "+
					"and a failure is drawn where it happened with widgets.Alert")
			}
		}
		return true
	})
	return found
}
