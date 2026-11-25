package lang

import (
	"errors"
	"fmt"
	"math"
)

// evalEnv is the evaluation context (upstream EvaluationContext).
type evalEnv struct {
	lib        *Library
	getState   func(name string) Value
	resolveRef func(name string) Value
	scope      *scopeVar
	extra      map[string]Value
	report     func(msg string)
	budget     *budget
	fenv       *FuncEnv
	limits     *Limits
	onImpure   func()
	vctx       vctx
	argPool    [4]Object
	argDepth   int
}

type scopeVar struct {
	name  string
	value Value
	next  *scopeVar
}

func (e *evalEnv) lookup(name string) (Value, bool) {
	for s := e.scope; s != nil; s = s.next {
		if s.name == name {
			return s.value, true
		}
	}
	return nil, false
}

func (e *evalEnv) with(name string, v Value) *evalEnv {
	c := *e
	c.scope = &scopeVar{name: name, value: v, next: e.scope}
	return &c
}

func (e *evalEnv) reportf(format string, args ...any) {
	if e.report != nil {
		e.report(fmt.Sprintf(format, args...))
	}
}

// ActionPlan is an evaluated action: one or more steps whose arguments are
// evaluated when the action runs (Whale runtime semantics). Hosts pass the
// plan back to Instance.Dispatch.
type ActionPlan struct {
	steps []planStep
	step0 [1]planStep
}

func singleStepPlan(c *CallNode) *ActionPlan {
	p := &ActionPlan{}
	p.step0[0] = planStep{call: c}
	p.steps = p.step0[:1]
	return p
}

// Len returns the number of steps.
func (p *ActionPlan) Len() int {
	if p == nil {
		return 0
	}
	return len(p.steps)
}

type planStep struct {
	call *CallNode // deferred call (built-in or custom step)
	data *Object   // a step read from data ({type: "open_url", url})
}

func (e *evalEnv) eval(n Node) Value {
	e.budget.enter(nodeKindName(n))
	var res Value
	switch t := n.(type) {
	case *StrNode:
		res = t.V
	case *NumNode:
		res = t.V
	case *BoolNode:
		res = t.V
	case *NullNode, *PhNode:
		res = nil
	case *ValueNode:
		res = t.V
	case *StateRefNode:
		if e.extra != nil {
			if v, ok := e.extra[t.N]; ok {
				e.budget.leave()
				return v
			}
		}
		res = e.getState(t.N)
	case *RefNode:
		if v, ok := e.lookup(t.N); ok {
			e.budget.leave()
			return v
		}
		if e.resolveRef != nil {
			res = e.resolveRef(t.N)
		}
	case *RuntimeRefNode:
		if e.resolveRef != nil {
			res = e.resolveRef(t.N)
		}
	case *ArrNode:
		out := make([]Value, len(t.Els))
		for i, x := range t.Els {
			out[i] = e.eval(x)
		}
		res = out
	case *ObjNode:
		o := NewObject(len(t.Entries))
		for _, x := range t.Entries {
			o.Set(x.Key, e.eval(x.Value))
		}
		res = o
	case *CallNode:
		res = e.evalCall(t)
	case *BinOpNode:
		res = e.evalBinOp(t)
	case *UnaryNode:
		x := e.eval(t.X)
		if t.Op == "!" {
			res = !Truthy(x)
		} else {
			res = -ToNumber(x)
		}
	case *TernaryNode:
		if Truthy(e.eval(t.Cond)) {
			res = e.eval(t.Then)
		} else {
			res = e.eval(t.Else)
		}
	case *MemberNode:
		x := e.eval(t.X)
		if x == nil {
			res = nil
		} else if arr, ok := x.([]Value); ok {
			if t.Field == "length" {
				res = float64(len(arr))
			} else {
				out := make([]Value, len(arr))
				for i, it := range arr {
					out[i] = memberOf(it, t.Field)
				}
				res = out
			}
		} else {
			res = memberOf(x, t.Field)
		}
	case *IndexNode:
		x := e.eval(t.X)
		idx := e.eval(t.Index)
		if x == nil || idx == nil {
			res = nil
		} else if arr, ok := x.([]Value); ok {
			f := ToNumber(idx)
			if f >= 0 && f == math.Trunc(f) && f < float64(len(arr)) {
				res = arr[int(f)]
			}
		} else {
			res = memberOf(x, ToString(idx))
		}
	case *AssignNode:
		expr := t.Value
		if e.scope != nil {
			expr = captureScope(expr, e.scope)
		}
		res = &Binding{Target: t.Target, expr: expr}
	}
	e.budget.leave()
	return res
}

