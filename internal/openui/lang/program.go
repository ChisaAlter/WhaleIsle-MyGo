package lang

import (
	"fmt"
	"regexp"
	"strings"
)

// StatementKind classifies a statement (spec section 2.1).
type StatementKind uint8

// Statement kinds.
const (
	ValueStatement StatementKind = iota
	StateStatement
	QueryStatementKind
	MutationStatementKind
)

// Statement is one parsed `name = expression` statement.
type Statement struct {
	Kind StatementKind
	ID   string
	Expr Node
	// Deps lists the $variables written literally in a query's args.
	Deps []string
}

// QueryStatement is `name = Query(tool, args, defaults, refreshSeconds?)`.
type QueryStatement struct {
	ID                            string
	Tool, Args, Defaults, Refresh Node
	Deps                          []string
}

// MutationStatement is `name = Mutation(tool, args)`.
type MutationStatement struct {
	ID         string
	Tool, Args Node
}

// ParseResult is the program tree and parse metadata (spec section 9.1).
type ParseResult struct {
	// Root is the materialized entry element; its props may hold Nodes.
	Root *Element
	// RootExpr is a runtime entry expression such as `$t ? A : B`.
	RootExpr Node
	// EntryID is the entry statement name.
	EntryID string
	// Incomplete is true when the input ended inside a string or bracket.
	Incomplete     bool
	Unresolved     []string
	Orphaned       []string
	StatementCount int
	Errors         []Diagnostic
	// StateDefaults holds the declared $variable defaults in declaration order.
	StateDefaults *Object
	Queries       []QueryStatement
	Mutations     []MutationStatement
	// Statements are the effective statements (later duplicates win).
	Statements []*Statement
	// Fatal is true when nothing renders.
	Fatal bool
}

func emptyResult(incomplete bool) *ParseResult {
	return &ParseResult{Incomplete: incomplete, StateDefaults: NewObject(0)}
}

func collectStateRefs(n Node) []string {
	if n == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	walkAST(n, func(x Node) {
		if s, ok := x.(*StateRefNode); ok && !seen[s.N] {
			seen[s.N] = true
			out = append(out, s.N)
		}
	})
	return out
}

func classify(raw rawStmt, expr Node) *Statement {
	if call, ok := expr.(*CallNode); ok {
		switch call.Name {
		case "Query":
			var deps []string
			if len(call.Args) > 1 {
				deps = collectStateRefs(call.Args[1])
			}
			return &Statement{Kind: QueryStatementKind, ID: raw.id, Expr: expr, Deps: deps}
		case "Mutation":
			return &Statement{Kind: MutationStatementKind, ID: raw.id, Expr: expr}
		}
	}
	if raw.state {
		return &Statement{Kind: StateStatement, ID: raw.id, Expr: expr}
	}
	return &Statement{Kind: ValueStatement, ID: raw.id, Expr: expr}
}

// stmtMap is an insertion-ordered statement table (JS Map semantics: a
// redefinition keeps the first position and takes the new value).
type stmtMap struct {
	order []string
	m     map[string]*Statement
}

func newStmtMap() *stmtMap { return &stmtMap{m: map[string]*Statement{}} }

func (s *stmtMap) set(st *Statement) {
	if _, ok := s.m[st.ID]; !ok {
		s.order = append(s.order, st.ID)
	}
	s.m[st.ID] = st
}

func (s *stmtMap) clone() *stmtMap {
	c := &stmtMap{order: append([]string(nil), s.order...), m: make(map[string]*Statement, len(s.m))}
	for k, v := range s.m {
		c.m[k] = v
	}
	return c
}

func (s *stmtMap) list() []*Statement {
	out := make([]*Statement, len(s.order))
	for i, id := range s.order {
		out[i] = s.m[id]
	}
	return out
}

// parser holds the per-library parse configuration.
type parser struct {
	lib    *Library
	limits Limits
}

func (p *parser) unicode() bool { return p.lib != nil && p.lib.Profile.UnicodeIdentifiers }

