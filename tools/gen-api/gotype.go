package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// goTypeer maps parsed zod schemas to Go type expressions and collects the
// named Go type declarations needed by them.
type goTypeer struct {
	schemas     map[string]*Schema
	curPkg      string
	docCtx      string
	used        map[string]bool
	defs        []goDef
	inProg      map[string]bool
	structCache map[string]string
	symbolNames map[string]string // typeSymbol/schemaKey -> declared Go type
}

type goDef struct {
	pkg  string
	code string
}

func newGoTypeer() *goTypeer {
	return &goTypeer{
		used:        map[string]bool{"SessionProjectionsValue": true},
		inProg:      map[string]bool{},
		structCache: map[string]string{},
		// This wire union has a handwritten decoder enforcing sequence authority.
		// Keep it at that boundary instead of regenerating a RawMessage alias.
		symbolNames: map[string]string{
			"@deepseek-ai/dsh-api-session-controller/types#SessionProjectionsValue": "SessionProjectionsValue",
		},
	}
}

// goType maps a schema to a Go type expression. The second return value is
// an optional one-line explanation when the mapping degraded to raw JSON.
func (g *goTypeer) goType(s *Schema, hint string) (string, string) {
	if s == nil {
		return "json.RawMessage", "missing schema"
	}
	if s.Unsupported {
		return "json.RawMessage", "zod construct not mapped; kept as raw JSON"
	}
	switch s.Kind {
	case "string", "litstr":
		return "string", ""
	case "number", "litnum":
		return "float64", ""
	case "boolean", "litbool":
		return "bool", ""
	case "any", "nulllit":
		return "any", ""
	case "enum":
		return "string", ""
	case "array":
		el, note := g.goType(s.Elem, hint+"Item")
		if note != "" {
			return "[]" + el, "array element: " + note
		}
		return "[]" + el, ""
	case "record":
		val, note := g.goType(s.Elem, hint+"Value")
		if note != "" {
			return "map[string]" + val, "record value: " + note
		}
		return "map[string]" + val, ""
	case "object":
		return g.structType(s, hint), ""
	case "union":
		return g.unionType(s, hint)
	case "intersection":
		return g.intersectionType(s, hint)
	case "lazy":
		return g.lazyType(s, hint)
	default:
		return "json.RawMessage", fmt.Sprintf("zod construct %q not mapped; kept as raw JSON", s.Kind)
	}
}

func (g *goTypeer) unionType(s *Schema, hint string) (string, string) {
	var rest []*Schema
	for _, m := range s.Members {
		if m.Kind == "nulllit" {
			continue
		}
		rest = append(rest, m)
	}
	if len(rest) == 0 {
		return "any", "union with no non-null members"
	}
	if len(rest) == 1 {
		return g.goType(rest[0], hint)
	}
	allStr, allNum, allBool := true, true, true
	for _, m := range rest {
		switch m.Kind {
		case "string", "litstr":
			allNum, allBool = false, false
		case "number", "litnum":
			allStr, allBool = false, false
		case "boolean", "litbool":
			allStr, allNum = false, false
		default:
			return "json.RawMessage", "union of complex members kept as raw JSON"
		}
	}
	switch {
	case allStr:
		return "string", ""
	case allNum:
		return "float64", ""
	case allBool:
		return "bool", ""
	}
	return "json.RawMessage", "union of mixed primitives kept as raw JSON"
}

func (g *goTypeer) intersectionType(s *Schema, hint string) (string, string) {
	var obj *Schema
	hasString := false
	for _, m := range s.Members {
		switch m.Kind {
		case "string", "litstr":
			hasString = true
		case "object":
			obj = m
		}
	}
	if hasString {
		// branded string types are emitted as plain string
		return "string", ""
	}
	if obj != nil {
		pure := true
		for _, m := range s.Members {
			if m != obj && m.Kind != "any" {
				pure = false
			}
		}
		if pure {
			return g.structType(obj, hint), ""
		}
	}
	return "json.RawMessage", "intersection kept as raw JSON"
}

