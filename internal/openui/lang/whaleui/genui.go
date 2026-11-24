package whaleui

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"whaleisle/internal/openui/lang"
)

// genuiSchema is one entry of the viewer's genui component schema table
// (host.cjs `li`, built by `kt(required, fields, aliases, opts)`), or a
// nested record schema (`Pi`).
type genuiSchema struct {
	Required            []string                     `json:"required"`
	Fields              map[string]string            `json:"fields"`
	Aliases             map[string]string            `json:"aliases"`
	ValueAliases        map[string]map[string]string `json:"valueAliases"`
	OneOfRequired       [][]string                   `json:"oneOfRequired"`
	ConditionalRequired []struct {
		Kind string `json:"kind"`
		When struct {
			Field  string `json:"field"`
			Equals any    `json:"equals"`
		} `json:"when"`
		Required []string `json:"required"`
		Message  string   `json:"message"`
	} `json:"conditionalRequired"`
	Enums     map[string][]string     `json:"enums"`
	Nested    map[string]*genuiSchema `json:"nested"`
	Validator *struct {
		Name string `json:"name"`
	} `json:"validator"`
}

// genuiLimits mirrors the viewer's genui limits (`X`).
type genuiLimits struct {
	MaxDepth          int `json:"maxDepth"`
	MaxNodes          int `json:"maxNodes"`
	MaxTabs           int `json:"maxTabs"`
	MaxAccordionItems int `json:"maxAccordionItems"`
}

// Genui validates the options of the specialized components and Genui
// specs against the extracted schema table.
type Genui struct {
	Schemas map[string]*genuiSchema
	// Types maps a component (for example "Plot") to its genui type ("plot").
	Types  map[string]string
	limits genuiLimits
}

func newGenui(raw lang.Value) (*Genui, error) {
	g := &Genui{Schemas: map[string]*genuiSchema{}, Types: map[string]string{}}
	o, _ := raw.(*lang.Object)
	if o == nil {
		return g, nil
	}
	if err := json.Unmarshal(lang.EncodeJSON(o.Value("schemas")), &g.Schemas); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(lang.EncodeJSON(o.Value("limits")), &g.limits); err != nil {
		return nil, err
	}
	if types, ok := o.Value("types").(*lang.Object); ok {
		for _, k := range types.Keys() {
			if t, ok := types.Value(k).(*lang.Object); ok {
				g.Types[k], _ = t.Value("type").(string)
			}
		}
	}
	if g.limits.MaxDepth == 0 {
		g.limits.MaxDepth = 8
	}
	if g.limits.MaxNodes == 0 {
		g.limits.MaxNodes = 200
	}
	return g, nil
}

// ValidateElement checks Genui(spec) and the specialized components'
// options. It returns the problems (empty when valid).
func (g *Genui) ValidateElement(el *lang.Element) []string {
	if el.Type == "Genui" {
		spec := el.Prop("spec")
		if spec == nil {
			return nil
		}
		return g.ValidateSpec(spec)
	}
	typ, ok := g.Types[el.Type]
	if !ok {
		return nil
	}
	opts, ok := el.Prop("options").(*lang.Object)
	if !ok {
		return nil
	}
	node := opts.Clone()
	node.Set("type", typ)
	return g.ValidateSpec(lang.ObjectFrom("items", []lang.Value{node}))
}

// ValidateOptions validates the options object of a specialized component
// type such as "plot".
func (g *Genui) ValidateOptions(typ string, options *lang.Object) []string {
	node := options.Clone()
	node.Set("type", typ)
	return g.ValidateSpec(lang.ObjectFrom("items", []lang.Value{node}))
}

func isObj(v lang.Value) (*lang.Object, bool) {
	o, ok := v.(*lang.Object)
	return o, ok
}

// inferType is the viewer's `B2e`.
func inferType(o *lang.Object) string {
	if t, ok := o.Value("type").(string); ok {
		return t
	}
	if o.Has("type") {
		return ""
	}
	var cands []string
	arr := func(k string) bool { _, ok := o.Value(k).([]lang.Value); return ok }
	if arr("columns") && arr("rows") {
		cands = append(cands, "table")
	}
	for _, p := range [][2]string{{"pairs", "keyvalue"}, {"steps", "steps"}, {"diffs", "diff"}, {"meshes", "scene3d"}, {"tabs", "tabs"}} {
		if arr(p[0]) {
			cands = append(cands, p[1])
		}
	}
	if len(cands) == 1 {
		return cands[0]
	}
	return ""
}

