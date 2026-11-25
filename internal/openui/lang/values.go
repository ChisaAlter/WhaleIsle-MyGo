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

const (
	smallObjCap  = 3
	mapThreshold = 16
)

// Object is an ordered string-keyed map that follows ECMAScript property
// order: array-index keys in ascending numeric order first, then the other
// keys in insertion order.
type Object struct {
	keys  []string
	vals  []Value
	index map[string]int
	kbuf  [smallObjCap]string
	vbuf  [smallObjCap]Value
}

// NewObject returns an empty object with room for n keys.
func NewObject(n int) *Object {
	o := &Object{}
	if n <= smallObjCap {
		o.keys = o.kbuf[:0]
		o.vals = o.vbuf[:0]
		return o
	}
	o.keys = make([]string, 0, n)
	o.vals = make([]Value, 0, n)
	if n > mapThreshold {
		o.index = make(map[string]int, n)
	}
	return o
}

func (o *Object) find(key string) int {
	if o == nil {
		return -1
	}
	if o.index != nil {
		if i, ok := o.index[key]; ok {
			return i
		}
		return -1
	}
	for i, k := range o.keys {
		if k == key {
			return i
		}
	}
	return -1
}

func (o *Object) buildIndex() {
	m := make(map[string]int, len(o.keys)+4)
	for i, k := range o.keys {
		m[k] = i
	}
	o.index = m
}

// appendOrdered appends a key known to be unique and already in property order.
func (o *Object) appendOrdered(key string, v Value) {
	pos := len(o.keys)
	o.keys = append(o.keys, key)
	o.vals = append(o.vals, v)
	if o.index != nil {
		o.index[key] = pos
	} else if len(o.keys) > mapThreshold {
		o.buildIndex()
	}
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
	if i := o.find(key); i >= 0 {
		return o.vals[i], true
	}
	return nil, false
}

// Value returns the value of key, or nil.
func (o *Object) Value(key string) Value {
	if i := o.find(key); i >= 0 {
		return o.vals[i]
	}
	return nil
}

// Has reports whether key is present.
func (o *Object) Has(key string) bool {
	return o.find(key) >= 0
}

// Set sets key to v, inserting it in ECMAScript order when it is new.
func (o *Object) Set(key string, v Value) {
	if i := o.find(key); i >= 0 {
		o.vals[i] = v
		return
	}
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
		o.vals = append(o.vals, nil)
		copy(o.keys[pos+1:], o.keys[pos:])
		copy(o.vals[pos+1:], o.vals[pos:])
		o.keys[pos] = key
		o.vals[pos] = v
		if o.index != nil {
			for j := pos; j < len(o.keys); j++ {
				o.index[o.keys[j]] = j
			}
		} else if len(o.keys) > mapThreshold {
			o.buildIndex()
		}
		return
	}
	pos := len(o.keys)
	o.keys = append(o.keys, key)
	o.vals = append(o.vals, v)
	if o.index != nil {
		o.index[key] = pos
	} else if len(o.keys) > mapThreshold {
		o.buildIndex()
	}
}

// Delete removes key.
func (o *Object) Delete(key string) {
	i := o.find(key)
	if i < 0 {
		return
	}
	copy(o.keys[i:], o.keys[i+1:])
	o.keys[len(o.keys)-1] = ""
	o.keys = o.keys[:len(o.keys)-1]
	copy(o.vals[i:], o.vals[i+1:])
	o.vals[len(o.vals)-1] = nil
	o.vals = o.vals[:len(o.vals)-1]
	if o.index != nil {
		delete(o.index, key)
		for j := i; j < len(o.keys); j++ {
			o.index[o.keys[j]] = j
		}
	}
}

// Clone returns a shallow copy.
func (o *Object) Clone() *Object {
	if o == nil {
		return NewObject(0)
	}
	n := len(o.keys)
	c := &Object{}
	if n < smallObjCap {
		c.keys = c.kbuf[:n]
		c.vals = c.vbuf[:n]
	} else {
		c.keys = make([]string, n, n+1)
		c.vals = make([]Value, n, n+1)
	}
	copy(c.keys, o.keys)
	copy(c.vals, o.vals)
	if n > mapThreshold {
		c.buildIndex()
	}
	return c
}

// arrayIndex reports whether key is a canonical array index (0 .. 2^32-2).
func arrayIndex(key string) (uint64, bool) {
	if key == "" || len(key) > 10 {
		return 0, false
	}
	c0 := key[0]
	if c0 < '0' || c0 > '9' {
		return 0, false
	}
	if len(key) > 1 && c0 == '0' {
		return 0, false
	}
	n := uint64(c0 - '0')
	for i := 1; i < len(key); i++ {
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
const (
	elMaterialized uint8 = iota
	elEphemeral
	elConsumed
)

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
	kind    uint8
	memo    *renderMemo
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
