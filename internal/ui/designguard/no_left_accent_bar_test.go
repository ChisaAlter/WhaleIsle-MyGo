package designguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// leftAccentBarWhitelist lists files where a left-edge vertical accent bar is
// an intentional typographic or diff convention (blockquote, diff stripe,
// openui quote/callout/toned card), NOT a navigation/tab/selection indicator.
var leftAccentBarWhitelist = map[string]bool{
	"widgets/markdown/document_layout.go": true,
	"widgets/markdown/tool_diffblock.go":  true,
	"openui/text.go":                      true,
	"openui/genui.go":                     true,
}

type guardViolation struct {
	File   string
	Line   int
	Reason string
}

func (v guardViolation) String() string {
	return fmt.Sprintf("%s:%d: %s", v.File, v.Line, v.Reason)
}

func TestNoLeftAccentBar(t *testing.T) {
	root := filepath.Join("..")
	var violations []guardViolation
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			switch rel {
			case "design", "designguard":
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
		if leftAccentBarWhitelist[rel] {
			return nil
		}
		vs, err := scanLeftAccentBar(path, rel)
		if err != nil {
			return err
		}
		violations = append(violations, vs...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
		t.Fatalf("left-edge accent/primary vertical bar detected (use background wash / borderless pill instead of a left bar):\n  %s", strings.Join(lines, "\n  "))
	}
}

func scanLeftAccentBar(path, rel string) ([]guardViolation, error) {
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

	var violations []guardViolation
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		scanLeftAccentBarBlock(fs, rel, fn.Body, &violations)
	}
	return violations, nil
}

func scanLeftAccentBarBlock(fs *token.FileSet, rel string, body *ast.BlockStmt, out *[]guardViolation) {
	// Track local scalar variables (narrow widths like barW := design.Sp(0.75))
	// and local color variables derived from accent/primary tokens.
	narrowVars := map[string]bool{}
	accentVars := map[string]bool{}

	recordAssign := func(lhs, rhs ast.Expr) {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name == "_" {
			return
		}
		if isNarrowBarWidthExpr(rhs, narrowVars) {
			narrowVars[id.Name] = true
		}
		if isAccentColorExpr(rhs, accentVars) {
			accentVars[id.Name] = true
		}
	}

	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == len(x.Rhs) {
				for i := range x.Lhs {
					recordAssign(x.Lhs[i], x.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			if len(x.Names) == len(x.Values) {
				for i := range x.Names {
					recordAssign(x.Names[i], x.Values[i])
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			// Rule A: p.Fill(ui.Rect{X: leftEdge, Y: ..., W: narrow, H: vertical}, accentColor, ...)
			if sel.Sel.Name == "Fill" && len(x.Args) >= 2 {
				if rect, ok := x.Args[0].(*ast.CompositeLit); ok && isUIRect(rect.Type) {
					if reason := inspectFillRectForLeftBar(rect, x.Args[1], narrowVars, accentVars); reason != "" {
						pos := fs.Position(x.Pos())
						*out = append(*out, guardViolation{File: rel, Line: pos.Line, Reason: reason})
					}
				}
			}
			// Rule B: left-only BorderWidth(0, 0, 0, w) with accent/primary color or thick stripe (w >= 2).
			if sel.Sel.Name == "BorderWidth" && len(x.Args) == 4 {
				if isZeroExpr(x.Args[0]) && isZeroExpr(x.Args[1]) && isZeroExpr(x.Args[2]) && !isZeroExpr(x.Args[3]) {
					if v, ok := evalDesignNum(x.Args[3]); ok && v >= 2 {
						pos := fs.Position(x.Pos())
						*out = append(*out, guardViolation{
							File:   rel,
							Line:   pos.Line,
							Reason: fmt.Sprintf("left-only BorderWidth(0, 0, 0, %g) paints a vertical accent bar", v),
						})
					}
				}
			}
			// Rule B2: BorderColor(accent) chained on BorderWidth(0, 0, 0, nonZero).
			if sel.Sel.Name == "BorderColor" && len(x.Args) >= 1 && isAccentColorExpr(x.Args[0], accentVars) {
				if hasLeftOnlyBorderWidthInChain(sel.X) {
					pos := fs.Position(x.Pos())
					*out = append(*out, guardViolation{
						File:   rel,
						Line:   pos.Line,
						Reason: "left-only BorderWidth(0, 0, 0, w) paired with accent/primary BorderColor",
					})
				}
			}
		}
		return true
	})
}

func isUIRect(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Rect" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "ui"
}

func inspectFillRectForLeftBar(rect *ast.CompositeLit, colorArg ast.Expr, narrowVars, accentVars map[string]bool) string {
	var xExpr, wExpr, hExpr ast.Expr
	for _, elt := range rect.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		k, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch k.Name {
		case "X":
			xExpr = kv.Value
		case "W":
			wExpr = kv.Value
		case "H":
			hExpr = kv.Value
		}
	}
	if xExpr == nil || wExpr == nil || hExpr == nil {
		return ""
	}
	if !isLeftEdgeX(xExpr) {
		return ""
	}
	if !isNarrowBarWidthExpr(wExpr, narrowVars) {
		return ""
	}
	if !isVerticalBarHeightExpr(hExpr, wExpr) {
		return ""
	}
	if !isAccentColorExpr(colorArg, accentVars) {
		return ""
	}
	return "Painter.Fill draws a left-edge narrow vertical accent/primary bar"
}

// isLeftEdgeX returns true for r.X, bounds.X, or r.X + smallOffset (<= 10 DIP),
// excluding right-edge expressions like r.X + r.W - 5.
func isLeftEdgeX(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name == "X"
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			if isLeftEdgeX(e.X) {
				if v, ok := evalDesignNum(e.Y); ok && v >= 0 && v <= 10 {
					return true
				}
			}
		}
	}
	return false
}

