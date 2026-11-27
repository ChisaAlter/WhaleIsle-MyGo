package designguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// interactiveLayoutShiftWhitelist lists verified zero-sibling-shift fixed-seat
// decorative elements that cannot alter the position or size of any sibling:
//   - chat/turn_rail.go: fixed turnRailWidth right-aligned tick indicator
//   - sidebar/rail.go: fixed 36×36 collapsed-rail toggle swapping avatar/icon
var interactiveLayoutShiftWhitelist = map[string]bool{
	"chat/turn_rail.go": true,
	"sidebar/rail.go":   true,
}

var (
	interactiveStateMethods = map[string]bool{
		"Hovered":      true,
		"FocusVisible": true,
		"FocusWithin":  true,
		"Pressed":      true,
	}

	// geometryMethods alter an element's box size, flow participation, or text
	// metrics. Calling them conditionally on hover/focus/pressed shifts siblings
	// or causes jitter. BorderColor, Background, TextColor, Opacity, and Shadow
	// are explicitly allowed.
	geometryMethods = map[string]bool{
		"Width":       true,
		"Height":      true,
		"Size":        true,
		"MinWidth":    true,
		"MinHeight":   true,
		"MaxWidth":    true,
		"MaxHeight":   true,
		"Padding":     true,
		"PaddingX":    true,
		"PaddingY":    true,
		"Margin":      true,
		"MarginX":     true,
		"MarginY":     true,
		"Gap":         true,
		"GapX":        true,
		"GapY":        true,
		"Border":      true,
		"BorderWidth": true,
		"FontWeight":  true,
		"FontSize":    true,
		"Absolute":    true,
	}

	// inFlowConstructors create layout-participating elements when called without
	// .Absolute() outside overlay/popover/dialog/tooltip containers.
	inFlowUIConstructors = map[string]bool{
		"Text":       true,
		"Button":     true,
		"ButtonBase": true,
		"Icon":       true,
		"Row":        true,
		"Column":     true,
		"Box":        true,
	}
	inFlowCossConstructors = map[string]bool{
		"Button": true,
		"Kbd":    true,
		"Badge":  true,
	}
	inFlowLocalConstructors = map[string]bool{
		"kText":       true,
		"kButton":     true,
		"kIconButton": true,
		"kGhost":      true,
	}

	// overlayWrappers render out-of-flow overlays (popovers, tooltips, dialogs,
	// menus, custom painter draws) whose children cannot cause layout shifts in
	// the underlying flow tree.
	overlayWrappers = map[string]bool{
		"Draw":                  true,
		"DrawOver":              true,
		"PopoverBase":           true,
		"Tooltip":               true,
		"DialogBase":            true,
		"DialogBaseWithOptions": true,
		"Menu":                  true,
		"Popover":               true,
		"Dialog":                true,
		"AlertDialog":           true,
	}
)

type parsedFile struct {
	rel  string
	path string
	fs   *token.FileSet
	file *ast.File
}

// animShiftHelper describes a package helper whose boolean parameter at
// ParamIndex drives an AnimateWith call fed into a geometry setter (e.g.
// Width(width * p)). Calling it with a hover/focus/pressed value shifts layout.
type animShiftHelper struct {
	File       string
	Line       int
	ParamIndex int
	Method     string
}

func TestNoInteractiveLayoutShift(t *testing.T) {
	root := filepath.Join("..")
	byDir := map[string][]parsedFile{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			switch rel {
			case "design", "coss", "designguard":
				return filepath.SkipDir
			}
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		if interactiveLayoutShiftWhitelist[rel] {
			return nil
		}
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, cg := range f.Comments {
			if strings.Contains(cg.Text(), "Code generated") {
				return nil
			}
		}
		dir := filepath.Dir(rel)
		byDir[dir] = append(byDir[dir], parsedFile{rel: rel, path: path, fs: fs, file: f})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var violations []guardViolation
	for _, files := range byDir {
		violations = append(violations, scanPackageInteractiveLayoutShift(files)...)
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return violations[i].Line < violations[j].Line
	})
	if len(violations) > 0 {
		lines := make([]string, len(violations))
		for i, v := range violations {
			lines[i] = v.String()
		}
		t.Fatalf("interactive (hover/focus/pressed) layout shift detected (keep geometry/weight/flow children constant; toggle Opacity/Background/BorderColor/Shadow or use an Absolute overlay instead):\n  %s", strings.Join(lines, "\n  "))
	}
}

