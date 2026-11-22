package markdown

import (
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/egoist/mygo/ui"
)

// highlightedCodeSpans tokenizes a fenced code block with chroma.
// Falls back to a single monospace span when the language is unknown.
func highlightedCodeSpans(lang, code string, st Style) []ui.Span {
	return codeSpans(lang, strings.TrimRight(code, "\n"), st, nil, nil)
}

// codeSpans shares tokenization while the owner chooses source normalization
// and token colors. A nil option/color pair retains the shiki-sheet preset.
func codeSpans(lang, code string, st Style, options *chroma.TokeniseOptions, color func(toks []chroma.Token, i int) ui.Color) []ui.Span {
	if code == "" {
		return nil
	}
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		return withLineNumbers([]ui.Span{{Text: code, Size: size, Background: codeBodyFill(st), Font: font}}, st)
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(options, code)
	if err != nil {
		return withLineNumbers([]ui.Span{{Text: code, Size: size, Background: codeBodyFill(st), Font: font}}, st)
	}
	toks := it.Tokens()
	ctx := shikiCtx{p: newShikiPalette(st.Dark, st.FG), lang: shikiLang(lang), toks: toks}
	var out []ui.Span
	for i, tok := range toks {
		if tok.Value == "" {
			continue
		}
		ink := ctx.color(i)
		if color != nil {
			ink = color(toks, i)
		}
		out = append(out, ui.Span{
			Text:       tok.Value,
			Size:       size,
			Color:      ink,
			Background: codeBodyFill(st),
			Font:       font,
		})
	}
	if len(out) == 0 {
		return withLineNumbers([]ui.Span{{Text: code, Size: size, Background: codeBodyFill(st), Font: font}}, st)
	}
	return withLineNumbers(out, st)
}

// withLineNumbers prefixes each source line with a right-aligned counter in the
// code gutter. Numbers are their own muted spans, so the block still renders as
// one wrapped RichText and stays selectable/copyable as one source.
func withLineNumbers(spans []ui.Span, st Style) []ui.Span {
	if !st.LineNumbers {
		return spans
	}
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	width := 2
	lines := 1
	for _, span := range spans {
		lines += strings.Count(span.Text, "\n")
	}
	for n := lines; n >= 10; n /= 10 {
		width++
	}
	out := make([]ui.Span, 0, len(spans)+lines)
	line := 1
	prefix := func() {
		number := padNumber(line, width)
		out = append(out, ui.Span{Text: number + "  ", Size: size, Color: st.Muted.Alpha(.75), Background: codeBodyFill(st), Font: font})
		line++
	}
	prefix()
	for _, span := range spans {
		parts := strings.Split(span.Text, "\n")
		for index, part := range parts {
			if index > 0 {
				out = append(out, ui.Span{Text: "\n", Size: size, Background: codeBodyFill(st), Font: font})
				prefix()
			}
			if part == "" {
				continue
			}
			copied := span
			copied.Text = part
			out = append(out, copied)
		}
	}
	return out
}

func padNumber(n, width int) string {
	text := ""
	for n > 0 {
		text = string(rune('0'+n%10)) + text
		n /= 10
	}
	if text == "" {
		text = "0"
	}
	for len(text) < width {
		text = " " + text
	}
	return text
}

// shikiPalette is the original's --shiki-* token sheet
// (ui-theme/src/styles/shiki.css): light values on :root, dark overrides on
// body[data-ds-dark-theme]. The sheet is fixed, not family-overridable.
type shikiPalette struct {
	fg, constant, str, comment           ui.Color
	keyword, param, function, stringExpr ui.Color
	punct, link                          ui.Color
}