func isNarrowBarWidthExpr(expr ast.Expr, narrowVars map[string]bool) bool {
	if id, ok := expr.(*ast.Ident); ok {
		return narrowVars[id.Name]
	}
	if v, ok := evalDesignNum(expr); ok {
		return v > 0 && v <= 6
	}
	return false
}

// isVerticalBarHeightExpr distinguishes a vertical bar (H derived from r.H /
// bounds.H / barH or constant > 8) from a square dot (W == H) or 1px hairline.
func isVerticalBarHeightExpr(hExpr, wExpr ast.Expr) bool {
	if hVal, ok := evalDesignNum(hExpr); ok {
		if wVal, wok := evalDesignNum(wExpr); wok && hVal <= wVal+1 {
			return false
		}
		return hVal > 8
	}
	// Non-constant height expression: ensure it doesn't equal the same narrow var.
	if hid, ok := hExpr.(*ast.Ident); ok {
		if wid, wok := wExpr.(*ast.Ident); wok && hid.Name == wid.Name {
			return false
		}
	}
	foundHeightRef := false
	ast.Inspect(hExpr, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if x.Sel.Name == "H" {
				foundHeightRef = true
			}
		case *ast.Ident:
			lower := strings.ToLower(x.Name)
			if strings.Contains(lower, "barh") || strings.Contains(lower, "height") || lower == "h" {
				foundHeightRef = true
			}
		}
		return true
	})
	return foundHeightRef
}

