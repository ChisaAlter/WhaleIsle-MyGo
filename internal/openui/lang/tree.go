package lang

import "fmt"

type stateDep struct {
	name string
	ver  uint64
}

type renderMemo struct {
	owner          *Instance
	form           string
	out            *Element
	stateDeps      []stateDep
	usesRef        bool
	queryVer       uint64
	errors         []Diagnostic
	elCount        int
	evalSteps      int
	maxRenderDepth int
	maxEvalDepth   int
}

// renderer evaluates a materialized tree into a renderer-neutral tree.
type renderer struct {
	in       *Instance
	store    *Store
	env      *evalEnv
	lib      *Library
	errors   []Diagnostic
	budget   *budget
	limits   *Limits
	elCount  int
	curEl    *Element
	curKey   string
	vctx     vctx
	useMemo  bool
	queryVer uint64
	depLog   []string
	depSeen  map[string]struct{}
	usesRef  bool
	impure   bool
}

func (r *renderer) getStateTracked(name string) Value {
	if r.useMemo {
		r.depLog = append(r.depLog, name)
	}
	return r.in.getState(name)
}

func (r *renderer) resolveRefTracked(name string) Value {
	if r.useMemo {
		r.usesRef = true
	}
	return r.in.resolveRef(name)
}

func (r *renderer) markImpure() {
	r.impure = true
}

func (r *renderer) memoValid(m *renderMemo) bool {
	if m.usesRef && m.queryVer != r.queryVer {
		return false
	}
	if r.limits != nil && r.limits.MaxRenderNodes > 0 && r.elCount+m.elCount > r.limits.MaxRenderNodes {
		return false
	}
	if r.budget.maxCount > 0 && r.budget.count+m.elCount > r.budget.maxCount {
		return false
	}
	if r.budget.maxDepth > 0 && r.budget.depth+m.maxRenderDepth > r.budget.maxDepth {
		return false
	}
	if r.env.budget.maxCount > 0 && r.env.budget.count+m.evalSteps > r.env.budget.maxCount {
		return false
	}
	if r.env.budget.maxDepth > 0 && r.env.budget.depth+m.maxEvalDepth > r.env.budget.maxDepth {
		return false
	}
	for _, d := range m.stateDeps {
		if r.store.keyVersion(d.name) != d.ver {
			return false
		}
	}
	return true
}

