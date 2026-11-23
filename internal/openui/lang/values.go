package lang

import (
	"math"
	"strconv"
	"strings"
	"unsafe"
)

// Value is a runtime value of OpenUI Lang. The dynamic types are:
//
//   - nil: JavaScript null (undefined is folded into null)
//   - bool, float64, string
//   - []Value: an array
//   - *Object: a plain object with JavaScript key order
//   - *Element: a component instance
//   - *ActionPlan: an action (one or more deferred steps)
//   - *Binding: a two-way binding handle for a bindable prop
//   - Node: an unevaluated expression inside a materialized tree
//
// Evaluated element trees (Frame.Root) never contain Node values.
type Value = any

// Object is an ordered string-keyed map that follows ECMAScript property
// order: array-index keys in ascending numeric order first, then the other
// keys in insertion order.
type Object struct {
	keys []string
	vals map[string]Value
}

// NewObject returns an empty object with room for n keys.
func NewObject(n int) *Object {
	return &Object{keys: make([]string, 0, n), vals: make(map[string]Value, n)}
}

// Len returns the number of keys.
func (o *Object) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Keys returns the keys in property order. The slice must not be modified.
func (o *Object) Keys() []string {
	if o == nil {
		return nil
	}
	return o.keys
}

// Get returns the value of key and whether it is present.
func (o *Object) Get(key string) (Value, bool) {
	if o == nil {
		return nil, false
	}
	v, ok := o.vals[key]
	return v, ok
}

// Value returns the value of key, or nil.
func (o *Object) Value(key string) Value {
	v, _ := o.Get(key)
	return v
}

// Has reports whether key is present.
func (o *Object) Has(key string) bool {
	if o == nil {
		return false
	}
	_, ok := o.vals[key]
	return ok
}

// Set sets key to v, inserting it in ECMAScript order when it is new.
func (o *Object) Set(key string, v Value) {
	if _, ok := o.vals[key]; ok {
		o.vals[key] = v
		return
	}
	o.vals[key] = v
	if idx, ok := arrayIndex(key); ok {
		// Insert among the leading index keys in ascending order.
		pos := 0
		for pos < len(o.keys) {
			k, isIdx := arrayIndex(o.keys[pos])
			if !isIdx || k > idx {
				break
			}
			pos++
		}
		o.keys = append(o.keys, "")
		copy(o.keys[pos+1:], o.keys[pos:])
		o.keys[pos] = key
		return
	}
	o.keys = append(o.keys, key)
}

// Delete removes key.
func (o *Object) Delete(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// Clone returns a shallow copy.
func (o *Object) Clone() *Object {
	if o == nil {
		return NewObject(0)
	}
	c := &Object{keys: make([]string, len(o.keys), len(o.keys)+1), vals: make(map[string]Value, len(o.vals)+1)}
	copy(c.keys, o.keys)
	for k, v := range o.vals {
		c.vals[k] = v
	}
	return c
}

// arrayIndex reports whether key is a canonical array index (0 .. 2^32-2).
func arrayIndex(key string) (uint64, bool) {
	if key == "" || len(key) > 10 {
		return 0, false
	}
	if len(key) > 1 && key[0] == '0' {
		return 0, false
	}
	var n uint64
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + uint64(c-'0')
	}
	if n >= 1<<32-1 {
		return 0, false
	}
	return n, true
}

// ObjectFrom builds an object from alternating key/value pairs.
func ObjectFrom(kv ...any) *Object {
	o := NewObject(len(kv) / 2)
	for i := 0; i+1 < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// Element is a component instance: a library component with resolved props.
type Element struct {
	// Type is the component name, for example "Button".
	Type string
	// Props holds the props in parameter order. Absent optional props are
	// not present. Child components appear as *Element values (or arrays of
	// them) inside props such as "children".
	Props *Object
	// StatementID is the statement that produced the element, when it was
	// produced by a named statement.
	StatementID string
	// SourceKey identifies the call site (statement and call ordinal).
	SourceKey string
	// InstanceKey distinguishes elements produced by one @Each template.
	InstanceKey string
	// FormName is the form scope the element belongs to, set while the tree
	// is evaluated. Empty means page level.
	FormName string
	// Partial is true when the element was read from a statement that is
	// still streaming.
	Partial bool
	// Issues lists schema problems found by the library element validator
	// (for example invalid options of a specialized component).
	Issues  []string
	dynamic bool
}

// Prop returns a prop value.
func (e *Element) Prop(name string) Value {
	if e == nil {
		return nil
	}
	return e.Props.Value(name)
}

// Children returns the element-valued props in prop order, flattening arrays.
func (e *Element) Children() []*Element {
	if e == nil {
		return nil
	}
	var out []*Element
	var walk func(v Value)
	walk = func(v Value) {
		switch t := v.(type) {
		case *Element:
			out = append(out, t)
		case []Value:
			for _, x := range t {
				walk(x)
			}
		}
	}
	for _, k := range e.Props.Keys() {
		walk(e.Props.Value(k))
	}
	return out
}

// Identity returns a stable identity for renderer state (statement, call
// site and @Each instance).
func (e *Element) Identity() string {
	id := e.StatementID
	if id == "" {
		id = e.SourceKey
	}
	if id == "" {
		id = e.Type
	}
	return id + ":" + e.Type + ":" + e.InstanceKey
}

// Binding is the two-way binding handle passed to a bindable prop
// ($binding<T>). Value is the current state value; writing goes through
// Instance.SetBinding(Target, v).
type Binding struct {
	Target string
	Value  Value
	expr   Node
}

// ── Type tests and coercion (ECMA-262 semantics where the spec requires) ──

func isPlainObject(v Value) bool {
	_, ok := v.(*Object)
	return ok
}

// isObjectLike reports values that behave as objects in expressions.
func isObjectLike(v Value) bool {
	switch v.(type) {
	case *Object, []Value, *Element, *ActionPlan, *Binding:
		return true
	}
	return false
}

// Truthy implements JavaScript ToBoolean.
func Truthy(v Value) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return !(t == 0 || math.IsNaN(t))
	case string:
		return t != ""
	case Node:
		return true
	default:
		return true
	}
}

