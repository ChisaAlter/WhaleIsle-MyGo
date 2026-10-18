package mathrender

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestOfflineMathJax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("integration test requires build-tool Node: ", err)
	}
	node, _ = filepath.Abs(node)
	for _, formula := range []string{`\frac{a+b}{c+d}`, `x_{i+1}^{n^2}`, `\sqrt[3]{\frac{x}{y}}`, `\begin{pmatrix}a&b\\c&d\end{pmatrix}`, `\sum_{i=1}^{n} i = \frac{n(n+1)}2`} {
		t.Run(formula, func(t *testing.T) {
			result, err := Render(context.Background(), node, Formula{TeX: formula, Display: true})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(result.SVG, "<path") || strings.Contains(result.SVG, "<text") {
				t.Fatalf("expected vector outlines: %.100s", result.SVG)
			}
			if _, err := ui.ParseSVG([]byte(result.SVG)); err != nil {
				t.Fatal("MyGo rejected equation SVG: ", err)
			}
		})
	}
	for _, bad := range []string{`\require{html}\href{https://example.com}{x}`, `\notARealCommand{x}`, strings.Repeat("x", MaxFormulaBytes+1)} {
		if _, err := Render(context.Background(), node, Formula{TeX: bad}); err == nil {
			t.Fatalf("unsafe/invalid formula accepted: %.60s", bad)
		}
	}
}

func TestEmbeddedMathProvenance(t *testing.T) {
	data, err := os.ReadFile("provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		BundleSHA256 string `json:"bundleSHA256"`
	}
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(renderer))
	if hex.EncodeToString(digest[:]) != provenance.BundleSHA256 {
		t.Fatal("embedded engine differs from recorded build")
	}
}
