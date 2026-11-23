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
	defer e.budget.leave()
	switch t := n.(type) {
	case *StrNode:
		return t.V
	case *NumNode:
		return t.V
	case *BoolNode:
		return t.V
	case *NullNode, *PhNode:
		return nil
	case *StateRefNode:
		if e.extra != nil {
			if v, ok := e.extra[t.N]; ok {
				return v
			}
		}
		return e.getState(t.N)
	case *RefNode:
		if v, ok := e.lookup(t.N); ok {
			return v
		}
		if e.resolveRef != nil {
			return e.resolveRef(t.N)
		}
		return nil
	case *RuntimeRefNode:
		if e.resolveRef != nil {
			return e.resolveRef(t.N)
		}
		return nil
	case *ArrNode:
		out := make([]Value, len(t.Els))
		for i, x := range t.Els {
			out[i] = e.eval(x)
		}
		return out
	case *ObjNode:
		o := NewObject(len(t.Entries))
		for _, x := range t.Entries {
			o.Set(x.Key, e.eval(x.Value))
		}
		return o
	case *CallNode:
		return e.evalCall(t)
	case *BinOpNode:
		return e.evalBinOp(t)
	case *UnaryNode:
		x := e.eval(t.X)
		if t.Op == "!" {
			return !Truthy(x)
		}
		return -ToNumber(x)
	case *TernaryNode:
		if Truthy(e.eval(t.Cond)) {
			return e.eval(t.Then)
		}
		return e.eval(t.Else)
	case *MemberNode:
		x := e.eval(t.X)
		if x == nil {
			return nil
		}
		if arr, ok := x.([]Value); ok {
			if t.Field == "length" {
				return float64(len(arr))
			}
			out := make([]Value, len(arr))
			for i, it := range arr {
				out[i] = memberOf(it, t.Field)
			}
			return out
		}
		return memberOf(x, t.Field)
	case *IndexNode:
		x := e.eval(t.X)
		idx := e.eval(t.Index)
		if x == nil || idx == nil {
			return nil
		}
		if arr, ok := x.([]Value); ok {
			f := ToNumber(idx)
			if f >= 0 && f == math.Trunc(f) && f < float64(len(arr)) {
				return arr[int(f)]
			}
			return nil
		}
		return memberOf(x, ToString(idx))
	case *AssignNode:
		return &Binding{Target: t.Target, expr: t.Value}
	}
	return nil
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
	var named []NamedArg
	if c.HasMapped {
		named = c.Mapped
	} else {
		named = nameArgs(c.Args, params)
	}
	args := NewObject(len(named))
	for _, a := range named {
		args.Set(a.Name, e.eval(a.Value))
	}
	return args
}

