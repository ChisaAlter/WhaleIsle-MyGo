package lang

import (
	"math"
	"regexp"
	"strings"
)

// ── Edit merge (spec section 7, upstream merge.ts) ──

type mergeStmt struct {
	id  string
	ast Node
	raw string
}

func splitStatementSource(input string) []string {
	var out []string
	depth := 0
	var inStr byte
	esc := false
	start := 0
	for i := 0; i < len(input); i++ {
		c := input[i]
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
		case '\n':
			if depth <= 0 {
				if s := strings.TrimSpace(input[start:i]); s != "" {
					out = append(out, s)
				}
				start = i + 1
			}
		}
	}
	if s := strings.TrimSpace(input[start:]); s != "" {
		out = append(out, s)
	}
	return out
}

func parseMergeStatements(input string, unicode bool) []mergeStmt {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	var out []mergeStmt
	lim := DefaultLimits
	for _, raw := range splitStatementSource(input) {
		toks, _ := tokenize(raw, unicode, lim.MaxTokens)
		stmts := splitStatements(toks)
		if len(stmts) == 0 {
			continue
		}
		ast, _ := parseExpression(stmts[0].toks, &lim, nil, nil)
		out = append(out, mergeStmt{id: stmts[0].id, ast: ast, raw: raw})
	}
	return out
}

// MergeStatements merges an edit-mode patch into a program by statement
// name: replace, append, `name = null` deletes, and statements root no
// longer reaches are removed (state declarations are kept).
func MergeStatements(existing, patch string, lib *Library) string {
	uni := lib != nil && lib.Profile.UnicodeIdentifiers
	cur := parseMergeStatements(existing, uni)
	pat := parseMergeStatements(StripFences(patch), uni)
	if len(cur) == 0 {
		parts := make([]string, len(pat))
		for i, s := range pat {
			parts[i] = s.raw
		}
		return strings.Join(parts, "\n")
	}
	if len(pat) == 0 {
		return existing
	}
	merged := map[string]string{}
	asts := map[string]Node{}
	var order []string
	for _, s := range cur {
		if _, ok := merged[s.id]; !ok {
			order = append(order, s.id)
		}
		merged[s.id], asts[s.id] = s.raw, s.ast
	}
	for _, s := range pat {
		if _, isNull := s.ast.(*NullNode); isNull {
			delete(merged, s.id)
			delete(asts, s.id)
			for i, id := range order {
				if id == s.id {
					order = append(order[:i], order[i+1:]...)
					break
				}
			}
			continue
		}
		if _, ok := merged[s.id]; !ok {
			order = append(order, s.id)
		}
		merged[s.id], asts[s.id] = s.raw, s.ast
	}
	if _, ok := asts["root"]; ok {
		reach := map[string]bool{"root": true}
		queue := []string{"root"}
		for len(queue) > 0 {
			id := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			walkAST(asts[id], func(n Node) {
				var name string
				switch t := n.(type) {
				case *RefNode:
					name = t.N
				case *RuntimeRefNode:
					name = t.N
				default:
					return
				}
				if _, ok := asts[name]; ok && !reach[name] {
					reach[name] = true
					queue = append(queue, name)
				}
			})
		}
		kept := order[:0]
		for _, id := range order {
			if reach[id] || strings.HasPrefix(id, "$") {
				kept = append(kept, id)
			} else {
				delete(merged, id)
			}
		}
		order = kept
	}
	lines := make([]string, 0, len(order))
	for _, id := range order {
		if raw, ok := merged[id]; ok {
			lines = append(lines, raw)
		}
	}
	return strings.Join(lines, "\n")
}

// ── Serialization (spec section 9.2, upstream serialize.ts) ──

var binPrec = map[string]int{"||": 2, "&&": 3, "==": 4, "!=": 4, ">": 5, "<": 5, ">=": 5, "<=": 5, "+": 6, "-": 6, "*": 7, "/": 7, "%": 7}

type serializer struct {
	lib   *Library
	order []string
	stmts map[string]string
}