func scanInteractiveLayoutShift(path, rel string) ([]guardViolation, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, cg := range f.Comments {
		if strings.Contains(cg.Text(), "Code generated") {
			return nil, nil
		}
	}
	return scanPackageInteractiveLayoutShift([]parsedFile{{rel: rel, path: path, fs: fs, file: f}}), nil
}

func scanPackageInteractiveLayoutShift(files []parsedFile) []guardViolation {
	// Pass 1: find package helpers where a bool param drives AnimateWith -> geometry.
	helpers := map[string]animShiftHelper{}
	for _, pf := range files {
		for _, decl := range pf.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || fn.Type.Params == nil {
				continue
			}
			if h, ok := detectAnimShiftHelper(pf.fs, pf.rel, fn); ok {
				helpers[fn.Name.Name] = h
			}
		}
	}

	// Pass 2: inspect every function body for direct interactive shifts and
	// interactive arguments passed into animShiftHelpers.
	var violations []guardViolation
	for _, pf := range files {
		for _, decl := range pf.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			w := &shiftWalker{
				fs:          pf.fs,
				rel:         pf.rel,
				helpers:     helpers,
				interactive: map[string]bool{},
				animVars:    map[string]bool{},
				shiftVars:   map[string]string{},
				out:         &violations,
			}
			w.walkBlock(fn.Body, false, false)
			w.checkTaintedVarUses(fn.Body)
		}
	}
	return violations
}

func detectAnimShiftHelper(fs *token.FileSet, rel string, fn *ast.FuncDecl) (animShiftHelper, bool) {
	paramIdx := map[string]int{}
	idx := 0
	for _, field := range fn.Type.Params.List {
		isBool := false
		if id, ok := field.Type.(*ast.Ident); ok && id.Name == "bool" {
			isBool = true
		}
		for _, name := range field.Names {
			if isBool {
				paramIdx[name.Name] = idx
			}
			idx++
		}
	}
	if len(paramIdx) == 0 {
		return animShiftHelper{}, false
	}

	// Track which bool param taints which local variable and AnimateWith output.
	taintedParam := map[string]int{}
	for name, i := range paramIdx {
		taintedParam[name] = i
	}
	animTainted := map[string]int{}

	exprParam := func(expr ast.Expr) (int, bool) {
		found, idx := false, -1
		ast.Inspect(expr, func(n ast.Node) bool {
			if found {
				return false
			}
			if id, ok := n.(*ast.Ident); ok {
				if p, ok := taintedParam[id.Name]; ok {
					found, idx = true, p
					return false
				}
			}
			return true
		})
		return idx, found
	}

	var walk func(stmt ast.Stmt, condParam int)
	walk = func(stmt ast.Stmt, condParam int) {
		switch s := stmt.(type) {
		case *ast.BlockStmt:
			for _, sub := range s.List {
				walk(sub, condParam)
			}
		case *ast.IfStmt:
			next := condParam
			if p, ok := exprParam(s.Cond); ok {
				next = p
			}
			walk(s.Body, next)
			if s.Else != nil {
				walk(s.Else, next)
			}
		case *ast.AssignStmt:
			if len(s.Lhs) != len(s.Rhs) {
				return
			}
			for i := range s.Lhs {
				id, ok := s.Lhs[i].(*ast.Ident)
				if !ok || id.Name == "_" {
					continue
				}
				rhs := s.Rhs[i]
				if condParam >= 0 {
					taintedParam[id.Name] = condParam
				} else if p, ok := exprParam(rhs); ok {
					taintedParam[id.Name] = p
				}
				if call, ok := rhs.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "AnimateWith" && len(call.Args) >= 2 {
						if p, ok := exprParam(call.Args[1]); ok {
							animTainted[id.Name] = p
						}
					}
				}
			}
		}
	}
	walk(fn.Body, -1)
	if len(animTainted) == 0 {
		return animShiftHelper{}, false
	}

	var hit animShiftHelper
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !geometryMethods[sel.Sel.Name] || (sel.Sel.Name != "Absolute" && len(call.Args) == 0) {
			return true
		}
		for _, arg := range call.Args {
			ast.Inspect(arg, func(an ast.Node) bool {
				if found {
					return false
				}
				if id, ok := an.(*ast.Ident); ok {
					if p, ok := animTainted[id.Name]; ok {
						hit = animShiftHelper{
							File:       rel,
							Line:       fs.Position(call.Pos()).Line,
							ParamIndex: p,
							Method:     sel.Sel.Name,
						}
						found = true
						return false
					}
				}
				return true
			})
		}
		return true
	})
	return hit, found
}

