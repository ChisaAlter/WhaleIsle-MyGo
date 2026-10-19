//go:build windows

package d3d11

import (
	"encoding/binary"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"testing"
)

// This entry uses D3DCompile only: it creates no device, HWND or swap chain.
// It validates real bytecode/layout, not GPU execution or fluid appearance.
func TestFluidShaderCompileAndLayout(t *testing.T) {
	for _, entry := range []string{"fluidVS", "copyPS", "splatPS", "curlPS", "vorticityPS", "divergencePS", "clearPS", "pressurePS", "gradientPS", "advectionPS", "displayPS"} {
		t.Run(entry, func(t *testing.T) {
			target := "ps_4_0"
			if entry == "fluidVS" {
				target = "vs_4_0"
			}
			code, err := compile(fluidSource, entry, target)
			if err != nil {
				t.Fatal(err)
			}
			if len(code) < 32 || string(code[:4]) != "DXBC" {
				t.Fatal("compiler did not return DXBC")
			}
		})
	}
	// Keep every cbuffer member live, so optimization cannot remove its ABI.
	code, err := compile(fluidSource+"\n"+`float4 layoutPS(Vertex v):SV_Target{return grid+splat+ink;}`, "layoutPS", "ps_4_0")
	if err != nil {
		t.Fatal(err)
	}
	rdef := fluidDXBCChunk(t, code, "RDEF")
	u32 := func(data []byte, off uint32) uint32 {
		t.Helper()
		if uint64(off)+4 > uint64(len(data)) {
			t.Fatal("truncated compiler metadata")
		}
		return binary.LittleEndian.Uint32(data[off:])
	}
	cstring := func(off uint32) string {
		t.Helper()
		if off >= uint32(len(rdef)) {
			t.Fatal("invalid name offset")
		}
		end := off
		for end < uint32(len(rdef)) && rdef[end] != 0 {
			end++
		}
		return string(rdef[off:end])
	}
	if u32(rdef, 0) != 1 {
		t.Fatal("expected one fluid constant buffer")
	}
	cb := u32(rdef, 4)
	if cstring(u32(rdef, cb)) != "FluidParameters" || u32(rdef, cb+4) != 3 || u32(rdef, cb+12) != 48 {
		t.Fatal("Go [3][4]float32 no longer matches HLSL cbuffer")
	}
	vars := u32(rdef, cb+8)
	for i, name := range []string{"grid", "splat", "ink"} {
		v := vars + uint32(i*24) // shader model 4 RDEF variable record
		if cstring(u32(rdef, v)) != name || u32(rdef, v+4) != uint32(i*16) || u32(rdef, v+8) != 16 {
			t.Fatalf("invalid %s layout", name)
		}
	}
}

func fluidDXBCChunk(t *testing.T, code []byte, name string) []byte {
	t.Helper()
	if len(code) < 32 {
		t.Fatal("truncated DXBC")
	}
	count := binary.LittleEndian.Uint32(code[28:])
	for i := uint32(0); i < count; i++ {
		index := 32 + uint64(i)*4
		if index+4 > uint64(len(code)) {
			t.Fatal("truncated chunk table")
		}
		off := uint64(binary.LittleEndian.Uint32(code[index:]))
		if off+8 > uint64(len(code)) {
			t.Fatal("truncated chunk")
		}
		n := uint64(binary.LittleEndian.Uint32(code[off+4:]))
		if off+8+n > uint64(len(code)) {
			t.Fatal("invalid chunk extent")
		}
		if string(code[off:off+4]) == name {
			return code[off+8 : off+8+n]
		}
	}
	t.Fatalf("missing %s chunk", name)
	return nil
}

func TestFluidWindowsResolutionAndDispatchContract(t *testing.T) {
	for _, tc := range []struct{ w, h, sw, sh, dw, dh int }{{1440, 900, 205, 128, 2304, 1440}, {900, 1440, 128, 205, 1440, 2304}, {100, 100, 128, 128, 1440, 1440}} {
		sw, sh := fluidResolution(tc.w, tc.h, 128)
		dw, dh := fluidResolution(tc.w, tc.h, 1440)
		if sw != tc.sw || sh != tc.sh || dw != tc.dw || dh != tc.dh || !FluidFits(tc.w, tc.h) {
			t.Fatalf("wrong source-resolution aspect: %+v => %dx%d/%dx%d", tc, sw, sh, dw, dh)
		}
	}
	for _, size := range [][2]int{{0, 100}, {1, 1_000_000}, {16384, 1}, {16384, 16384}} {
		if FluidFits(size[0], size[1]) {
			t.Fatalf("unbounded GPU allocation accepted: %v", size)
		}
	}
	data, err := os.ReadFile("fluid_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "fluid_windows.go", data, 0)
	if err != nil {
		t.Fatal(err)
	}
	var passes []string
	iterations := ""
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Step" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if loop, ok := n.(*ast.RangeStmt); ok {
				if literal, ok := loop.X.(*ast.BasicLit); ok {
					iterations = literal.Value
				}
			}
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "pass" {
					if name, ok := call.Args[0].(*ast.BasicLit); ok {
						passes = append(passes, name.Value)
					}
				}
			}
			return true
		})
	}
	want := []string{`"curlPS"`, `"vorticityPS"`, `"divergencePS"`, `"clearPS"`, `"pressurePS"`, `"gradientPS"`, `"advectionPS"`, `"advectionPS"`}
	if !reflect.DeepEqual(passes, want) || iterations != "20" {
		t.Fatalf("simulation dispatch changed: %v, iterations %s", passes, iterations)
	}
}
