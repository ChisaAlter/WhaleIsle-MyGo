package htmldocument

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

const (
	maxCSSTokens = 200000
	maxCSSBytes  = 25 << 20
	emptyCSSURL  = `url("data:,")`
)

type cssLexeme struct {
	kind css.TokenType
	text string
}

// inlineCSS only resolves dependencies through the caller's confined loaders.
// base always belongs to this stylesheet; loadCSS selects each import's base.
func inlineCSS(source, base string, depth int, loadCSS func(base, reference string, depth int) (string, error), loadURL func(base, reference string) (string, error), warn func(string)) (string, error) {
	if len(source) > maxCSSBytes {
		return "", errors.New("CSS snapshot exceeds the size limit")
	}
	if !utf8.ValidString(source) || strings.IndexByte(source, 0) >= 0 {
		return "", errors.New("CSS snapshot requires UTF-8 text without null bytes")
	}
	if warn == nil {
		warn = func(string) {}
	}
	// Match CSS input preprocessing, including CRLF following a hex escape.
	// The lexer consumes one whitespace byte after escapes, so normalize first.
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	source = strings.ReplaceAll(source, "\f", "\n")
	lexer := css.NewLexer(parse.NewInputString(source))
	tokens := make([]cssLexeme, 0)
	for {
		kind, data := lexer.Next()
		if kind == css.ErrorToken {
			if lexer.Err() != io.EOF {
				return "", errors.New("CSS snapshot could not be parsed")
			}
			break
		}
		if len(tokens) >= maxCSSTokens {
			return "", errors.New("CSS snapshot exceeds the token limit")
		}
		tokens = append(tokens, cssLexeme{kind: kind, text: string(data)})
	}

	var out strings.Builder
	write := func(value string) error {
		if len(value) > maxCSSBytes-out.Len() {
			return errors.New("CSS snapshot exceeds the size limit")
		}
		out.WriteString(value)
		return nil
	}
	resolveURL := func(reference string, valid bool) (string, error) {
		if !valid || reference == "" {
			warn("A malformed CSS resource was omitted")
			return emptyCSSURL, nil
		}
		if loadURL == nil {
			warn("A CSS resource could not be included")
			return emptyCSSURL, nil
		}
		value, err := loadURL(base, reference)
		if err != nil || !(strings.HasPrefix(value, "#") || (len(value) >= 5 && strings.EqualFold(value[:5], "data:"))) {
			warn("A CSS resource could not be included")
			return emptyCSSURL, nil
		}
		return cssQuotedURL(value)
	}

	stringFunctions := cssFunctionsWithStrings(tokens)
	braceDepth := 0
	for i := 0; i < len(tokens); {
		token := tokens[i]
		value := token.text
		next := i + 1
		switch token.kind {
		case css.AtKeywordToken:
			name, valid := decodeCSSEscapes(strings.TrimPrefix(token.text, "@"), false)
			if valid && strings.EqualFold(name, "import") {
				end, after, complete := cssImportEnd(tokens, next)
				next = after
				value = "\n/* CSS import omitted. */\n"
				if !complete || braceDepth != 0 {
					warn("A malformed CSS import was omitted")
					break
				}
				reference, media, valid := cssImportParts(tokens[i+1 : end])
				if !valid {
					warn("An unsupported CSS import was omitted")
					break
				}
				if loadCSS == nil {
					warn("A CSS import could not be included")
					break
				}
				imported, err := loadCSS(base, reference, depth+1)
				if err != nil {
					warn("A CSS import could not be included")
					break
				}
				if len(imported) > maxCSSBytes-32-len(media) {
					return "", errors.New("CSS snapshot exceeds the size limit")
				}
				value = "\n" + imported + "\n"
				if media != "" {
					value = "\n@media " + media + " {\n" + imported + "\n}\n"
				}
			}
		case css.URLToken:
			reference, valid := cssURLReference(token.text)
			var err error
			value, err = resolveURL(reference, valid)
			if err != nil {
				return "", err
			}
		case css.FunctionToken:
			// The lexer returns FunctionToken for hex-escaped spellings of url.
			// Browsers decode the identifier, so these must use the same loader.
			name, valid := decodeCSSEscapes(strings.TrimSuffix(token.text, "("), false)
			if valid && (strings.EqualFold(name, "image-set") || strings.EqualFold(name, "-webkit-image-set")) && stringFunctions[i] {
				next, _ = cssFunctionEnd(tokens, i)
				value = emptyCSSURL
				warn("CSS image-set string resources are not supported")
				break
			}
			if valid && strings.EqualFold(name, "url") {
				end, complete := cssFunctionEnd(tokens, i)
				next = end
				reference := ""
				if complete {
					reference, valid = cssURLReference("url(" + cssJoin(tokens[i+1:end-1]) + ")")
				} else {
					valid = false
				}
				var err error
				value, err = resolveURL(reference, valid)
				if err != nil {
					return "", err
				}
			}
		case css.BadURLToken:
			warn("A malformed CSS resource was omitted")
			value = emptyCSSURL
		case css.BadStringToken:
			warn("A malformed CSS string was omitted")
			value = `""`
		case css.LeftBraceToken:
			braceDepth++
		case css.RightBraceToken:
			if braceDepth > 0 {
				braceDepth--
			}
		}
		if err := write(value); err != nil {
			return "", err
		}
		i = next
	}

	// Escape across token boundaries as well as inside strings and comments.
	// HTML raw-text parsing recognizes closing tags before CSS parsing runs.
	result := out.String()
	if strings.Count(result, "</") > (maxCSSBytes-len(result))/3 {
		return "", errors.New("CSS snapshot exceeds the size limit")
	}
	return strings.ReplaceAll(result, "</", `\3c /`), nil
}

