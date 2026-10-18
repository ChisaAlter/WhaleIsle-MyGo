package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// typeMapper maps a CanonicalSchema graph onto Go type expressions and named
// struct declarations. Complex unions / tuples / never degrade to
// json.RawMessage, matching tools/gen-api.
type typeMapper struct {
	nodes   []SchemaNode
	prefix  string // root type name, used to namespace nested types
	used    map[string]bool
	names   map[int]string // object/map node index -> declared Go name
	inProg  map[int]bool
	defs    []string
	needJSON bool
	rootDoc string
}

func newTypeMapper(nodes []SchemaNode, prefix, rootDoc string) *typeMapper {
	return &typeMapper{
		nodes:   nodes,
		prefix:  prefix,
		used:    map[string]bool{},
		names:   map[int]string{},
		inProg:  map[int]bool{},
		rootDoc: rootDoc,
	}
}

func (m *typeMapper) declareRoot(rootIdx int, exactName string) (string, error) {
	if rootIdx < 0 || rootIdx >= len(m.nodes) {
		return "", fmt.Errorf("root index %d out of range", rootIdx)
	}
	if exactName == "" {
		exactName = m.prefix
	}
	if m.used[exactName] {
		return "", fmt.Errorf("Go type name %q already used", exactName)
	}
	m.used[exactName] = true
	n := m.nodes[rootIdx]
	switch n.Kind {
	case "object":
		m.names[rootIdx] = exactName
		m.inProg[rootIdx] = true
		code := m.renderObject(exactName, n, m.rootDoc)
		m.inProg[rootIdx] = false
		m.defs = append(m.defs, code)
		return exactName, nil
	default:
		expr, note := m.goType(rootIdx, exactName)
		var b strings.Builder
		fmt.Fprintf(&b, "// %s is %s.\n", exactName, m.rootDoc)
		if note != "" {
			fmt.Fprintf(&b, "// %s\n", note)
		}
		fmt.Fprintf(&b, "type %s = %s\n", exactName, expr)
		m.defs = append(m.defs, b.String())
		return exactName, nil
	}
}

func (m *typeMapper) goType(idx int, hint string) (string, string) {
	if idx < 0 || idx >= len(m.nodes) {
		m.needJSON = true
		return "json.RawMessage", fmt.Sprintf("node index %d out of range", idx)
	}
	n := m.nodes[idx]
	switch n.Kind {
	case "primitive":
		switch n.Type {
		case "string":
			return "string", ""
		case "number":
			return "float64", ""
		case "boolean":
			return "bool", ""
		case "null":
			return "any", "JSON null"
		case "never":
			m.needJSON = true
			return "json.RawMessage", "primitive never kept as raw JSON"
		default:
			m.needJSON = true
			return "json.RawMessage", fmt.Sprintf("unknown primitive %q", n.Type)
		}
	case "literal":
		return m.literalType(n.Value)
	case "opaque":
		return "any", "opaque " + n.Reason
	case "array":
		if n.Element == nil {
			m.needJSON = true
			return "json.RawMessage", "array missing element"
		}
		el, note := m.goType(*n.Element, hint+"Item")
		if note != "" {
			return "[]" + el, "array element: " + note
		}
		return "[]" + el, ""
	case "tuple":
		m.needJSON = true
		return "json.RawMessage", "tuple kept as raw JSON"
	case "union":
		return m.unionType(n, hint)
	case "object":
		return m.objectType(idx, hint), ""
	default:
		m.needJSON = true
		return "json.RawMessage", fmt.Sprintf("schema kind %q not mapped", n.Kind)
	}
}

func (m *typeMapper) literalType(raw json.RawMessage) (string, string) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		m.needJSON = true
		return "json.RawMessage", "empty literal"
	}
	switch raw[0] {
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			m.needJSON = true
			return "json.RawMessage", "bad string literal"
		}
		return "string", "literal " + strconv.Quote(s)
	case 't', 'f':
		return "bool", "literal " + string(raw)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return "float64", "literal " + string(raw)
	case 'n':
		return "any", "literal null"
	default:
		m.needJSON = true
		return "json.RawMessage", "unsupported literal"
	}
}