func nodeKindName(n Node) string {
	switch n.(type) {
	case *CallNode:
		return "Comp"
	case *BinOpNode:
		return "BinOp"
	case *MemberNode:
		return "Member"
	}
	return "expr"
}

func (e *evalEnv) evalBinOp(t *BinOpNode) Value {
	switch t.Op {
	case "&&":
		l := e.eval(t.L)
		if !Truthy(l) {
			return l
		}
		return e.eval(t.R)
	case "||":
		l := e.eval(t.L)
		if Truthy(l) {
			return l
		}
		return e.eval(t.R)
	}
	l := e.eval(t.L)
	r := e.eval(t.R)
	switch t.Op {
	case "+":
		ls, lok := l.(string)
		rs, rok := r.(string)
		if lok || rok {
			if !lok {
				ls = concatString(l)
			}
			if !rok {
				rs = concatString(r)
			}
			return ls + rs
		}
		return ToNumber(l) + ToNumber(r)
	case "-":
		return ToNumber(l) - ToNumber(r)
	case "*":
		return ToNumber(l) * ToNumber(r)
	case "/":
		d := ToNumber(r)
		if d == 0 {
			return 0.0
		}
		return ToNumber(l) / d
	case "%":
		d := ToNumber(r)
		if d == 0 {
			return 0.0
		}
		return jsMod(ToNumber(l), d)
	case "==":
		return LooseEqual(l, r)
	case "!=":
		return !LooseEqual(l, r)
	case ">":
		return ToNumber(l) > ToNumber(r)
	case "<":
		return ToNumber(l) < ToNumber(r)
	case ">=":
		return ToNumber(l) >= ToNumber(r)
	case "<=":
		return ToNumber(l) <= ToNumber(r)
	}
	return nil
}

func concatString(v Value) string {
	if v == nil {
		return ""
	}
	return ToString(v)
}

func (e *evalEnv) evalArgs(c *CallNode, params []*Param) *Object {
	if c.HasMapped {
		args := NewObject(len(c.Mapped))
		for _, a := range c.Mapped {
			args.Set(a.Name, e.eval(a.Value))
		}
		return args
	}
	n := len(params)
	if len(c.Args) < n {
		n = len(c.Args)
	}
	args := NewObject(n)
	for i := 0; i < n; i++ {
		args.Set(params[i].Name, e.eval(c.Args[i]))
	}
	return args
}