func (e *evalEnv) evalCall(c *CallNode) Value {
	if b := builtinFuncs[c.Name]; b != nil {
		def := builtinCalls[c.Name]
		args := e.evalArgs(c, def.params)
		m := make(map[string]Value, args.Len())
		for _, k := range args.Keys() {
			m[k] = args.vals[k]
		}
		return b.fn(m)
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
		return &ActionPlan{steps: []planStep{{call: c}}}
	}
	if fn := e.lib.Functions[c.Name]; fn != nil {
		return e.callFunction(fn, c)
	}
	if e.lib.Actions[c.Name] != nil {
		return &ActionPlan{steps: []planStep{{call: c}}}
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
		return &Element{Type: c.Name, Props: props, SourceKey: c.SourceKey, dynamic: true}
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
func (e *evalEnv) callArgs(c *CallNode, params []*Param) *Object {
	args := e.evalArgs(c, params)
	for _, p := range params {
		if v, ok := args.Get(p.Name); ok && v == nil && !p.Required {
			args.Delete(p.Name)
		}
	}
	return args
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
func (e *evalEnv) checkValue(v Value, schema *Object, name, path string, strict bool) Value {
	if schema == nil {
		return v
	}
	c := &vctx{lib: e.lib, source: SourceRuntime}
	out := c.validateSchemaValue(v, schema, name, path)
	for _, d := range c.errors {
		e.reportf("@%s: %s", name, d.Message)
	}
	if strict && len(c.errors) > 0 {
		return invalid
	}
	return out
}

func (e *evalEnv) callFunction(fn *Function, c *CallNode) (result Value) {
	args := e.callArgs(c, fn.Params)
	if missesRequired(args, fn.Params) {
		return nil
	}
	checked := e.checkValue(args, fn.ParamsSchema, fn.Name, "", false)
	if isInvalid(checked) {
		return nil
	}
	obj, _ := checked.(*Object)
	if obj == nil {
		obj = args
	}
	for _, p := range fn.Params {
		if p.HasDefault && !obj.Has(p.Name) {
			obj = obj.Clone()
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
	out := e.checkValue(result, fn.Returns, fn.Name, "/returns", false)
	if isInvalid(out) || isOmit(out) {
		return nil
	}
	return out
}

// checkValueSize enforces the value-node, depth and collection budgets on
// data produced by functions.
func checkValueSize(v Value, lim *Limits, what string) *budgetError {
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
			for _, k := range t.keys {
				if err := walk(t.vals[k], depth+1); err != nil {
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
	for i, item := range arr {
		sub := substituteRef(tmpl, name, item)
		v := e.with(name, item).eval(sub)
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

func refersTo(n Node, name string) bool {
	found := false
	walkAST(n, func(x Node) {
		if r, ok := x.(*RefNode); ok && r.N == name {
			found = true
		}
	})
	return found
}

// substituteRef replaces Ref(name) with the loop element as literal AST, so
// deferred steps capture their own element.
func substituteRef(n Node, name string, v Value) Node {
	if !refersTo(n, name) {
		return n
	}
	sub := func(x Node) Node { return substituteRef(x, name, v) }
	switch t := n.(type) {
	case *RefNode:
		if t.N == name {
			return toLiteralAST(v)
		}
		return t
	case *MemberNode:
		obj := sub(t.X)
		if o, ok := obj.(*ObjNode); ok {
			for _, en := range o.Entries {
				if en.Key == t.Field {
					return en.Value
				}
			}
		}
		return &MemberNode{X: obj, Field: t.Field}
	case *IndexNode:
		return &IndexNode{X: sub(t.X), Index: sub(t.Index)}
	case *BinOpNode:
		return &BinOpNode{Op: t.Op, L: sub(t.L), R: sub(t.R)}
	case *UnaryNode:
		return &UnaryNode{Op: t.Op, X: sub(t.X)}
	case *TernaryNode:
		return &TernaryNode{Cond: sub(t.Cond), Then: sub(t.Then), Else: sub(t.Else)}
	case *ArrNode:
		els := make([]Node, len(t.Els))
		for i, x := range t.Els {
			els[i] = sub(x)
		}
		return &ArrNode{Els: els}
	case *ObjNode:
		ents := make([]ObjEntry, len(t.Entries))
		for i, x := range t.Entries {
			ents[i] = ObjEntry{Key: x.Key, Value: sub(x.Value)}
		}
		return &ObjNode{Entries: ents}
	case *CallNode:
		shadow := t.Name == "Each" && eachVarName(t) == name
		cp := *t
		cp.Args = make([]Node, len(t.Args))
		for i, a := range t.Args {
			if shadow && i > 0 {
				cp.Args[i] = a
			} else {
				cp.Args[i] = sub(a)
			}
		}
		if t.HasMapped {
			cp.Mapped = make([]NamedArg, len(t.Mapped))
			for i, a := range t.Mapped {
				cp.Mapped[i] = NamedArg{Name: a.Name, Value: sub(a.Value)}
			}
		}
		return &cp
	case *AssignNode:
		return &AssignNode{Target: t.Target, Value: sub(t.Value)}
	}
	return n
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
			for _, k := range t.Keys() {
				o.Set(k, walk(t.vals[k]))
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
		for _, k := range t.Keys() {
			o.Set(k, dataOnly(t.vals[k]))
		}
		return o
	}
	return v
}
