package lang

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// tokKind is a token type (upstream lexer.ts numbering is not relied on).
type tokKind uint8

const (
	tNewline tokKind = iota
	tLParen
	tRParen
	tLBrack
	tRBrack
	tLBrace
	tRBrace
	tComma
	tColon
	tEquals
	tTrue
	tFalse
	tNull
	tEOF
	tStr
	tNum
	tIdent    // reference identifier (lowercase / _ / non-ASCII-uppercase start)
	tType     // identifier starting with A-Z
	tStateVar // $name (value includes $)
	tDot
	tPlus
	tMinus
	tStar
	tSlash
	tPercent
	tEqEq
	tNotEq
	tGreater
	tLess
	tGreaterEq
	tLessEq
	tAnd
	tOr
	tNot
	tQuestion
	tBuiltin // @name (value excludes @)
)

type token struct {
	k tokKind
	s string  // identifiers, strings
	n float64 // numbers
}

func isIdentStartByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

// identStart reports whether the rune at src[i:] starts an identifier.
// ASCII letters and _ always do; with unicode, letters (ID_Start) do too.
func identStart(src string, i int, unicodeIdents bool) (int, bool) {
	if i >= len(src) {
		return 0, false
	}
	c := src[i]
	if c < utf8.RuneSelf {
		return 1, isIdentStartByte(c)
	}
	if !unicodeIdents {
		return 0, false
	}
	r, size := utf8.DecodeRuneInString(src[i:])
	if unicode.In(r, unicode.L, unicode.Nl, unicode.Other_ID_Start) {
		return size, true
	}
	return 0, false
}

func identContinue(src string, i int, unicodeIdents bool) int {
	c := src[i]
	if c < utf8.RuneSelf {
		if isIdentStartByte(c) || c >= '0' && c <= '9' {
			return 1
		}
		return 0
	}
	if !unicodeIdents {
		return 0
	}
	r, size := utf8.DecodeRuneInString(src[i:])
	if r == 0x200C || r == 0x200D || unicode.In(r, unicode.L, unicode.Nl, unicode.Other_ID_Start, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue) {
		return size
	}
	return 0
}

func scanIdent(src string, i int, unicodeIdents bool) int {
	for i < len(src) {
		n := identContinue(src, i, unicodeIdents)
		if n == 0 {
			break
		}
		i += n
	}
	return i
}

var tokSlicePool = sync.Pool{
	New: func() any {
		s := make([]token, 0, 256)
		return &s
	},
}

func acquireTokBuf(hint int) *[]token {
	bp := tokSlicePool.Get().(*[]token)
	s := *bp
	if cap(s) < hint {
		s = make([]token, 0, hint)
	} else {
		s = s[:0]
	}
	*bp = s
	return bp
}

func releaseTokBuf(bp *[]token) {
	s := *bp
	if cap(s) > 65536 {
		return
	}
	clear(s)
	*bp = s[:0]
	tokSlicePool.Put(bp)
}

// tokenize lexes source text. Lexing never fails: characters that match no
// rule are skipped. It returns false when the token budget is exceeded.
func tokenize(src string, unicodeIdents bool, maxTokens int) ([]token, bool) {
	return tokenizeInto(make([]token, 0, len(src)/3+8), src, unicodeIdents, maxTokens)
}