func (r *renderer) reportCurrent(msg string) {
	if r.curEl != nil {
		r.runtimeError(r.curEl, r.curKey, msg)
	}
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
	memoize := r.useMemo && el.kind == elMaterialized
	if memoize {
		if m := el.memo; m != nil && m.owner == r.in && m.form == form && r.memoValid(m) {
			r.elCount += m.elCount
			r.budget.count += m.elCount
			if d := r.budget.depth + m.maxRenderDepth; d > r.budget.peakDepth {
				r.budget.peakDepth = d
			}
			r.env.budget.count += m.evalSteps
			if d := r.env.budget.depth + m.maxEvalDepth; d > r.env.budget.peakDepth {
				r.env.budget.peakDepth = d
			}
			if len(m.errors) > 0 {
				r.errors = append(r.errors, m.errors...)
			}
			for _, d := range m.stateDeps {
				r.depLog = append(r.depLog, d.name)
			}
			if m.usesRef {
				r.usesRef = true
			}
			return m.out
		}
	}
	var (
		depMark          int
		errMark          int
		prevUsesRef      bool
		prevImpure       bool
		beforeElCount    int
		beforeEvalSteps  int
		entryRenderDepth int
		prevRenderPeak   int
		entryEvalDepth   int
		prevEvalPeak     int
	)
	if memoize {
		depMark = len(r.depLog)
		errMark = len(r.errors)
		prevUsesRef, r.usesRef = r.usesRef, false
		prevImpure, r.impure = r.impure, false
		beforeElCount = r.elCount
		beforeEvalSteps = r.env.budget.count
		entryRenderDepth = r.budget.depth
		prevRenderPeak = r.budget.peakDepth
		r.budget.peakDepth = entryRenderDepth
		entryEvalDepth = r.env.budget.depth
		prevEvalPeak = r.env.budget.peakDepth
		r.env.budget.peakDepth = entryEvalDepth
	}
	r.budget.enter(el.Type)
	r.elCount++
	if r.limits != nil && r.elCount > r.limits.MaxRenderNodes {
		r.budget.leave()
		panic(&budgetError{phase: "render nodes", limit: r.limits.MaxRenderNodes})
	}
	inPlace := el.kind == elEphemeral
	var out *Element
	if inPlace {
		el.kind = elConsumed
		el.FormName = form
		out = el
	} else {
		out = &Element{Type: el.Type, StatementID: el.StatementID, SourceKey: el.SourceKey, InstanceKey: el.InstanceKey, Partial: el.Partial, FormName: form}
	}
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
	if r.env.report == nil {
		r.env.report = r.reportCurrent
	}
	prevEl, prevKey := r.curEl, r.curKey
	if inPlace {
		if el.Props != nil {
			write := 0
			for i, key := range el.Props.keys {
				val := el.Props.vals[i]
				switch val.(type) {
				case nil, bool, float64, string, *ActionPlan:
					if write != i {
						el.Props.keys[write] = key
						el.Props.vals[write] = val
					}
					write++
					continue
				}
				var p *Param
				if comp != nil {
					p = paramByName(comp, key)
				}
				bindable := p != nil && p.Bindable
				r.curEl, r.curKey = el, key
				v, err := r.safeProp(val, bindable, childForm)
				if err != nil {
					r.runtimeError(el, key, err.Error())
					continue
				}
				el.Props.keys[write] = key
				el.Props.vals[write] = v
				write++
			}
			if write < len(el.Props.keys) {
				clear(el.Props.keys[write:])
				clear(el.Props.vals[write:])
				el.Props.keys = el.Props.keys[:write]
				el.Props.vals = el.Props.vals[:write]
				if el.Props.index != nil {
					el.Props.buildIndex()
				}
			}
		}
	} else {
		props := NewObject(el.Props.Len())
		for i, key := range el.Props.Keys() {
			val := el.Props.vals[i]
			// Fast path: plain primitives and action plans cannot panic or bind.
			switch val.(type) {
			case nil, bool, float64, string, *ActionPlan:
				props.appendOrdered(key, val)
				continue
			}
			var p *Param
			if comp != nil {
				p = paramByName(comp, key)
			}
			bindable := p != nil && p.Bindable
			r.curEl, r.curKey = el, key
			v, err := r.safeProp(val, bindable, childForm)
			if err != nil {
				r.runtimeError(el, key, err.Error())
				continue
			}
			props.appendOrdered(key, v)
		}
		out.Props = props
	}
	r.curEl, r.curKey = prevEl, prevKey
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
	r.budget.leave()
	if memoize {
		myUsesRef := r.usesRef
		myImpure := r.impure
		r.usesRef = prevUsesRef || myUsesRef
		r.impure = prevImpure || myImpure
		relRenderDepth := r.budget.peakDepth - entryRenderDepth
		if prevRenderPeak > r.budget.peakDepth {
			r.budget.peakDepth = prevRenderPeak
		}
		relEvalDepth := r.env.budget.peakDepth - entryEvalDepth
		if prevEvalPeak > r.env.budget.peakDepth {
			r.env.budget.peakDepth = prevEvalPeak
		}
		if !myImpure {
			m := el.memo
			if m == nil {
				m = &renderMemo{}
				el.memo = m
			}
			m.owner = r.in
			m.form = form
			m.out = out
			m.stateDeps = r.compactDepsInto(m.stateDeps[:0], depMark)
			m.usesRef = myUsesRef
			m.queryVer = r.queryVer
			if len(r.errors) > errMark {
				m.errors = append(m.errors[:0], r.errors[errMark:]...)
			} else {
				m.errors = m.errors[:0]
			}
			m.elCount = r.elCount - beforeElCount
			m.evalSteps = r.env.budget.count - beforeEvalSteps
			m.maxRenderDepth = relRenderDepth
			m.maxEvalDepth = relEvalDepth
		}
	}
	return out
}

