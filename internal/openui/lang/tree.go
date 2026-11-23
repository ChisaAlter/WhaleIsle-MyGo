package lang

import "fmt"

// renderer evaluates a materialized tree into a renderer-neutral tree.
type renderer struct {
	env     *evalEnv
	lib     *Library
	errors  []Diagnostic
	budget  *budget
	limits  *Limits
	elCount int
}

func (r *renderer) runtimeError(el *Element, key, msg string) {
	r.errors = append(r.errors, Diagnostic{
		Source:      SourceRuntime,
		Code:        CodeRuntimeError,
		Component:   el.Type,
		StatementID: el.StatementID,
		Message:     fmt.Sprintf("Evaluating prop %q on %s failed: %s", key, el.Type, msg),
		Hint:        fmt.Sprintf("Check the expression used for prop %q", key),
	})
}

// renderRoot evaluates the entry. It returns nil when nothing renders.
func (r *renderer) renderRoot(res *ParseResult) *Element {
	if res.Root != nil {
		return r.renderElement(res.Root, "")
	}
	if res.RootExpr == nil {
		return nil
	}
	v := r.env.eval(res.RootExpr)
	if el, ok := v.(*Element); ok {
		return r.renderElement(el, "")
	}
	r.errors = append(r.errors, Diagnostic{Source: SourceRuntime, Code: "no-root", StatementID: "root", Message: "root does not evaluate to a component"})
	return nil
}

func (r *renderer) renderElement(el *Element, form string) *Element {
	r.budget.enter(el.Type)
	defer r.budget.leave()
	r.elCount++
	if r.limits != nil && r.elCount > r.limits.MaxRenderNodes {
		panic(&budgetError{phase: "render nodes", limit: r.limits.MaxRenderNodes})
	}
	out := &Element{Type: el.Type, StatementID: el.StatementID, SourceKey: el.SourceKey, InstanceKey: el.InstanceKey, Partial: el.Partial, FormName: form}
	comp := r.lib.Components[el.Type]
	childForm := form
	if r.lib.FormComponents != nil {
		if prop, ok := r.lib.FormComponents[el.Type]; ok {
			childForm = out.Identity()
			if prop != "" {
				if v, ok := r.evalPlain(el.Props.Value(prop)).(string); ok && v != "" {
					childForm = v
				}
			}
		}
	}
	props := NewObject(el.Props.Len())
	for _, key := range el.Props.Keys() {
		var p *Param
		if comp != nil {
			p = paramByName(comp, key)
		}
		bindable := p != nil && p.Bindable
		val := el.Props.vals[key]
		prev := r.env.report
		r.env.report = func(msg string) { r.runtimeError(el, key, msg) }
		v, err := r.safeProp(val, bindable, childForm)
		r.env.report = prev
		if err != nil {
			r.runtimeError(el, key, err.Error())
			continue
		}
		props.Set(key, v)
	}
	out.Props = props
	if r.lib.Profile.RuntimeValidation && comp != nil {
		r.validateRuntime(out, comp)
	}
	if r.lib.ValidateElement != nil {
		if issues := r.lib.ValidateElement(out); len(issues) > 0 {
			out.Issues = issues
			for _, msg := range issues {
				r.errors = append(r.errors, Diagnostic{Source: SourceRuntime, Code: CodeSchemaInvalid, Component: out.Type, StatementID: out.StatementID, Message: msg})
			}
		}
	}
	return out
}

// evalPlain evaluates a value for a structural read (form names).
func (r *renderer) evalPlain(v Value) Value {
	if n, ok := v.(Node); ok {
		return r.env.eval(n)
	}
	return v
}

func (r *renderer) safeProp(v Value, bindable bool, form string) (out Value, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if be, ok := rec.(*budgetError); ok {
				panic(be)
			}
			err = fmt.Errorf("%v", rec)
		}
	}()
	return r.evalProp(v, bindable, form), nil
}

