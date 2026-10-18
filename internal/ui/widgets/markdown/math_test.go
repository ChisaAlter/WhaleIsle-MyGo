package markdown

import (
	"strings"
	"testing"
)

func TestMathParsingPreservesTeX(t *testing.T) {
	tests := []struct {
		source, tex string
		display     bool
	}{
		{`Before $x_{i+1}^{n^2}$ after`, `x_{i+1}^{n^2}`, false},
		{`Before \(\frac{a}{b}\) after`, `\frac{a}{b}`, false},
		{"Before\n\n$$\n\\begin{pmatrix}a&b\\\\c&d\\end{pmatrix}\n$$\n\nAfter", `\begin{pmatrix}a&b\\c&d\end{pmatrix}`, true},
		{"\\[\n\\sqrt[3]{x}\n\\]\nAfter", `\sqrt[3]{x}`, true},
		{`| Formula |` + "\n" + `| --- |` + "\n" + `| $a_b$ |`, `a_b`, false},
		{`- **value** $\frac{a}{b}$`, `\frac{a}{b}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.tex, func(t *testing.T) {
			formulas := map[int]mathFormula{}
			spans := Spans(tt.source, Style{maths: formulas})
			if len(formulas) != 1 {
				t.Fatalf("formulas=%#v spans=%q", formulas, concat(spans))
			}
			for index, formula := range formulas {
				if strings.TrimSpace(formula.TeX) != tt.tex || formula.Display != tt.display {
					t.Fatalf("formula=%#v", formula)
				}
				if spans[index].Text != formula.Raw {
					t.Fatal("span fallback lost source")
				}
			}
		})
	}
}

func TestMathSkipsCodeAndIncompleteDelimiters(t *testing.T) {
	for _, source := range []string{"`$x$`", "```tex\n$x$\n```", `cost \$5`, `unfinished $x`, "$$\nnot closed", `cost $5 and $10`} {
		formulas := map[int]mathFormula{}
		Spans(source, Style{maths: formulas})
		if len(formulas) != 0 {
			t.Fatalf("unexpected formulas for %q: %#v", source, formulas)
		}
	}
}

func TestMathLayoutBaselinesAndWrap(t *testing.T) {
	atoms := []mathAtom{
		{width: 30, ascent: 10, descent: 3},
		{width: 35, ascent: 25, descent: 12},
		{width: 40, ascent: 10, descent: 3},
	}
	layout := layoutMathAtoms(atoms, 80, 16)
	if len(layout.items) != 3 {
		t.Fatal(layout)
	}
	a, b, c := layout.items[0], layout.items[1], layout.items[2]
	if a.rect.Y+a.atom.ascent != b.rect.Y+b.atom.ascent {
		t.Fatal("inline baselines differ")
	}
	if c.rect.X != 0 || c.rect.Y < b.rect.H {
		t.Fatal("text did not wrap below fraction")
	}
	if layout.height < 53 {
		t.Fatalf("fraction extent clipped: %#v", layout)
	}
}

func TestMathDisplayLayout(t *testing.T) {
	layout := layoutMathAtoms([]mathAtom{{width: 20, ascent: 10, descent: 3}, {width: 40, ascent: 30, descent: 15, display: true}, {width: 20, ascent: 10, descent: 3}}, 100, 16)
	if layout.items[1].rect.X != 30 || layout.items[1].rect.Y < 16 || layout.items[2].rect.Y < 61 {
		t.Fatalf("display placement=%#v", layout)
	}
}
