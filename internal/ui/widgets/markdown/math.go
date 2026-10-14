package markdown

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Curated TeX command → Unicode display map (KaTeX stand-in; dependency-light).
var texSymbols = map[string]string{
	"alpha":      "α",
	"beta":       "β",
	"gamma":      "γ",
	"delta":      "δ",
	"epsilon":    "ε",
	"theta":      "θ",
	"lambda":     "λ",
	"mu":         "μ",
	"pi":         "π",
	"sigma":      "σ",
	"phi":        "φ",
	"omega":      "ω",
	"Gamma":      "Γ",
	"Delta":      "Δ",
	"Theta":      "Θ",
	"Lambda":     "Λ",
	"Pi":         "Π",
	"Sigma":      "Σ",
	"Omega":      "Ω",
	"times":      "×",
	"cdot":       "·",
	"pm":         "±",
	"mp":         "∓",
	"div":        "÷",
	"leq":        "≤",
	"le":         "≤",
	"geq":        "≥",
	"ge":         "≥",
	"neq":        "≠",
	"ne":         "≠",
	"approx":     "≈",
	"equiv":      "≡",
	"rightarrow": "→",
	"to":         "→",
	"leftarrow":  "←",
	"Rightarrow": "⇒",
	"infty":      "∞",
	"sum":        "∑",
	"prod":       "∏",
	"int":        "∫",
	"partial":    "∂",
	"nabla":      "∇",
	"in":         "∈",
	"notin":      "∉",
	"subset":     "⊂",
	"subseteq":   "⊆",
	"cup":        "∪",
	"cap":        "∩",
	"forall":     "∀",
	"exists":     "∃",
	"emptyset":   "∅",
	"ldots":      "…",
	"cdots":      "⋯",
	"ast":        "∗",
	"circ":       "∘",
	// spacing / no-ops → ordinary space or empty
	",": " ",
	";": " ",
	":": " ",
	" ": " ",
	"quad":  "  ",
	"qquad": "    ",
}

var superMap = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
	'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ', 'd': 'ᵈ', 'e': 'ᵉ',
	'f': 'ᶠ', 'g': 'ᵍ', 'h': 'ʰ', 'i': 'ⁱ', 'j': 'ʲ',
	'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ', 'n': 'ⁿ', 'o': 'ᵒ',
	'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ',
	'v': 'ᵛ', 'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
}

var subMap = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
	'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
	'a': 'ₐ', 'e': 'ₑ', 'h': 'ₕ', 'i': 'ᵢ', 'j': 'ⱼ',
	'k': 'ₖ', 'l': 'ₗ', 'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ',
	'p': 'ₚ', 'r': 'ᵣ', 's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ',
	'v': 'ᵥ', 'x': 'ₓ',
}

// transformTeX expands a small curated set of TeX into readable Unicode text.
func transformTeX(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	i := 0
	for i < len(src) {
		switch src[i] {
		case '\\':
			if i+1 >= len(src) {
				b.WriteByte('\\')
				i++
				continue
			}
			next := src[i+1]
			// Escaped single characters: \{ \} \$ \_ \^ \\ \%
			if next == '{' || next == '}' || next == '$' || next == '_' ||
				next == '^' || next == '\\' || next == '%' || next == '#' || next == '&' {
				b.WriteByte(next)
				i += 2
				continue
			}
			cmd, n := readTeXCommand(src[i+1:])
			i += 1 + n
			switch cmd {
			case "frac":
				num, nn := readTeXGroup(src[i:])
				i += nn
				den, dn := readTeXGroup(src[i:])
				i += dn
				b.WriteString(transformTeX(num))
				b.WriteByte('/')
				b.WriteString(transformTeX(den))
			case "sqrt":
				arg, an := readTeXGroup(src[i:])
				i += an
				if arg == "" {
					b.WriteRune('√')
				} else {
					b.WriteString("√(")
					b.WriteString(transformTeX(arg))
					b.WriteByte(')')
				}
			default:
				if u, ok := texSymbols[cmd]; ok {
					b.WriteString(u)
				} else if cmd != "" {
					b.WriteByte('\\')
					b.WriteString(cmd)
				} else {
					b.WriteByte('\\')
				}
			}
		case '^':
			content, n := readTeXGroup(src[i+1:])
			i += 1 + n
			b.WriteString(mapScript(transformTeX(content), superMap, "^"))
		case '_':
			content, n := readTeXGroup(src[i+1:])
			i += 1 + n
			b.WriteString(mapScript(transformTeX(content), subMap, "_"))
		case '{', '}':
			// grouping braces are structural only
			i++
		default:
			r, size := utf8.DecodeRuneInString(src[i:])
			b.WriteRune(r)
			i += size
		}
	}
	return b.String()
}

