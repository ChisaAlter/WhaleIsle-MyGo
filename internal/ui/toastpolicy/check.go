package toastpolicy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}

// receiverIs reports whether call is name(...) on the named package ident —
// widgets.Notice(c, …), ui.Toast{…} — so an unrelated field or method of the
// same spelling elsewhere does not read as a toast.
func receiverIs(call *ast.CallExpr, pkg, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg
}

// violations lists the toast sites in one file's source. The product has no
// floating toast layer: a confirmation lands on the control or status seat
// that caused it (widgets.AckLine/widgets.AckLive/widgets.StatusLine), and a
// failure is drawn where it happened (widgets.Alert). The framework's toast
// primitives stay available in third_party/mygo; nothing in the app may use
// them. rel is the file's repository-relative path, used only to report where
// each site is.
func violations(rel, src string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil
	}
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			where := rel + ":" + strconv.Itoa(fset.Position(node.Pos()).Line)
			switch calleeName(node) {
			case "AddToast", "Toast", "ToastAction":
				found = append(found, where+": "+calleeName(node)+" is the floating toast layer; "+
					"a confirmation goes on its control or status seat (widgets.AckLive/widgets.AckLine/widgets.StatusLine) "+
					"and a failure is drawn where it happened (widgets.Alert)")
			case "Notice":
				if receiverIs(node, "widgets", "Notice") {
					found = append(found, where+": widgets.Notice was removed with the toast layer; "+
						"use widgets.AckLine for an outcome or widgets.StatusLine for a live condition")
				}
			}
		case *ast.CompositeLit:
			sel, ok := node.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Toast" {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "ui" {
				where := rel + ":" + strconv.Itoa(fset.Position(node.Pos()).Line)
				found = append(found, where+": ui.Toast{...} is the floating toast layer; "+
					"a confirmation goes on its control or status seat and a failure is drawn where it happened")
			}
		}
		return true
	})
	return found
}