func (g *goTypeer) lazyType(s *Schema, hint string) (string, string) {
	if s.Str == "" || g.inProg[s.Str] {
		// recursive reference: Go maps cannot express it inline here without
		// named indirection, so degrade to any
		return "any", ""
	}
	def, ok := g.schemas[s.Str]
	if !ok {
		return "json.RawMessage", "unresolved schema reference " + s.Str
	}
	g.inProg[s.Str] = true
	defer delete(g.inProg, s.Str)
	return g.goType(def, hint)
}

func (g *goTypeer) structType(s *Schema, hint string) string {
	key := schemaKey(s)
	if name, ok := g.structCache[key]; ok {
		return name
	}
	name := g.uniqueName(hint)
	doc := ""
	if g.docCtx != "" {
		doc = fmt.Sprintf("// %s is the wire type of %s.\n", name, g.docCtx)
	}
	savedDoc := g.docCtx
	g.docCtx = ""
	defer func() { g.docCtx = savedDoc }()

	seen := map[string]int{}
	var fields []string
	for _, f := range s.Fields {
		gname := goFieldName(f.Name)
		if gname == "" {
			gname = "Field"
		}
		if n := seen[gname]; n > 0 {
			gname = fmt.Sprintf("%s%d", gname, n+1)
		}
		seen[gname]++
		expr, note := g.goType(f.Schema, name+gname)
		if f.Schema.Optional && pointerWorthy(expr) || f.Schema.Optional && isStructName(expr) {
			expr = "*" + expr
		}
		tag := f.Name
		if f.Schema.Optional || f.Schema.Nullable {
			tag += ",omitempty"
		}
		line := fmt.Sprintf("\t%s %s `json:%s`", gname, expr, strconv.Quote(tag))
		if note != "" {
			line += " // " + singleLine(note)
		}
		fields = append(fields, line)
	}
	var b strings.Builder
	b.WriteString(doc)
	fmt.Fprintf(&b, "type %s struct {\n", name)
	for _, l := range fields {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	b.WriteString("}\n")
	g.defs = append(g.defs, goDef{pkg: g.curPkg, code: b.String()})
	g.structCache[key] = name
	return name
}

// rootType declares the Go type for one parameter/result/uplink schema and
// returns its Go type name. ref is the referenced schema constant name and
// may be empty (e.g. a src-json codec), in which case nothing is declared.
func (g *goTypeer) rootType(ref, typeSymbol, what, hint string) (string, error) {
	if ref == "" {
		return "", nil
	}
	s, ok := g.schemas[ref]
	if !ok {
		return "", fmt.Errorf("schema constant %q not found", ref)
	}
	// The same type symbol always denotes the same wire type; declare it once.
	dedupeKey := typeSymbol
	if dedupeKey == "" {
		dedupeKey = schemaKey(s)
	}
	if name, ok := g.symbolNames[dedupeKey]; ok {
		return name, nil
	}
	g.docCtx = what
	defer func() { g.docCtx = "" }()
	g.inProg[ref] = true
	defer delete(g.inProg, ref)

	var name string
	if s.Kind == "object" && !s.Unsupported {
		name = g.structType(s, hint)
		// Deduplicated fields may share a struct, but named public schemas still
		// need their own stable entry point (for example SettingsNamespaceView).
		if typeSymbol != "" && name != goTypeName(hint) {
			target := name
			name = g.uniqueName(hint)
			g.defs = append(g.defs, goDef{pkg: g.curPkg, code: fmt.Sprintf("// %s is the wire type of %s.\ntype %s = %s\n", name, what, name, target)})
		}
	} else {
		expr, note := g.goType(s, hint)
		name = g.uniqueName(hint)
		var b strings.Builder
		fmt.Fprintf(&b, "// %s is the wire type of %s.\n", name, g.docCtx)
		if note != "" {
			fmt.Fprintf(&b, "// %s\n", note)
		}
		fmt.Fprintf(&b, "type %s = %s\n", name, expr)
		g.defs = append(g.defs, goDef{pkg: g.curPkg, code: b.String()})
	}
	g.symbolNames[dedupeKey] = name
	return name, nil
}

func (g *goTypeer) uniqueName(hint string) string {
	base := goTypeName(hint)
	name := base
	for i := 2; g.used[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	g.used[name] = true
	return name
}

func pointerWorthy(expr string) bool {
	switch expr {
	case "string", "float64", "bool":
		return true
	}
	return false
}

// isStructName reports whether expr is a bare named struct type (its first
// character is an uppercase letter), which is safe to point at when optional.
func isStructName(expr string) bool {
	if expr == "" {
		return false
	}
	c := expr[0]
	return c >= 'A' && c <= 'Z'
}

// goFieldName converts a JSON property name into an exported Go field name.
func goFieldName(name string) string {
	var b strings.Builder
	upper := true
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				b.WriteRune(r)
			}
		} else {
			upper = true
		}
	}
	s := b.String()
	if s == "" {
		return ""
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "F" + s
	}
	return s
}