// parseStatements lexes, splits and parses statement text into the table.
// It returns the number of statements and false on a budget overrun.
func (p *parser) parseInto(text string, into *stmtMap, calls *int, nodes *int) (int, error) {
	toks, ok := tokenize(text, p.unicode(), p.limits.MaxTokens)
	if !ok {
		return 0, &budgetError{phase: "tokens", limit: p.limits.MaxTokens}
	}
	raws := splitStatements(toks)
	count := 0
	for _, r := range raws {
		expr, ok := parseExpression(r.toks, &p.limits, calls, nodes)
		if !ok {
			return count, &budgetError{phase: "AST", limit: p.limits.MaxASTNodes, path: r.id}
		}
		into.set(classify(r, expr))
		count++
		if len(into.order) > p.limits.MaxStatements {
			return count, &budgetError{phase: "statements", limit: p.limits.MaxStatements}
		}
	}
	return count, nil
}

// Parse parses a complete program (batch parse).
func Parse(input string, lib *Library, limits Limits) *ParseResult {
	p := &parser{lib: lib, limits: limits.withDefaults()}
	return p.parse(input, true)
}

func (p *parser) parse(input string, complete bool) (res *ParseResult) {
	defer func() {
		if r := recover(); r != nil {
			res = failureResult(r)
		}
	}()
	if len(input) > p.limits.MaxProgramBytes {
		return budgetResult(&budgetError{phase: "source", limit: p.limits.MaxProgramBytes})
	}
	trimmed := Preprocess(input)
	if trimmed == "" {
		return p.finish(emptyResult(true), complete)
	}
	text, incomplete := autoClose(trimmed)
	m := newStmtMap()
	calls, nodes := 0, 0
	n, err := p.parseInto(text, m, &calls, &nodes)
	if err != nil {
		return budgetResult(err)
	}
	if n == 0 {
		return p.finish(emptyResult(incomplete), complete)
	}
	return p.finish(p.build(m, incomplete, len(m.order), nil), complete)
}

func budgetResult(err error) *ParseResult {
	r := emptyResult(false)
	d := Diagnostic{Source: SourceParser, Code: CodeBudgetExceeded, Message: err.Error()}
	if be, ok := err.(*budgetError); ok {
		d.Path = be.path
	}
	r.Errors = append(r.Errors, d)
	r.Fatal = true
	return r
}

// failureResult converts a recovered panic into a fatal result.
func failureResult(r any) *ParseResult {
	if be, ok := r.(*budgetError); ok {
		return budgetResult(be)
	}
	res := emptyResult(false)
	res.Fatal = true
	res.Errors = append(res.Errors, Diagnostic{Source: SourceParser, Code: CodeParseException, Message: fmt.Sprintf("Parser crashed: %v", r), Hint: "The response may contain syntax the parser cannot handle"})
	return res
}

// finish adds the completion diagnostics of the Whale profile.
func (p *parser) finish(r *ParseResult, complete bool) *ParseResult {
	if !complete || p.lib == nil || !p.lib.Profile.CompletionDiagnostics {
		return r
	}
	if r.StatementCount == 0 && !hasCode(r.Errors, "no-root") {
		r.Errors = append(r.Errors, Diagnostic{Source: SourceParser, Code: "no-root", StatementID: "root", Message: "Program has no root component statement"})
		r.Fatal = true
	}
	if r.Incomplete {
		r.Errors = append(r.Errors, Diagnostic{Source: SourceParser, Code: CodeIncompleteInput, StatementID: "root", Message: "Program ended with an incomplete expression or string"})
	}
	seen := map[string]bool{}
	for _, u := range r.Unresolved {
		if seen[u] {
			continue
		}
		seen[u] = true
		r.Errors = append(r.Errors, Diagnostic{Source: SourceParser, Code: CodeUnresolvedRef, StatementID: u, Message: `Reference "` + u + `" is unresolved or circular`})
	}
	return r
}

func hasCode(ds []Diagnostic, code string) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}

func isComponentCall(lib *Library, n Node) (*CallNode, bool) {
	c, ok := n.(*CallNode)
	if !ok || c.Fn || isBuiltinName(c.Name) || isReservedCall(c.Name) {
		return nil, false
	}
	return c, true
}