func newShikiPalette(dark bool, fg ui.Color) shikiPalette {
	p := shikiPalette{fg: fg}
	if dark {
		p.constant = ui.Hex("#4dabf7")
		p.str = ui.Hex("#69db7c")
		p.comment = ui.Hex("#adb5bd")
		p.keyword = ui.Hex("#faa2c1")
		p.param = ui.Hex("#ffa94d")
		p.function = ui.Hex("#b197fc")
		p.stringExpr = ui.Hex("#8ce99a")
		p.punct = ui.Hex("#ced4da")
		p.link = ui.Hex("#74c0fc")
		return p
	}
	p.constant = ui.Hex("#1c7ed6")
	p.str = ui.Hex("#2f9e44")
	p.comment = ui.Hex("#868e96")
	p.keyword = ui.Hex("#d6336c")
	p.param = ui.Hex("#e8590c")
	p.function = ui.Hex("#6741d9")
	p.stringExpr = ui.Hex("#2b8a3e")
	p.punct = ui.Hex("#495057")
	p.link = ui.Hex("#1971c2")
	return p
}

// shikiCtx colors one chroma token stream against the palette. Chroma emits
// flat token types where TextMate emits a scope stack, so a handful of
// language rules and identifier context heuristics reproduce the original's
// scope→token mapping (highlight-engine.ts + createCssVariablesTheme).
type shikiCtx struct {
	p         shikiPalette
	lang      string
	toks      []chroma.Token
	declConst bool
}

func shikiLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "ts", "tsx", "typescript", "typescriptreact", "js", "jsx", "javascript", "javascriptreact", "mjs", "cjs":
		return "js"
	case "go", "golang":
		return "go"
	case "py", "python", "python3":
		return "python"
	case "css", "scss", "less", "sass":
		return "css"
	case "json", "jsonc", "json5", "jsonl":
		return "json"
	case "bash", "sh", "shell", "shellscript", "zsh", "console":
		return "bash"
	}
	return strings.ToLower(strings.TrimSpace(lang))
}

func (s *shikiCtx) skip(i, dir int) int {
	for j := i + dir; j >= 0 && j < len(s.toks); j += dir {
		if strings.TrimSpace(s.toks[j].Value) != "" {
			return j
		}
	}
	return -1
}

func (s *shikiCtx) color(i int) ui.Color {
	tok := s.toks[i]
	// const-declaration tracking for the js family: `const x` marks x
	// variable.other.constant → constant (the original's blue).
	if s.lang == "js" {
		value := strings.TrimSpace(tok.Value)
		if strings.Contains(value, "\n") || value == ";" {
			s.declConst = false
		}
		if value == "const" && tok.Type.InCategory(chroma.Keyword) {
			s.declConst = true
		}
	}
	if out, ok := s.langColor(i); ok {
		return out
	}
	return s.baseColor(i)
}

// langColor carries the per-language cases where chroma's categories diverge
// from the original's scope names.
func (s *shikiCtx) langColor(i int) (ui.Color, bool) {
	tok := s.toks[i]
	switch s.lang {
	case "json":
		switch {
		case tok.Type == chroma.NameTag:
			return s.p.keyword, true // support.type.property-name.json
		case tok.Type == chroma.Punctuation && tok.Value == ":":
			return s.p.keyword, true // punctuation.separator.key-value
		case tok.Type == chroma.Punctuation && tok.Value == ",":
			return s.p.punct, true
		case tok.Type == chroma.Punctuation:
			return s.p.fg, true
		}
	case "bash":
		switch {
		case tok.Type == chroma.NameBuiltin:
			return s.p.function, true // meta.function-call / support.function
		case tok.Type == chroma.Text && strings.TrimSpace(tok.Value) != "":
			return s.p.str, true // string.unquoted bare-word arguments
		}
	case "css":
		switch {
		case tok.Type.InCategory(chroma.Keyword) || tok.Type.InSubCategory(chroma.LiteralNumber):
			return s.p.constant, true // meta.property-name / property values + units
		case tok.Type == chroma.NameClass || tok.Type == chroma.NameDecorator ||
			tok.Type == chroma.NameLabel || tok.Type == chroma.NameAttribute:
			return s.p.function, true // entity.other.attribute-name selectors
		case tok.Type == chroma.NameTag:
			return s.p.stringExpr, true // entity.name.tag element selectors
		case tok.Type == chroma.Punctuation:
			return s.p.fg, true
		}
	case "js", "go", "python":
		if tok.Type == chroma.Operator && tok.Value == "." {
			return s.p.fg, true // punctuation.accessor
		}
		if tok.Type == chroma.Punctuation && tok.Value == "." && s.callsFunction(i) {
			return s.p.function, true // `.log(` paints inside the call span
		}
	}
	return ui.Color{}, false
}

