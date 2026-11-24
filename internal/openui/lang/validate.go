package lang

import (
	"strings"
)

// invalidMarker is the INVALID sentinel of upstream validation.ts.
type invalidMarker struct{}

var invalid Value = &invalidMarker{}

func isInvalid(v Value) bool { _, ok := v.(*invalidMarker); return ok }

// vctx collects validation errors (upstream MaterializeCtx subset).
type vctx struct {
	lib       *Library
	errors    []Diagnostic
	partial   bool
	statement string
	source    string
}

func (c *vctx) push(component, path, code, message string) {
	src := c.source
	if src == "" {
		src = SourceParser
	}
	c.errors = append(c.errors, Diagnostic{Source: src, Code: code, Component: component, Path: path, Message: message, StatementID: c.statement})
}

func (c *vctx) typeMismatch(component, path, expected, actual string) {
	c.push(component, path, "type-mismatch", `field "`+path+`" expects `+expected+` but got `+actual)
}

func schemaRefName(s *Object) string {
	ref, ok := s.Value("$ref").(string)
	if !ok {
		return ""
	}
	if i := strings.LastIndexByte(ref, '/'); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

func isCompositeSchema(s *Object) bool {
	return s.Has("$ref") || s.Has("anyOf") || s.Has("oneOf") || s.Has("allOf")
}

// scalarInfo returns the scalar leaf type ("string", "number", "boolean")
// or enum values of a schema.
func scalarInfo(s *Object) (string, []Value, bool) {
	if e, ok := s.Value("enum").([]Value); ok {
		return "", e, true
	}
	if c, ok := s.Get("const"); ok {
		return "", []Value{c}, true
	}
	switch s.Value("type") {
	case "string":
		return "string", nil, false
	case "number", "integer":
		return "number", nil, false
	case "boolean":
		return "boolean", nil, false
	}
	return "", nil, false
}

// schemaTypeName renders a short type for signatures and hints.
func schemaTypeName(s *Object) string {
	if s == nil {
		return ""
	}
	if r := schemaRefName(s); r != "" {
		return r
	}
	t, enum, isEnum := scalarInfo(s)
	if isEnum {
		parts := make([]string, len(enum))
		for i, v := range enum {
			parts[i] = string(EncodeJSON(v))
		}
		return strings.Join(parts, "|")
	}
	if t != "" {
		return t
	}
	if ts, ok := s.Value("type").(string); ok {
		return ts
	}
	return ""
}

func schemaRequired(s *Object) []string {
	arr, _ := s.Value("required").([]Value)
	out := make([]string, 0, len(arr))
	for _, r := range arr {
		if k, ok := r.(string); ok {
			out = append(out, k)
		}
	}
	return out
}

func schemaDefault(s Value) (Value, bool) {
	o, ok := s.(*Object)
	if !ok {
		return nil, false
	}
	return o.Get("default")
}

// resolveInvalid applies the edge rule: default, else INVALID when required,
// else omit (returned as omitValue).
func resolveInvalid(required bool, def Value, hasDef bool) Value {
	if hasDef {
		return def
	}
	if required {
		return invalid
	}
	return omit
}

// omit marks a value to remove (upstream `undefined`).
type omitMarker struct{}

var omit Value = &omitMarker{}

func isOmit(v Value) bool { _, ok := v.(*omitMarker); return ok }

func describeValueKind(v Value) string {
	switch v.(type) {
	case []Value:
		return "array"
	case *Object:
		return "plain object"
	}
	return typeofName(v)
}

// validateSchemaValue checks a materialized value against a JSON Schema
// fragment (upstream validation.ts). It returns the value, a cleaned copy,
// or invalid. Nodes (runtime expressions) and null are not checked.
func (c *vctx) validateSchemaValue(v Value, s *Object, component, path string) Value {
	if s == nil || v == nil {
		return v
	}
	if _, ok := v.(Node); ok {
		return v
	}
	if el, ok := v.(*Element); ok {
		if isCompositeSchema(s) {
			return v
		}
		_, isType := s.Value("type").(string)
		_, hasEnum := s.Value("enum").([]Value)
		if isType || hasEnum || s.Has("const") {
			exp := "a literal value"
			if t, ok := s.Value("type").(string); ok {
				exp = t
			}
			c.typeMismatch(component, path, exp, `component "`+el.Type+`"`)
			return invalid
		}
		return v
	}
	if _, ok := v.(*ActionPlan); ok {
		return v
	}
	strict := c.lib != nil && c.lib.Profile.StrictUnions
	if strict {
		if types, ok := s.Value("type").([]Value); ok {
			actual := jsTypeName(v)
			okType := false
			for _, t := range types {
				ts, _ := t.(string)
				if ts == actual || ts == "integer" && actual == "number" && isInteger(v) {
					okType = true
				}
			}
			if !okType {
				names := make([]string, 0, len(types))
				for _, t := range types {
					names = append(names, ToString(t))
				}
				c.typeMismatch(component, path, strings.Join(names, " | "), actual)
				return invalid
			}
			return v
		}
		members, _ := s.Value("anyOf").([]Value)
		if members == nil {
			members, _ = s.Value("oneOf").([]Value)
		}
		if members != nil {
			var names []string
			for _, m := range members {
				ms, _ := m.(*Object)
				sub := &vctx{lib: c.lib, partial: c.partial, statement: c.statement, source: c.source}
				out := sub.validateSchemaValue(v, ms, component, path)
				if !isInvalid(out) && len(sub.errors) == 0 {
					return out
				}
				if n := schemaTypeName(ms); n != "" {
					names = append(names, n)
				}
			}
			exp := strings.Join(names, " | ")
			if exp == "" {
				exp = "an allowed union member"
			}
			c.typeMismatch(component, path, exp, jsTypeName(v))
			return invalid
		}
		if all, ok := s.Value("allOf").([]Value); ok {
			cur := v
			for _, m := range all {
				ms, _ := m.(*Object)
				cur = c.validateSchemaValue(cur, ms, component, path)
				if isInvalid(cur) {
					break
				}
			}
			return cur
		}
	}
	if isCompositeSchema(s) {
		if !c.isOnlyComponentSlot(s) {
			return v
		}
		c.typeMismatch(component, path, strings.Join(slotRefNames(s), " | "), describeValueKind(v))
		return invalid
	}
	switch s.Value("type") {
	case "object":
		return c.validateObject(v, s, component, path)
	case "array":
		return c.validateArray(v, s, component, path)
	}
	if c.validateLeaf(v, s, component, path) {
		return invalid
	}
	return v
}

func isInteger(v Value) bool {
	f, ok := v.(float64)
	return ok && f == float64(int64(f))
}

func slotRefNames(s *Object) []string {
	members, _ := s.Value("anyOf").([]Value)
	if members == nil {
		members, _ = s.Value("oneOf").([]Value)
	}
	if members == nil {
		return []string{schemaRefName(s)}
	}
	out := make([]string, 0, len(members))
	for _, m := range members {
		ms, _ := m.(*Object)
		out = append(out, schemaRefName(ms))
	}
	return out
}

func (c *vctx) isOnlyComponentSlot(s *Object) bool {
	names := slotRefNames(s)
	if len(names) == 0 || c.lib == nil {
		return false
	}
	for _, n := range names {
		if n == "" || c.lib.Components[n] == nil {
			return false
		}
	}
	return true
}

func (c *vctx) validateObject(v Value, s *Object, component, path string) Value {
	obj, ok := v.(*Object)
	if !ok {
		c.typeMismatch(component, path, "object", jsTypeName(v))
		return invalid
	}
	props, _ := s.Value("properties").(*Object)
	required := schemaRequired(s)
	var out *Object
	bad := false
	if !c.partial {
		for _, key := range required {
			cur, present := obj.Get(key)
			if !present || cur == nil {
				if d, ok := schemaDefault(props.Value(key)); ok {
					if out == nil {
						out = obj.Clone()
					}
					out.Set(key, d)
					continue
				}
				code := "missing-required"
				msg := `missing required field "` + path + "/" + key + `"`
				if present {
					code = "null-required"
					msg = `required field "` + path + "/" + key + `" cannot be null`
				}
				c.push(component, path+"/"+key, code, msg)
				bad = true
			}
		}
	}
	for _, key := range props.Keys() {
		cur, present := obj.Get(key)
		if !present {
			continue
		}
		sub, _ := props.Value(key).(*Object)
		next := c.validateSchemaValue(cur, sub, component, path+"/"+key)
		if isInvalid(next) {
			d, hasD := schemaDefault(sub)
			next = resolveInvalid(containsStr(required, key), d, hasD)
			if isInvalid(next) {
				bad = true
				continue
			}
		}
		if isOmit(next) {
			if out == nil {
				out = obj.Clone()
			}
			out.Delete(key)
			continue
		}
		if sameValueOrNode(next, cur) {
			continue
		}
		if out == nil {
			out = obj.Clone()
		}
		out.Set(key, next)
	}
	if bad {
		return invalid
	}
	if out != nil {
		return out
	}
	return obj
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (c *vctx) validateArray(v Value, s *Object, component, path string) Value {
	arr, ok := v.([]Value)
	if !ok {
		c.typeMismatch(component, path, "array", jsTypeName(v))
		return invalid
	}
	items, ok := s.Value("items").(*Object)
	if !ok {
		return v
	}
	var out []Value
	changed := false
	for i, x := range arr {
		next := c.validateSchemaValue(x, items, component, path+"/"+itoa(i))
		if !changed && !sameValueOrNode(next, x) {
			changed = true
			out = append(make([]Value, 0, len(arr)), arr[:i]...)
		}
		if changed && !isInvalid(next) && !isOmit(next) {
			out = append(out, next)
		}
	}
	if changed {
		return out
	}
	return arr
}

func (c *vctx) validateLeaf(v Value, s *Object, component, path string) bool {
	typ, enum, isEnum := scalarInfo(s)
	if typ == "" && !isEnum {
		return false
	}
	if isEnum && c.partial {
		return false
	}
	actual := typeofName(v)
	if isEnum {
		if actual != "string" && actual != "number" && actual != "boolean" {
			return false
		}
		for _, e := range enum {
			if StrictEqual(e, v) {
				return false
			}
		}
		parts := make([]string, len(enum))
		for i, e := range enum {
			parts[i] = string(EncodeJSON(e))
		}
		c.typeMismatch(component, path, "one of ["+strings.Join(parts, ", ")+"]", string(EncodeJSON(v)))
		return true
	}
	if typ != actual {
		c.typeMismatch(component, path, typ, jsTypeName(v))
		return true
	}
	return false
}