func pickEntry(m *stmtMap, root string) (string, bool, bool) {
	if _, ok := m.m["root"]; ok {
		return "root", false, true
	}
	for _, id := range m.order {
		st := m.m[id]
		if st.Kind != ValueStatement {
			continue
		}
		if c, ok := isComponentCall(nil, st.Expr); ok && c.Name == root && root != "" {
			return id, true, true
		}
	}
	return "", false, false
}

func resolvesToComponentCall(n Node, syms map[string]Node) bool {
	seen := map[string]bool{}
	for {
		r, ok := n.(*RefNode)
		if !ok || seen[r.N] {
			break
		}
		seen[r.N] = true
		n = syms[r.N]
	}
	_, ok := isComponentCall(nil, n)
	return ok
}

func containsComponent(n Node) bool {
	found := false
	walkAST(n, func(x Node) {
		if _, ok := isComponentCall(nil, x); ok {
			found = true
		}
	})
	return found
}

// partialRef is a $name at the very end of streamed text that may still grow.
type partialRef struct {
	name  string
	stmts map[*Statement]bool
}

func (p *parser) build(m *stmtMap, incomplete bool, count int, pref *partialRef) *ParseResult {
	lib := p.lib
	rootName := ""
	if lib != nil {
		rootName = lib.Root
	}
	entry, fallback, hasEntry := pickEntry(m, rootName)
	syms := make(map[string]Node, len(m.order))
	for _, id := range m.order {
		syms[id] = m.m[id].Expr
	}
	unreached := map[string]bool{}
	for _, id := range m.order {
		st := m.m[id]
		if hasEntry && id == entry || st.Kind != ValueStatement {
			continue
		}
		unreached[id] = true
	}
	c := &mctx{
		vctx:      vctx{lib: lib, partial: incomplete, statement: entry},
		syms:      syms,
		visited:   map[string]bool{},
		unreached: unreached,
		cacheExpr: map[string]Node{},
		cacheVal:  map[string]Node{},
		budget:    &budget{phase: "materialization", maxCount: p.limits.MaxMaterializedNodes, maxDepth: p.limits.MaxDepth},
	}
	res := &ParseResult{Incomplete: incomplete, StatementCount: count, StateDefaults: NewObject(4), EntryID: entry}
	if hasEntry {
		v := c.materializeValue(syms[entry])
		if el, ok := v.(*Element); ok {
			res.Root = el
			el.StatementID = entry
		} else if n, ok := v.(Node); ok && containsComponent(n) {
			res.RootExpr = n
		}
	}
	p.extract(m, c, res, pref)
	c.statement = entry
	noRoot := func(reason string) {
		msg := "No root statement"
		switch reason {
		case "not-component":
			msg = "root does not evaluate to a component"
		case "fallback":
			msg = "No root statement. Rendering the first statement that calls the root component " + rootName
		default:
			if rootName != "" {
				msg = "No root statement, and no statement calls the root component " + rootName
			}
		}
		d := Diagnostic{Source: SourceParser, Code: "no-root", Component: rootName, Message: msg, StatementID: c.statement}
		if reason == "fallback" {
			d.Severity = "warning"
		}
		c.errors = append(c.errors, d)
	}
	switch {
	case !hasEntry:
		noRoot("missing")
	case res.Root == nil && res.RootExpr == nil && !resolvesToComponentCall(syms[entry], syms):
		noRoot("not-component")
	case fallback:
		if res.Root != nil {
			noRoot("fallback")
		} else {
			noRoot("missing")
		}
	}
	if hasEntry {
		for _, id := range m.order {
			if unreached[id] {
				res.Orphaned = append(res.Orphaned, id)
			}
		}
	}
	res.Unresolved = c.unres
	res.Errors = c.errors
	res.Statements = m.list()
	res.Fatal = res.Root == nil && res.RootExpr == nil
	return res
}

