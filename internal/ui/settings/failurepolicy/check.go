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
			if isCauseText(call.Args[sentenceArg]) {
				found = append(found, where+": settingsAlert draws the cause as its message; "+
					"write what failed and what to do next, and put the cause behind 详情")
			}
		case "Alert":
			if message := compositeField(call, "Message"); message != nil && isRawFailure(message) {
				found = append(found, where+": Alert Message is the store's own failure text; "+
					"write what failed and what to do next, and put the raw text in Detail")
			}
			if message := compositeField(call, "Message"); message != nil && isCauseText(message) {
				found = append(found, where+": Alert Message is the cause; "+
					"write what failed and what to do next, and put the cause in Detail")
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

// errcopyNames are the classifiers that turn a failure into the sentence a
// person reads. A surface that calls one has the error in hand, so it also has
// the text 详情 should show.
var errcopyNames = map[string]bool{
	"Sentence":     true,
	"SentenceText": true,
	"SentenceFor":  true,
}

// classifiedSeats are the names that mean the surface already has the cause in
// hand: a field a store classified before the page drew it, and the sentence half
// of a settingsRefusal, whose other half is the detail. Showing one of these with
// an empty detail has dropped the cause on purpose or by accident.
var classifiedSeats = map[string]bool{"Status": true, "sentence": true}

// DroppedCause reports a settings surface that showed a classified failure as
// its sentence and passed nothing to 详情. The person reads what happened but
// cannot open the cause, and whoever they report it to has nothing to point at.
func DroppedCause(rel, src string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil
	}
	classified := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) || !isClassification(rhs) {
				continue
			}
			if name := seatName(assign.Lhs[i]); name != "" {
				classified[name] = true
			}
		}
		return true
	})
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := calleeName(call)
		if name != "settingsAlert" && name != "settingsFailureLine" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		detail := call.Args[len(call.Args)-1]
		if !isEmptyString(detail) {
			return true
		}
		sentence := seatName(call.Args[len(call.Args)-2])
		if !classified[sentence] && !classifiedSeats[sentence] {
			return true
		}
		where := rel + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
		found = append(found, where+": "+name+" shows the sentence "+sentence+" and passes nothing to 详情; "+
			"keep the error and pass its own text as the last argument so the cause opens under the sentence")
		return true
	})
	return found
}

// isClassification reports whether expr is a call to an errcopy classifier.
func isClassification(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "errcopy" && errcopyNames[sel.Sel.Name]
}

// seatName is the name a surface keeps a sentence under: failure, *failure,
// state.Err, or editor.Status.
func seatName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return seatName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.ParenExpr:
		return seatName(e.X)
	}
	return ""
}

// isEmptyString reports whether expr is the empty string literal.
func isEmptyString(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `""`
}

// UntranslatedStatus reports a theme editor status written in code rather than
// in the interface language. A status is the message a person reads when the
// editor refuses or fails, so a literal here is copy that never gets translated.
func UntranslatedStatus(rel, src string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil
	}
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) || seatName(assign.Lhs[i]) != "Status" {
				continue
			}
			if !writesCopyInCode(rhs) {
				continue
			}
			where := rel + ":" + strconv.Itoa(fset.Position(assign.Pos()).Line)
			found = append(found, where+": Status is written in code; "+
				"register the sentence with i18n and set it with i18n.T, or refuse the preference through invalid() so the words are translated")
		}
		return true
	})
	return found
}

// writesCopyInCode reports whether expr builds a sentence from a string literal
// with words in it.
func writesCopyInCode(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Kind == token.STRING && hasCopyWords(e.Value)
	case *ast.CallExpr:
		if calleeName(e) != "Sprintf" || len(e.Args) == 0 {
			return false
		}
		lit, ok := e.Args[0].(*ast.BasicLit)
		return ok && lit.Kind == token.STRING && hasCopyWords(lit.Value)
	}
	return false
}

// hasCopyWords reports whether a Go string literal carries words: ASCII
// letters or CJK characters. A key, an ID or a format of punctuation alone has
// none, so a literal with them is copy that never reaches the translator.
func hasCopyWords(lit string) bool {
	for _, r := range lit {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			return true
		case r >= 0x4E00 && r <= 0x9FFF, r >= 0x3400 && r <= 0x4DBF:
			return true
		}
	}
	return false
}

// StoreCopy reports a store that words a refusal itself. The store names the
// problem; the sentence is copy in the interface language and belongs to the
// surface that draws it, so a literal here is a sentence in the wrong layer.
func StoreCopy(rel, src string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil
	}
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		where := rel + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
		switch calleeName(call) {
		case "New":
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && toString(sel.X) == "errors" && isWordy(call.Args[0]) {
				found = append(found, where+": the store words a refusal itself; "+
					"name the problem with a Problem constant and pass the store's own text to refuse() so the surface writes the sentence")
			}
		case "Errorf":
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && toString(sel.X) == "fmt" && isWordy(call.Args[0]) {
				found = append(found, where+": the store words a refusal itself; "+
					"name the problem with a Problem constant and pass the store's own text to refuse() so the surface writes the sentence")
			}
		}
		return true
	})
	return found
}

// isWordy reports whether expr is a string literal carrying words: the marker
// that the literal is a sentence, not a key or a format of punctuation.
func isWordy(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && hasCopyWords(lit.Value)
}

func toString(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// UnnamedProblems reports a Problem the store can name that the panel view has
// no sentence for. A missing key would not leak anything; it would quietly turn
// a precise refusal back into the generic failure sentence.
func UnnamedProblems(storeSrc, viewSrc string) []string {
	named := problemConstants(storeSrc)
	if len(named) == 0 {
		return nil
	}
	worded := map[string]bool{}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "view.go", viewSrc, 0)
	if err != nil {
		return nil
	}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if sel, ok := kv.Key.(*ast.SelectorExpr); ok {
				worded[sel.Sel.Name] = true
			}
		}
		return true
	})
	var found []string
	for _, name := range named {
		if !worded[name] {
			found = append(found, "internal/store/custompanel: "+name+
				" has no sentence in internal/ui/custompanel; register one and add it to problemKeys, or the panel falls back to the generic failure line")
		}
	}
	return found
}

// problemConstants lists the constants the store declares with the Problem type.
func problemConstants(src string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "store.go", src, 0)
	if err != nil {
		return nil
	}
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		if ident, ok := spec.Type.(*ast.Ident); !ok || ident.Name != "Problem" {
			return true
		}
		for _, name := range spec.Names {
			names = append(names, name.Name)
		}
		return true
	})
	return names
}

// causeSeats are the names a surface keeps a cause under, for the 详情
// disclosure. Drawing one as the message turns the evidence back into the
// sentence: the person reads the store's words and nothing says what to do.
var causeSeats = map[string]bool{"Detail": true, "detail": true}

// isCauseText reports whether expr reads a failure's cause rather than its sentence.
func isCauseText(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return causeSeats[e.Sel.Name]
	case *ast.ParenExpr:
		return isCauseText(e.X)
	}
	return false
}
