package lang

import (
	"math"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Built-in step and reserved names (spec section 1.3).
const (
	stepRun         = "Run"
	stepToAssistant = "ToAssistant"
	stepOpenURL     = "OpenUrl"
	stepSet         = "Set"
	stepReset       = "Reset"
)

type builtinFn func(args []Value) Value

type builtinFunc struct {
	params []string
	fn     builtinFn
}

func argAt(a []Value, i int) Value {
	if i < len(a) {
		return a[i]
	}
	return nil
}

var builtinFuncs = map[string]*builtinFunc{
	"Count": {[]string{"array"}, func(a []Value) Value {
		if arr, ok := argAt(a, 0).([]Value); ok {
			return float64(len(arr))
		}
		return 0.0
	}},
	"First": {[]string{"array"}, func(a []Value) Value {
		if arr, ok := argAt(a, 0).([]Value); ok && len(arr) > 0 {
			return arr[0]
		}
		return nil
	}},
	"Last": {[]string{"array"}, func(a []Value) Value {
		if arr, ok := argAt(a, 0).([]Value); ok && len(arr) > 0 {
			return arr[len(arr)-1]
		}
		return nil
	}},
	"Sum": {[]string{"numbers"}, func(a []Value) Value {
		arr, ok := argAt(a, 0).([]Value)
		if !ok {
			return 0.0
		}
		return sumOf(arr)
	}},
	"Avg": {[]string{"numbers"}, func(a []Value) Value {
		arr, ok := argAt(a, 0).([]Value)
		if !ok || len(arr) == 0 {
			return 0.0
		}
		return sumOf(arr) / float64(len(arr))
	}},
	"Min": {[]string{"numbers"}, func(a []Value) Value {
		return reduceMinMax(argAt(a, 0), math.Min)
	}},
	"Max": {[]string{"numbers"}, func(a []Value) Value {
		return reduceMinMax(argAt(a, 0), math.Max)
	}},
	"Sort":   {[]string{"array", "field", "direction"}, builtinSort},
	"Filter": {[]string{"array", "field", "operator", "value"}, builtinFilter},
	"Round": {[]string{"number", "decimals"}, func(a []Value) Value {
		num := ToNumber(argAt(a, 0))
		d := 0.0
		if dec := argAt(a, 1); dec != nil {
			d = ToNumber(dec)
		}
		f := math.Pow(10, d)
		return jsRound(num*f) / f
	}},
	"Abs":   {[]string{"number"}, func(a []Value) Value { return math.Abs(ToNumber(argAt(a, 0))) }},
	"Floor": {[]string{"number"}, func(a []Value) Value { return math.Floor(ToNumber(argAt(a, 0))) }},
	"Ceil":  {[]string{"number"}, func(a []Value) Value { return math.Ceil(ToNumber(argAt(a, 0))) }},
}

// builtinCalls is the base call registry of every library.
var builtinCalls = func() map[string]*callDef {
	m := map[string]*callDef{}
	params := func(names ...string) []*Param {
		out := make([]*Param, len(names))
		for i, n := range names {
			out[i] = &Param{Name: n}
		}
		return out
	}
	for name, b := range builtinFuncs {
		m[name] = &callDef{kind: callFunction, params: params(b.params...), builtin: true}
	}
	m["Each"] = &callDef{kind: callFunction, params: params("array", "varName", "template"), builtin: true, lazy: true}
	m[stepRun] = &callDef{kind: callAction, params: params("ref"), builtin: true, lazy: true}
	m[stepToAssistant] = &callDef{kind: callAction, params: params("message", "context"), builtin: true}
	m[stepOpenURL] = &callDef{kind: callAction, params: params("url"), builtin: true}
	m[stepSet] = &callDef{kind: callAction, params: params("variable", "value"), builtin: true, lazy: true}
	m[stepReset] = &callDef{kind: callAction, params: params("variable"), builtin: true, lazy: true}
	m["Action"] = &callDef{kind: callAction, params: []*Param{{Name: "steps", Required: true}}, builtin: true, lazy: true}
	return m
}()

// BuiltinNames lists the reserved built-in functions and steps.
func BuiltinNames() []string {
	out := make([]string, 0, len(builtinCalls))
	for k := range builtinCalls {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func isBuiltinName(name string) bool {
	_, ok := builtinCalls[name]
	return ok
}

func isReservedCall(name string) bool { return name == "Query" || name == "Mutation" }

func isStepName(name string) bool {
	d := builtinCalls[name]
	return d != nil && d.kind == callAction
}

func sumOf(arr []Value) float64 {
	s := 0.0
	for _, x := range arr {
		s += ToNumber(x)
	}
	return s
}

func reduceMinMax(v Value, op func(a, b float64) float64) Value {
	arr, ok := v.([]Value)
	if !ok || len(arr) == 0 {
		return 0.0
	}
	acc := ToNumber(arr[0])
	for _, x := range arr {
		n := ToNumber(x)
		if math.IsNaN(acc) || math.IsNaN(n) {
			acc = math.NaN()
			continue
		}
		acc = op(acc, n)
	}
	return acc
}

// resolveField reads a dot path on a value.
func resolveField(v Value, path string) Value {
	if path == "" || v == nil {
		return nil
	}
	if !strings.Contains(path, ".") {
		return memberOf(v, path)
	}
	cur := v
	for _, p := range strings.Split(path, ".") {
		if cur == nil {
			return nil
		}
		cur = memberOf(cur, p)
	}
	return cur
}

// memberOf reads an own property: object keys, array indices and length,
// string indices and length.
func memberOf(v Value, key string) Value {
	switch t := v.(type) {
	case *Object:
		return t.Value(key)
	case []Value:
		if key == "length" {
			return float64(len(t))
		}
		if i, ok := arrayIndex(key); ok && i < uint64(len(t)) {
			return t[i]
		}
	case string:
		u := utf16Units(t)
		if key == "length" {
			return float64(len(u))
		}
		if i, ok := arrayIndex(key); ok && i < uint64(len(u)) {
			return utf16String(u[i : i+1])
		}
	}
	return nil
}

func isNumericValue(v Value) bool {
	switch t := v.(type) {
	case float64:
		return true
	case string:
		return t != "" && !math.IsNaN(StringToNumber(t))
	}
	return false
}

var localeCollator = collate.New(language.Und)

// localeCompare approximates ECMAScript String.prototype.localeCompare with
// the root (CLDR) collation: "a" < "b" < "B" < "é" < "Z".
func localeCompare(a, b string) int {
	collatorMu.Lock()
	defer collatorMu.Unlock()
	return localeCollator.CompareString(a, b)
}

func builtinSort(a []Value) Value {
	arr, ok := argAt(a, 0).([]Value)
	if !ok {
		return argAt(a, 0)
	}
	f := ""
	if fv := argAt(a, 1); fv != nil {
		f = ToString(fv)
	}
	desc := false
	if dv := argAt(a, 2); dv != nil {
		desc = ToString(dv) == "desc"
	}
	n := len(arr)
	out := make([]Value, n)
	copy(out, arr)
	if n <= 1 {
		return out
	}
	if f == "" {
		sort.SliceStable(out, func(i, j int) bool {
			return sortCompare(out[i], out[j], desc) < 0
		})
		return out
	}
	var keyBuf [8]Value
	var keys []Value
	if n <= len(keyBuf) {
		keys = keyBuf[:n]
	} else {
		keys = make([]Value, n)
	}
	for i, it := range out {
		keys[i] = resolveField(it, f)
	}
	// Insertion sort for small slices avoids interface/closure boxing overhead.
	if n <= 16 {
		for i := 1; i < n; i++ {
			kv, ov := keys[i], out[i]
			j := i - 1
			for j >= 0 && sortCompare(kv, keys[j], desc) < 0 {
				keys[j+1] = keys[j]
				out[j+1] = out[j]
				j--
			}
			keys[j+1] = kv
			out[j+1] = ov
		}
		return out
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		return sortCompare(keys[idx[i]], keys[idx[j]], desc) < 0
	})
	for i, pos := range idx {
		out[i] = arr[pos]
	}
	return out
}

func sortCompare(av, bv Value, desc bool) int {
	var c float64
	if isNumericValue(av) && isNumericValue(bv) {
		c = ToNumber(av) - ToNumber(bv)
	} else {
		as, bs := "", ""
		if av != nil {
			as = ToString(av)
		}
		if bv != nil {
			bs = ToString(bv)
		}
		c = float64(localeCompare(as, bs))
	}
	if desc {
		c = -c
	}
	switch {
	case c < 0:
		return -1
	case c > 0:
		return 1
	}
	return 0
}

func builtinFilter(a []Value) Value {
	arr, ok := argAt(a, 0).([]Value)
	if !ok {
		return []Value{}
	}
	f := ""
	if fv := argAt(a, 1); fv != nil {
		f = ToString(fv)
	}
	op := "=="
	if ov := argAt(a, 2); ov != nil {
		op = ToString(ov)
	}
	val := argAt(a, 3)
	out := make([]Value, 0, len(arr))
	for _, item := range arr {
		v := item
		if f != "" {
			v = resolveField(item, f)
		}
		keep := false
		switch op {
		case "==":
			keep = LooseEqual(v, val)
		case "!=":
			keep = !LooseEqual(v, val)
		case ">":
			keep = ToNumber(v) > ToNumber(val)
		case "<":
			keep = ToNumber(v) < ToNumber(val)
		case ">=":
			keep = ToNumber(v) >= ToNumber(val)
		case "<=":
			keep = ToNumber(v) <= ToNumber(val)
		case "contains":
			vs, ws := "", ""
			if v != nil {
				vs = ToString(v)
			}
			if val != nil {
				ws = ToString(val)
			}
			keep = strings.Contains(vs, ws)
		}
		if keep {
			out = append(out, item)
		}
	}
	return out
}