// ValidateSpec validates a declarative genui spec ({items:[...]} or a
// single component node).
func (g *Genui) ValidateSpec(spec lang.Value) []string {
	if s, ok := spec.(string); ok {
		if v, err := lang.ParseJSON([]byte(s)); err == nil {
			spec = v
		}
	}
	if arr, ok := spec.([]lang.Value); ok && len(arr) > 0 {
		spec = lang.ObjectFrom("items", arr)
	}
	root, ok := isObj(spec)
	if !ok {
		return []string{"spec root must be an object"}
	}
	if _, isNode := root.Value("type").(string); isNode {
		if _, hasItems := root.Value("items").([]lang.Value); !hasItems || g.Schemas[str(root.Value("type"))] != nil {
			root = lang.ObjectFrom("items", []lang.Value{root})
		}
	}
	items, ok := root.Value("items").([]lang.Value)
	if !ok {
		return []string{"spec.items must be an array"}
	}
	v := &genuiValidator{g: g}
	for i, it := range items {
		v.node(it, 0, "items["+itoa(i)+"]")
		if v.stopped {
			break
		}
	}
	return dedupe(v.errs)
}

func str(v lang.Value) string { s, _ := v.(string); return s }

func itoa(i int) string { return lang.NumberToString(float64(i)) }

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

type genuiValidator struct {
	g       *Genui
	errs    []string
	count   int
	stopped bool
}

func (v *genuiValidator) push(s string) { v.errs = append(v.errs, s) }

func (v *genuiValidator) node(val lang.Value, depth int, path string) {
	if v.stopped {
		return
	}
	if v.count >= v.g.limits.MaxNodes {
		v.push("spec exceeds " + itoa(v.g.limits.MaxNodes) + " nodes; tail elided")
		v.stopped = true
		return
	}
	v.count++
	if depth > v.g.limits.MaxDepth {
		v.push(path + ": exceeds max depth " + itoa(v.g.limits.MaxDepth))
		return
	}
	o, ok := isObj(val)
	if !ok {
		v.push(path + ": must be an object")
		return
	}
	typ := inferType(o)
	if typ == "" {
		v.push(path + ": missing string 'type'")
		return
	}
	s := v.g.Schemas[typ]
	if s == nil {
		v.push(path + ": unsupported component type '" + typ + "' (no renderer registered)")
		return
	}
	n := v.normalize(o, typ, s)
	v.check(n, typ, s, path)
	for _, c := range v.children(n, typ, path) {
		v.node(c.value, depth+1, c.path)
	}
}