func (s *serializer) node(n Node) string {
	switch t := n.(type) {
	case *StrNode:
		return jsonQuote(t.V)
	case *NumNode:
		return NumberToString(t.V)
	case *BoolNode:
		if t.V {
			return "true"
		}
		return "false"
	case *NullNode:
		return "null"
	case *PhNode:
		return t.N
	case *ValueNode:
		return s.value(t.V)
	case *StateRefNode:
		return t.N
	case *RuntimeRefNode:
		return t.N
	case *RefNode:
		return t.N
	case *ArrNode:
		parts := make([]string, len(t.Els))
		for i, e := range t.Els {
			parts[i] = s.node(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *ObjNode:
		parts := make([]string, len(t.Entries))
		for i, e := range t.Entries {
			parts[i] = e.Key + ": " + s.node(e.Value)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *BinOpNode:
		child := func(c Node, right bool) string {
			out := s.node(c)
			if b, ok := c.(*BinOpNode); ok {
				if binPrec[b.Op] < binPrec[t.Op] || right && binPrec[b.Op] == binPrec[t.Op] {
					return "(" + out + ")"
				}
			}
			return out
		}
		return child(t.L, false) + " " + t.Op + " " + child(t.R, true)
	case *UnaryNode:
		x := s.node(t.X)
		switch t.X.(type) {
		case *BinOpNode, *TernaryNode:
			x = "(" + x + ")"
		}
		return t.Op + x
	case *TernaryNode:
		return s.node(t.Cond) + " ? " + s.node(t.Then) + " : " + s.node(t.Else)
	case *MemberNode:
		return s.node(t.X) + "." + t.Field
	case *IndexNode:
		return s.node(t.X) + "[" + s.node(t.Index) + "]"
	case *AssignNode:
		return t.Target + " = " + s.node(t.Value)
	case *CallNode:
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			parts[i] = s.node(a)
		}
		if s.lib.callDef(t.Name) != nil && t.Name != "Action" {
			return "@" + t.Name + "(" + strings.Join(parts, ", ") + ")"
		}
		return t.Name + "(" + strings.Join(parts, ", ") + ")"
	}
	return "null"
}

func (s *serializer) value(v Value) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return jsonQuote(t)
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return "null"
		}
		return NumberToString(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case *Element:
		if t.StatementID != "" {
			if _, ok := s.stmts[t.StatementID]; !ok {
				s.stmts[t.StatementID] = ""
				expr := s.element(t)
				s.stmts[t.StatementID] = t.StatementID + " = " + expr
				s.order = append(s.order, t.StatementID)
			}
			return t.StatementID
		}
		return s.element(t)
	case Node:
		return s.node(t)
	case []Value:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = s.value(x)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Object:
		parts := make([]string, 0, t.Len())
		for i, k := range t.Keys() {
			parts = append(parts, k+": "+s.value(t.vals[i]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Binding:
		return t.Target
	}
	return "null"
}

func (s *serializer) element(el *Element) string {
	c := s.lib.Components[el.Type]
	if c == nil {
		parts := make([]string, 0, el.Props.Len())
		for i := range el.Props.Keys() {
			parts = append(parts, s.value(el.Props.vals[i]))
		}
		return el.Type + "(" + strings.Join(parts, ", ") + ")"
	}
	args := make([]string, len(c.Params))
	for i, p := range c.Params {
		args[i] = s.value(el.Props.Value(p.Name))
	}
	for len(args) > 0 && args[len(args)-1] == "null" && !c.Params[len(args)-1].Required {
		args = args[:len(args)-1]
	}
	return el.Type + "(" + strings.Join(args, ", ") + ")"
}

// Serialize writes an element tree back to source text: the entry first,
// then each named component as its own statement, then state declarations.
func Serialize(root *Element, lib *Library, state *Object) string {
	s := &serializer{lib: lib, stmts: map[string]string{}}
	rootID := root.StatementID
	if rootID == "" {
		rootID = "root"
	}
	s.stmts[rootID] = ""
	expr := s.element(root)
	lines := []string{rootID + " = " + expr}
	for _, id := range s.order {
		if id != rootID {
			lines = append(lines, s.stmts[id])
		}
	}
	for i, k := range state.Keys() {
		if v := state.vals[i]; v != nil {
			lines = append(lines, k+" = "+s.value(v))
		}
	}
	return strings.Join(lines, "\n")
}

// ── Validation rules (spec section 5.4) ──

// ValidateRules checks a field value against a rules object such as
// {required: true, email: true, maxLength: 80}. It returns the message of
// the first failing rule, or "".
func ValidateRules(value Value, rules *Object) string {
	for i, k := range rules.Keys() {
		arg := rules.vals[i]
		if arg == nil || arg == false {
			continue
		}
		if msg := validateRule(k, value, arg); msg != "" {
			return msg
		}
	}
	return ""
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func isEmptyValue(v Value) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []Value:
		return len(t) == 0
	case *Field:
		return isEmptyValue(t.Value)
	case *Object:
		if t.Has("value") {
			return isEmptyValue(t.Value("value"))
		}
		return t.Len() == 0
	}
	return false
}

func validateRule(rule string, v Value, arg Value) string {
	if rule != "required" && isEmptyValue(v) {
		return ""
	}
	switch rule {
	case "required":
		if isEmptyValue(v) {
			return "This field is required"
		}
		if o, ok := v.(*Object); ok && o.Len() > 0 {
			allBool, any := true, false
			for i := range o.Keys() {
				b, ok := o.vals[i].(bool)
				if !ok {
					allBool = false
					break
				}
				any = any || b
			}
			if allBool && !any {
				return "At least one option is required"
			}
		}
	case "email":
		s, ok := v.(string)
		if !ok || !emailRe.MatchString(s) {
			return "Please enter a valid email"
		}
	case "url":
		s, ok := v.(string)
		if !ok || !isAbsoluteURL(s) {
			return "Please enter a valid URL"
		}
	case "numeric":
		switch t := v.(type) {
		case float64:
			if !math.IsNaN(t) {
				return ""
			}
		case string:
			if !math.IsNaN(ParseFloat(t)) && strings.TrimSpace(t) != "" {
				return ""
			}
		}
		return "Must be a number"
	case "min", "max":
		n, ok := v.(float64)
		if !ok {
			n = ParseFloat(ToString(v))
		}
		if math.IsNaN(n) {
			return ""
		}
		lim := jsToNumber(arg)
		if rule == "min" && !(n >= lim) {
			return "Must be at least " + NumberToString(lim)
		}
		if rule == "max" && !(n <= lim) {
			return "Must be no more than " + NumberToString(lim)
		}
	case "minLength", "maxLength":
		s, ok := v.(string)
		if !ok {
			return ""
		}
		lim := jsToNumber(arg)
		n := float64(utf16Len(s))
		if rule == "minLength" && !(n >= lim) {
			return "Must be at least " + NumberToString(lim) + " characters"
		}
		if rule == "maxLength" && !(n <= lim) {
			return "Must be no more than " + NumberToString(lim) + " characters"
		}
	case "pattern":
		s, ok := v.(string)
		p, ok2 := arg.(string)
		if !ok || !ok2 {
			return ""
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return ""
		}
		if !re.MatchString(s) {
			return "Invalid format"
		}
	}
	return ""
}

// isAbsoluteURL approximates `new URL(value)` succeeding: a scheme followed
// by a non-empty remainder (special schemes need a host).
func isAbsoluteURL(s string) bool {
	s = strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return false
	}
	scheme := strings.ToLower(s[:i])
	for j, c := range scheme {
		if !(c >= 'a' && c <= 'z' || j > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	rest := s[i+1:]
	switch scheme {
	case "http", "https", "ws", "wss", "ftp":
		rest = strings.TrimLeft(rest, "/\\")
		host := rest
		if k := strings.IndexAny(host, "/\\?#"); k >= 0 {
			host = host[:k]
		}
		if at := strings.LastIndexByte(host, '@'); at >= 0 {
			host = host[at+1:]
		}
		return host != "" && !strings.ContainsAny(host, " <>^|")
	case "file":
		return true
	}
	return true
}
