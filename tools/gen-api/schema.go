package main

import (
	"errors"
	"fmt"
	"strings"
)

// Schema is a decoded zod schema expression from a generated descriptor file.
type Schema struct {
	Kind string // string number boolean any nulllit litstr litbool litnum enum
	//           union intersection array object record lazy unsupported
	Str         string  // litstr value / lazy target constant name
	Bool        bool    // litbool value
	Num         float64 // litnum value
	Members     []*Schema
	Elem        *Schema // array element / record value
	Key         *Schema // record key
	Fields      []SchemaField
	Optional    bool
	Nullable    bool
	Readonly    bool
	Unsupported bool
}

type SchemaField struct {
	Name   string
	Schema *Schema
}

// parsedFile is one parsed typert.remote-client.js.
type parsedFile struct {
	file    *typertFile
	schemas map[string]*Schema // const name -> schema
}

type typertFile struct {
	Package     string
	Descriptors []*endpointDesc
}

type endpointDesc struct {
	ID               string
	Service          string
	Namespace        string
	Method           string
	FullName         string // "<namespace>/<method>"
	Stream           bool
	Kind             string // "direct" | "context"
	Context          string
	ContextWire      string
	ScopeContext     string
	ScopeWire        string
	Cancellation     string
	HasUplink        bool
	UplinkTypeSymbol string
	UplinkSchemaRef  string
	Params           []paramDesc
	ResultMode       string
	ResultTypeSymbol string
	ResultSchemaRef  string
}

type paramDesc struct {
	Name             string
	Wire             string
	Source           string // "json" | "lookup"
	Lookup           string
	AcceptsUndefined bool
	TypeSymbol       string
	SchemaRef        string
}

// parseTypertFile parses a generated typert.remote-client.js source.
func parseTypertFile(src string) (*parsedFile, error) {
	toks, err := tokenize(src)
	if err != nil {
		return nil, err
	}
	p := &jsParser{toks: toks}
	pf := &parsedFile{schemas: map[string]*Schema{}}
	var root *jsNode
	for p.peek().kind != tkEOF {
		t := p.peek()
		if t.kind == tkIdent && t.val == "export" {
			p.pos++
			continue
		}
		if t.kind == tkIdent && t.val == "const" &&
			p.peekAt(1).kind == tkIdent &&
			p.peekAt(2).kind == tkPunct && p.peekAt(2).val == "=" {
			name := p.peekAt(1).val
			p.pos += 3
			if s, ok, err := p.tryParseSchemaConst(); ok {
				if err != nil {
					return nil, fmt.Errorf("const %s: %w", name, err)
				}
				pf.schemas[name] = s
			} else if err := p.skipOrParseConstValue(name, &root); err != nil {
				return nil, fmt.Errorf("const %s: %w", name, err)
			}
			continue
		}
		p.pos++
	}
	if root == nil || root.kind != 'o' {
		return nil, errors.New("TYPERT_REMOTE object literal not found")
	}
	pf.file = &typertFile{Package: nodeString(root, "package")}
	if methods := root.get("methods"); methods != nil && methods.kind == 'o' {
		// createTypertRemoteInfo derives these descriptor fields from the
		// method-map key. Read the generated facts without executing JavaScript.
		for i, name := range methods.keys {
			d := methods.vals[i]
			if d == nil || d.kind != 'o' {
				return nil, fmt.Errorf("methods[%q]: not an object", name)
			}
			ns, method, ok := strings.Cut(name, "/")
			if !ok || ns == "" || method == "" || strings.Contains(method, "/") {
				return nil, fmt.Errorf("methods[%q]: expected namespace/method", name)
			}
			normalized := &jsNode{
				kind: 'o',
				keys: []string{"id", "namespace", "method"},
				vals: []*jsNode{
					{kind: 's', str: pf.file.Package + "#" + name},
					{kind: 's', str: ns},
					{kind: 's', str: method},
				},
			}
			normalized.keys = append(normalized.keys, d.keys...)
			normalized.vals = append(normalized.vals, d.vals...)
			e, err := walkEndpoint(normalized)
			if err != nil {
				return nil, fmt.Errorf("methods[%q]: %w", name, err)
			}
			pf.file.Descriptors = append(pf.file.Descriptors, e)
		}
		return pf, nil
	}
	descs := root.get("descriptors")
	if descs == nil || descs.kind != 'a' {
		return nil, errors.New("TYPERT_REMOTE.descriptors not found")
	}
	for i, d := range descs.vals {
		if d == nil || d.kind != 'o' {
			return nil, fmt.Errorf("descriptors[%d]: not an object", i)
		}
		e, err := walkEndpoint(d)
		if err != nil {
			return nil, fmt.Errorf("descriptors[%d]: %w", i, err)
		}
		pf.file.Descriptors = append(pf.file.Descriptors, e)
	}
	return pf, nil
}