// callsFunction reports whether a `.`/`(` context turns the adjacent name
// into a call: `.name(`.
func (s *shikiCtx) callsFunction(i int) bool {
	next := s.skip(i, 1)
	if next < 0 || !isNameToken(s.toks[next].Type) {
		return false
	}
	after := s.skip(next, 1)
	return after >= 0 && s.toks[after].Value == "("
}

func isNameToken(t chroma.TokenType) bool {
	switch t {
	case chroma.Name, chroma.NameOther, chroma.NameVariable,
		chroma.NameVariableGlobal, chroma.NameVariableInstance,
		chroma.NameVariableClass, chroma.NameNamespace:
		return true
	}
	return false
}

func (s *shikiCtx) baseColor(i int) ui.Color {
	t := s.toks[i].Type
	switch {
	case t == chroma.KeywordConstant:
		return s.p.constant // constant.language (true/null/this-like literals)
	case t.InCategory(chroma.Keyword) || t.InCategory(chroma.Operator):
		return s.p.keyword // keyword.* covers operators in TextMate scope order
	case t == chroma.LiteralStringDoc:
		return s.p.comment // string.quoted.docstring.multi
	case t.InCategory(chroma.Comment):
		return s.p.comment
	case t == chroma.LiteralStringDouble || t == chroma.LiteralStringSingle ||
		t == chroma.LiteralStringChar || t == chroma.LiteralStringInterpol ||
		t == chroma.LiteralStringBacktick || t == chroma.LiteralStringRegex ||
		t == chroma.LiteralStringHeredoc || t == chroma.LiteralStringEscape:
		return s.p.stringExpr // string.quoted / interpolated / regexp / template
	case t.InSubCategory(chroma.LiteralString):
		return s.p.str
	case t.InSubCategory(chroma.LiteralNumber) || t == chroma.Literal || t == chroma.LiteralDate:
		return s.p.constant
	case t == chroma.NameBuiltin || t == chroma.NameBuiltinPseudo || t == chroma.NameConstant:
		return s.p.constant // support.* / variable.language / constant.language
	case t == chroma.NameProperty:
		return s.p.constant // meta.property-name
	case t == chroma.NameFunction || t == chroma.NameFunctionMagic ||
		t == chroma.NameDecorator || t == chroma.NameClass ||
		t == chroma.NameEntity || t == chroma.NameAttribute:
		return s.p.function // entity.name.function / support.function / attribute-name
	case t == chroma.NameTag:
		return s.p.stringExpr // entity.name.tag (html/xml/jsx element names)
	case isNameToken(t):
		return s.nameColor(i)
	}
	return s.p.fg
}

// nameColor reproduces variable.other.* decisions from token context:
// call sites are functions, object references and property access are
// constant, everything else is plain foreground.
func (s *shikiCtx) nameColor(i int) ui.Color {
	next, prev := s.skip(i, 1), s.skip(i, -1)
	if next >= 0 && s.toks[next].Value == "(" {
		return s.p.function // meta.function-call
	}
	if prev >= 0 && s.toks[prev].Value == "." {
		return s.p.constant // meta.property-name member read
	}
	if next >= 0 && s.toks[next].Value == "." {
		return s.p.constant // variable.other.object
	}
	if s.declConst {
		return s.p.constant // variable.other.constant (const declaration)
	}
	return s.p.fg
}