func (e *evalEnv) evalCall(c *CallNode) Value {
	if b := builtinFuncs[c.Name]; b != nil {
		var buf [4]Value
		var argv []Value
		if c.HasMapped {
			argv = buf[:len(b.params)]
			for _, a := range c.Mapped {
				for i, pn := range b.params {
					if a.Name == pn {
						argv[i] = e.eval(a.Value)
						break
					}
				}
			}
		} else {
			n := len(b.params)
			if len(c.Args) < n {
				n = len(c.Args)
			}
			argv = buf[:n]
			for i := 0; i < n; i++ {
				argv[i] = e.eval(c.Args[i])
			}
		}
		return b.fn(argv)
	}
	if c.Name == "Each" {
		return e.evalEach(c)
	}
	if isStepName(c.Name) {
		if c.Name == "Action" {
			var list Value
			if len(c.Args) > 0 {
				list = e.eval(c.Args[0])
			}
			arr, _ := list.([]Value)
			return flattenPlans(arr)
		}
		call := c
		if e.scope != nil {
			call = captureScopeCall(c, e.scope)
		}
		return singleStepPlan(call)
	}
	if fn := e.lib.Functions[c.Name]; fn != nil {
		return e.callFunction(fn, c)
	}
	if e.lib.Actions[c.Name] != nil {
		call := c
		if e.scope != nil {
			call = captureScopeCall(c, e.scope)
		}
		return singleStepPlan(call)
	}
	if comp := e.lib.Components[c.Name]; comp != nil && c.HasMapped {
		props := NewObject(len(c.Mapped))
		for _, a := range c.Mapped {
			p := paramByName(comp, a.Name)
			if sr, ok := a.Value.(*StateRefNode); ok {
				if p != nil && p.Bindable {
					props.Set(a.Name, &Binding{Target: sr.N, expr: &StateRefNode{N: "$value"}})
				} else {
					props.Set(a.Name, e.getState(sr.N))
				}
				continue
			}
			props.Set(a.Name, e.eval(a.Value))
		}
		return &Element{Type: c.Name, Props: props, SourceKey: c.SourceKey, dynamic: true, kind: elEphemeral}
	}
	return nil
}

func paramByName(c *Component, name string) *Param {
	for _, p := range c.Params {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func flattenPlans(vals []Value) *ActionPlan {
	plan := &ActionPlan{}
	for _, v := range vals {
		switch t := v.(type) {
		case *ActionPlan:
			plan.steps = append(plan.steps, t.steps...)
		case *Object:
			if typ, ok := t.Value("type").(string); ok && (typ == "continue_conversation" || typ == "open_url") {
				plan.steps = append(plan.steps, planStep{data: t})
			}
		}
	}
	return plan
}

// callArgs evaluates library function/action args; an explicit null for an
// optional param is dropped so its default applies.
func (e *evalEnv) callArgsInto(dst *Object, c *CallNode, params []*Param) *Object {
	if c.HasMapped {
		for _, a := range c.Mapped {
			v := e.eval(a.Value)
			if v == nil {
				opt := false
				for _, p := range params {
					if p.Name == a.Name {
						opt = !p.Required
						break
					}
				}
				if opt {
					continue
				}
			}
			dst.Set(a.Name, v)
		}
		return dst
	}
	n := len(params)
	if len(c.Args) < n {
		n = len(c.Args)
	}
	for i := 0; i < n; i++ {
		p := params[i]
		v := e.eval(c.Args[i])
		if v == nil && !p.Required {
			continue
		}
		dst.Set(p.Name, v)
	}
	return dst
}

func (e *evalEnv) callArgs(c *CallNode, params []*Param) *Object {
	n := len(params)
	if c.HasMapped {
		n = len(c.Mapped)
	} else if len(c.Args) < n {
		n = len(c.Args)
	}
	return e.callArgsInto(NewObject(n), c, params)
}

func (e *evalEnv) acquireArgObj() (*Object, bool) {
	if e.argDepth < len(e.argPool) {
		o := &e.argPool[e.argDepth]
		e.argDepth++
		if cap(o.keys) == 0 {
			o.keys = o.kbuf[:0]
			o.vals = o.vbuf[:0]
		} else {
			o.keys = o.keys[:0]
			o.vals = o.vals[:0]
			if o.index != nil {
				clear(o.index)
				o.index = nil
			}
		}
		return o, true
	}
	return NewObject(4), false
}

func (e *evalEnv) releaseArgObj(o *Object) {
	clear(o.keys)
	clear(o.vals)
	o.keys = o.keys[:0]
	o.vals = o.vals[:0]
	o.index = nil
	e.argDepth--
}

func missesRequired(args *Object, params []*Param) bool {
	for _, p := range params {
		if p.Required && args.Value(p.Name) == nil && !p.HasDefault {
			return true
		}
	}
	return false
}

// checkValue validates a runtime value against a schema, reporting errors.
func (e *evalEnv) checkValue(v Value, schema *Object, compiled *compiledSchema, name, path string, strict bool) Value {
	if schema == nil {
		return v
	}
	c := &e.vctx
	c.lib = e.lib
	c.source = SourceRuntime
	c.errors = c.errors[:0]
	var out Value
	if compiled != nil && compiled.raw == schema {
		c.resetPath(path)
		out = c.validateCompiled(v, compiled, name)
	} else {
		out = c.validateSchemaValue(v, schema, name, path)
	}
	for _, d := range c.errors {
		e.reportf("@%s: %s", name, d.Message)
	}
	if strict && len(c.errors) > 0 {
		return invalid
	}
	return out
}

func (e *evalEnv) callFunction(fn *Function, c *CallNode) (result Value) {
	if !fn.Pure && e.onImpure != nil {
		e.onImpure()
	}
	args, pooled := e.acquireArgObj()
	if pooled {
		defer func() {
			if result == args {
				// Defensive: if a custom FuncImpl returned its args object, detach the pool slot.
				e.argPool[e.argDepth-1] = Object{}
			}
			e.releaseArgObj(args)
		}()
	}
	e.callArgsInto(args, c, fn.Params)
	if missesRequired(args, fn.Params) {
		return nil
	}
	checked := e.checkValue(args, fn.ParamsSchema, fn.paramsCompiled, fn.Name, "", false)
	if isInvalid(checked) {
		return nil
	}
	obj, _ := checked.(*Object)
	if obj == nil {
		obj = args
	}
	cloned := obj == args // args is owned here
	for _, p := range fn.Params {
		if p.HasDefault && !obj.Has(p.Name) {
			if !cloned {
				obj = obj.Clone()
				cloned = true
			}
			obj.Set(p.Name, p.Default)
		}
	}
	if fn.Impl == nil {
		e.reportf("@%s threw: no implementation", fn.Name)
		return nil
	}
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if be, ok := r.(*budgetError); ok {
					panic(be)
				}
				err = fmt.Errorf("%v", r)
			}
		}()
		result, err = fn.Impl(obj, e.fenv)
	}()
	if err != nil {
		var be *budgetError
		if errors.As(err, &be) {
			panic(be)
		}
		e.reportf("@%s threw: %s", fn.Name, err.Error())
		return nil
	}
	if e.limits != nil {
		if err := checkValueSize(result, e.limits, "function @"+fn.Name); err != nil {
			panic(err)
		}
	}
	if fn.Returns == nil {
		return result
	}
	out := e.checkValue(result, fn.Returns, fn.returnsCompiled, fn.Name, "/returns", false)
	if isInvalid(out) || isOmit(out) {
		return nil
	}
	return out
}