func (p *parser) extract(m *stmtMap, c *mctx, res *ParseResult, pref *partialRef) {
	literal := p.lib != nil && p.lib.Profile.LiteralStateDefaults
	allowData := p.lib == nil || p.lib.Profile.AllowData
	for _, id := range m.order {
		st := m.m[id]
		switch st.Kind {
		case StateStatement:
			if literal {
				if v, ok := literalValue(st.Expr); ok {
					res.StateDefaults.Set(id, v)
				} else {
					c.errors = append(c.errors, Diagnostic{Source: SourceParser, Code: CodeInvalidStateDflt, StatementID: id, Message: "State default " + id + " must be a literal (string, number, boolean, null, array, or object)"})
					res.StateDefaults.Set(id, nil)
				}
			} else {
				prev := c.statement
				c.statement = id
				v := c.materializeValue(st.Expr)
				c.statement = prev
				if containsDynamic(v) {
					v = evalStandalone(v, res.StateDefaults, p.lib, p.limits)
				}
				res.StateDefaults.Set(id, v)
			}
		case QueryStatementKind, MutationStatementKind:
			call := st.Expr.(*CallNode)
			c.nameRawCallArgs(call.Args, id)
			arg := func(i int) Node {
				if i < len(call.Args) {
					return call.Args[i]
				}
				return nil
			}
			if !allowData {
				kind := "Query"
				if st.Kind == MutationStatementKind {
					kind = "Mutation"
				}
				c.errors = append(c.errors, Diagnostic{Source: SourceParser, Code: CodeDataUnavailable, StatementID: id, Component: kind, Message: "This interface has no business-tool provider. Obtain data using the conversation tools first."})
			}
			if st.Kind == QueryStatementKind {
				res.Queries = append(res.Queries, QueryStatement{ID: id, Tool: arg(0), Args: arg(1), Defaults: arg(2), Refresh: arg(3), Deps: st.Deps})
			} else {
				res.Mutations = append(res.Mutations, MutationStatement{ID: id, Tool: arg(0), Args: arg(1)})
			}
		}
	}
	// Auto-declare every referenced $variable with null.
	for _, id := range m.order {
		st := m.m[id]
		var nodes []Node
		switch st.Kind {
		case StateStatement, ValueStatement:
			nodes = []Node{st.Expr}
		default:
			nodes = st.Expr.(*CallNode).Args
		}
		for _, n := range nodes {
			for _, dep := range collectStateRefs(n) {
				if pref != nil && dep == pref.name && pref.stmts[st] {
					continue
				}
				if !res.StateDefaults.Has(dep) {
					res.StateDefaults.Set(dep, nil)
				}
			}
		}
	}
}

var tailStateRef = regexp.MustCompile(`\$\w+$`)

// StreamParser parses a growing program text, reusing completed statements
// (spec section 4). It is not safe for concurrent use.
type StreamParser struct {
	p            *parser
	buf          string
	cleaned      string
	completedEnd int
	completed    *stmtMap
	count        int
	calls        int
	nodes        int
	last         *ParseResult
}

// NewStreamParser returns a streaming parser for lib.
func NewStreamParser(lib *Library, limits Limits) *StreamParser {
	return &StreamParser{p: &parser{lib: lib, limits: limits.withDefaults()}, completed: newStmtMap()}
}

// Reset discards all cached state.
func (s *StreamParser) Reset() {
	s.buf, s.cleaned, s.completedEnd, s.count, s.calls, s.nodes = "", "", 0, 0, 0, 0
	s.completed = newStmtMap()
}

// Push appends a chunk. complete marks the end of the stream.
func (s *StreamParser) Push(chunk string, complete bool) *ParseResult {
	s.buf += chunk
	return s.result(complete)
}

// Set replaces the full text so far; an appended text reuses the cache,
// any other change starts over.
func (s *StreamParser) Set(full string, complete bool) *ParseResult {
	if len(full) < len(s.buf) || !strings.HasPrefix(full, s.buf) {
		s.Reset()
	}
	s.buf = full
	return s.result(complete)
}

// Result returns the current result.
func (s *StreamParser) Result(complete bool) *ParseResult { return s.result(complete) }

