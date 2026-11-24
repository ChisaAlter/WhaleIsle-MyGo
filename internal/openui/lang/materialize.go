package lang

// budgetError aborts a parse or evaluation that exceeded a Limits budget.
type budgetError struct {
	phase string
	limit int
	path  string
}

func (e *budgetError) Error() string {
	s := "Whale UI " + e.phase + " budget exceeded (" + itoa(e.limit) + ")"
	if e.path != "" {
		s += " at " + e.path
	}
	return s
}

// budget counts steps and nesting depth.
type budget struct {
	phase    string
	count    int
	depth    int
	maxCount int
	maxDepth int
}

func (b *budget) enter(path string) {
	if b == nil {
		return
	}
	b.count++
	if b.maxCount > 0 && b.count > b.maxCount {
		panic(&budgetError{phase: b.phase, limit: b.maxCount, path: path})
	}
	b.depth++
	if b.maxDepth > 0 && b.depth > b.maxDepth {
		b.depth--
		panic(&budgetError{phase: b.phase + " depth", limit: b.maxDepth, path: path})
	}
}

func (b *budget) leave() {
	if b != nil {
		b.depth--
	}
}

// mctx is the materialization context (upstream MaterializeCtx).
type mctx struct {
	vctx
	syms      map[string]Node
	visited   map[string]bool
	unres     []string
	unreached map[string]bool
	cacheExpr map[string]Node
	cacheVal  map[string]Node
	budget    *budget
}

func (c *mctx) issue(component, path, code, message string) { c.push(component, path, code, message) }

func (c *mctx) resolveRef(name string, valueMode bool) Value {
	if c.visited[name] || c.syms[name] == nil {
		c.unres = append(c.unres, name)
		if valueMode {
			return nil
		}
		return &PhNode{N: name}
	}
	target := c.syms[name]
	delete(c.unreached, name)
	if call, ok := target.(*CallNode); ok && isReservedCall(call.Name) {
		return &RuntimeRefNode{N: name, Mutation: call.Name == "Mutation"}
	}
	cache := c.cacheExpr
	if valueMode {
		cache = c.cacheVal
	}
	if n, ok := cache[name]; ok {
		return n
	}
	c.visited[name] = true
	prev := c.statement
	c.statement = name
	before := len(c.unres)
	defer func() {
		c.statement = prev
		delete(c.visited, name)
	}()
	var res Value
	if valueMode {
		res = c.materializeValue(target)
		if el, ok := res.(*Element); ok {
			el.StatementID = name
		}
	} else {
		res = c.materializeExpr(target, nil)
	}
	if n, ok := res.(Node); ok && len(c.unres) == before {
		cache[name] = n
	}
	return res
}

func nameArgs(args []Node, params []*Param) []NamedArg {
	out := make([]NamedArg, 0, len(params))
	for i := 0; i < len(params) && i < len(args); i++ {
		out = append(out, NamedArg{Name: params[i].Name, Value: args[i]})
	}
	return out
}

// mapCallArgs checks literal library-call arguments at parse time. nil
// means the call is invalid.
func (c *mctx) mapCallArgs(name string, args []Node, def *callDef) ([]NamedArg, bool) {
	params := def.params
	if len(args) > len(params) {
		c.issue(name, "", "excess-args", name+" takes "+itoa(len(params))+" arg(s), got "+itoa(len(args))+" ("+itoa(len(args)-len(params))+" excess dropped)")
	}
	mapped := nameArgs(args, params)
	bad := false
	before := len(c.errors)
	for _, p := range params {
		var arg Node
		idx := -1
		for i, m := range mapped {
			if m.Name == p.Name {
				arg, idx = m.Value, i
			}
		}
		var lit Value
		if arg != nil {
			v, ok := literalValue(arg)
			if !ok {
				continue
			}
			lit = v
		}
		if lit == nil {
			if !p.Required || p.HasDefault {
				continue
			}
			code, msg := "missing-required", `missing required field "/`+p.Name+`"`
			if arg != nil {
				code, msg = "null-required", `required field "/`+p.Name+`" cannot be null`
			}
			c.issue(name, "/"+p.Name, code, msg+" — signature: "+paramsSignature(name, params))
			bad = true
			continue
		}
		next := c.validateSchemaValue(lit, p.Schema, name, "/"+p.Name)
		if isInvalid(next) {
			next = resolveInvalid(p.Required, p.Default, p.HasDefault)
		}
		switch {
		case isInvalid(next):
			bad = true
		case isOmit(next):
			mapped = append(mapped[:idx], mapped[idx+1:]...)
		case !sameValueOrNode(next, lit):
			mapped[idx].Value = toLiteralAST(next)
		}
	}
	if def.kind == callAction && len(c.errors) > before {
		bad = true
	}
	if bad {
		return nil, false
	}
	return mapped, true
}

var emptyPlanNode = func() Node { return &CallNode{Name: "Action", Args: []Node{&ArrNode{}}} }