func schemaRefOf(codec *jsNode) string {
	// Older typert builds used codec.schema; current builds use codec.create
	// (a zero-arg factory that returns the zod schema).
	for _, key := range []string{"schema", "create"} {
		v := codec.get(key)
		if v != nil && v.kind == 'i' {
			return v.str
		}
	}
	return ""
}

// skipOrParseConstValue handles non-schema const initializers: TYPERT_REMOTE
// object literals or lazy Remote metadata, and arrow-function helpers (binary
// result codecs) emitted alongside the schema factories.
func (p *jsParser) skipOrParseConstValue(name string, root **jsNode) error {
	if name == "TYPERT_REMOTE" && p.peek().kind == tkIdent && p.peek().val == "createTypertRemoteInfo" {
		p.pos++
		if t := p.next(); t.kind != tkPunct || t.val != "(" {
			return fmt.Errorf("line %d: expected ( after createTypertRemoteInfo", t.line)
		}
		v, err := p.parseValue()
		if err != nil {
			return err
		}
		if t := p.next(); t.kind != tkPunct || t.val != "," {
			return fmt.Errorf("line %d: expected , before Remote Zod argument", t.line)
		}
		if t := p.next(); t.kind != tkIdent || t.val != "z" {
			return fmt.Errorf("line %d: expected Remote Zod argument z", t.line)
		}
		if t := p.next(); t.kind != tkPunct || t.val != ")" {
			return fmt.Errorf("line %d: expected ) after Remote Zod argument", t.line)
		}
		*root = v
		return nil
	}
	// Arrow helpers: (args...) => { ... }
	if p.peek().kind == tkPunct && p.peek().val == "(" {
		line := p.peek().line
		if err := p.skipParenArgs(line); err != nil {
			return err
		}
		if !(p.peek().kind == tkPunct && p.peek().val == "=>") {
			return fmt.Errorf("line %d: expected => after helper params", p.peek().line)
		}
		p.pos++
		if !(p.peek().kind == tkPunct && p.peek().val == "{") {
			return fmt.Errorf("line %d: expected { after => in helper", p.peek().line)
		}
		return p.skipBalancedBrace()
	}
	v, err := p.parseValue()
	if err != nil {
		return err
	}
	if name == "TYPERT_REMOTE" {
		*root = v
	}
	return nil
}

// skipBalancedBrace consumes a `{...}` token span, nesting-aware.
func (p *jsParser) skipBalancedBrace() error {
	t := p.next()
	if !(t.kind == tkPunct && t.val == "{") {
		return fmt.Errorf("line %d: expected {", t.line)
	}
	depth := 1
	for depth > 0 {
		t := p.next()
		if t.kind == tkEOF {
			return fmt.Errorf("unbalanced {")
		}
		if t.kind == tkPunct && t.val == "{" {
			depth++
		} else if t.kind == tkPunct && t.val == "}" {
			depth--
		}
	}
	return nil
}

// tryParseSchemaConst recognizes the two const-initializer shapes used by
// typert.remote-client.js:
//
//	const Foo$schema = z....
//	const Foo$schema = () => (Foo$schema$value ??= z....)
//
// ok is false when the initializer is neither form (e.g. TYPERT_REMOTE).
func (p *jsParser) tryParseSchemaConst() (s *Schema, ok bool, err error) {
	// Direct: z.<ctor>(...)
	if pk := p.peek(); pk.kind == tkIdent && pk.val == "z" &&
		p.peekAt(1).kind == tkPunct && p.peekAt(1).val == "." {
		s, err = p.parseSchemaExpr()
		return s, true, err
	}
	// Lazy factory: () => (cacheIdent ??= z.<ctor>(...))
	if !(p.peek().kind == tkPunct && p.peek().val == "(" &&
		p.peekAt(1).kind == tkPunct && p.peekAt(1).val == ")" &&
		p.peekAt(2).kind == tkPunct && p.peekAt(2).val == "=>") {
		return nil, false, nil
	}
	p.pos += 3 // () =>
	if !(p.peek().kind == tkPunct && p.peek().val == "(") {
		return nil, true, fmt.Errorf("line %d: expected ( after => in lazy schema", p.peek().line)
	}
	p.pos++ // (
	if p.peek().kind != tkIdent {
		return nil, true, fmt.Errorf("line %d: expected cache ident in lazy schema", p.peek().line)
	}
	p.pos++ // cache ident
	if !(p.peek().kind == tkPunct && p.peek().val == "?" &&
		p.peekAt(1).kind == tkPunct && p.peekAt(1).val == "?" &&
		p.peekAt(2).kind == tkPunct && p.peekAt(2).val == "=") {
		return nil, true, fmt.Errorf("line %d: expected ??= in lazy schema", p.peek().line)
	}
	p.pos += 3 // ??=
	s, err = p.parseSchemaExpr()
	if err != nil {
		return nil, true, err
	}
	if !(p.peek().kind == tkPunct && p.peek().val == ")") {
		return nil, true, fmt.Errorf("line %d: expected ) to close lazy schema", p.peek().line)
	}
	p.pos++ // )
	return s, true, nil
}