// checkValueSize enforces the value-node, depth and collection budgets on
// data produced by functions.
func checkValueSize(v Value, lim *Limits, what string) *budgetError {
	// Fast path for scalar values (most function returns: strings, numbers, bools, nil).
	switch v.(type) {
	case nil, bool, float64, string:
		if lim.MaxValueNodes > 0 && 1 > lim.MaxValueNodes {
			return &budgetError{phase: what + " nodes", limit: lim.MaxValueNodes}
		}
		return nil
	}
	nodes := 0
	var walk func(v Value, depth int) *budgetError
	walk = func(v Value, depth int) *budgetError {
		nodes++
		if nodes > lim.MaxValueNodes {
			return &budgetError{phase: what + " nodes", limit: lim.MaxValueNodes}
		}
		if depth > lim.MaxDepth {
			return &budgetError{phase: what + " depth", limit: lim.MaxDepth}
		}
		switch t := v.(type) {
		case []Value:
			if len(t) > lim.MaxListItems {
				return &budgetError{phase: what + " collection", limit: lim.MaxListItems}
			}
			for _, x := range t {
				if err := walk(x, depth+1); err != nil {
					return err
				}
			}
		case *Object:
			if t.Len() > lim.MaxListItems {
				return &budgetError{phase: what + " collection", limit: lim.MaxListItems}
			}
			for i := range t.keys {
				if err := walk(t.vals[i], depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(v, 0)
}

// evalEach implements @Each(array, varName, template).
func (e *evalEnv) evalEach(c *CallNode) Value {
	if len(c.Args) < 3 {
		return []Value{}
	}
	arr, ok := e.eval(c.Args[0]).([]Value)
	if !ok {
		return []Value{}
	}
	name := eachVarName(c)
	if name == "" {
		return []Value{}
	}
	if e.limits != nil && len(arr) > e.limits.MaxListItems {
		panic(&budgetError{phase: "@Each collection", limit: e.limits.MaxListItems})
	}
	tmpl := c.Args[2]
	out := make([]Value, len(arr))
	prev := e.scope
	sv := scopeVar{name: name, next: prev}
	e.scope = &sv
	defer func() { e.scope = prev }()
	for i, item := range arr {
		sv.value = item
		v := e.eval(tmpl)
		if el, ok := v.(*Element); ok {
			key := Value(float64(i))
			if o, ok := item.(*Object); ok {
				if id := o.Value("id"); id != nil {
					key = id
				} else if k := o.Value("key"); k != nil {
					key = k
				}
			}
			el.InstanceKey = name + ":" + ToString(key)
		}
		out[i] = v
	}
	return out
}

type shadowVar struct {
	name string
	next *shadowVar
}

func isShadowed(sh *shadowVar, name string) bool {
	for s := sh; s != nil; s = s.next {
		if s.name == name {
			return true
		}
	}
	return false
}

func lookupUnshadowed(scope *scopeVar, sh *shadowVar, name string) (Value, bool) {
	if isShadowed(sh, name) {
		return nil, false
	}
	for s := scope; s != nil; s = s.next {
		if s.name == name {
			return s.value, true
		}
	}
	return nil, false
}

// captureScope substitutes active @Each loop variables inside a deferred AST
// node (ActionPlan step or AssignNode expression), sharing unchanged subtrees.
func captureScope(n Node, scope *scopeVar) Node {
	out, _ := substScope(n, scope, nil)
	return out
}

func captureScopeCall(c *CallNode, scope *scopeVar) *CallNode {
	out, ch := substScope(c, scope, nil)
	if !ch {
		return c
	}
	return out.(*CallNode)
}

func substScope(n Node, scope *scopeVar, sh *shadowVar) (Node, bool) {
	switch t := n.(type) {
	case *RefNode:
		if v, ok := lookupUnshadowed(scope, sh, t.N); ok {
			return &ValueNode{V: v}, true
		}
		return t, false
	case *MemberNode:
		obj, ch := substScope(t.X, scope, sh)
		if !ch {
			return t, false
		}
		if vn, ok := obj.(*ValueNode); ok {
			if o, ok := vn.V.(*Object); ok {
				if x, ok := o.Get(t.Field); ok {
					return &ValueNode{V: x}, true
				}
			}
		}
		return &MemberNode{X: obj, Field: t.Field}, true
	case *IndexNode:
		x, c1 := substScope(t.X, scope, sh)
		i, c2 := substScope(t.Index, scope, sh)
		if !c1 && !c2 {
			return t, false
		}
		return &IndexNode{X: x, Index: i}, true
	case *BinOpNode:
		l, c1 := substScope(t.L, scope, sh)
		r, c2 := substScope(t.R, scope, sh)
		if !c1 && !c2 {
			return t, false
		}
		return &BinOpNode{Op: t.Op, L: l, R: r}, true
	case *UnaryNode:
		x, ch := substScope(t.X, scope, sh)
		if !ch {
			return t, false
		}
		return &UnaryNode{Op: t.Op, X: x}, true
	case *TernaryNode:
		c, c1 := substScope(t.Cond, scope, sh)
		th, c2 := substScope(t.Then, scope, sh)
		el, c3 := substScope(t.Else, scope, sh)
		if !c1 && !c2 && !c3 {
			return t, false
		}
		return &TernaryNode{Cond: c, Then: th, Else: el}, true
	case *ArrNode:
		var els []Node
		for i, x := range t.Els {
			y, ch := substScope(x, scope, sh)
			if ch && els == nil {
				els = append(make([]Node, 0, len(t.Els)), t.Els[:i]...)
			}
			if els != nil {
				els = append(els, y)
			}
		}
		if els == nil {
			return t, false
		}
		return &ArrNode{Els: els}, true
	case *ObjNode:
		var ents []ObjEntry
		for i, x := range t.Entries {
			y, ch := substScope(x.Value, scope, sh)
			if ch && ents == nil {
				ents = append(make([]ObjEntry, 0, len(t.Entries)), t.Entries[:i]...)
			}
			if ents != nil {
				ents = append(ents, ObjEntry{Key: x.Key, Value: y})
			}
		}
		if ents == nil {
			return t, false
		}
		return &ObjNode{Entries: ents}, true
	case *CallNode:
		innerSh := sh
		var eachShadow shadowVar
		isEach := t.Name == "Each"
		if isEach {
			if vn := eachVarName(t); vn != "" {
				eachShadow = shadowVar{name: vn, next: sh}
				innerSh = &eachShadow
			}
		}
		var args []Node
		for i, a := range t.Args {
			curSh := sh
			if isEach && i > 0 {
				curSh = innerSh
			}
			y, ch := substScope(a, scope, curSh)
			if ch && args == nil {
				args = append(make([]Node, 0, len(t.Args)), t.Args[:i]...)
			}
			if args != nil {
				args = append(args, y)
			}
		}
		var mapped []NamedArg
		if t.HasMapped {
			for i, a := range t.Mapped {
				y, ch := substScope(a.Value, scope, sh)
				if ch && mapped == nil {
					mapped = append(make([]NamedArg, 0, len(t.Mapped)), t.Mapped[:i]...)
				}
				if mapped != nil {
					mapped = append(mapped, NamedArg{Name: a.Name, Value: y})
				}
			}
		}
		if args == nil && mapped == nil {
			return t, false
		}
		cp := *t
		if args != nil {
			cp.Args = args
		}
		if mapped != nil {
			cp.Mapped = mapped
		}
		return &cp, true
	case *AssignNode:
		x, ch := substScope(t.Value, scope, sh)
		if !ch {
			return t, false
		}
		return &AssignNode{Target: t.Target, Value: x}, true
	}
	return n, false
}

// evalStandalone evaluates a parse-time state default that is an
// expression (spec 1.1 semantics): built-ins only, earlier defaults visible.
func evalStandalone(v Value, defaults *Object, lib *Library, lim Limits) Value {
	if lib == nil {
		lib = &Library{}
	}
	env := &evalEnv{
		lib:      &Library{Components: lib.Components, calls: builtinCalls},
		getState: func(name string) Value { return defaults.Value(name) },
		budget:   &budget{phase: "evaluation", maxCount: lim.MaxEvalSteps, maxDepth: lim.MaxDepth},
		limits:   &lim,
	}
	var walk func(v Value) Value
	walk = func(v Value) Value {
		switch t := v.(type) {
		case Node:
			return dataOnly(env.eval(t))
		case []Value:
			out := make([]Value, len(t))
			for i, x := range t {
				out[i] = walk(x)
			}
			return out
		case *Object:
			o := NewObject(t.Len())
			for i, k := range t.Keys() {
				o.appendOrdered(k, walk(t.vals[i]))
			}
			return o
		case *Element:
			return nil
		}
		return v
	}
	return walk(v)
}

// dataOnly replaces non-data values (elements, plans, bindings) with null.
func dataOnly(v Value) Value {
	switch t := v.(type) {
	case *Element, *ActionPlan, *Binding, Node:
		return nil
	case []Value:
		out := make([]Value, len(t))
		for i, x := range t {
			out[i] = dataOnly(x)
		}
		return out
	case *Object:
		o := NewObject(t.Len())
		for i, k := range t.Keys() {
			o.appendOrdered(k, dataOnly(t.vals[i]))
		}
		return o
	}
	return v
}
