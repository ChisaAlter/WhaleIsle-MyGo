package chat

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// These compact interface surfaces must preserve their relative sizes while
// following the saved interface preference. Content renderers are separate.
func TestCompactSurfacesUseInterfaceFontSizes(t *testing.T) {
	for _, name := range []string{"trajectory.go", "citations.go", "todopanel.go", "planboard.go"} {
		t.Run(name, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, name, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "FontSize" {
					return true
				}
				count++
				if len(call.Args) != 1 {
					t.Fatalf("invalid FontSize at %s", fs.Position(call.Pos()))
				}
				size, ok := call.Args[0].(*ast.CallExpr)
				if !ok {
					t.Errorf("%s bypasses saved interface size", fs.Position(call.Pos()))
					return true
				}
				helper, ok := size.Fun.(*ast.SelectorExpr)
				if !ok {
					t.Errorf("%s does not use interface size helper", fs.Position(call.Pos()))
					return true
				}
				pkg, ok := helper.X.(*ast.Ident)
				if !ok || pkg.Name != "theme" || helper.Sel.Name != "InterfaceSize" {
					t.Errorf("%s uses incorrect typography role", fs.Position(call.Pos()))
				}
				return true
			})
			if count == 0 {
				t.Fatal("no font consumers checked")
			}
		})
	}
}