func walkEndpoint(n *jsNode) (*endpointDesc, error) {
	e := &endpointDesc{
		ID:      nodeString(n, "id"),
		Service: nodeString(n, "service"),
		Method:  nodeString(n, "method"),
	}
	if e.Method == "" {
		return nil, errors.New("endpoint without method name")
	}
	e.Namespace = nodeString(n, "namespace")
	if e.Namespace == "" {
		e.Namespace = e.Service
	}
	e.FullName = e.Namespace + "/" + e.Method
	e.Stream = nodeString(n, "mode") == "stream"
	if inv := n.get("invocation"); inv != nil && inv.kind == 'o' {
		e.Kind = nodeString(inv, "kind")
		e.Context = nodeString(inv, "context")
		e.ContextWire = nodeString(inv, "wire")
	}
	if e.Kind == "" {
		e.Kind = "direct"
	}
	if scope := n.get("scope"); scope != nil && scope.kind == 'o' {
		e.ScopeContext = nodeString(scope, "context")
		e.ScopeWire = nodeString(scope, "wire")
	}
	if canc := n.get("cancellation"); canc != nil && canc.kind == 'o' {
		e.Cancellation = nodeString(canc, "parameter")
	}
	if upl := n.get("uplink"); upl != nil && upl.kind == 'o' {
		e.HasUplink = true
		codec := upl.get("codec")
		if codec != nil && codec.kind == 'o' {
			e.UplinkTypeSymbol = nodeString(codec, "typeSymbol")
			e.UplinkSchemaRef = schemaRefOf(codec)
		}
	}
	if params := n.get("parameters"); params != nil && params.kind == 'a' {
		for i, pn := range params.vals {
			if pn == nil || pn.kind != 'o' {
				return nil, fmt.Errorf("parameter %d: not an object", i)
			}
			prm := paramDesc{
				Name:             nodeString(pn, "name"),
				Wire:             nodeString(pn, "wire"),
				Source:           nodeString(pn, "source"),
				Lookup:           nodeString(pn, "lookup"),
				AcceptsUndefined: nodeBool(pn, "acceptsUndefined"),
			}
			if prm.Wire == "" {
				prm.Wire = prm.Name
			}
			if prm.Source == "" {
				prm.Source = "json"
			}
			codec := pn.get("codec")
			if codec != nil && codec.kind == 'o' {
				prm.TypeSymbol = nodeString(codec, "typeSymbol")
				prm.SchemaRef = schemaRefOf(codec)
			}
			e.Params = append(e.Params, prm)
		}
	}
	if res := n.get("result"); res != nil && res.kind == 'o' {
		e.ResultMode = nodeString(res, "mode")
		e.ResultTypeSymbol = nodeString(res, "typeSymbol")
		e.ResultSchemaRef = schemaRefOf(res)
	}
	return e, nil
}

// parseSchemaExpr parses `z.<method>(...)` with a postfix chain.
func (p *jsParser) parseSchemaExpr() (*Schema, error) {
	t := p.peek()
	if t.kind != tkIdent || t.val != "z" {
		return nil, fmt.Errorf("line %d: expected zod expression", t.line)
	}
	p.pos++
	s, err := p.parseZodCtor()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tkPunct && p.peek().val == "." {
		p.pos++
		m := p.next()
		if m.kind != tkIdent {
			return nil, fmt.Errorf("line %d: expected method name after '.'", m.line)
		}
		switch m.val {
		case "optional":
			if err := p.skipParenArgs(m.line); err != nil {
				return nil, err
			}
			s.Optional = true
		case "readonly":
			if err := p.skipParenArgs(m.line); err != nil {
				return nil, err
			}
			s.Readonly = true
		case "nullable":
			if err := p.skipParenArgs(m.line); err != nil {
				return nil, err
			}
			s.Nullable = true
		case "describe", "default", "catch", "brand":
			// metadata or type-preserving decorators
			if err := p.skipParenArgs(m.line); err != nil {
				return nil, err
			}
		default:
			// unknown or type-changing decorator: keep the value as raw JSON
			if err := p.skipParenArgs(m.line); err != nil {
				return nil, err
			}
			s.Unsupported = true
		}
	}
	return s, nil
}