func isAccentColorExpr(expr ast.Expr, accentVars map[string]bool) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		switch x := n.(type) {
		case *ast.Ident:
			if accentVars[x.Name] {
				found = true
				return false
			}
			lower := strings.ToLower(x.Name)
			if strings.Contains(lower, "primary") || strings.Contains(lower, "accent") || strings.Contains(lower, "indicator") {
				found = true
				return false
			}
		case *ast.SelectorExpr:
			lower := strings.ToLower(x.Sel.Name)
			if strings.Contains(lower, "primary") || lower == "accent" || lower == "ring" || lower == "sidebarring" || lower == "info" {
				// Exclude subtle surface fills like SidebarAccent unless paired with Alpha/Primary?
				// Wait: a 3px left bar in Accent/Primary/SidebarPrimary/SidebarAccent is STILL a left bar!
				found = true
				return false
			}
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "ui" && colorFuncs[x.Sel.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hasLeftOnlyBorderWidthInChain(expr ast.Expr) bool {
	cur := expr
	for cur != nil {
		call, ok := cur.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if sel.Sel.Name == "BorderWidth" && len(call.Args) == 4 {
			return isZeroExpr(call.Args[0]) && isZeroExpr(call.Args[1]) && isZeroExpr(call.Args[2]) && !isZeroExpr(call.Args[3])
		}
		cur = sel.X
	}
	return false
}

func isZeroExpr(expr ast.Expr) bool {
	v, ok := evalDesignNum(expr)
	return ok && v == 0
}

// evalDesignNum statically evaluates simple numeric literals, negative
// literals, arithmetic (+,-,*,/), and design.Sp(n) / design.BorderWidth.
func evalDesignNum(expr ast.Expr) (float64, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT || e.Kind == token.FLOAT {
			v, err := strconv.ParseFloat(e.Value, 64)
			return v, err == nil
		}
	case *ast.UnaryExpr:
		if e.Op == token.SUB {
			if v, ok := evalDesignNum(e.X); ok {
				return -v, true
			}
		}
	case *ast.ParenExpr:
		return evalDesignNum(e.X)
	case *ast.BinaryExpr:
		l, lok := evalDesignNum(e.X)
		r, rok := evalDesignNum(e.Y)
		if lok && rok {
			switch e.Op {
			case token.ADD:
				return l + r, true
			case token.SUB:
				return l - r, true
			case token.MUL:
				return l * r, true
			case token.QUO:
				if r != 0 {
					return l / r, true
				}
			}
		}
	case *ast.SelectorExpr:
		if id, ok := e.X.(*ast.Ident); ok && id.Name == "design" {
			switch e.Sel.Name {
			case "BorderWidth":
				return 1, true
			case "Unit":
				return 4, true
			}
		}
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "design" && sel.Sel.Name == "Sp" && len(e.Args) == 1 {
				if v, ok := evalDesignNum(e.Args[0]); ok {
					return v * 4, true
				}
			}
		}
	}
	return 0, false
}

func TestNoLeftAccentBarScanner(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		src     string
		wantHit bool
	}{
		{
			name: "bad-sidebar-primary-bar",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(btn ui.Element, t *design.Tokens) {
	indicator := t.SidebarPrimary
	btn.DrawOver(func(p *ui.Painter, r ui.Rect) {
		barH := min(design.Sp(4), max(0, r.H-design.Sp(3)))
		barW := design.Sp(0.75)
		p.Fill(ui.Rect{X: r.X + design.Sp(0.5), Y: r.Y + (r.H-barH)/2, W: barW, H: barH}, indicator, barW/2)
	})
}`,
			wantHit: true,
		},
		{
			name: "bad-search-target-bar",
			src: `package p
import "github.com/egoist/mygo/ui"
func f(c *ui.Context, row ui.Element) {
	row.DrawOver(func(p *ui.Painter, bounds ui.Rect) {
		p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y + 4, W: 3, H: max(0, bounds.H-8)}, c.Theme().Accent, 1.5)
	})
}`,
			wantHit: true,
		},
		{
			name: "bad-left-only-accent-border",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(c *ui.Context, t *design.Tokens) {
	ui.Row(c).BorderWidth(0, 0, 0, 3).BorderColor(t.Primary)
}`,
			wantHit: true,
		},
		{
			name: "good-horizontal-divider",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(e ui.Element, t *design.Tokens) {
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, t.Primary, 0)
	})
}`,
			wantHit: false,
		},
		{
			name: "good-right-edge-scrollbar",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(e ui.Element, t *design.Tokens) {
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X + r.W - 5, Y: r.Y + 4, W: 4, H: r.H - 8}, t.Primary, 2)
	})
}`,
			wantHit: false,
		},
		{
			name: "good-square-dot",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(e ui.Element, t *design.Tokens) {
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 2, W: 4, H: 4}, t.Primary, 2)
	})
}`,
			wantHit: false,
		},
		{
			name: "good-neutral-hairline-divider",
			src: `package p
import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)
func f(c *ui.Context, t *design.Tokens) {
	ui.Row(c).BorderWidth(0, 0, 0, design.BorderWidth).BorderColor(t.Border)
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
			vs, err := scanLeftAccentBar(p, tc.name+".go")
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