type shiftWalker struct {
	fs          *token.FileSet
	rel         string
	helpers     map[string]animShiftHelper
	interactive map[string]bool   // local vars tainted by Hovered/FocusVisible/FocusWithin/Pressed
	animVars    map[string]bool   // AnimateWith outputs driven by interactive vars
	shiftVars   map[string]string // local scalar vars mutated inside an interactive branch (e.g. weight = 600)
	out         *[]guardViolation
}

func (w *shiftWalker) addViolation(pos token.Pos, reason string) {
	*w.out = append(*w.out, guardViolation{
		File:   w.rel,
		Line:   w.fs.Position(pos).Line,
		Reason: reason,
	})
}

func (w *shiftWalker) exprUsesInteractive(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && interactiveStateMethods[sel.Sel.Name] {
				found = true
				return false
			}
		case *ast.Ident:
			if w.interactive[x.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func (w *shiftWalker) recordAssign(stmt *ast.AssignStmt, inInteractive bool) {
	if len(stmt.Lhs) != len(stmt.Rhs) {
		return
	}
	for i := range stmt.Lhs {
		id, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		rhs := stmt.Rhs[i]
		if w.exprUsesInteractive(rhs) {
			w.interactive[id.Name] = true
		}
		if call, ok := rhs.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "AnimateWith" && len(call.Args) >= 2 {
				if w.exprUsesInteractive(call.Args[1]) {
					w.animVars[id.Name] = true
				}
			}
		}
		// Plain assignment (not :=) to an outer scalar var inside an interactive
		// branch: track it so if later passed to FontWeight/Width/Padding etc.,
		// we catch indirect shifts like `case hovered: weight = 600`.
		if inInteractive && stmt.Tok == token.ASSIGN {
			if _, isLit := rhs.(*ast.BasicLit); isLit || isDesignMetricSelector(rhs) {
				if _, exists := w.shiftVars[id.Name]; !exists {
					w.shiftVars[id.Name] = fmt.Sprintf("variable %q assigned inside hover/focus/pressed branch at line %d", id.Name, w.fs.Position(stmt.Pos()).Line)
				}
			}
		}
	}
}

func isDesignMetricSelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != "design" {
		return false
	}
	name := sel.Sel.Name
	return strings.HasPrefix(name, "Weight") || name == "BorderWidth"
}

func (w *shiftWalker) walkBlock(block *ast.BlockStmt, inInteractive, inOverlay bool) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		w.walkStmt(stmt, inInteractive, inOverlay)
	}
}