// normalize applies aliases and value aliases (viewer `b6t`, `ZJ`, `HJ`).
func (v *genuiValidator) normalize(o *lang.Object, typ string, s *genuiSchema) *lang.Object {
	n := o.Clone()
	n.Set("type", typ)
	for _, alias := range sortedKeys(s.Aliases) {
		canon := s.Aliases[alias]
		if !n.Has(alias) {
			continue
		}
		if !n.Has(canon) {
			n.Set(canon, n.Value(alias))
		}
		n.Delete(alias)
	}
	for field, m := range s.ValueAliases {
		if cur, ok := n.Value(field).(string); ok {
			if to, ok := m[cur]; ok {
				n.Set(field, to)
			}
		}
	}
	for field, ns := range s.Nested {
		if x := n.Value(field); x != nil {
			n.Set(field, nestedAliases(x, ns))
		}
	}
	switch typ {
	case "file-tree":
		n.Set("items", fileTreeDirs(n.Value("items")))
	case "keyvalue":
		if pairs, ok := n.Value("pairs").([]lang.Value); ok && len(pairs) > 0 {
			allArr := true
			for _, p := range pairs {
				if _, ok := p.([]lang.Value); !ok {
					allArr = false
				}
			}
			if allArr {
				out := make([]lang.Value, len(pairs))
				for i, p := range pairs {
					a := p.([]lang.Value)
					var val lang.Value = ""
					if len(a) > 1 {
						val = a[1]
					}
					var key lang.Value
					if len(a) > 0 {
						key = a[0]
					}
					out[i] = lang.ObjectFrom("key", key, "value", val)
				}
				n.Set("pairs", out)
			}
		}
	case "tabs":
		if tabs, ok := n.Value("tabs").([]lang.Value); ok {
			out := make([]lang.Value, len(tabs))
			for i, t := range tabs {
				to, ok := isObj(t)
				if !ok {
					out[i] = t
					continue
				}
				c := to.Clone()
				if c.Has("content") {
					if !c.Has("items") {
						c.Set("items", c.Value("content"))
					}
					c.Delete("content")
				}
				switch it := c.Value("items").(type) {
				case []lang.Value:
				case nil:
					c.Set("items", []lang.Value{})
				default:
					c.Set("items", []lang.Value{it})
				}
				out[i] = c
			}
			n.Set("tabs", out)
		}
	}
	return n
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func nestedAliases(x lang.Value, s *genuiSchema) lang.Value {
	if arr, ok := x.([]lang.Value); ok {
		out := make([]lang.Value, len(arr))
		for i, e := range arr {
			out[i] = nestedAliases(e, s)
		}
		return out
	}
	o, ok := isObj(x)
	if !ok {
		return x
	}
	c := o.Clone()
	for _, alias := range sortedKeys(s.Aliases) {
		canon := s.Aliases[alias]
		if c.Has(alias) && !c.Has(canon) {
			c.Set(canon, c.Value(alias))
			c.Delete(alias)
		}
	}
	for field, ns := range s.Nested {
		if v := c.Value(field); v != nil {
			c.Set(field, nestedAliases(v, ns))
		}
	}
	return c
}

func fileTreeDirs(x lang.Value) lang.Value {
	if arr, ok := x.([]lang.Value); ok {
		out := make([]lang.Value, len(arr))
		for i, e := range arr {
			out[i] = fileTreeDirs(e)
		}
		return out
	}
	o, ok := isObj(x)
	if !ok {
		return x
	}
	c := o.Clone()
	if _, isArr := c.Value("children").([]lang.Value); isArr && !c.Has("type") {
		c.Set("type", "dir")
	}
	if ch := c.Value("children"); ch != nil {
		c.Set("children", fileTreeDirs(ch))
	}
	return c
}

func typeOK(v lang.Value, t string) bool {
	switch t {
	case "string":
		_, ok := v.(string)
		return ok
	case "string-or-null":
		_, ok := v.(string)
		return ok || v == nil
	case "number":
		f, ok := v.(float64)
		return ok && !math.IsNaN(f) && !math.IsInf(f, 0)
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "nodes", "array":
		_, ok := v.([]lang.Value)
		return ok
	case "object":
		_, ok := v.(*lang.Object)
		return ok
	}
	return true
}

func typeDesc(t string) string {
	switch t {
	case "string-or-null":
		return "a string or null"
	case "number":
		return "a finite number"
	case "boolean":
		return "a boolean"
	case "nodes", "array":
		return "an array"
	case "object":
		return "an object"
	case "unknown":
		return "a value"
	}
	return "a string"
}

// check is the viewer's per-node schema validation (`w6t`).
func (v *genuiValidator) check(n *lang.Object, typ string, s *genuiSchema, path string) {
	already := func(field string) bool {
		for _, e := range v.errs {
			if strings.HasPrefix(e, path+"."+field+" ") || strings.Contains(e, "'"+field+"'") || strings.Contains(e, "requires "+field) {
				return true
			}
		}
		return false
	}
	skipColumns := false
	if typ == "table" {
		rows := n.Value("rows")
		if rows == nil {
			rows = n.Value("data")
		}
		if arr, ok := rows.([]lang.Value); ok && len(arr) > 0 {
			_, skipColumns = arr[0].([]lang.Value)
		}
	}
	for _, f := range s.Required {
		ft, known := s.Fields[f]
		val, present := n.Get(f)
		if !known || !present {
			if !(skipColumns && f == "columns") && !already(f) {
				msg := path + ": type '" + typ + "' requires " + f
				if known {
					msg += " (" + typeDesc(ft) + ")"
				}
				v.push(msg)
			}
			continue
		}
		if !typeOK(val, ft) && !already(f) {
			v.push(path + "." + f + " must be " + typeDesc(ft))
		}
	}
	for _, f := range sortedKeys(s.Fields) {
		if f == "type" {
			continue
		}
		val, present := n.Get(f)
		if !present {
			continue
		}
		if !typeOK(val, s.Fields[f]) && !already(f) {
			v.push(path + "." + f + " must be " + typeDesc(s.Fields[f]))
		}
	}
	for _, f := range sortedKeys(s.Enums) {
		sv, ok := n.Value(f).(string)
		if !ok {
			continue
		}
		found := false
		for _, e := range s.Enums[f] {
			if e == sv {
				found = true
			}
		}
		if !found && !already(f) {
			v.push(path + "." + f + " must be one of " + strings.Join(s.Enums[f], ", "))
		}
	}
	for _, group := range s.OneOfRequired {
		any := false
		for _, f := range group {
			if n.Has(f) {
				any = true
			}
		}
		if !any {
			v.push(path + ": type '" + typ + "' requires one of " + strings.Join(group, " or "))
		}
	}
	for _, c := range s.ConditionalRequired {
		if lang.StrictEqual(n.Value(c.When.Field), lang.FromGo(c.When.Equals)) {
			for _, f := range c.Required {
				if !n.Has(f) && !already(f) {
					msg := path + ": type '" + typ + "' requires " + f
					if c.Message != "" {
						msg += " (" + c.Message + ")"
					}
					v.push(msg)
				}
			}
		}
	}
	for _, f := range sortedKeys(s.Nested) {
		val := n.Value(f)
		if val == nil {
			continue
		}
		if arr, ok := val.([]lang.Value); ok {
			for i, e := range arr {
				v.record(e, path+"."+f+"["+itoa(i)+"]", s.Nested[f])
			}
		} else {
			v.record(val, path+"."+f, s.Nested[f])
		}
	}
}

// record validates a nested record (viewer `Kq`).
func (v *genuiValidator) record(val lang.Value, path string, s *genuiSchema) {
	o, ok := isObj(val)
	if !ok {
		v.push(path + " must be an object")
		return
	}
	for _, f := range s.Required {
		ft, known := s.Fields[f]
		x, present := o.Get(f)
		if !present {
			msg := path + ": requires " + f
			if known {
				msg += " (" + typeDesc(ft) + ")"
			}
			v.push(msg)
		} else if known && !typeOK(x, ft) {
			v.push(path + "." + f + " must be " + typeDesc(ft))
		}
	}
	for _, f := range sortedKeys(s.Fields) {
		if containsStr(s.Required, f) {
			continue
		}
		if x, present := o.Get(f); present && !typeOK(x, s.Fields[f]) {
			v.push(path + "." + f + " must be " + typeDesc(s.Fields[f]))
		}
	}
	for _, f := range sortedKeys(s.Enums) {
		if sv, ok := o.Value(f).(string); ok && !containsStr(s.Enums[f], sv) {
			v.push(path + "." + f + " must be one of " + strings.Join(s.Enums[f], ", "))
		}
	}
	for _, f := range sortedKeys(s.Nested) {
		x := o.Value(f)
		if x == nil {
			continue
		}
		if arr, ok := x.([]lang.Value); ok {
			for i, e := range arr {
				v.record(e, path+"."+f+"["+itoa(i)+"]", s.Nested[f])
			}
		} else {
			v.record(x, path+"."+f, s.Nested[f])
		}
	}
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type child struct {
	value lang.Value
	path  string
}

// children lists nested component nodes (viewer `wa`).
func (v *genuiValidator) children(n *lang.Object, typ, path string) []child {
	var out []child
	add := func(arr lang.Value, key string, onlyNodes bool) {
		a, ok := arr.([]lang.Value)
		if !ok {
			return
		}
		for i, d := range a {
			if onlyNodes {
				o, ok := isObj(d)
				if !ok || !o.Has("type") {
					continue
				}
			}
			out = append(out, child{d, path + "." + key + "[" + itoa(i) + "]"})
		}
	}
	family := func(arr lang.Value, key string, cap int) {
		a, ok := arr.([]lang.Value)
		if !ok {
			return
		}
		if cap > 0 && len(a) > cap {
			a = a[:cap]
		}
		for i, d := range a {
			if o, ok := isObj(d); ok {
				add(o.Value("items"), key+"["+itoa(i)+"].items", false)
			}
		}
	}
	switch typ {
	case "row", "col", "grid", "card":
		add(n.Value("items"), "items", false)
	case "list":
		add(n.Value("items"), "items", true)
	case "tabs":
		family(n.Value("tabs"), "tabs", v.g.limits.MaxTabs)
	case "accordion":
		family(n.Value("items"), "items", v.g.limits.MaxAccordionItems)
	}
	return out
}