// compactDepsInto deduplicates r.depLog[mark:] in place and fills dst with
// the current per-key store versions.
func (r *renderer) compactDepsInto(dst []stateDep, mark int) []stateDep {
	raw := r.depLog[mark:]
	if len(raw) == 0 {
		return dst[:0]
	}
	if len(raw) == 1 {
		name := raw[0]
		return append(dst[:0], stateDep{name: name, ver: r.store.keyVersion(name)})
	}
	write := mark
	if len(raw) <= 12 {
		for _, name := range raw {
			dup := false
			for j := mark; j < write; j++ {
				if r.depLog[j] == name {
					dup = true
					break
				}
			}
			if !dup {
				r.depLog[write] = name
				write++
			}
		}
	} else {
		if r.depSeen == nil {
			r.depSeen = make(map[string]struct{}, len(raw))
		} else {
			clear(r.depSeen)
		}
		for _, name := range raw {
			if _, ok := r.depSeen[name]; !ok {
				r.depSeen[name] = struct{}{}
				r.depLog[write] = name
				write++
			}
		}
	}
	r.depLog = r.depLog[:write]
	n := write - mark
	if cap(dst) < n {
		dst = make([]stateDep, n)
	} else {
		dst = dst[:n]
	}
	for i := 0; i < n; i++ {
		name := r.depLog[mark+i]
		dst[i] = stateDep{name: name, ver: r.store.keyVersion(name)}
	}
	return dst
}

// evalPlain evaluates a value for a structural read (form names).
func (r *renderer) evalPlain(v Value) Value {
	if n, ok := v.(Node); ok {
		return r.env.eval(n)
	}
	return v
}

func (r *renderer) safeProp(v Value, bindable bool, form string) (out Value, err error) {
	savedEvalDepth := r.env.budget.depth
	savedRenderDepth := r.budget.depth
	defer func() {
		if rec := recover(); rec != nil {
			if be, ok := rec.(*budgetError); ok {
				panic(be)
			}
			r.env.budget.depth = savedEvalDepth
			r.budget.depth = savedRenderDepth
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
		for i := range t.keys {
			switch t.vals[i].(type) {
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
		for i, k := range t.keys {
			o.appendOrdered(k, r.evalProp(t.vals[i], bindable, form))
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
	c := &r.vctx
	c.lib = r.lib
	c.source = SourceRuntime
	c.statement = el.StatementID
	c.errors = c.errors[:0]
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
			nullable := true
			if p.compiled != nil && p.compiled.raw == p.Schema {
				nullable = p.compiled.nullable
			} else if p.Schema != nil {
				nullable = schemaNullable(p.Schema)
			}
			if present && nullable {
				continue
			}
			if p.Required && !p.HasDefault {
				kind := "undefined"
				if present {
					kind = "null"
				}
				c.errors = append(c.errors, Diagnostic{Source: SourceRuntime, Code: "missing-required", Component: el.Type, Path: "/" + p.Name, StatementID: el.StatementID, Message: "Required prop " + p.Name + " evaluated to " + kind})
			}
			continue
		}
		var m Value
		if p.compiled != nil && p.compiled.raw == p.Schema {
			c.resetPath("")
			c.pathSegs = append(c.pathSegs, pathSeg{seg: p.Name, idx: -1})
			m = c.validateCompiled(h, p.compiled, el.Type)
		} else {
			m = c.validateSchemaValue(h, p.Schema, el.Type, "/"+p.Name)
		}
		if isInvalid(m) {
			g := resolveInvalid(p.Required, p.Default, p.HasDefault)
			if isInvalid(g) || isOmit(g) {
				el.Props.Delete(p.Name)
			} else {
				el.Props.Set(p.Name, g)
			}
		} else if !isBind && !isOmit(m) && !sameValueOrNode(m, h) {
			el.Props.Set(p.Name, m)
		}
	}
	if len(c.errors) > 0 {
		r.errors = append(r.errors, c.errors...)
	}
}