func readTeXCommand(s string) (string, int) {
	if s == "" {
		return "", 0
	}
	if !isTeXLetter(s[0]) {
		return string(s[0]), 1
	}
	i := 1
	for i < len(s) && isTeXLetter(s[i]) {
		i++
	}
	return s[:i], i
}

func isTeXLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// readTeXGroup reads {…} or a single token (rune / \cmd). Returns content and bytes consumed.
func readTeXGroup(s string) (string, int) {
	if s == "" {
		return "", 0
	}
	if s[0] == '{' {
		depth := 0
		for i := 0; i < len(s); i++ {
			if s[i] == '\\' && i+1 < len(s) {
				i++
				continue
			}
			switch s[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return s[1:i], i + 1
				}
			}
		}
		return s[1:], len(s)
	}
	if s[0] == '\\' {
		cmd, n := readTeXCommand(s[1:])
		return "\\" + cmd, 1 + n
	}
	_, size := utf8.DecodeRuneInString(s)
	return s[:size], size
}

func mapScript(s string, m map[rune]rune, fallback string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		u, ok := m[r]
		if !ok {
			return fallback + "{" + s + "}"
		}
		b.WriteRune(u)
	}
	return b.String()
}

// appendMathAwareText splits plain text on $...$ / $$...$$ (with \$ escapes)
// and paints transformed formulas as accent monospace spans.
func appendMathAwareText(text string, st Style, out *[]ui.Span, mutators ...spanMutator) {
	if !strings.Contains(text, "$") {
		appendPlainSpan(text, st, out, mutators...)
		return
	}
	rest := text
	for rest != "" {
		plain, math, block, after, ok := splitNextMath(rest)
		if plain != "" {
			appendPlainSpan(plain, st, out, mutators...)
		}
		if !ok {
			return
		}
		display := transformTeX(math)
		if block {
			display = strings.TrimSpace(display)
		}
		appendMathSpan(display, st, out)
		rest = after
	}
}

func splitNextMath(s string) (plain, math string, block bool, after string, ok bool) {
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '$' {
			plain += s[:i] + "$"
			s = s[i+2:]
			i = 0
			continue
		}
		if s[i] == '$' {
			plain += s[:i]
			s = s[i:]
			break
		}
		i++
	}
	if i >= len(s) || !strings.HasPrefix(s, "$") {
		return plain + s, "", false, "", false
	}

	block = strings.HasPrefix(s, "$$")
	openLen := 1
	if block {
		openLen = 2
	}
	body := s[openLen:]
	end := indexMathClose(body, block)
	if end < 0 {
		return plain + s, "", false, "", false
	}
	closeLen := 1
	if block {
		closeLen = 2
	}
	return plain, body[:end], block, body[end+closeLen:], true
}

func indexMathClose(s string, block bool) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] != '$' {
			continue
		}
		if block {
			if i+1 < len(s) && s[i+1] == '$' {
				return i
			}
			continue
		}
		// inline: a lone $; do not treat the first half of $$ as close
		if i+1 < len(s) && s[i+1] == '$' {
			continue
		}
		return i
	}
	return -1
}

func appendPlainSpan(text string, st Style, out *[]ui.Span, mutators ...spanMutator) {
	if text == "" {
		return
	}
	span := ui.Span{Text: text, Size: st.FontSize, Color: st.FG}
	for _, m := range mutators {
		m(&span)
	}
	*out = append(*out, span)
}

func appendMathSpan(formula string, st Style, out *[]ui.Span) {
	color := st.Accent
	if color.A == 0 && color.R == 0 && color.G == 0 && color.B == 0 {
		color = st.FG
	}
	*out = append(*out, ui.Span{
		Text:   formula,
		Size:   st.FontSize,
		Color:  color,
		Font:   "Consolas",
		Italic: true,
	})
}