func goTypeName(hint string) string {
	s := camel(hint)
	if s == "" {
		return "Anon"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "T" + s
	}
	return s
}

func camel(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				b.WriteRune(r)
			}
		} else {
			upper = true
		}
	}
	return b.String()
}

func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// nameFromSymbol derives a Go type name from a codec type symbol.
// Recognized forms:
//   - "<pkg>/types#ExportedName"          -> ExportedName
//   - "<pkg>#<ns>/<method>:result"        -> <Method>Result
//   - "<pkg>#<ns>/<method>:<param>"       -> <Method><Param>
//   - "<pkg>#<ns>/<method>"               -> <Method><Role>
//   - ""                                  -> <Method><Role>
func nameFromSymbol(typeSymbol, method, argName string, isResult bool) string {
	role := "Args"
	if isResult {
		role = "Result"
	}
	if argName != "" && !isResult {
		role = camel(argName)
	}
	if typeSymbol == "" {
		return camel(method) + role
	}
	tail := typeSymbol
	if i := strings.LastIndex(tail, "#"); i >= 0 {
		tail = tail[i+1:]
	}
	if i := strings.Index(tail, ":"); i >= 0 {
		left, right := tail[:i], tail[i+1:]
		if right == "result" {
			return camel(method) + "Result"
		}
		if right == "" {
			return camel(lastSegment(left)) + role
		}
		return camel(lastSegment(left)) + camel(right)
	}
	if strings.Contains(tail, "/") {
		return camel(lastSegment(tail)) + role
	}
	return camel(tail)
}

// schemaKey produces a canonical serialization of a schema used to detect
// structurally identical objects so the same Go type is reused.
func schemaKey(s *Schema) string {
	var b strings.Builder
	writeSchemaKey(&b, s)
	return b.String()
}

func writeSchemaKey(b *strings.Builder, s *Schema) {
	if s == nil {
		b.WriteString("<nil>")
		return
	}
	b.WriteString(s.Kind)
	if s.Unsupported {
		b.WriteString("!")
	}
	if s.Optional {
		b.WriteString("?")
	}
	if s.Nullable {
		b.WriteString("~")
	}
	switch s.Kind {
	case "litstr", "lazy":
		b.WriteString("=" + s.Str)
	case "litbool":
		if s.Bool {
			b.WriteString("=t")
		} else {
			b.WriteString("=f")
		}
	case "litnum":
		fmt.Fprintf(b, "=%g", s.Num)
	case "array", "record":
		b.WriteString("[")
		if s.Key != nil {
			writeSchemaKey(b, s.Key)
		}
		b.WriteString("|")
		writeSchemaKey(b, s.Elem)
		b.WriteString("]")
	case "union", "intersection":
		for _, m := range s.Members {
			b.WriteString("(")
			writeSchemaKey(b, m)
			b.WriteString(")")
		}
	case "object":
		for _, f := range s.Fields {
			b.WriteString(f.Name)
			b.WriteString(":")
			writeSchemaKey(b, f.Schema)
			b.WriteString(";")
		}
	}
}

func singleLine(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}