func (w *shiftWalker) walkStmt(stmt ast.Stmt, inInteractive, inOverlay bool) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		w.walkBlock(s, inInteractive, inOverlay)
	case *ast.AssignStmt:
		w.recordAssign(s, inInteractive)
		for _, rhs := range s.Rhs {
			w.walkExpr(rhs, inInteractive, inOverlay)
		}
	case *ast.ExprStmt:
		w.walkExpr(s.X, inInteractive, inOverlay)
	case *ast.DeclStmt:
		if gd, ok := s.Decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for i, val := range vs.Values {
						if i < len(vs.Names) && w.exprUsesInteractive(val) {
							w.interactive[vs.Names[i].Name] = true
						}
						w.walkExpr(val, inInteractive, inOverlay)
					}
				}
			}
		}
	case *ast.IfStmt:
		if s.Init != nil {
			w.walkStmt(s.Init, inInteractive, inOverlay)
		}
		branchInteractive := inInteractive || w.exprUsesInteractive(s.Cond)
		w.walkBlock(s.Body, branchInteractive, inOverlay)
		if s.Else != nil {
			w.walkStmt(s.Else, branchInteractive, inOverlay)
		}
	case *ast.SwitchStmt:
		if s.Init != nil {
			w.walkStmt(s.Init, inInteractive, inOverlay)
		}
		tagInteractive := inInteractive || w.exprUsesInteractive(s.Tag)
		for _, clause := range s.Body.List {
			cc, ok := clause.(*ast.CaseClause)
			if !ok {
				continue
			}
			caseInteractive := tagInteractive
			for _, e := range cc.List {
				if w.exprUsesInteractive(e) {
					caseInteractive = true
					break
				}
			}
			for _, sub := range cc.Body {
				w.walkStmt(sub, caseInteractive, inOverlay)
			}
		}
	case *ast.ForStmt:
		if s.Init != nil {
			w.walkStmt(s.Init, inInteractive, inOverlay)
		}
		w.walkBlock(s.Body, inInteractive, inOverlay)
	case *ast.RangeStmt:
		w.walkBlock(s.Body, inInteractive, inOverlay)
	case *ast.ReturnStmt:
		for _, r := range s.Results {
			w.walkExpr(r, inInteractive, inOverlay)
		}
	case *ast.DeferStmt:
		w.walkExpr(s.Call, inInteractive, inOverlay)
	}
}

func (w *shiftWalker) walkExpr(expr ast.Expr, inInteractive, inOverlay bool) {
	if expr == nil {
		return
	}
	// If this expression is a chained call that creates a brand-new element and
	// marks it .Absolute() (e.g. ui.Box(c).Absolute().Top(0).Border(2, brand)),
	// that entire element is an out-of-flow overlay and does NOT shift layout.
	if call, ok := expr.(*ast.CallExpr); ok && isNewAbsoluteOverlayChain(call) {
		w.walkCallCallbacks(call, inInteractive, true)
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && !inOverlay {
			if helper, ok := w.helpers[id.Name]; ok && helper.ParamIndex < len(e.Args) {
				if inInteractive || w.exprUsesInteractive(e.Args[helper.ParamIndex]) {
					w.addViolation(e.Pos(), fmt.Sprintf("hover/focus/pressed state passed to %s(...) animates .%s(...) at %s:%d and causes layout shift", id.Name, helper.Method, helper.File, helper.Line))
				}
			}
		}
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			name := sel.Sel.Name
			if overlayWrappers[name] {
				for _, arg := range e.Args {
					w.walkExpr(arg, inInteractive, true)
				}
				w.walkExpr(sel.X, inInteractive, inOverlay)
				return
			}
			if inInteractive && !inOverlay {
				// Require arguments for geometry setters (except .Absolute()) so
				// 0-arg getters like c.Size() or layout.Size() never false-positive.
				if geometryMethods[name] && (name == "Absolute" || len(e.Args) > 0) {
					w.addViolation(e.Pos(), fmt.Sprintf("calling .%s(...) inside hover/focus/pressed branch causes layout shift", name))
				}
				if isInFlowConstructor(e.Fun) {
					w.addViolation(e.Pos(), fmt.Sprintf("creating in-flow element %s(...) inside hover/focus/pressed branch causes layout shift", renderFuncName(e.Fun)))
				}
			}
			w.walkExpr(sel.X, inInteractive, inOverlay)
			for _, arg := range e.Args {
				w.walkExpr(arg, inInteractive, inOverlay)
			}
			return
		}
		if inInteractive && !inOverlay && isInFlowConstructor(e.Fun) {
			w.addViolation(e.Pos(), fmt.Sprintf("creating in-flow element %s(...) inside hover/focus/pressed branch causes layout shift", renderFuncName(e.Fun)))
		}
		for _, arg := range e.Args {
			w.walkExpr(arg, inInteractive, inOverlay)
		}
	case *ast.FuncLit:
		w.walkBlock(e.Body, inInteractive, inOverlay)
	case *ast.UnaryExpr:
		w.walkExpr(e.X, inInteractive, inOverlay)
	case *ast.BinaryExpr:
		w.walkExpr(e.X, inInteractive, inOverlay)
		w.walkExpr(e.Y, inInteractive, inOverlay)
	case *ast.ParenExpr:
		w.walkExpr(e.X, inInteractive, inOverlay)
	case *ast.CompositeLit:
		for _, elt := range e.Elts {
			w.walkExpr(elt, inInteractive, inOverlay)
		}
	case *ast.KeyValueExpr:
		w.walkExpr(e.Value, inInteractive, inOverlay)
	}
}

