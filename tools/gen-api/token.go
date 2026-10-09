package main

import (
	"fmt"
	"strings"
)

// A minimal JavaScript tokenizer good enough for machine-generated
// typert.remote-client.js files. It skips comments, string/template/regex
// literals and emits identifiers, strings, numbers and punctuation.

type tokKind int

const (
	tkEOF tokKind = iota
	tkIdent
	tkString
	tkNumber
	tkPunct
)

type token struct {
	kind tokKind
	val  string // identifier text, decoded string value or punct
	line int
}

type tokenizer struct {
	src  string
	pos  int
	line int
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func tokenize(src string) ([]token, error) {
	t := &tokenizer{src: src, line: 1}
	var out []token
	for {
		t.skipTrivia()
		if t.pos >= len(t.src) {
			return append(out, token{kind: tkEOF, line: t.line}), nil
		}
		c := t.src[t.pos]
		switch {
		case isIdentStart(c):
			start := t.pos
			for t.pos < len(t.src) && isIdentPart(t.src[t.pos]) {
				t.pos++
			}
			out = append(out, token{kind: tkIdent, val: t.src[start:t.pos], line: t.line})
		case c >= '0' && c <= '9':
			start := t.pos
			for t.pos < len(t.src) && (isIdentPart(t.src[t.pos]) || t.src[t.pos] == '.') {
				t.pos++
			}
			out = append(out, token{kind: tkNumber, val: t.src[start:t.pos], line: t.line})
		case c == '"' || c == '\'' || c == '`':
			s, err := t.readString(c)
			if err != nil {
				return nil, err
			}
			out = append(out, token{kind: tkString, val: s, line: t.line})
		case c == '/' && regexAllowed(out):
			if err := t.skipRegex(); err != nil {
				return nil, err
			}
		default:
			if t.pos+1 < len(t.src) && t.src[t.pos:t.pos+2] == "=>" {
				out = append(out, token{kind: tkPunct, val: "=>", line: t.line})
				t.pos += 2
				continue
			}
			if strings.IndexByte("()[]{},.;:=+-*/%<>!&|^?~", c) < 0 {
				return nil, fmt.Errorf("line %d: unexpected character %q", t.line, c)
			}
			out = append(out, token{kind: tkPunct, val: string(c), line: t.line})
			t.pos++
		}
	}
}

func (t *tokenizer) skipTrivia() {
	for t.pos < len(t.src) {
		c := t.src[t.pos]
		switch {
		case c == '\n':
			t.line++
			t.pos++
		case c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f':
			t.pos++
		case c == '/' && t.pos+1 < len(t.src) && t.src[t.pos+1] == '/':
			for t.pos < len(t.src) && t.src[t.pos] != '\n' {
				t.pos++
			}
		case c == '/' && t.pos+1 < len(t.src) && t.src[t.pos+1] == '*':
			t.pos += 2
			for t.pos < len(t.src) {
				if t.src[t.pos] == '\n' {
					t.line++
				}
				if t.src[t.pos] == '*' && t.pos+1 < len(t.src) && t.src[t.pos+1] == '/' {
					t.pos += 2
					break
				}
				t.pos++
			}
		default:
			return
		}
	}
}

// readString consumes a quoted string and returns its decoded value.
// Template literals (backticks) are consumed but their content is not
// decoded; generated descriptor files keep data in plain quotes.
func (t *tokenizer) readString(q byte) (string, error) {
	startLine := t.line
	t.pos++
	var b strings.Builder
	if q == '`' {
		for t.pos < len(t.src) {
			c := t.src[t.pos]
			if c == '\\' {
				t.pos += 2
				continue
			}
			if c == '`' {
				t.pos++
				return "", nil
			}
			if c == '$' && t.pos+1 < len(t.src) && t.src[t.pos+1] == '{' {
				t.pos += 2
				depth := 1
				for t.pos < len(t.src) && depth > 0 {
					switch t.src[t.pos] {
					case '{':
						depth++
					case '}':
						depth--
					case '\n':
						t.line++
					}
					t.pos++
				}
				continue
			}
			if c == '\n' {
				t.line++
			}
			t.pos++
		}
		return "", fmt.Errorf("line %d: unterminated template literal", startLine)
	}
	for t.pos < len(t.src) {
		c := t.src[t.pos]
		if c == q {
			t.pos++
			return b.String(), nil
		}
		if c == '\n' {
			return "", fmt.Errorf("line %d: unterminated string literal", startLine)
		}
		if c == '\\' {
			t.pos++
			if t.pos >= len(t.src) {
				break
			}
			switch e := t.src[t.pos]; e {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'v':
				b.WriteByte('\v')
			case '\n':
				t.line++
			default:
				b.WriteByte(e)
			}
			t.pos++
			continue
		}
		b.WriteByte(c)
		t.pos++
	}
	return "", fmt.Errorf("line %d: unterminated string literal", startLine)
}

// regexAllowed decides whether a '/' starts a regex literal given the
// previously emitted token (standard JS division-vs-regex heuristic).
func regexAllowed(out []token) bool {
	if len(out) == 0 {
		return true
	}
	t := out[len(out)-1]
	switch t.kind {
	case tkIdent:
		switch t.val {
		case "return", "typeof", "case", "in", "of", "new", "delete", "void", "do", "else", "yield", "await":
			return true
		}
		return false
	case tkString, tkNumber:
		return false
	case tkPunct:
		return t.val != ")" && t.val != "]"
	}
	return true
}

func (t *tokenizer) skipRegex() error {
	startLine := t.line
	t.pos++
	inClass := false
	for t.pos < len(t.src) {
		c := t.src[t.pos]
		switch {
		case c == '\\':
			t.pos += 2
			continue
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case c == '/' && !inClass:
			t.pos++
			for t.pos < len(t.src) && isIdentPart(t.src[t.pos]) {
				t.pos++
			}
			return nil
		case c == '\n':
			return fmt.Errorf("line %d: unterminated regex literal", startLine)
		}
		t.pos++
	}
	return fmt.Errorf("line %d: unterminated regex literal", startLine)
}