// cssImportEnd consumes a whole at-rule, including an invalid block, while
// preserving a containing rule's closing brace for the caller.
func cssImportEnd(tokens []cssLexeme, start int) (end, next int, complete bool) {
	parens := 0
	for i := start; i < len(tokens); i++ {
		switch tokens[i].kind {
		case css.FunctionToken, css.LeftParenthesisToken:
			parens++
		case css.RightParenthesisToken:
			if parens == 0 {
				return i, i + 1, false
			}
			parens--
		case css.SemicolonToken:
			if parens == 0 {
				return i, i + 1, true
			}
		case css.RightBraceToken:
			return i, i, false
		case css.LeftBraceToken:
			braces := 1
			for j := i + 1; j < len(tokens); j++ {
				switch tokens[j].kind {
				case css.LeftBraceToken:
					braces++
				case css.RightBraceToken:
					braces--
					if braces == 0 {
						return i, j + 1, false
					}
				}
			}
			return i, len(tokens), false
		}
	}
	return len(tokens), len(tokens), parens == 0
}

func cssImportParts(tokens []cssLexeme) (reference, media string, valid bool) {
	i := 0
	for i < len(tokens) && cssTrivia(tokens[i].kind) {
		i++
	}
	if i == len(tokens) {
		return "", "", false
	}
	switch tokens[i].kind {
	case css.StringToken:
		reference, valid = cssStringValue(tokens[i].text)
		i++
	case css.URLToken:
		reference, valid = cssURLReference(tokens[i].text)
		i++
	case css.FunctionToken:
		name, ok := decodeCSSEscapes(strings.TrimSuffix(tokens[i].text, "("), false)
		if !ok || !strings.EqualFold(name, "url") {
			return "", "", false
		}
		end, complete := cssFunctionEnd(tokens, i)
		if !complete {
			return "", "", false
		}
		reference, valid = cssURLReference("url(" + cssJoin(tokens[i+1:end-1]) + ")")
		i = end
	default:
		return "", "", false
	}
	if !valid || reference == "" {
		return "", "", false
	}
	// Support media types, boolean operators, and parenthesized feature tests.
	// layer()/supports() need different wrappers and must never become @media.
	parens := 0
	hasMedia := false
	for _, token := range tokens[i:] {
		if cssTrivia(token.kind) {
			continue
		}
		hasMedia = true
		switch token.kind {
		case css.NumberToken, css.DimensionToken, css.PercentageToken, css.ColonToken, css.CommaToken:
		case css.IdentToken:
			name, ok := decodeCSSEscapes(token.text, false)
			if !ok || strings.EqualFold(name, "layer") || strings.EqualFold(name, "supports") {
				return "", "", false
			}
		case css.LeftParenthesisToken:
			parens++
		case css.RightParenthesisToken:
			parens--
			if parens < 0 {
				return "", "", false
			}
		case css.DelimToken:
			if !strings.Contains("><=/", token.text) || len(token.text) != 1 {
				return "", "", false
			}
		default:
			return "", "", false
		}
	}
	if parens != 0 {
		return "", "", false
	}
	if !hasMedia {
		return reference, "", true
	}
	return reference, strings.TrimSpace(cssJoin(tokens[i:])), true
}

