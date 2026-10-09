package main

import (
	"fmt"
	"strconv"
)

// jsNode is a parsed JavaScript value: object ('o'), array ('a'), string
// ('s'), number ('n'), boolean ('b'), null/undefined ('x') or bare
// identifier ('i', e.g. a schema constant reference).
type jsNode struct {
	kind byte
	str  string
	num  float64
	keys []string
	vals []*jsNode
}

func (n *jsNode) get(key string) *jsNode {
	if n == nil || n.kind != 'o' {
		return nil
	}
	for i, k := range n.keys {
		if k == key {
			return n.vals[i]
		}
	}
	return nil
}

func nodeString(n *jsNode, key string) string {
	v := n.get(key)
	if v == nil || v.kind != 's' {
		return ""
	}
	return v.str
}

func nodeBool(n *jsNode, key string) bool {
	v := n.get(key)
	return v != nil && v.kind == 'b' && v.str == "true"
}

type jsParser struct {
	toks []token
	pos  int
}

func (p *jsParser) peek() token { return p.toks[p.pos] }

func (p *jsParser) peekAt(n int) token {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n]
	}
	return p.toks[len(p.toks)-1]
}

func (p *jsParser) next() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *jsParser) parseValue() (*jsNode, error) {
	t := p.peek()
	switch {
	case t.kind == tkPunct && t.val == "{":
		return p.parseObject()
	case t.kind == tkPunct && t.val == "[":
		return p.parseArray()
	case t.kind == tkString:
		p.pos++
		return &jsNode{kind: 's', str: t.val}, nil
	case t.kind == tkNumber:
		p.pos++
		f, err := strconv.ParseFloat(t.val, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad number %q", t.line, t.val)
		}
		return &jsNode{kind: 'n', num: f}, nil
	case t.kind == tkPunct && (t.val == "-" || t.val == "+"):
		n := p.peekAt(1)
		if n.kind == tkNumber {
			p.pos += 2
			f, err := strconv.ParseFloat(t.val+n.val, 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: bad number %q", t.line, t.val+n.val)
			}
			return &jsNode{kind: 'n', num: f}, nil
		}
		return nil, fmt.Errorf("line %d: unexpected %q", t.line, t.val)
	case t.kind == tkIdent:
		p.pos++
		switch t.val {
		case "true":
			return &jsNode{kind: 'b', str: "true"}, nil
		case "false":
			return &jsNode{kind: 'b', str: "false"}, nil
		case "null", "undefined":
			return &jsNode{kind: 'x'}, nil
		}
		return &jsNode{kind: 'i', str: t.val}, nil
	}
	return nil, fmt.Errorf("line %d: unexpected token %q", t.line, t.val)
}

func (p *jsParser) parseObject() (*jsNode, error) {
	p.pos++ // {
	n := &jsNode{kind: 'o'}
	for {
		t := p.peek()
		if t.kind == tkPunct && t.val == "}" {
			p.pos++
			return n, nil
		}
		if t.kind == tkEOF {
			return nil, fmt.Errorf("unexpected EOF in object literal")
		}
		var key string
		switch {
		case t.kind == tkIdent:
			key = t.val
			p.pos++
		case t.kind == tkString:
			key = t.val
			p.pos++
		default:
			return nil, fmt.Errorf("line %d: bad object key %q", t.line, t.val)
		}
		if c := p.peek(); c.kind == tkPunct && c.val == ":" {
			p.pos++
			v, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			n.keys = append(n.keys, key)
			n.vals = append(n.vals, v)
		} else {
			// shorthand property
			n.keys = append(n.keys, key)
			n.vals = append(n.vals, &jsNode{kind: 'i', str: key})
		}
		t = p.peek()
		if t.kind == tkPunct && t.val == "," {
			p.pos++
			continue
		}
		if t.kind == tkPunct && t.val == "}" {
			p.pos++
			return n, nil
		}
		return nil, fmt.Errorf("line %d: expected , or } in object literal", t.line)
	}
}

func (p *jsParser) parseArray() (*jsNode, error) {
	p.pos++ // [
	n := &jsNode{kind: 'a'}
	for {
		t := p.peek()
		if t.kind == tkPunct && t.val == "]" {
			p.pos++
			return n, nil
		}
		if t.kind == tkEOF {
			return nil, fmt.Errorf("unexpected EOF in array literal")
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		n.vals = append(n.vals, v)
		t = p.peek()
		if t.kind == tkPunct && t.val == "," {
			p.pos++
			continue
		}
		if t.kind == tkPunct && t.val == "]" {
			p.pos++
			return n, nil
		}
		return nil, fmt.Errorf("line %d: expected , or ] in array literal", t.line)
	}
}

// skipParenArgs consumes a parenthesized argument list without interpreting it.
func (p *jsParser) skipParenArgs(line int) error {
	t := p.peek()
	if !(t.kind == tkPunct && t.val == "(") {
		return nil
	}
	depth := 0
	for {
		t := p.next()
		if t.kind == tkEOF {
			return fmt.Errorf("line %d: unbalanced parentheses", line)
		}
		if t.kind == tkPunct && t.val == "(" {
			depth++
		}
		if t.kind == tkPunct && t.val == ")" {
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}
