package designguard

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// counts are a file's design literals.
type counts struct {
	Colors  int `json:"colors"`
	Metrics int `json:"metrics"`
}

var (
	hexString = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)
	rgbString = regexp.MustCompile(`^rgba?\(`)
	// metricMethods take radii and spacing: a number literal there is a
	// metric that belongs to design tokens.
	metricMethods = map[string]bool{
		"Radius": true, "Padding": true, "PaddingX": true, "PaddingY": true,
		"Gap": true, "GapX": true, "GapY": true, "Margin": true, "MarginX": true, "MarginY": true,
	}
	colorFuncs = map[string]bool{"RGB": true, "RGBA": true, "Hex": true}
)

// scan counts the design literals of one Go file.
func scan(path string) (counts, error) {
	var n counts
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return n, err
	}
	for _, cg := range f.Comments {
		if strings.Contains(cg.Text(), "Code generated") {
			return counts{}, nil
		}
	}
	ast.Inspect(f, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "ui" && colorFuncs[sel.Sel.Name] {
				n.Colors++
				return false // its string argument is the same color
			}
			if metricMethods[sel.Sel.Name] {
				for _, a := range x.Args {
					if lit, ok := a.(*ast.BasicLit); ok && (lit.Kind == token.INT || lit.Kind == token.FLOAT) {
						if v, err := strconv.ParseFloat(lit.Value, 64); err == nil && v != 0 {
							n.Metrics++
						}
					}
				}
			}
		case *ast.CompositeLit:
			if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Color" {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "ui" && len(x.Elts) > 0 {
					n.Colors++
				}
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if s, err := strconv.Unquote(x.Value); err == nil && (hexString.MatchString(s) || rgbString.MatchString(s)) {
					n.Colors++
				}
			}
		}
		return true
	})
	return n, nil
}

// walk counts every non-test Go file of internal/ui outside the design
// system.
func walk(t *testing.T) map[string]counts {
	t.Helper()
	root := filepath.Join("..")
	out := map[string]counts{}
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
		n, err := scan(path)
		if err != nil {
			return err
		}
		if n != (counts{}) {
			out[rel] = n
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const baselinePath = "testdata/baseline.json"

func TestDesignLiteralsOnlyShrink(t *testing.T) {
	got := walk(t)
	if os.Getenv("DESIGNGUARD_UPDATE") != "" {
		data, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(baselinePath, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	var base map[string]counts
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatal(err)
	}
	var files []string
	for f := range got {
		files = append(files, f)
	}
	sort.Strings(files)
	var grew []string
	shrank := 0
	for _, f := range files {
		g, b := got[f], base[f]
		if g.Colors > b.Colors || g.Metrics > b.Metrics {
			grew = append(grew, f+": colors "+strconv.Itoa(b.Colors)+"→"+strconv.Itoa(g.Colors)+", metrics "+strconv.Itoa(b.Metrics)+"→"+strconv.Itoa(g.Metrics))
		}
	}
	for f, b := range base {
		if g := got[f]; g.Colors < b.Colors || g.Metrics < b.Metrics {
			shrank++
		}
	}
	if len(grew) > 0 {
		t.Fatalf("new hard-coded colors or radius/spacing literals in internal/ui; use internal/ui/design tokens or internal/ui/coss components:\n  %s", strings.Join(grew, "\n  "))
	}
	if shrank > 0 {
		t.Logf("%d files have fewer literals than the baseline; lock the gain with DESIGNGUARD_UPDATE=1", shrank)
	}
}

// TestScanCounts pins what the scanner counts.
func TestScanCounts(t *testing.T) {
	dir := t.TempDir()
	src := `package x

import "github.com/egoist/mygo/ui"

func f(c *ui.Context) {
	_ = ui.RGBA(1, 2, 3, 0.5)
	_ = ui.Hex("#fff")
	_ = ui.Color{R: 1}
	_ = "#123456"
	_ = "rgba(0, 0, 0, 0.1)"
	ui.Box(c).Radius(6).Padding(4, 8).Gap(0).Margin(x)
}

var x float32
`
	p := filepath.Join(dir, "x.go")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := scan(p)
	if err != nil {
		t.Fatal(err)
	}
	if n.Colors != 5 || n.Metrics != 3 {
		t.Fatalf("scan = %+v, want 5 colors and 3 metrics", n)
	}
}