// ToNumber is the language's lenient numeric coercion (spec section 3.1): a
// number is itself, a string parses as a number or else 0, true is 1, and
// everything else is 0.
func ToNumber(v Value) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		n := StringToNumber(t)
		if math.IsNaN(n) {
			return 0
		}
		return n
	case bool:
		if t {
			return 1
		}
		return 0
	}
	return 0
}

// jsToNumber is ECMAScript ToNumber for primitives (NaN allowed).
func jsToNumber(v Value) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		return StringToNumber(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case nil:
		return 0
	}
	return math.NaN()
}

// ToString implements JavaScript String(v) for language values.
func ToString(v Value) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return NumberToString(t)
	case string:
		return t
	case []Value:
		var b strings.Builder
		for i, x := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			if x != nil {
				b.WriteString(ToString(x))
			}
		}
		return b.String()
	default:
		return "[object Object]"
	}
}

// toPrimitive converts objects for loose equality (arrays join, others
// become "[object Object]").
func toPrimitive(v Value) Value {
	if isObjectLike(v) {
		return ToString(v)
	}
	return v
}

// sameRef reports reference identity for object-like values.
func sameRef(a, b Value) bool {
	switch x := a.(type) {
	case []Value:
		y, ok := b.([]Value)
		if !ok || len(x) != len(y) {
			return false
		}
		if len(x) == 0 {
			return unsafe.SliceData(x) == unsafe.SliceData(y) && x != nil
		}
		return &x[0] == &y[0]
	case *Object:
		y, ok := b.(*Object)
		return ok && x == y
	case *Element:
		y, ok := b.(*Element)
		return ok && x == y
	case *ActionPlan:
		y, ok := b.(*ActionPlan)
		return ok && x == y
	case *Binding:
		y, ok := b.(*Binding)
		return ok && x == y
	}
	return false
}

// StrictEqual implements JavaScript ===.
func StrictEqual(a, b Value) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	}
	return sameRef(a, b)
}

// SameValue implements Object.is.
func SameValue(a, b Value) bool {
	x, ok1 := a.(float64)
	y, ok2 := b.(float64)
	if ok1 && ok2 {
		if math.IsNaN(x) && math.IsNaN(y) {
			return true
		}
		if x == 0 && y == 0 {
			return math.Signbit(x) == math.Signbit(y)
		}
		return x == y
	}
	return StrictEqual(a, b)
}

// SameValueZero is the comparison of Array.prototype.includes and Set.
func SameValueZero(a, b Value) bool {
	x, ok1 := a.(float64)
	y, ok2 := b.(float64)
	if ok1 && ok2 && math.IsNaN(x) && math.IsNaN(y) {
		return true
	}
	return StrictEqual(a, b)
}

// LooseEqual implements ECMAScript IsLooselyEqual (==).
func LooseEqual(a, b Value) bool {
	for range 4 {
		if typeTag(a) == typeTag(b) {
			return StrictEqual(a, b)
		}
		if a == nil || b == nil {
			return false
		}
		switch x := a.(type) {
		case float64:
			if s, ok := b.(string); ok {
				return x == StringToNumber(s)
			}
		case string:
			if y, ok := b.(float64); ok {
				return StringToNumber(x) == y
			}
		}
		if bb, ok := a.(bool); ok {
			a = boolNum(bb)
			continue
		}
		if bb, ok := b.(bool); ok {
			b = boolNum(bb)
			continue
		}
		aObj, bObj := isObjectLike(a), isObjectLike(b)
		if aObj && !bObj {
			a = toPrimitive(a)
			continue
		}
		if bObj && !aObj {
			b = toPrimitive(b)
			continue
		}
		return false
	}
	return false
}

func boolNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func typeTag(v Value) int {
	switch v.(type) {
	case nil:
		return 0
	case bool:
		return 1
	case float64:
		return 2
	case string:
		return 3
	default:
		return 4
	}
}

// jsTypeName returns typeof-like names used in diagnostics ("array" for arrays).
func jsTypeName(v Value) string {
	switch v.(type) {
	case nil:
		return "object"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []Value:
		return "array"
	}
	return "object"
}

// typeofName is JavaScript typeof.
func typeofName(v Value) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	}
	return "object"
}

// jsonQuote quotes s like JSON.stringify.
func jsonQuote(s string) string {
	var b strings.Builder
	writeJSONString(&b, s)
	return b.String()
}

// numKey formats a number used as an object key or member name.
func numKey(f float64) string { return NumberToString(f) }

// itoa is strconv.Itoa.
func itoa(i int) string { return strconv.Itoa(i) }