func cssFunctionEnd(tokens []cssLexeme, start int) (int, bool) {
	depth := 1
	for i := start + 1; i < len(tokens); i++ {
		switch tokens[i].kind {
		case css.FunctionToken, css.LeftParenthesisToken:
			depth++
		case css.RightParenthesisToken:
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
	}
	return len(tokens), false
}

// Record direct string arguments in one pass. In image-set(), type("...")
// describes a MIME type, while a direct "..." is an unsupported resource URL.
func cssFunctionsWithStrings(tokens []cssLexeme) map[int]bool {
	result := make(map[int]bool)
	var stack []int
	for i, token := range tokens {
		switch token.kind {
		case css.FunctionToken, css.LeftParenthesisToken:
			stack = append(stack, i)
		case css.RightParenthesisToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case css.StringToken, css.BadStringToken:
			if len(stack) > 0 {
				result[stack[len(stack)-1]] = true
			}
		}
	}
	return result
}

func cssTrivia(kind css.TokenType) bool {
	return kind == css.WhitespaceToken || kind == css.CommentToken
}

func cssJoin(tokens []cssLexeme) string {
	var out strings.Builder
	for _, token := range tokens {
		out.WriteString(token.text)
	}
	return out.String()
}

func cssURLReference(raw string) (string, bool) {
	open := strings.IndexByte(raw, '(')
	if open < 0 || !strings.HasSuffix(raw, ")") {
		return "", false
	}
	value := strings.Trim(raw[open+1:len(raw)-1], " \t\r\n\f")
	if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
		return cssStringValue(value)
	}
	if !css.IsURLUnquoted([]byte(value)) {
		return "", false
	}
	return decodeCSSEscapes(value, false)
}

func cssStringValue(raw string) (string, bool) {
	if len(raw) < 2 || (raw[0] != '\'' && raw[0] != '"') || raw[len(raw)-1] != raw[0] {
		return "", false
	}
	return decodeCSSEscapes(raw[1:len(raw)-1], true)
}

// Decode CSS escapes rather than URL percent escapes. Workspace URL validation
// remains the loader's job. Invalid Unicode and control characters fail closed.
func decodeCSSEscapes(raw string, allowContinuation bool) (string, bool) {
	if !utf8.ValidString(raw) {
		return "", false
	}
	var out strings.Builder
	for i := 0; i < len(raw); {
		c := raw[i]
		if c != '\\' {
			if c < 0x20 || c == 0x7f {
				return "", false
			}
			out.WriteByte(c)
			i++
			continue
		}
		i++
		if i == len(raw) {
			return "", false
		}
		if raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\f' {
			if !allowContinuation {
				return "", false
			}
			if raw[i] == '\r' && i+1 < len(raw) && raw[i+1] == '\n' {
				i++
			}
			i++
			continue
		}
		value, digits := rune(0), 0
		for i < len(raw) && digits < 6 {
			digit := strings.IndexByte("0123456789abcdef", raw[i]|0x20)
			if digit < 0 {
				break
			}
			value = value*16 + rune(digit)
			digits++
			i++
		}
		if digits > 0 {
			if value < 0x20 || value == 0x7f || !utf8.ValidRune(value) {
				return "", false
			}
			out.WriteRune(value)
			if i < len(raw) && strings.IndexByte(" \t\r\n\f", raw[i]) >= 0 {
				if raw[i] == '\r' && i+1 < len(raw) && raw[i+1] == '\n' {
					i++
				}
				i++
			}
		} else {
			r, size := utf8.DecodeRuneInString(raw[i:])
			if r < 0x20 || r == 0x7f {
				return "", false
			}
			out.WriteRune(r)
			i += size
		}
	}
	return out.String(), true
}

func cssQuotedURL(value string) (string, error) {
	if len(value) > maxCSSBytes || !utf8.ValidString(value) {
		return "", errors.New("CSS resource exceeds the encoding limit")
	}
	var out strings.Builder
	out.WriteString(`url("`)
	for _, r := range value {
		if out.Len() > maxCSSBytes-12 {
			return "", errors.New("CSS resource exceeds the encoding limit")
		}
		switch {
		case r == '\\' || r == '"':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r < 0x20 || r == 0x7f || r == '<':
			out.WriteByte('\\')
			out.WriteString(strconv.FormatInt(int64(r), 16))
			out.WriteByte(' ')
		default:
			out.WriteRune(r)
		}
	}
	out.WriteString(`")`)
	return out.String(), nil
}
