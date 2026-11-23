package lang

// Node is an expression AST node.
type Node interface{ isNode() }

type (
	// StrNode is a string literal.
	StrNode struct{ V string }
	// NumNode is a number literal.
	NumNode struct{ V float64 }
	// BoolNode is true or false.
	BoolNode struct{ V bool }
	// NullNode is null.
	NullNode struct{}
	// ArrNode is an array literal.
	ArrNode struct{ Els []Node }
	// ObjNode is an object literal.
	ObjNode struct{ Entries []ObjEntry }
	// RefNode references a statement (or an @Each loop variable).
	RefNode struct{ N string }
	// PhNode is an unresolved reference placeholder.
	PhNode struct{ N string }
	// StateRefNode reads a $variable.
	StateRefNode struct{ N string }
	// RuntimeRefNode reads a query or mutation result.
	RuntimeRefNode struct {
		N        string
		Mutation bool
	}
	// BinOpNode is a binary operation.
	BinOpNode struct {
		Op   string
		L, R Node
	}
	// UnaryNode is ! or unary -.
	UnaryNode struct {
		Op string
		X  Node
	}
	// TernaryNode is cond ? then : else.
	TernaryNode struct{ Cond, Then, Else Node }
	// MemberNode is x.field.
	MemberNode struct {
		X     Node
		Field string
	}
	// IndexNode is x[index].
	IndexNode struct{ X, Index Node }
	// AssignNode is $name = value inside an expression (binding assignment).
	AssignNode struct {
		Target string
		Value  Node
	}
	// CallNode is a component call, an @function/@action call, Action(...),
	// Query(...) or Mutation(...).
	CallNode struct {
		Name string
		Args []Node
		// Mapped holds args named by parameter after materialization.
		Mapped    []NamedArg
		HasMapped bool
		// Fn marks an @Name call whose name is not a built-in.
		Fn        bool
		SourceKey string
	}
)

// ObjEntry is one object literal entry.
type ObjEntry struct {
	Key   string
	Value Node
}

// NamedArg is a call argument mapped to a parameter name.
type NamedArg struct {
	Name  string
	Value Node
}

func (*StrNode) isNode()        {}
func (*NumNode) isNode()        {}
func (*BoolNode) isNode()       {}
func (*NullNode) isNode()       {}
func (*ArrNode) isNode()        {}
func (*ObjNode) isNode()        {}
func (*RefNode) isNode()        {}
func (*PhNode) isNode()         {}
func (*StateRefNode) isNode()   {}
func (*RuntimeRefNode) isNode() {}
func (*BinOpNode) isNode()      {}
func (*UnaryNode) isNode()      {}
func (*TernaryNode) isNode()    {}
func (*MemberNode) isNode()     {}
func (*IndexNode) isNode()      {}
func (*AssignNode) isNode()     {}
func (*CallNode) isNode()       {}

var nullNode = &NullNode{}

func isRuntimeExpr(n Node) bool {
	switch n.(type) {
	case *StateRefNode, *RuntimeRefNode, *BinOpNode, *UnaryNode, *TernaryNode, *MemberNode, *IndexNode, *AssignNode:
		return true
	}
	return false
}

// mappedArg returns a mapped argument.
func (c *CallNode) mappedArg(name string) (Node, bool) {
	for _, a := range c.Mapped {
		if a.Name == name {
			return a.Value, true
		}
	}
	return nil, false
}

// walkAST visits every node depth-first.
func walkAST(n Node, visit func(Node)) {
	if n == nil {
		return
	}
	visit(n)
	switch t := n.(type) {
	case *CallNode:
		for _, a := range t.Args {
			walkAST(a, visit)
		}
		for _, a := range t.Mapped {
			walkAST(a.Value, visit)
		}
	case *ArrNode:
		for _, e := range t.Els {
			walkAST(e, visit)
		}
	case *ObjNode:
		for _, e := range t.Entries {
			walkAST(e.Value, visit)
		}
	case *BinOpNode:
		walkAST(t.L, visit)
		walkAST(t.R, visit)
	case *UnaryNode:
		walkAST(t.X, visit)
	case *TernaryNode:
		walkAST(t.Cond, visit)
		walkAST(t.Then, visit)
		walkAST(t.Else, visit)
	case *MemberNode:
		walkAST(t.X, visit)
	case *IndexNode:
		walkAST(t.X, visit)
		walkAST(t.Index, visit)
	case *AssignNode:
		walkAST(t.Value, visit)
	}
}

// toLiteralAST converts a plain value into literal nodes.
func toLiteralAST(v Value) Node {
	switch t := v.(type) {
	case nil:
		return nullNode
	case string:
		return &StrNode{V: t}
	case float64:
		return &NumNode{V: t}
	case bool:
		return &BoolNode{V: t}
	case []Value:
		els := make([]Node, len(t))
		for i, x := range t {
			els[i] = toLiteralAST(x)
		}
		return &ArrNode{Els: els}
	case *Object:
		ents := make([]ObjEntry, 0, t.Len())
		for _, k := range t.Keys() {
			ents = append(ents, ObjEntry{Key: k, Value: toLiteralAST(t.vals[k])})
		}
		return &ObjNode{Entries: ents}
	}
	return nullNode
}

// literalValue returns the plain value of a literal-only node.
func literalValue(n Node) (Value, bool) {
	switch t := n.(type) {
	case *StrNode:
		return t.V, true
	case *NumNode:
		return t.V, true
	case *BoolNode:
		return t.V, true
	case *NullNode:
		return nil, true
	case *ArrNode:
		out := make([]Value, 0, len(t.Els))
		for _, e := range t.Els {
			v, ok := literalValue(e)
			if !ok {
				return nil, false
			}
			out = append(out, v)
		}
		return out, true
	case *ObjNode:
		o := NewObject(len(t.Entries))
		for _, e := range t.Entries {
			v, ok := literalValue(e.Value)
			if !ok {
				return nil, false
			}
			o.Set(e.Key, v)
		}
		return o, true
	}
	return nil, false
}
