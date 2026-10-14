package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTransformTeXSymbols(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`\alpha`, "α"},
		{`\beta`, "β"},
		{`a \times b`, "a × b"},
		{`a \cdot b`, "a · b"},
		{`x \pm y`, "x ± y"},
		{`a \leq b \geq c \neq d`, "a ≤ b ≥ c ≠ d"},
		{`x \rightarrow \infty`, "x → ∞"},
		{`\sum`, "∑"},
		{`\prod`, "∏"},
		{`\pi`, "π"},
	}
	for _, tc := range cases {
		if got := transformTeX(tc.in); got != tc.want {
			t.Fatalf("transformTeX(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestTransformTeXFracSqrtScripts(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`\frac{a}{b}`, "a/b"},
		{`\frac{1}{2}`, "1/2"},
		{`\frac{\alpha}{\beta}`, "α/β"},
		{`\sqrt{x}`, "√(x)"},
		{`a^2`, "a²"},
		{`x^{10}`, "x¹⁰"},
		{`x_i`, "xᵢ"},
		{`a_{12}`, "a₁₂"},
		{`E=mc^2`, "E=mc²"},
	}
	for _, tc := range cases {
		if got := transformTeX(tc.in); got != tc.want {
			t.Fatalf("transformTeX(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestTransformTeXEscapesAndUnknown(t *testing.T) {
	if got := transformTeX(`\{a\}`); got != "{a}" {
		t.Fatalf("braces: %q", got)
	}
	if got := transformTeX(`\unknown`); got != `\unknown` {
		t.Fatalf("unknown: %q", got)
	}
	// letters without subscript forms fall back
	if got := transformTeX(`x_{yz}`); got != "x_{yz}" {
		t.Fatalf("fallback sub: %q", got)
	}
}

func TestSplitNextMathInlineBlockEscape(t *testing.T) {
	plain, math, block, after, ok := splitNextMath(`cost $a+b$ more`)
	if !ok || plain != "cost " || math != "a+b" || block || after != " more" {
		t.Fatalf("inline: plain=%q math=%q block=%v after=%q ok=%v", plain, math, block, after, ok)
	}

	plain, math, block, after, ok = splitNextMath("before $$\\frac{1}{2}$$ after")
	if !ok || plain != "before " || math != `\frac{1}{2}` || !block || after != " after" {
		t.Fatalf("block: plain=%q math=%q block=%v after=%q ok=%v", plain, math, block, after, ok)
	}

	plain, math, block, after, ok = splitNextMath(`price is \$5 today`)
	if ok || plain != "price is $5 today" || math != "" {
		t.Fatalf("escape: plain=%q math=%q ok=%v", plain, math, ok)
	}
}

func TestAppendMathAwareText(t *testing.T) {
	st := Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Accent: ui.RGB(0, 80, 200)}
	var out []ui.Span
	appendMathAwareText("area $a^2$ ok", st, &out)
	if len(out) != 3 {
		t.Fatalf("spans=%d %#v", len(out), out)
	}
	if out[1].Text != "a²" || !out[1].Italic {
		t.Fatalf("%#v", out[1])
	}
}

func TestAppendMathAwareTextBlockAndEscape(t *testing.T) {
	st := Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Accent: ui.RGB(0, 80, 200)}
	var out []ui.Span
	appendMathAwareText(`use \$ and $$\alpha+\beta$$ end`, st, &out)
	if len(out) != 3 {
		t.Fatalf("spans=%d %#v", len(out), out)
	}
	if out[0].Text != "use $ and " {
		t.Fatalf("plain=%q", out[0].Text)
	}
	if out[1].Text != "α+β" || !out[1].Italic {
		t.Fatalf("math=%#v", out[1])
	}
	if out[2].Text != " end" {
		t.Fatalf("tail=%q", out[2].Text)
	}
}
