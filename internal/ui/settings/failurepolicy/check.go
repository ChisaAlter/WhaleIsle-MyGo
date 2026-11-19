// Package failurepolicy keeps a settings failure's own text out of the sentence
// a person reads.
//
// A settings surface reports a failed read or write as a sentence the view
// wrote: what did not happen and what to do next. The store's text — a path, a
// status code, an err.Error() in English — is evidence, and it belongs behind
// the alert's 详情 disclosure or in the log, never in the sentence. The checker
// below finds the shapes that put it in the sentence so a reviewer does not
// have to.
package failurepolicy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

// rawFailureFields are the store and Host fields that carry a failure's own
// text: err.Error(), an exit code, a path a person cannot act on. They are
// evidence, not copy, so none of them may stand as a sentence.
var rawFailureFields = map[string]bool{
	"Err":     true,
	"Error":   true,
	"Failure": true,
	"ErrText": true,
}

// sentenceArg is the argument position of settingsAlert's sentence:
// settingsAlert(c, id, show, sentence, text).
const sentenceArg = 3

// settingsAlertArgs is how many arguments settingsAlert takes once the sentence
// and the store's text are separate.
const settingsAlertArgs = 5

// Violations reports every place in one settings source file where a failure's
// own text reaches a position a person reads as the sentence. rel is the file's
// path, used only to report the location.
func Violations(rel, src string) []string {
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
		switch name := calleeName(call); name {
		case "settingsAlert":
			if len(call.Args) != settingsAlertArgs {
				found = append(found, where+": settingsAlert takes the sentence and the store's text as separate arguments; "+
					"pass the view's own sentence and the raw failure as its fifth argument so the raw text lands behind 详情")
				return true
			}
			if isRawFailure(call.Args[sentenceArg]) {
				found = append(found, where+": settingsAlert sentence is the store's own failure text; "+
					"write what failed and what to do next, and pass the raw text as the fifth argument")
			}
		case "Alert":
			if message := compositeField(call, "Message"); message != nil && isRawFailure(message) {
				found = append(found, where+": Alert Message is the store's own failure text; "+
					"write what failed and what to do next, and put the raw text in Detail")
			}
		case "Text":
			if len(call.Args) > 1 && isRawFailure(call.Args[1]) {
				found = append(found, where+": Text draws the store's own failure text as the sentence; "+
					"write what failed and what to do next, and put the raw text behind a 详情 disclosure")
			}
		}
		return true
	})
	return found
}

func calleeName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.Ident:
		return fun.Name
	}
	return ""
}

// compositeField returns the value of key in the struct literal the call passes,
// or nil when the call passes no such literal.
func compositeField(call *ast.CallExpr, key string) ast.Expr {
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == key {
				return kv.Value
			}
		}
	}
	return nil
}

// isRawFailure reports whether expr reads a failure's own text: a store field
// such as state.Err, or an err.Error() call.
func isRawFailure(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return rawFailureFields[e.Sel.Name]
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "Error"
	case *ast.ParenExpr:
		return isRawFailure(e.X)
	}
	return false
}