func (p *jsParser) parseZodCtor() (*Schema, error) {
	d := p.next() // "."
	if d.kind != tkPunct || d.val != "." {
		return nil, fmt.Errorf("line %d: expected '.' after z", d.line)
	}
	m := p.next()
	if m.kind != tkIdent {
		return nil, fmt.Errorf("line %d: expected zod method name", m.line)
	}
	op := p.peek()
	if !(op.kind == tkPunct && op.val == "(") {
		return nil, fmt.Errorf("line %d: expected ( after z.%s", m.line, m.val)
	}
	switch m.val {
	case "string":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "string"})
	case "number":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "number"})
	case "boolean":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "boolean"})
	case "unknown", "any":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "any"})
	case "null", "undefined":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "nulllit"})
	case "nan", "bigint", "date", "symbol", "void":
		p.pos++
		return p.closeCall(m.line, &Schema{Kind: "unsupported"})
	case "literal":
		p.pos++
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		var s *Schema
		switch v.kind {
		case 's':
			s = &Schema{Kind: "litstr", Str: v.str}
		case 'b':
			s = &Schema{Kind: "litbool", Bool: v.str == "true"}
		case 'n':
			s = &Schema{Kind: "litnum", Num: v.num}
		case 'x':
			s = &Schema{Kind: "nulllit"}
		default:
			s = &Schema{Kind: "unsupported"}
		}
		return p.closeCall(m.line, s)
	case "enum":
		p.pos++
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		s := &Schema{Kind: "enum"}
		if v.kind == 'a' {
			for _, mv := range v.vals {
				if mv.kind == 's' {
					s.Members = append(s.Members, &Schema{Kind: "litstr", Str: mv.str})
				}
			}
		}
		return p.closeCall(m.line, s)
	case "union":
		return p.parseZodMemberList(m.val, m.line)
	case "intersection":
		args, err := p.parseSchemaArgs()
		if err != nil {
			return nil, err
		}
		return &Schema{Kind: "intersection", Members: args}, nil
	case "array":
		args, err := p.parseSchemaArgs()
		if err != nil {
			return nil, err
		}
		s := &Schema{Kind: "array"}
		if len(args) > 0 {
			s.Elem = args[0]
		}
		return s, nil
	case "record":
		args, err := p.parseSchemaArgs()
		if err != nil {
			return nil, err
		}
		s := &Schema{Kind: "record"}
		if len(args) > 0 {
			s.Key = args[0]
		}
		if len(args) > 1 {
			s.Elem = args[1]
		}
		return s, nil
	case "object":
		return p.parseZodObject()
	case "lazy":
		p.pos++
		t1 := p.next() // (
		t2 := p.next() // )
		t3 := p.next() // =>
		id := p.next() // referenced constant
		if t1.val != "(" || t2.val != ")" || t3.val != "=>" || id.kind != tkIdent {
			return nil, fmt.Errorf("line %d: malformed z.lazy expression", t1.line)
		}
		// Current builds invoke schema factories: z.lazy(() => Foo$schema())
		// Older builds reference the schema const directly: z.lazy(() => Foo$schema)
		if p.peek().kind == tkPunct && p.peek().val == "(" &&
			p.peekAt(1).kind == tkPunct && p.peekAt(1).val == ")" {
			p.pos += 2
		}
		return p.closeCall(m.line, &Schema{Kind: "lazy", Str: id.val})
	default:
		// unsupported constructor (z.tuple, z.map, ...)
		if err := p.skipParenArgs(m.line); err != nil {
			return nil, err
		}
		return &Schema{Kind: "unsupported"}, nil
	}
}