func tokenizeInto(toks []token, src string, unicodeIdents bool, maxTokens int) ([]token, bool) {
	i, n := 0, len(src)
	for i < n {
		if maxTokens > 0 && len(toks) >= maxTokens {
			return toks, false
		}
		for i < n && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r') {
			i++
		}
		if i >= n {
			break
		}
		c := src[i]
		switch c {
		case '\n':
			toks = append(toks, token{k: tNewline})
			i++
			continue
		case '(':
			toks = append(toks, token{k: tLParen})
			i++
			continue
		case ')':
			toks = append(toks, token{k: tRParen})
			i++
			continue
		case '[':
			toks = append(toks, token{k: tLBrack})
			i++
			continue
		case ']':
			toks = append(toks, token{k: tRBrack})
			i++
			continue
		case '{':
			toks = append(toks, token{k: tLBrace})
			i++
			continue
		case '}':
			toks = append(toks, token{k: tRBrace})
			i++
			continue
		case ',':
			toks = append(toks, token{k: tComma})
			i++
			continue
		case ':':
			toks = append(toks, token{k: tColon})
			i++
			continue
		case '=':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{k: tEqEq})
				i += 2
			} else {
				toks = append(toks, token{k: tEquals})
				i++
			}
			continue
		case '!':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{k: tNotEq})
				i += 2
			} else {
				toks = append(toks, token{k: tNot})
				i++
			}
			continue
		case '>':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{k: tGreaterEq})
				i += 2
			} else {
				toks = append(toks, token{k: tGreater})
				i++
			}
			continue
		case '<':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{k: tLessEq})
				i += 2
			} else {
				toks = append(toks, token{k: tLess})
				i++
			}
			continue
		case '&':
			toks = append(toks, token{k: tAnd})
			if i+1 < n && src[i+1] == '&' {
				i += 2
			} else {
				i++
			}
			continue
		case '|':
			toks = append(toks, token{k: tOr})
			if i+1 < n && src[i+1] == '|' {
				i += 2
			} else {
				i++
			}
			continue
		case '.':
			toks = append(toks, token{k: tDot})
			i++
			continue
		case '?':
			toks = append(toks, token{k: tQuestion})
			i++
			continue
		case '+':
			toks = append(toks, token{k: tPlus})
			i++
			continue
		case '*':
			toks = append(toks, token{k: tStar})
			i++
			continue
		case '/':
			toks = append(toks, token{k: tSlash})
			i++
			continue
		case '%':
			toks = append(toks, token{k: tPercent})
			i++
			continue
		case '"':
			start := i
			i++
			closed := false
			hasEscape := false
			hasCtrl := false
			for i < n {
				b := src[i]
				if b == '\\' {
					hasEscape = true
					i += 2
				} else if b == '"' {
					i++
					closed = true
					break
				} else {
					if b < 0x20 {
						hasCtrl = true
					}
					i++
				}
			}
			if i > n {
				i = n
			}
			if !hasEscape && !hasCtrl {
				end := i
				if closed {
					end--
				}
				toks = append(toks, token{k: tStr, s: src[start+1 : end]})
				continue
			}
			raw := src[start:i]
			lit := raw
			if !closed {
				lit = raw + `"`
			}
			if v, ok := decodeJSONStringLiteral(lit); ok {
				toks = append(toks, token{k: tStr, s: v})
			} else {
				// JSON.parse failed: keep the raw text without its quotes.
				s := strings.TrimPrefix(raw, `"`)
				s = strings.TrimSuffix(s, `"`)
				toks = append(toks, token{k: tStr, s: s})
			}
			continue
		case '\'':
			i++
			start := i
			hasEscape := false
			for i < n {
				if src[i] == '\\' {
					hasEscape = true
					break
				}
				if src[i] == '\'' {
					toks = append(toks, token{k: tStr, s: src[start:i]})
					i++
					goto nextTok
				}
				i++
			}
			if !hasEscape {
				toks = append(toks, token{k: tStr, s: src[start:i]})
				continue
			}
			var b strings.Builder
			b.WriteString(src[start:i])
			for i < n {
				if src[i] == '\\' {
					i++
					if i < n {
						switch src[i] {
						case '\'':
							b.WriteByte('\'')
						case '\\':
							b.WriteByte('\\')
						case 'n':
							b.WriteByte('\n')
						case 't':
							b.WriteByte('\t')
						default:
							b.WriteByte(src[i])
						}
						i++
					}
				} else if src[i] == '\'' {
					i++
					break
				} else {
					b.WriteByte(src[i])
					i++
				}
			}
			toks = append(toks, token{k: tStr, s: b.String()})
			continue
		case '-':
			afterValue := false
			if len(toks) > 0 {
				switch toks[len(toks)-1].k {
				case tNum, tStr, tIdent, tType, tRParen, tRBrack, tTrue, tFalse, tNull, tStateVar, tBuiltin:
					afterValue = true
				}
			}
			if afterValue || !(i+1 < n && src[i+1] >= '0' && src[i+1] <= '9') {
				toks = append(toks, token{k: tMinus})
				i++
				continue
			}
		}
		if c >= '0' && c <= '9' || c == '-' {
			start := i
			neg := false
			if src[i] == '-' {
				neg = true
				i++
			}
			var iv int64
			intDigits := 0
			for i < n && src[i] >= '0' && src[i] <= '9' {
				if intDigits < 15 {
					iv = iv*10 + int64(src[i]-'0')
				}
				intDigits++
				i++
			}
			isFloat := false
			if i < n && src[i] == '.' && i+1 < n && src[i+1] >= '0' && src[i+1] <= '9' {
				isFloat = true
				i++
				for i < n && src[i] >= '0' && src[i] <= '9' {
					i++
				}
			}
			if i < n && (src[i] == 'e' || src[i] == 'E') {
				isFloat = true
				i++
				if i < n && (src[i] == '+' || src[i] == '-') {
					i++
				}
				for i < n && src[i] >= '0' && src[i] <= '9' {
					i++
				}
			}
			var num float64
			if !isFloat && intDigits > 0 && intDigits <= 15 {
				num = float64(iv)
				if neg {
					num = -num
				}
			} else {
				raw := src[start:i]
				f, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
						num = f
					} else {
						num = math.NaN()
					}
				} else {
					num = f
				}
			}
			toks = append(toks, token{k: tNum, n: num})
			continue
		}
		if c == '$' {
			if _, ok := identStart(src, i+1, unicodeIdents); ok {
				start := i
				i = scanIdent(src, i+1, unicodeIdents)
				toks = append(toks, token{k: tStateVar, s: src[start:i]})
				continue
			}
		}
		if _, ok := identStart(src, i, unicodeIdents); ok {
			start := i
			i = scanIdent(src, i, unicodeIdents)
			word := src[start:i]
			switch word {
			case "true":
				toks = append(toks, token{k: tTrue})
			case "false":
				toks = append(toks, token{k: tFalse})
			case "null":
				toks = append(toks, token{k: tNull})
			default:
				k := tIdent
				if c >= 'A' && c <= 'Z' {
					k = tType
				}
				toks = append(toks, token{k: k, s: word})
			}
			continue
		}
		if c == '@' {
			if _, ok := identStart(src, i+1, unicodeIdents); ok {
				start := i + 1
				i = scanIdent(src, start, unicodeIdents)
				toks = append(toks, token{k: tBuiltin, s: src[start:i]})
				continue
			}
		}
		// Skip any other character (a whole UTF-8 sequence).
		if c < utf8.RuneSelf {
			i++
		} else {
			_, size := utf8.DecodeRuneInString(src[i:])
			i += size
		}
	nextTok:
	}
	toks = append(toks, token{k: tEOF})
	return toks, true
}

func sortStrings(s []string) { sort.Strings(s) }