func (w *shiftWalker) walkCallCallbacks(call *ast.CallExpr, inInteractive, inOverlay bool) {
	ast.Inspect(call, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok {
			w.walkBlock(fl.Body, inInteractive, inOverlay)
			return false
		}
		return true
	})
}

// isNewAbsoluteOverlayChain returns true when a call chain starts from a fresh
// element constructor (ui.Box/ui.Row/ui.Column/ui.Text/etc.) AND calls
// .Absolute() somewhere in the chain, making the new element out-of-flow.
// Calling .Absolute() on an existing variable (e.g. row.Absolute()) returns false.
func isNewAbsoluteOverlayChain(call *ast.CallExpr) bool {
	hasAbsolute := false
	cur := ast.Expr(call)
	for cur != nil {
		c, ok := cur.(*ast.CallExpr)
		if !ok {
			return false
		}
		if isInFlowConstructor(c.Fun) {
			return hasAbsolute
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if sel.Sel.Name == "Absolute" {
			hasAbsolute = true
		}
		cur = sel.X
	}
	return false
}

func isInFlowConstructor(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			if id.Name == "ui" && inFlowUIConstructors[f.Sel.Name] {
				return true
			}
			if id.Name == "coss" && inFlowCossConstructors[f.Sel.Name] {
				return true
			}
		}
	case *ast.Ident:
		return inFlowLocalConstructors[f.Name]
	}
	return false
}

func renderFuncName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			return id.Name + "." + f.Sel.Name
		}
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	}
	return "element"
}

// checkTaintedVarUses catches indirect shifts where a hover/focus/pressed
// branch assigns a local scalar (e.g. weight = design.WeightSemibold) or an
// interactive AnimateWith value is passed into a geometry/weight method.
func (w *shiftWalker) checkTaintedVarUses(body *ast.BlockStmt) {
	if len(w.animVars) == 0 && len(w.shiftVars) == 0 {
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if overlayWrappers[sel.Sel.Name] {
			return false
		}
		if !geometryMethods[sel.Sel.Name] || (sel.Sel.Name != "Absolute" && len(call.Args) == 0) {
			return true
		}
		for _, arg := range call.Args {
			ast.Inspect(arg, func(an ast.Node) bool {
				id, ok := an.(*ast.Ident)
				if !ok {
					return true
				}
				if w.animVars[id.Name] {
					w.addViolation(call.Pos(), fmt.Sprintf("hover/focus/pressed AnimateWith value %q fed into .%s(...) causes layout shift", id.Name, sel.Sel.Name))
				}
				if origin, ok := w.shiftVars[id.Name]; ok {
					w.addViolation(call.Pos(), fmt.Sprintf("%s fed into .%s(...) causes layout shift", origin, sel.Sel.Name))
				}
				return true
			})
		}
		return true
	})
}