func (m *typeMapper) unionType(n SchemaNode, hint string) (string, string) {
	var rest []int
	for _, t := range n.Types {
		if t < 0 || t >= len(m.nodes) {
			continue
		}
		if m.nodes[t].Kind == "primitive" && m.nodes[t].Type == "null" {
			continue
		}
		if m.nodes[t].Kind == "literal" {
			v := strings.TrimSpace(string(m.nodes[t].Value))
			if v == "null" {
				continue
			}
		}
		rest = append(rest, t)
	}
	if len(rest) == 0 {
		return "any", "union with no non-null members"
	}
	if len(rest) == 1 {
		return m.goType(rest[0], hint)
	}
	allStr, allNum, allBool := true, true, true
	for _, t := range rest {
		nn := m.nodes[t]
		switch nn.Kind {
		case "primitive":
			switch nn.Type {
			case "string":
				allNum, allBool = false, false
			case "number":
				allStr, allBool = false, false
			case "boolean":
				allStr, allNum = false, false
			default:
				m.needJSON = true
				return "json.RawMessage", "union of complex members kept as raw JSON"
			}
		case "literal":
			expr, _ := m.literalType(nn.Value)
			switch expr {
			case "string":
				allNum, allBool = false, false
			case "float64":
				allStr, allBool = false, false
			case "bool":
				allStr, allNum = false, false
			default:
				m.needJSON = true
				return "json.RawMessage", "union of complex members kept as raw JSON"
			}
		default:
			m.needJSON = true
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
	m.needJSON = true
	return "json.RawMessage", "union of mixed primitives kept as raw JSON"
}

func (m *typeMapper) objectType(idx int, hint string) string {
	if name, ok := m.names[idx]; ok {
		return name
	}
	n := m.nodes[idx]
	// Pure index-signature objects become maps when the key is a string.
	if len(n.Properties) == 0 && len(n.Indices) == 1 {
		ix := n.Indices[0]
		keyExpr, keyNote := m.goType(ix.Key, hint+"Key")
		valExpr, valNote := m.goType(ix.Value, hint+"Value")
		if keyExpr == "string" && keyNote == "" {
			note := ""
			if valNote != "" {
				note = valNote
			}
			_ = note
			return "map[string]" + valExpr
		}
		m.needJSON = true
		name := m.uniqueName(hint)
		m.names[idx] = name
		var b strings.Builder
		fmt.Fprintf(&b, "// %s is a non-string-keyed index object kept as raw JSON.\n", name)
		fmt.Fprintf(&b, "type %s = json.RawMessage\n", name)
		m.defs = append(m.defs, b.String())
		return name
	}
	if len(n.Indices) > 0 {
		// Properties plus index signatures: degrade.
		m.needJSON = true
		name := m.uniqueName(hint)
		m.names[idx] = name
		var b strings.Builder
		fmt.Fprintf(&b, "// %s mixes fixed properties and index signatures; kept as raw JSON.\n", name)
		fmt.Fprintf(&b, "type %s = json.RawMessage\n", name)
		m.defs = append(m.defs, b.String())
		return name
	}

	name := m.uniqueName(hint)
	m.names[idx] = name
	m.inProg[idx] = true
	code := m.renderObject(name, n, "")
	m.inProg[idx] = false
	m.defs = append(m.defs, code)
	return name
}

func (m *typeMapper) renderObject(name string, n SchemaNode, doc string) string {
	var b strings.Builder
	if doc != "" {
		fmt.Fprintf(&b, "// %s is %s.\n", name, doc)
	}
	fmt.Fprintf(&b, "type %s struct {\n", name)
	seen := map[string]int{}
	for _, p := range n.Properties {
		gname := goFieldName(p.Name)
		if gname == "" {
			gname = "Field"
		}
		if c := seen[gname]; c > 0 {
			gname = fmt.Sprintf("%s%d", gname, c+1)
		}
		seen[gname]++

		hint := name + gname
		expr, note := m.goType(p.Type, hint)
		if p.Optional && (pointerWorthy(expr) || isStructName(expr)) {
			expr = "*" + expr
		}
		tag := p.Name
		if p.Optional {
			tag += ",omitempty"
		}
		line := fmt.Sprintf("\t%s %s `json:%s`", gname, expr, strconv.Quote(tag))
		if note != "" {
			line += " // " + singleLine(note)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("}\n")
	return b.String()
}

func (m *typeMapper) uniqueName(hint string) string {
	base := goTypeName(hint)
	name := base
	for i := 2; m.used[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	m.used[name] = true
	return name
}

func pointerWorthy(expr string) bool {
	switch expr {
	case "string", "float64", "bool":
		return true
	}
	return false
}

func isStructName(expr string) bool {
	if expr == "" {
		return false
	}
	c := expr[0]
	return c >= 'A' && c <= 'Z'
}

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

func singleLine(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "")
}

// rootGoName picks the exported Go type name for a persistence root.
func rootGoName(r PersistenceRoot) string {
	switch r.Kind {
	case "event":
		return goTypeName(r.Event)
	default:
		return goTypeName(r.Key)
	}
}