func (s *StreamParser) result(complete bool) (res *ParseResult) {
	defer func() {
		if r := recover(); r != nil {
			res = failureResult(r)
		}
		s.last = res
	}()
	if len(s.buf) > s.p.limits.MaxProgramBytes {
		return budgetResult(&budgetError{phase: "source", limit: s.p.limits.MaxProgramBytes})
	}
	next := Preprocess(s.buf)
	if !strings.HasPrefix(next, s.cleaned[:s.completedEnd]) {
		s.completedEnd, s.count, s.calls, s.nodes = 0, 0, 0, 0
		s.completed = newStmtMap()
	}
	s.cleaned = next
	pendingStart, err := s.scanCompleted()
	if err != nil {
		return budgetResult(err)
	}
	pending := trimJSSpace(s.cleaned[pendingStart:])
	if pending == "" {
		if s.count == 0 {
			return s.p.finish(emptyResult(true), complete)
		}
		return s.p.finish(s.p.build(s.completed.clone(), false, s.count, nil), complete)
	}
	closed, incomplete := autoClose(pending)
	toks, ok := tokenize(closed, s.p.unicode(), s.p.limits.MaxTokens)
	if !ok {
		return budgetResult(&budgetError{phase: "tokens", limit: s.p.limits.MaxTokens})
	}
	raws := splitStatements(toks)
	if len(raws) == 0 {
		if s.count == 0 {
			return s.p.finish(emptyResult(incomplete), complete)
		}
		return s.p.finish(s.p.build(s.completed.clone(), incomplete, s.count, nil), complete)
	}
	all := s.completed.clone()
	var added []*Statement
	calls, nodes := s.calls, s.nodes
	for _, r := range raws {
		if _, done := s.completed.m[r.id]; done && incomplete {
			continue
		}
		expr, ok := parseExpression(r.toks, &s.p.limits, &calls, &nodes)
		if !ok {
			return budgetResult(&budgetError{phase: "AST", limit: s.p.limits.MaxASTNodes, path: r.id})
		}
		st := classify(r, expr)
		all.set(st)
		added = append(added, st)
	}
	if len(all.order) > s.p.limits.MaxStatements {
		return budgetResult(&budgetError{phase: "statements", limit: s.p.limits.MaxStatements})
	}
	var pref *partialRef
	if !complete {
		if tail := tailStateRef.FindString(s.buf); tail != "" && strings.HasSuffix(pending, tail) {
			head := pending[:len(pending)-len(tail)]
			if !containsWholeWord(head, tail) {
				set := map[*Statement]bool{}
				for _, st := range added {
					set[st] = true
				}
				pref = &partialRef{name: tail, stmts: set}
			}
		}
	}
	return s.p.finish(s.p.build(all, incomplete, s.count+len(raws), pref), complete)
}

// containsWholeWord reports whether word occurs in s not followed by a word
// character.
func containsWholeWord(s, word string) bool {
	for i := 0; ; {
		j := strings.Index(s[i:], word)
		if j < 0 {
			return false
		}
		end := i + j + len(word)
		if end >= len(s) || !isWordByte(s[end]) {
			return true
		}
		i = i + j + 1
	}
}

func isWordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// scanCompleted moves completed statements into the cache and returns the
// start of the pending tail.
func (s *StreamParser) scanCompleted() (int, error) {
	depth, ternary := 0, 0
	var inStr byte
	esc := false
	stmtStart := s.completedEnd
	text := s.cleaned
	for i := s.completedEnd; i < len(text); i++ {
		c := text[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' && inStr != 0 {
			esc = true
			continue
		}
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '?':
			if depth == 0 {
				ternary++
			}
		case ':':
			if depth == 0 && ternary > 0 {
				ternary--
			}
		case '\n':
			if depth > 0 || ternary > 0 {
				continue
			}
			peek := i + 1
			for peek < len(text) && (text[peek] == ' ' || text[peek] == '\t' || text[peek] == '\r' || text[peek] == '\n') {
				peek++
			}
			if peek < len(text) && (text[peek] == '?' || text[peek] == ':' && ternary > 0) {
				continue
			}
			if t := trimJSSpace(text[stmtStart:i]); t != "" {
				n, err := s.p.parseInto(t, s.completed, &s.calls, &s.nodes)
				s.count += n
				if err != nil {
					return 0, err
				}
			}
			stmtStart = i + 1
			s.completedEnd = i + 1
		}
	}
	return stmtStart, nil
}