// parseZodMemberList parses z.union([...]) / z.intersection([...]) whose
// single argument is an array literal of schemas.
func (p *jsParser) parseZodMemberList(kind string, line int) (*Schema, error) {
	op := p.next() // (
	if !(op.kind == tkPunct && op.val == "(") {
		return nil, fmt.Errorf("line %d: expected ( after z.%s", op.line, kind)
	}
	ob := p.next() // [
	if !(ob.kind == tkPunct && ob.val == "[") {
		return nil, fmt.Errorf("line %d: expected [ in z.%s", ob.line, kind)
	}
	s := &Schema{Kind: kind}
	for {
		t := p.peek()
		if t.kind == tkPunct && t.val == "]" {
			p.pos++
			break
		}
		if t.kind == tkEOF {
			return nil, fmt.Errorf("unexpected EOF in z.%s", kind)
		}
		member, err := p.parseSchemaArg()
		if err != nil {
			return nil, err
		}
		s.Members = append(s.Members, member)
		t = p.peek()
		if t.kind == tkPunct && t.val == "," {
			p.pos++
			continue
		}
		if t.kind == tkPunct && t.val == "]" {
			p.pos++
			break
		}
		return nil, fmt.Errorf("line %d: expected , or ] in z.%s", t.line, kind)
	}
	return p.closeCall(line, s)
}

// closeCall consumes the closing paren of a zero-argument zod call.
func (p *jsParser) closeCall(line int, s *Schema) (*Schema, error) {
	t := p.next()
	if !(t.kind == tkPunct && t.val == ")") {
		return nil, fmt.Errorf("line %d: expected ) to close zod call", t.line)
	}
	return s, nil
}

// parseSchemaArgs parses a comma-separated schema argument list including
// the enclosing parentheses.
func (p *jsParser) parseSchemaArgs() ([]*Schema, error) {
	op := p.next() // (
	if !(op.kind == tkPunct && op.val == "(") {
		return nil, fmt.Errorf("line %d: expected ( in argument list", op.line)
	}
	var out []*Schema
	for {
		t := p.peek()
		if t.kind == tkPunct && t.val == ")" {
			p.pos++
			return out, nil
		}
		if t.kind == tkEOF {
			return nil, fmt.Errorf("unexpected EOF in zod argument list")
		}
		s, err := p.parseSchemaArg()
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		t = p.peek()
		if t.kind == tkPunct && t.val == "," {
			p.pos++
			continue
		}
		if t.kind == tkPunct && t.val == ")" {
			p.pos++
			return out, nil
		}
		return nil, fmt.Errorf("line %d: expected , or ) in zod argument list", t.line)
	}
}

func (p *jsParser) parseSchemaArg() (*Schema, error) {
	t := p.peek()
	if t.kind == tkIdent && t.val == "z" &&
		p.peekAt(1).kind == tkPunct && p.peekAt(1).val == "." {
		return p.parseSchemaExpr()
	}
	if t.kind == tkIdent && t.val != "true" && t.val != "false" && t.val != "null" && t.val != "undefined" {
		// bare reference to another schema constant
		p.pos++
		return &Schema{Kind: "lazy", Str: t.val}, nil
	}
	if err := p.skipParenArgsValue(); err != nil {
		return nil, err
	}
	return &Schema{Kind: "unsupported"}, nil
}

// skipParenArgsValue consumes any value expression (best effort) for
// arguments the generator does not model.
func (p *jsParser) skipParenArgsValue() error {
	if _, err := p.parseValue(); err != nil {
		return err
	}
	return nil
}

func (p *jsParser) parseZodObject() (*Schema, error) {
	op := p.next() // (
	if !(op.kind == tkPunct && op.val == "(") {
		return nil, fmt.Errorf("line %d: expected ( after z.object", op.line)
	}
	ob := p.next() // {
	if !(ob.kind == tkPunct && ob.val == "{") {
		return nil, fmt.Errorf("line %d: expected { in z.object", ob.line)
	}
	s := &Schema{Kind: "object"}
	for {
		t := p.peek()
		if t.kind == tkPunct && t.val == "}" {
			p.pos++
			break
		}
		if t.kind == tkEOF {
			return nil, fmt.Errorf("unexpected EOF in z.object")
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
			return nil, fmt.Errorf("line %d: bad key in z.object", t.line)
		}
		c := p.next()
		if !(c.kind == tkPunct && c.val == ":") {
			return nil, fmt.Errorf("line %d: expected : in z.object", c.line)
		}
		fv, err := p.parseSchemaArg()
		if err != nil {
			return nil, err
		}
		s.Fields = append(s.Fields, SchemaField{Name: key, Schema: fv})
		t = p.peek()
		if t.kind == tkPunct && t.val == "," {
			p.pos++
			continue
		}
		if t.kind == tkPunct && t.val == "}" {
			p.pos++
			break
		}
		return nil, fmt.Errorf("line %d: expected , or } in z.object", t.line)
	}
	cl := p.next() // )
	if !(cl.kind == tkPunct && cl.val == ")") {
		return nil, fmt.Errorf("line %d: expected ) after z.object", cl.line)
	}
	return s, nil
}