func (c *mctx) materializeCall(n *CallNode, argFn func(Node) Node) Node {
	args := make([]Node, len(n.Args))
	for i, a := range n.Args {
		args[i] = argFn(a)
	}
	def := c.lib.callDef(n.Name)
	cp := *n
	cp.Args = args
	if def == nil || def.lazy {
		return &cp
	}
	if def.builtin {
		cp.Mapped, cp.HasMapped = nameArgs(args, def.params), true
		return &cp
	}
	mapped, ok := c.mapCallArgs(n.Name, args, def)
	if ok {
		cp.Mapped, cp.HasMapped = mapped, true
		return &cp
	}
	if def.kind == callAction {
		return emptyPlanNode()
	}
	return nullNode
}

// nameRawCallArgs names the args of calls inside Query/Mutation args in place.
func (c *mctx) nameRawCallArgs(args []Node, statementID string) {
	prev := c.statement
	c.statement = statementID
	seen := map[*CallNode]bool{}
	for _, a := range args {
		walkAST(a, func(n Node) {
			call, ok := n.(*CallNode)
			if !ok || seen[call] {
				return
			}
			seen[call] = true
			def := c.lib.callDef(call.Name)
			if def == nil || def.lazy {
				return
			}
			if !def.builtin {
				if m, ok := c.mapCallArgs(call.Name, call.Args, def); ok {
					call.Mapped, call.HasMapped = m, true
					return
				}
			}
			call.Mapped, call.HasMapped = nameArgs(call.Args, def.params), true
		})
	}
	c.statement = prev
}

func eachVarName(n *CallNode) string {
	if len(n.Args) < 2 {
		return ""
	}
	switch v := n.Args[1].(type) {
	case *RefNode:
		return v.N
	case *StrNode:
		return v.V
	}
	return ""
}

func (c *mctx) materializeLazy(n *CallNode, scoped map[string]bool) Node {
	def := c.lib.callDef(n.Name)
	if def == nil || def.kind != callFunction || !def.lazy || len(n.Args) < 3 {
		return nil
	}
	v := eachVarName(n)
	if v == "" {
		return nil
	}
	next := make(map[string]bool, len(scoped)+1)
	for k := range scoped {
		next[k] = true
	}
	next[v] = true
	cp := *n
	cp.Args = make([]Node, len(n.Args))
	for i, a := range n.Args {
		if i == 1 {
			cp.Args[i] = a
		} else {
			cp.Args[i] = c.materializeExpr(a, next)
		}
	}
	return &cp
}

func (c *mctx) sourceKey(call *CallNode) string {
	stmt := c.statement
	if stmt == "" {
		stmt = "root"
	}
	k := call.SourceKey
	if k == "" {
		k = "inline"
	}
	return stmt + "/" + k
}