func TestNoInteractiveLayoutShiftScanner(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		src     string
		wantHit bool
	}{
		{
			name: "bad-hover-padding",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	btn := ui.ButtonBase(c).Padding(4, 8)
	if btn.Hovered() {
		btn.Padding(4, 12)
	}
}`,
			wantHit: true,
		},
		{
			name: "bad-hover-border-width",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	card := ui.Column(c).Border(0.5, c.Theme().Border)
	hovered := card.Hovered()
	if hovered {
		card.Border(2, c.Theme().Accent)
	}
}`,
			wantHit: true,
		},
		{
			name: "bad-hover-in-flow-text",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	row := ui.Row(c)
	hovered := row.Hovered()
	row.Children(func() {
		if hovered {
			ui.Text(c, "Enter")
		}
	})
}`,
			wantHit: true,
		},
		{
			name: "bad-hover-in-flow-kbd",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
)
func f(c *ui.Context) {
	row := ui.Row(c)
	row.Children(func() {
		if row.Hovered() {
			coss.Kbd(c, "⌘K")
		}
	})
}`,
			wantHit: true,
		},
		{
			name: "bad-hover-fontweight-var",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(c *ui.Context) {
	btn := ui.ButtonBase(c)
	weight := design.WeightNormal
	if btn.Hovered() {
		weight = design.WeightSemibold
	}
	ui.Text(c, "Label").FontWeight(weight)
}`,
			wantHit: true,
		},
		{
			name: "bad-hover-animate-width",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(c *ui.Context) {
	row := ui.Row(c)
	hovered := row.Hovered()
	t := row.AnimateWith("w", hovered, design.MotionFast, design.EaseOut)
	ui.Box(c).Width(24 * t)
}`,
			wantHit: true,
		},
		{
			name: "bad-helper-animate-width-on-focus",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func revealLabel(c *ui.Context, owner ui.Element, shown bool) {
	target := float32(0)
	if shown {
		target = 1
	}
	p := owner.AnimateWith("reveal", target, design.MotionFast, design.EaseOut)
	ui.Row(c).Width(120 * p)
}
func caller(c *ui.Context) {
	btn := ui.ButtonBase(c)
	shown := btn.Hovered() || btn.FocusVisible()
	revealLabel(c, btn, shown)
}`,
			wantHit: true,
		},
		{
			name: "good-hover-color-opacity-shadow",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	card := ui.Column(c).Border(0.5, c.Theme().Border)
	hovered := card.Hovered()
	if hovered {
		card.Background(c.Theme().SurfaceHover).BorderColor(c.Theme().Accent).Shadow(0, 4, 12, 0, c.Theme().Border)
	}
	actions := ui.Row(c)
	if !hovered && !card.FocusWithin() {
		actions.Opacity(0)
	}
}`,
			wantHit: false,
		},
		{
			name: "good-hover-zero-arg-size-getter",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	btn := ui.ButtonBase(c)
	if btn.Hovered() {
		_, _ = c.Size()
		layout := ui.ShapeTextLine("hi", ui.Font{Size: 12}, 0)
		_, _ = layout.Size()
	}
}`,
			wantHit: false,
		},
		{
			name: "good-focus-visible-absolute-ring",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	root := ui.ButtonBase(c)
	root.Children(func() {
		if root.FocusVisible() {
			ui.Box(c).Absolute().Top(3).Right(3).Bottom(3).Left(3).Border(2, c.Theme().Accent).PassThrough()
		}
	})
}`,
			wantHit: false,
		},
		{
			name: "good-hover-animate-opacity",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(c *ui.Context) {
	row := ui.Row(c)
	hovered := row.Hovered()
	alpha := row.AnimateWith("fade", hovered, design.MotionFast, design.EaseOut)
	ui.Box(c).Opacity(alpha)
}`,
			wantHit: false,
		},
		{
			name: "good-hover-drawover-paint",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context) {
	btn := ui.ButtonBase(c)
	if btn.Hovered() {
		btn.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Stroke(r, c.Theme().Accent, 1, 8)
		})
	}
}`,
			wantHit: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(dir, tc.name+".go")
			if err := os.WriteFile(p, []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			vs, err := scanInteractiveLayoutShift(p, tc.name+".go")
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantHit && len(vs) == 0 {
				t.Fatalf("expected violation for %s, got none", tc.name)
			}
			if !tc.wantHit && len(vs) > 0 {
				t.Fatalf("unexpected violation for %s: %v", tc.name, vs)
			}
		})
	}
}