func (r *renderer) binding(target string, expr Node) *Binding {
	return &Binding{Target: target, expr: expr, Value: r.env.getState(target)}
}

// evalProp is upstream evaluatePropValue (with Whale's binding handling).
func (r *renderer) evalProp(v Value, bindable bool, form string) Value {
	switch t := v.(type) {
	case nil, bool, float64, string:
		return v
	case *Binding:
		if bindable {
			return r.binding(t.Target, t.expr)
		}
		return r.env.getState(t.Target)
	case Node:
		if sr, ok := t.(*StateRefNode); ok && bindable {
			return r.binding(sr.N, &StateRefNode{N: "$value"})
		}
		res := r.env.eval(t)
		switch x := res.(type) {
		case *Element:
			return r.renderElement(x, form)
		case []Value:
			out := make([]Value, len(x))
			for i, it := range x {
				if el, ok := it.(*Element); ok {
					out[i] = r.renderElement(el, form)
				} else {
					out[i] = it
				}
			}
			return out
		case *Binding:
			if bindable {
				return r.binding(x.Target, x.expr)
			}
			return r.env.getState(x.Target)
		}
		return res
	case []Value:
		out := make([]Value, len(t))
		for i, x := range t {
			out[i] = r.evalProp(x, bindable, form)
		}
		return out
	case *Element:
		return r.renderElement(t, form)
	case *ActionPlan:
		return t
	case *Object:
		needs := false
		for _, k := range t.keys {
			switch t.vals[k].(type) {
			case nil, bool, float64, string:
			default:
				needs = true
			}
			if needs {
				break
			}
		}
		if !needs {
			return t
		}
		o := NewObject(t.Len())
		for _, k := range t.keys {
			o.Set(k, r.evalProp(t.vals[k], bindable, form))
		}
		return o
	}
	return v
}

func schemaNullable(s *Object) bool {
	if s == nil {
		return true
	}
	if s.Value("type") == "null" {
		return true
	}
	if ts, ok := s.Value("type").([]Value); ok {
		for _, t := range ts {
			if t == "null" {
				return true
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		if ms, ok := s.Value(key).([]Value); ok {
			for _, m := range ms {
				mo, _ := m.(*Object)
				if schemaNullable(mo) {
					return true
				}
			}
			return false
		}
	}
	return !s.Has("type") && !s.Has("$ref") && !s.Has("enum") && !s.Has("const")
}

// validateRuntime checks evaluated props (Whale render-time validation).
func (r *renderer) validateRuntime(el *Element, comp *Component) {
	c := &vctx{lib: r.lib, source: SourceRuntime, statement: el.StatementID}
	for _, p := range comp.Params {
		f, present := el.Props.Get(p.Name)
		if _, isPlan := f.(*ActionPlan); isPlan {
			continue
		}
		h := f
		b, isBind := f.(*Binding)
		if isBind && p.Bindable {
			h = b.Value
			if h == nil {
				continue
			}
		}
		if h == nil {
			if present && schemaNullable(p.Schema) {
				continue
			}
			if p.Required && !p.HasDefault {
				c.errors = append(c.errors, Diagnostic{Source: SourceRuntime, Code: "missing-required", Component: el.Type, Path: "/" + p.Name, StatementID: el.StatementID, Message: "Required prop " + p.Name + " evaluated to " + map[bool]string{true: "null", false: "undefined"}[present]})
			}
			continue
		}
		m := c.validateSchemaValue(h, p.Schema, el.Type, "/"+p.Name)
		if isInvalid(m) {
			g := resolveInvalid(p.Required, p.Default, p.HasDefault)
			if isInvalid(g) || isOmit(g) {
				el.Props.Delete(p.Name)
			} else {
				el.Props.Set(p.Name, g)
			}
		} else if !isBind && !isOmit(m) {
			el.Props.Set(p.Name, m)
		}
	}
	r.errors = append(r.errors, c.errors...)
}