func (c *mctx) unknownComponent(name string) {
	msg := `Unknown component "` + name + `" — not found in catalog or builtins`
	if len(c.lib.ComponentOrder) > 0 {
		msg += ". Available components: " + joinNames(c.lib.ComponentOrder)
	}
	c.issue(name, "", "unknown-component", msg)
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

func (c *mctx) unknownFunction(name string) {
	c.issue(name, "", "unknown-function", `Unknown function "@`+name+`": not a built-in or action step. The call evaluates to null`)
}

// materializeExpr normalizes a node for runtime evaluation: resolves refs,
// names catalog component args, checks library calls.
func (c *mctx) materializeExpr(n Node, scoped map[string]bool) Node {
	c.budget.enter(c.statement)
	defer c.budget.leave()
	switch t := n.(type) {
	case *RefNode:
		if scoped[t.N] {
			return t
		}
		return c.resolveRef(t.N, false).(Node)
	case *PhNode:
		return t
	case *CallNode:
		if t.Fn && !c.lib.isRuntimeCall(t.Name) {
			c.unknownFunction(t.Name)
			return nullNode
		}
		if lz := c.materializeLazy(t, scoped); lz != nil {
			return lz
		}
		if c.lib.isRuntimeCall(t.Name) {
			return c.materializeCall(t, func(a Node) Node { return c.materializeExpr(a, scoped) })
		}
		args := make([]Node, len(t.Args))
		for i, a := range t.Args {
			args[i] = c.materializeExpr(a, scoped)
		}
		cp := *t
		cp.Args = args
		if def := c.lib.Components[t.Name]; def != nil {
			cp.Mapped, cp.HasMapped = nameArgs(args, def.Params), true
			cp.SourceKey = c.sourceKey(t)
			return &cp
		}
		c.unknownComponent(t.Name)
		return &cp
	case *ArrNode:
		els := make([]Node, len(t.Els))
		for i, e := range t.Els {
			els[i] = c.materializeExpr(e, scoped)
		}
		return &ArrNode{Els: els}
	case *ObjNode:
		ents := make([]ObjEntry, len(t.Entries))
		for i, e := range t.Entries {
			ents[i] = ObjEntry{Key: e.Key, Value: c.materializeExpr(e.Value, scoped)}
		}
		return &ObjNode{Entries: ents}
	case *BinOpNode:
		return &BinOpNode{Op: t.Op, L: c.materializeExpr(t.L, scoped), R: c.materializeExpr(t.R, scoped)}
	case *UnaryNode:
		return &UnaryNode{Op: t.Op, X: c.materializeExpr(t.X, scoped)}
	case *TernaryNode:
		return &TernaryNode{Cond: c.materializeExpr(t.Cond, scoped), Then: c.materializeExpr(t.Then, scoped), Else: c.materializeExpr(t.Else, scoped)}
	case *MemberNode:
		return &MemberNode{X: c.materializeExpr(t.X, scoped), Field: t.Field}
	case *IndexNode:
		return &IndexNode{X: c.materializeExpr(t.X, scoped), Index: c.materializeExpr(t.Index, scoped)}
	case *AssignNode:
		return &AssignNode{Target: t.Target, Value: c.materializeExpr(t.Value, scoped)}
	}
	return n
}

// containsDynamic reports values that need runtime evaluation.
func containsDynamic(v Value) bool {
	switch t := v.(type) {
	case Node:
		return true
	case []Value:
		for _, x := range t {
			if containsDynamic(x) {
				return true
			}
		}
	case *Element:
		for _, k := range t.Props.Keys() {
			if containsDynamic(t.Props.vals[k]) {
				return true
			}
		}
	case *Object:
		if _, ok := t.Value("steps").([]Value); ok {
			return true
		}
		for _, k := range t.Keys() {
			if containsDynamic(t.vals[k]) {
				return true
			}
		}
	}
	return false
}

// materializeValue lowers a node into a plain value, an *Element, or a Node
// that stays for the runtime (upstream materializeValue).
func (c *mctx) materializeValue(n Node) Value {
	c.budget.enter(c.statement)
	defer c.budget.leave()
	switch t := n.(type) {
	case *RefNode:
		return c.resolveRef(t.N, true)
	case *StrNode:
		return t.V
	case *NumNode:
		return t.V
	case *BoolNode:
		return t.V
	case *NullNode, *PhNode:
		return nil
	case *ArrNode:
		items := make([]Value, 0, len(t.Els))
		for _, e := range t.Els {
			if _, ph := e.(*PhNode); ph {
				continue
			}
			v := c.materializeValue(e)
			if v == nil {
				switch e.(type) {
				case *CallNode, *RefNode:
					continue
				}
			}
			items = append(items, v)
		}
		return items
	case *ObjNode:
		o := NewObject(len(t.Entries))
		for _, e := range t.Entries {
			o.Set(e.Key, c.materializeValue(e.Value))
		}
		return o
	case *CallNode:
		name := t.Name
		if t.Fn && !c.lib.isRuntimeCall(name) {
			c.unknownFunction(name)
			return nil
		}
		if isReservedCall(name) {
			c.issue(name, "", "inline-reserved", name+"() must be declared as a top-level statement, not used inline as a value")
			return nil
		}
		if c.lib.isRuntimeCall(name) {
			if lz := c.materializeLazy(t, nil); lz != nil {
				return lz
			}
			return c.materializeCall(t, func(a Node) Node { return c.materializeExpr(a, nil) })
		}
		def := c.lib.Components[name]
		if def == nil {
			c.unknownComponent(name)
			return nil
		}
		props := NewObject(len(def.Params))
		drop := false
		for i := 0; i < len(def.Params) && i < len(t.Args); i++ {
			p := def.Params[i]
			v := c.materializeValue(t.Args[i])
			props.Set(p.Name, v)
			if p.Schema == nil {
				continue
			}
			next := c.validateSchemaValue(v, p.Schema, name, "/"+p.Name)
			if isInvalid(next) {
				next = resolveInvalid(p.Required, p.Default, p.HasDefault)
				if isInvalid(next) {
					drop = true
				} else if isOmit(next) {
					props.Delete(p.Name)
				}
			}
			if !isInvalid(next) && !isOmit(next) {
				props.Set(p.Name, next)
			}
		}
		if len(t.Args) > len(def.Params) {
			c.issue(name, "", "excess-args", name+" takes "+itoa(len(def.Params))+" arg(s), got "+itoa(len(t.Args))+" ("+itoa(len(t.Args)-len(def.Params))+" excess dropped)")
		}
		var still []*Param
		for _, p := range def.Params {
			v, present := props.Get(p.Name)
			if p.Required && (!present || v == nil) {
				if p.HasDefault {
					props.Set(p.Name, p.Default)
					continue
				}
				still = append(still, p)
			}
		}
		if len(still) > 0 {
			for _, p := range still {
				code, msg := "missing-required", `missing required field "/`+p.Name+`"`
				if props.Has(p.Name) {
					code, msg = "null-required", `required field "/`+p.Name+`" cannot be null`
				}
				c.issue(name, "/"+p.Name, code, msg+" — signature: "+paramsSignature(name, def.Params))
			}
			return nil
		}
		if drop {
			return nil
		}
		return &Element{Type: name, Props: props, Partial: c.partial, dynamic: containsDynamic(props), SourceKey: c.sourceKey(t)}
	}
	if isRuntimeExpr(n) {
		return c.materializeExpr(n, nil)
	}
	return n
}
