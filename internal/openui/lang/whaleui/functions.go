package whaleui

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"whaleisle/internal/openui/lang"
)

// Ports of the Whale UI viewer library functions (host.cjs `$o`). Arguments
// arrive validated against each function's JSON Schema.

type args = *lang.Object
type env = *lang.FuncEnv

func num(a args, k string) float64 {
	f, _ := a.Value(k).(float64)
	return f
}

func numOr(a args, k string, d float64) float64 {
	if f, ok := a.Value(k).(float64); ok {
		return f
	}
	return d
}

func strArg(a args, k string) string {
	s, _ := a.Value(k).(string)
	return s
}

func strOr(a args, k, d string) string {
	if s, ok := a.Value(k).(string); ok {
		return s
	}
	return d
}

func list(a args, k string) []lang.Value {
	l, _ := a.Value(k).([]lang.Value)
	return l
}

func maxItems(e env) int {
	if e != nil && e.MaxListItems > 0 {
		return e.MaxListItems
	}
	return lang.DefaultLimits.MaxListItems
}

// ec is the viewer's result budget check: a finite count within the
// collection budget, truncated and clamped at zero.
func ec(e env, n float64) (int, error) {
	lim := maxItems(e)
	if math.IsNaN(n) || math.IsInf(n, 0) || n > float64(lim) {
		return 0, fmt.Errorf("Result exceeds %d items", lim)
	}
	return int(math.Max(0, math.Trunc(n))), nil
}

// vi reads a field ("" is the item itself).
func vi(v lang.Value, field string) lang.Value {
	if field == "" {
		return v
	}
	if v == nil {
		return nil
	}
	return lang.Member(v, field)
}

func clampF(x, lo, hi float64) float64 { return math.Min(math.Max(x, lo), hi) }

func finiteNumbers(vs []lang.Value) []float64 {
	out := make([]float64, 0, len(vs))
	for _, v := range vs {
		if f, ok := v.(float64); ok && !math.IsNaN(f) && !math.IsInf(f, 0) {
			out = append(out, f)
		}
	}
	return out
}

func aggregate(rows []lang.Value, field, agg string) float64 {
	if agg == "count" || field == "" {
		return float64(len(rows))
	}
	vals := make([]lang.Value, len(rows))
	for i, r := range rows {
		vals[i] = vi(r, field)
	}
	nums := finiteNumbers(vals)
	if len(nums) == 0 {
		return 0
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	switch agg {
	case "avg":
		return sum / float64(len(nums))
	case "min":
		m := nums[0]
		for _, n := range nums {
			m = math.Min(m, n)
		}
		return m
	case "max":
		m := nums[0]
		for _, n := range nums {
			m = math.Max(m, n)
		}
		return m
	}
	return sum
}

// group keys rows by a field in first-seen order (JS Map, SameValueZero).
func group(rows []lang.Value, key string) ([]lang.Value, [][]lang.Value) {
	var keys []lang.Value
	var groups [][]lang.Value
	for _, r := range rows {
		k := vi(r, key)
		idx := -1
		for i, x := range keys {
			if lang.SameValueZero(x, k) {
				idx = i
				break
			}
		}
		if idx < 0 {
			keys = append(keys, k)
			groups = append(groups, nil)
			idx = len(keys) - 1
		}
		groups[idx] = append(groups[idx], r)
	}
	return keys, groups
}

// toIntOrInf is ECMAScript ToIntegerOrInfinity.
func toIntOrInf(f float64) float64 {
	if math.IsNaN(f) {
		return 0
	}
	return math.Trunc(f)
}

// sliceBounds implements Array.prototype.slice index resolution.
func sliceBounds(n int, start lang.Value, end lang.Value) (int, int) {
	rel := func(v lang.Value, dflt float64) int {
		f := dflt
		if x, ok := v.(float64); ok {
			f = toIntOrInf(x)
		}
		ln := float64(n)
		if f < 0 {
			f = math.Max(ln+f, 0)
		} else {
			f = math.Min(f, ln)
		}
		return int(f)
	}
	s := rel(start, 0)
	e := rel(end, float64(n))
	if e < s {
		e = s
	}
	return s, e
}

func copyList(l []lang.Value) []lang.Value { return append([]lang.Value(nil), l...) }

var upperCaser = cases.Upper(language.Und)
var lowerCaser = cases.Lower(language.Und)

func fastUpper(s string) string {
	hasLower := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 {
			return upperCaser.String(s)
		}
		if c >= 'a' && c <= 'z' {
			hasLower = true
		}
	}
	if !hasLower {
		return s
	}
	buf := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		buf[i] = c
	}
	return string(buf)
}

func fastLower(s string) string {
	hasUpper := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 {
			return lowerCaser.String(s)
		}
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
	}
	if !hasUpper {
		return s
	}
	buf := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[i] = c
	}
	return string(buf)
}

func jsSplit(s, sep string) []string {
	if sep == "" {
		u := []rune(s)
		out := make([]string, 0, len(u))
		for _, r := range u {
			out = append(out, string(r))
		}
		return out
	}
	return strings.Split(s, sep)
}

// relational implements the abstract relational comparison a < b
// (undefined when either side is NaN).
func lessThan(a, b lang.Value) (bool, bool) {
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return compareUTF16(as, bs) < 0, true
	}
	x, y := relNumber(a), relNumber(b)
	if math.IsNaN(x) || math.IsNaN(y) {
		return false, false
	}
	return x < y, true
}

func relNumber(v lang.Value) float64 {
	switch t := v.(type) {
	case nil:
		return 0
	case bool, float64:
		return lang.ToNumber(t)
	case string:
		return lang.StringToNumber(t)
	case []lang.Value:
		return lang.StringToNumber(lang.ToString(t))
	}
	return math.NaN()
}

func compareUTF16(a, b string) int {
	ua, ub := []rune(a), []rune(b)
	// Code-unit order differs from code-point order only for supplementary
	// characters against U+E000..U+FFFF; map those through UTF-16.
	key := func(r rune) rune {
		if r >= 0x10000 {
			return 0xD800 + ((r - 0x10000) >> 10)
		}
		return r
	}
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			ka, kb := key(ua[i]), key(ub[i])
			if ka != kb {
				if ka < kb {
					return -1
				}
				return 1
			}
			if ua[i] < ub[i] {
				return -1
			}
			return 1
		}
	}
	return len(ua) - len(ub)
}

func ge(a, b lang.Value) bool {
	lt, ok := lessThan(a, b)
	return ok && !lt
}

func le(a, b lang.Value) bool {
	lt, ok := lessThan(b, a)
	return ok && !lt
}

var hexColor = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

func parseHex(s string) ([3]float64, error) {
	c := strings.TrimPrefix(s, "#")
	if len(c) == 3 {
		c = string([]byte{c[0], c[0], c[1], c[1], c[2], c[2]})
	}
	if !hexColor.MatchString(c) {
		return [3]float64{}, errors.New("Expected a hex color")
	}
	var out [3]float64
	for i := range 3 {
		v, _ := strconv.ParseUint(c[i*2:i*2+2], 16, 8)
		out[i] = float64(v)
	}
	return out, nil
}

var units = map[string][3]any{
	"mm": {"length", 0.001, 0.0}, "cm": {"length", 0.01, 0.0}, "m": {"length", 1.0, 0.0}, "km": {"length", 1e3, 0.0},
	"in": {"length", 0.0254, 0.0}, "ft": {"length", 0.3048, 0.0}, "yd": {"length", 0.9144, 0.0}, "mi": {"length", 1609.344, 0.0},
	"mg": {"mass", 1e-6, 0.0}, "g": {"mass", 0.001, 0.0}, "kg": {"mass", 1.0, 0.0}, "oz": {"mass", 0.028349523125, 0.0},
	"lb": {"mass", 0.45359237, 0.0}, "st": {"mass", 6.35029318, 0.0},
	"ml": {"volume", 0.001, 0.0}, "l": {"volume", 1.0, 0.0}, "tsp": {"volume", 0.00492892159375, 0.0}, "tbsp": {"volume", 0.01478676478125, 0.0},
	"floz": {"volume", 0.0295735295625, 0.0}, "cup": {"volume", 0.2365882365, 0.0}, "pt": {"volume", 0.473176473, 0.0}, "qt": {"volume", 0.946352946, 0.0},
	"gal": {"volume", 3.785411784, 0.0},
	"m/s": {"speed", 1.0, 0.0}, "km/h": {"speed", 1 / 3.6, 0.0}, "mph": {"speed", 0.44704, 0.0}, "kn": {"speed", 1852.0 / 3600, 0.0},
	"m2": {"area", 1.0, 0.0}, "km2": {"area", 1e6, 0.0}, "ft2": {"area", 0.09290304, 0.0}, "ha": {"area", 1e4, 0.0}, "acre": {"area", 4046.8564224, 0.0},
	"c": {"temperature", 1.0, 273.15}, "f": {"temperature", 5.0 / 9, 255.3722222222222}, "k": {"temperature", 1.0, 0.0},
}

func isWhite(r rune) bool { return unicode.IsSpace(r) || r == 0xFEFF }

// Functions returns the implementations keyed by library function name.
func Functions() map[string]lang.FuncImpl {
	return map[string]lang.FuncImpl{
		"Append": func(a args, e env) (lang.Value, error) {
			l := list(a, "list")
			if _, err := ec(e, float64(len(l)+1)); err != nil {
				return nil, err
			}
			return append(copyList(l), a.Value("item")), nil
		},
		"Prepend": func(a args, e env) (lang.Value, error) {
			l := list(a, "list")
			if _, err := ec(e, float64(len(l)+1)); err != nil {
				return nil, err
			}
			return append([]lang.Value{a.Value("item")}, l...), nil
		},
		"Concat": func(a args, e env) (lang.Value, error) {
			l, m := list(a, "list"), list(a, "more")
			if _, err := ec(e, float64(len(l)+len(m))); err != nil {
				return nil, err
			}
			return append(copyList(l), m...), nil
		},
		"Remove": func(a args, _ env) (lang.Value, error) {
			f, v := strArg(a, "field"), a.Value("value")
			out := []lang.Value{}
			for _, it := range list(a, "list") {
				if !lang.SameValue(vi(it, f), v) {
					out = append(out, it)
				}
			}
			return out, nil
		},
		"RemoveAt": func(a args, _ env) (lang.Value, error) {
			l := list(a, "list")
			t := num(a, "index")
			r := math.Trunc(t)
			if t < 0 {
				r = float64(len(l)) + math.Trunc(t)
			}
			out := []lang.Value{}
			for i, it := range l {
				if float64(i) != r {
					out = append(out, it)
				}
			}
			return out, nil
		},
		"Update": func(a args, _ env) (lang.Value, error) {
			f, v, sf, sv := strArg(a, "field"), a.Value("value"), strArg(a, "setField"), a.Value("setValue")
			l := list(a, "list")
			out := make([]lang.Value, len(l))
			for i, it := range l {
				if lang.SameValue(vi(it, f), v) {
					o := lang.NewObject(4)
					if src, ok := it.(*lang.Object); ok {
						o = src.Clone()
					} else if arr, ok := it.([]lang.Value); ok {
						for j, x := range arr {
							o.Set(strconv.Itoa(j), x)
						}
					}
					o.Set(sf, sv)
					out[i] = o
				} else {
					out[i] = it
				}
			}
			return out, nil
		},
		"Toggle": func(a args, _ env) (lang.Value, error) {
			l, item := list(a, "list"), a.Value("item")
			for _, it := range l {
				if lang.SameValueZero(it, item) {
					out := []lang.Value{}
					for _, x := range l {
						if !lang.SameValue(x, item) {
							out = append(out, x)
						}
					}
					return out, nil
				}
			}
			return append(copyList(l), item), nil
		},
		"Includes": func(a args, _ env) (lang.Value, error) {
			item := a.Value("item")
			if s, ok := a.Value("list").(string); ok {
				return strings.Contains(s, lang.ToString(item)), nil
			}
			for _, it := range list(a, "list") {
				if lang.SameValueZero(it, item) {
					return true, nil
				}
			}
			return false, nil
		},
		"Find": func(a args, _ env) (lang.Value, error) {
			f, v := strArg(a, "field"), a.Value("value")
			for _, it := range list(a, "list") {
				if lang.SameValue(vi(it, f), v) {
					return it, nil
				}
			}
			return nil, nil
		},
		"IndexOf": func(a args, _ env) (lang.Value, error) {
			item := a.Value("item")
			for i, it := range list(a, "list") {
				if lang.StrictEqual(it, item) {
					return float64(i), nil
				}
			}
			return -1.0, nil
		},
		"Take": func(a args, _ env) (lang.Value, error) {
			l, n := list(a, "list"), num(a, "n")
			var s, e int
			if n < 0 {
				s, e = sliceBounds(len(l), math.Trunc(n), nil)
			} else {
				s, e = sliceBounds(len(l), 0.0, math.Trunc(n))
			}
			return copyList(l[s:e]), nil
		},
		"Slice": func(a args, _ env) (lang.Value, error) {
			end := a.Value("end")
			if str, ok := a.Value("list").(string); ok {
				s, e := sliceBounds(lang.UTF16Len(str), a.Value("start"), end)
				return lang.UTF16Slice(str, s, e), nil
			}
			l := list(a, "list")
			s, e := sliceBounds(len(l), a.Value("start"), end)
			return copyList(l[s:e]), nil
		},
		"Unique": func(a args, _ env) (lang.Value, error) {
			out := []lang.Value{}
			for _, it := range list(a, "list") {
				dup := false
				for _, x := range out {
					if lang.SameValueZero(x, it) {
						dup = true
						break
					}
				}
				if !dup {
					out = append(out, it)
				}
			}
			return out, nil
		},
		"Reverse": func(a args, _ env) (lang.Value, error) {
			l := copyList(list(a, "list"))
			for i, j := 0, len(l)-1; i < j; i, j = i+1, j-1 {
				l[i], l[j] = l[j], l[i]
			}
			if l == nil {
				l = []lang.Value{}
			}
			return l, nil
		},
		"Range": func(a args, e env) (lang.Value, error) {
			start, end, step := num(a, "start"), num(a, "end"), numOr(a, "step", 1)
			if step == 0 || math.IsNaN(step) || math.IsInf(step, 0) {
				return nil, errors.New("Range step must be finite and nonzero")
			}
			if math.IsNaN(start) || math.IsInf(start, 0) || math.IsNaN(end) || math.IsInf(end, 0) {
				return nil, errors.New("Range bounds must be finite")
			}
			n, err := ec(e, math.Ceil((end-start)/step))
			if err != nil {
				return nil, err
			}
			out := make([]lang.Value, n)
			for i := range n {
				out[i] = start + float64(i)*step
			}
			return out, nil
		},
		"Pick": func(a args, e env) (lang.Value, error) {
			l := list(a, "list")
			if len(l) == 0 {
				return nil, nil
			}
			return l[int(math.Floor(e.Random()*float64(len(l))))], nil
		},
		"Shuffle": func(a args, e env) (lang.Value, error) {
			t := copyList(list(a, "list"))
			for r := len(t) - 1; r > 0; r-- {
				i := int(math.Floor(e.Random() * float64(r+1)))
				t[r], t[i] = t[i], t[r]
			}
			if t == nil {
				t = []lang.Value{}
			}
			return t, nil
		},
		"Flatten": func(a args, e env) (lang.Value, error) {
			l := list(a, "list")
			total := 0
			for _, it := range l {
				if arr, ok := it.([]lang.Value); ok {
					total += len(arr)
				} else {
					total++
				}
			}
			if _, err := ec(e, float64(total)); err != nil {
				return nil, err
			}
			out := make([]lang.Value, 0, total)
			for _, it := range l {
				if arr, ok := it.([]lang.Value); ok {
					out = append(out, arr...)
				} else {
					out = append(out, it)
				}
			}
			return out, nil
		},
		"Chunk": func(a args, e env) (lang.Value, error) {
			l := list(a, "list")
			r := math.Trunc(num(a, "size"))
			if math.IsNaN(r) || math.IsInf(r, 0) || r < 1 {
				return nil, errors.New("Chunk size must be positive and finite")
			}
			n, err := ec(e, math.Ceil(float64(len(l))/r))
			if err != nil {
				return nil, err
			}
			size := int(r)
			out := make([]lang.Value, n)
			for i := range n {
				s, end := i*size, (i+1)*size
				if end > len(l) {
					end = len(l)
				}
				out[i] = copyList(l[s:end])
			}
			return out, nil
		},
		"Pow": func(a args, _ env) (lang.Value, error) {
			return jsPow(num(a, "base"), num(a, "exponent")), nil
		},
		"Sqrt": func(a args, _ env) (lang.Value, error) {
			n := num(a, "number")
			if n < 0 {
				return nil, nil
			}
			return math.Sqrt(n), nil
		},
		"Random": func(a args, e env) (lang.Value, error) {
			lo, hi := num(a, "min"), num(a, "max")
			if b, _ := a.Value("integer").(bool); b {
				return math.Floor(e.Random()*(math.Floor(hi)-math.Ceil(lo)+1)) + math.Ceil(lo), nil
			}
			return lo + e.Random()*(hi-lo), nil
		},
		"Clamp": func(a args, _ env) (lang.Value, error) {
			return clampF(num(a, "number"), num(a, "min"), num(a, "max")), nil
		},
		"Percent": func(a args, _ env) (lang.Value, error) {
			t := num(a, "total")
			if t == 0 {
				return 0.0, nil
			}
			return num(a, "part") / t * 100, nil
		},
		"Lerp": func(a args, _ env) (lang.Value, error) {
			x, y, t := num(a, "a"), num(a, "b"), num(a, "t")
			return x + (y-x)*t, nil
		},
		"Allocate": allocate,
		"Payment": func(a args, _ env) (lang.Value, error) {
			p, rate, n := num(a, "principal"), num(a, "rate"), num(a, "periods")
			if n <= 0 {
				return nil, nil
			}
			i := rate / 100
			if i == 0 {
				return p / n, nil
			}
			return p * i / (1 - math.Pow(1+i, -n)), nil
		},
		"Convert": func(a args, _ env) (lang.Value, error) {
			from, ok1 := units[strings.ToLower(strArg(a, "from"))]
			to, ok2 := units[strings.ToLower(strArg(a, "to"))]
			if !ok1 || !ok2 || from[0] != to[0] {
				return nil, nil
			}
			v := num(a, "value")
			return (v*from[1].(float64) + from[2].(float64) - to[2].(float64)) / to[1].(float64), nil
		},
		"Format": func(a args, e env) (lang.Value, error) {
			var dec *int
			if d, ok := a.Value("decimals").(float64); ok {
				x := int(clampF(math.Trunc(d), 0, 20))
				dec = &x
			}
			return formatNumber(num(a, "value"), strOr(a, "kind", "number"), dec, strArg(a, "currency"), locale(e)), nil
		},
		"Join": func(a args, _ env) (lang.Value, error) {
			sep, field := strOr(a, "sep", ", "), strArg(a, "field")
			parts := []string{}
			for _, it := range list(a, "list") {
				v := it
				if field != "" {
					v = vi(it, field)
				}
				if v != nil {
					parts = append(parts, lang.ToString(v))
				}
			}
			return strings.Join(parts, sep), nil
		},
		"Plural": func(a args, _ env) (lang.Value, error) {
			n, word := num(a, "n"), strArg(a, "word")
			w := word + "s"
			if p, ok := a.Value("plural").(string); ok {
				w = p
			}
			if n == 1 {
				w = word
			}
			return lang.NumberToString(n) + " " + w, nil
		},
		"Pad": func(a args, e env) (lang.Value, error) {
			n, err := ec(e, num(a, "length"))
			if err != nil {
				return nil, err
			}
			s := lang.ToString(a.Value("value"))
			ch := strOr(a, "char", "0")
			cur := lang.UTF16Len(s)
			if cur >= n || ch == "" {
				return s, nil
			}
			fill := strings.Repeat(ch, (n-cur)/lang.UTF16Len(ch)+1)
			return lang.UTF16Slice(fill, 0, n-cur) + s, nil
		},
		"Upper": func(a args, _ env) (lang.Value, error) { return fastUpper(strArg(a, "text")), nil },
		"Lower": func(a args, _ env) (lang.Value, error) { return fastLower(strArg(a, "text")), nil },
		"Trim":  func(a args, _ env) (lang.Value, error) { return lang.TrimSpace(strArg(a, "text")), nil },
		"Capitalize": func(a args, _ env) (lang.Value, error) {
			rs := []rune(strArg(a, "text"))
			var b strings.Builder
			for i, r := range rs {
				if !isWhite(r) && (i == 0 || isWhite(rs[i-1])) {
					b.WriteString(upperCaser.String(string(r)))
				} else {
					b.WriteRune(r)
				}
			}
			return b.String(), nil
		},
		"Split": func(a args, _ env) (lang.Value, error) {
			out := []lang.Value{}
			for _, p := range jsSplit(strArg(a, "text"), strOr(a, "sep", ",")) {
				if t := lang.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out, nil
		},
		"Replace": func(a args, _ env) (lang.Value, error) {
			return strings.Join(jsSplit(strArg(a, "text"), strArg(a, "find")), strArg(a, "replacement")), nil
		},
		"Duration": func(a args, _ env) (lang.Value, error) {
			r := math.Max(0, math.Floor(num(a, "seconds")))
			h := math.Floor(r / 3600)
			m := math.Mod(math.Floor(r/60), 60)
			s := math.Mod(r, 60)
			p2 := func(x float64) string {
				if x < 10 {
					return "0" + lang.NumberToString(x)
				}
				return lang.NumberToString(x)
			}
			ns := lang.NumberToString
			if strOr(a, "style", "short") == "clock" {
				if h != 0 {
					return ns(h) + ":" + p2(m) + ":" + p2(s), nil
				}
				return ns(m) + ":" + p2(s), nil
			}
			switch {
			case h != 0:
				return ns(h) + "h " + ns(m) + "m", nil
			case m != 0:
				return ns(m) + "m " + p2(s) + "s", nil
			}
			return ns(s) + "s", nil
		},
		"Default": func(a args, _ env) (lang.Value, error) {
			v := a.Value("value")
			if f, ok := v.(float64); ok && (math.IsNaN(f) || math.IsInf(f, 0)) {
				return a.Value("fallback"), nil
			}
			if v == nil || v == "" {
				return a.Value("fallback"), nil
			}
			return v, nil
		},
		"MixColor": func(a args, _ env) (lang.Value, error) {
			x, err := parseHex(strArg(a, "from"))
			if err != nil {
				return nil, err
			}
			y, err := parseHex(strArg(a, "to"))
			if err != nil {
				return nil, err
			}
			t := clampF(num(a, "t"), 0, 1)
			var b strings.Builder
			b.WriteByte('#')
			for i := range 3 {
				fmt.Fprintf(&b, "%02x", int(lang.JSRound(x[i]+(y[i]-x[i])*t)))
			}
			return b.String(), nil
		},
		"Now": func(_ args, e env) (lang.Value, error) {
			return now(e).UTC().Format("2006-01-02T15:04:05.000Z"), nil
		},
		"Today": func(_ args, e env) (lang.Value, error) {
			return isoDate(now(e).In(loc(e))), nil
		},
		"FormatDate": func(a args, e env) (lang.Value, error) {
			t, err := toDate(a.Value("date"), loc(e))
			if err != nil {
				return nil, err
			}
			style := strOr(a, "style", "date")
			if style == "iso" {
				return isoDate(t), nil
			}
			return formatDate(t, style, locale(e)), nil
		},
		"AddDays": func(a args, e env) (lang.Value, error) {
			t, err := toDate(a.Value("date"), loc(e))
			if err != nil {
				return nil, err
			}
			return isoDate(t.AddDate(0, 0, int(math.Trunc(num(a, "days"))))), nil
		},
		"AddMonths": func(a args, e env) (lang.Value, error) {
			t, err := addMonths(a.Value("date"), num(a, "months"), loc(e))
			if err != nil {
				return nil, err
			}
			return isoDate(t), nil
		},
		"DaysBetween": func(a args, e env) (lang.Value, error) {
			f, err := toDate(a.Value("from"), loc(e))
			if err != nil {
				return nil, err
			}
			t, err := toDate(a.Value("to"), loc(e))
			if err != nil {
				return nil, err
			}
			return utcDays(t) - utcDays(f), nil
		},
		"MonthsBetween": func(a args, e env) (lang.Value, error) {
			r, err := toDate(a.Value("from"), loc(e))
			if err != nil {
				return nil, err
			}
			i, err := toDate(a.Value("to"), loc(e))
			if err != nil {
				return nil, err
			}
			sign := 1.0
			lo, hi := r, i
			if i.Before(r) {
				sign, lo, hi = -1, i, r
			}
			l := float64((hi.Year()-lo.Year())*12 + int(hi.Month()) - int(lo.Month()))
			back, _ := addMonths(float64(lo.UnixMilli()), l, loc(e))
			if back.After(hi) {
				l--
			}
			return sign * l, nil
		},
		"StartOf": func(a args, e env) (lang.Value, error) {
			t, err := startOf(a.Value("date"), strArg(a, "unit"), loc(e))
			if err != nil {
				return nil, err
			}
			return isoDate(t), nil
		},
		"GroupBy": func(a args, _ env) (lang.Value, error) {
			keys, groups := group(list(a, "rows"), strArg(a, "key"))
			field, agg := strArg(a, "field"), strOr(a, "agg", "sum")
			out := make([]lang.Value, len(keys))
			for i, k := range keys {
				out[i] = lang.ObjectFrom("key", k, "value", aggregate(groups[i], field, agg))
			}
			return out, nil
		},
		"Pivot": func(a args, _ env) (lang.Value, error) {
			rows := list(a, "rows")
			rowKey, colKey, field, agg := strArg(a, "rowKey"), strArg(a, "colKey"), strArg(a, "field"), strOr(a, "agg", "sum")
			var cols []lang.Value
			for _, r := range rows {
				c := vi(r, colKey)
				dup := false
				for _, x := range cols {
					if lang.SameValueZero(x, c) {
						dup = true
						break
					}
				}
				if !dup {
					cols = append(cols, c)
				}
			}
			keys, groups := group(rows, rowKey)
			out := make([]lang.Value, len(keys))
			for i, k := range keys {
				o := lang.NewObject(len(cols) + 1)
				o.Set(rowKey, k)
				for _, c := range cols {
					var sub []lang.Value
					for _, r := range groups[i] {
						if lang.SameValue(vi(r, colKey), c) {
							sub = append(sub, r)
						}
					}
					o.Set(lang.ToString(c), aggregate(sub, field, agg))
				}
				out[i] = o
			}
			return out, nil
		},
		"Top": func(a args, _ env) (lang.Value, error) {
			rows := copyList(list(a, "rows"))
			field := strArg(a, "field")
			asc := strOr(a, "direction", "desc") == "asc"
			sort.SliceStable(rows, func(i, j int) bool {
				x, y := vi(rows[i], field), vi(rows[j], field)
				c := 0
				if lt, _ := lessThan(x, y); lt {
					c = -1
				} else if gt, _ := lessThan(y, x); gt {
					c = 1
				}
				if !asc {
					c = -c
				}
				return c < 0
			})
			n := int(math.Max(0, num(a, "n")))
			if n < len(rows) {
				rows = rows[:n]
			}
			if rows == nil {
				rows = []lang.Value{}
			}
			return rows, nil
		},
		"Between": func(a args, _ env) (lang.Value, error) {
			field, lo, hi := strArg(a, "field"), a.Value("min"), a.Value("max")
			out := []lang.Value{}
			for _, r := range list(a, "rows") {
				v := vi(r, field)
				if v != nil && (lo == nil || ge(v, lo)) && (hi == nil || le(v, hi)) {
					out = append(out, r)
				}
			}
			return out, nil
		},
		"Bucket":     bucket,
		"Cumulative": cumulative,
		"MovingAvg":  movingAvg,
		"Change": func(a args, _ env) (lang.Value, error) {
			p := num(a, "previous")
			if p == 0 {
				return nil, nil
			}
			return (num(a, "current") - p) / p * 100, nil
		},
		"Histogram": histogram,
		"Stats":     stats,
	}
}

func jsPow(b, e float64) float64 {
	if math.IsNaN(e) {
		return math.NaN()
	}
	if (b == 1 || b == -1) && math.IsInf(e, 0) {
		return math.NaN()
	}
	return math.Pow(b, e)
}

func allocate(a args, e env) (lang.Value, error) {
	weights := list(a, "weights")
	if _, err := ec(e, float64(len(weights))); err != nil {
		return nil, err
	}
	total := num(a, "total")
	scale := math.Pow(10, clampF(math.Trunc(numOr(a, "decimals", 2)), 0, 10))
	sign := 1.0
	if total < 0 {
		sign = -1
	}
	ws := make([]float64, len(weights))
	sum := 0.0
	for i, w := range weights {
		ws[i] = math.Max(0, lang.ToNumber(w))
		sum += ws[i]
	}
	if len(ws) == 0 {
		return []lang.Value{}, nil
	}
	out := make([]lang.Value, len(ws))
	if sum == 0 {
		for i := range out {
			out[i] = 0.0
		}
		return out, nil
	}
	l := lang.JSRound(math.Abs(total) * scale)
	exact := make([]float64, len(ws))
	floor := make([]float64, len(ws))
	assigned := 0.0
	for i, w := range ws {
		exact[i] = l * w / sum
		floor[i] = math.Floor(exact[i])
		assigned += floor[i]
	}
	idx := make([]int, len(ws))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(x, y int) bool {
		rx, ry := exact[idx[x]]-floor[idx[x]], exact[idx[y]]-floor[idx[y]]
		if rx != ry {
			return rx > ry
		}
		return idx[x] < idx[y]
	})
	for p, d := 0, l-assigned; d > 0; p, d = p+1, d-1 {
		floor[idx[p%len(idx)]]++
	}
	for i, f := range floor {
		out[i] = sign * f / scale
	}
	return out, nil
}

func bucket(a args, e env) (lang.Value, error) {
	rows := list(a, "rows")
	dateField, unit, field, agg := strArg(a, "dateField"), strArg(a, "unit"), strArg(a, "field"), strOr(a, "agg", "sum")
	groups := map[string][]lang.Value{}
	var keys []string
	for _, r := range rows {
		t, err := startOf(vi(r, dateField), unit, loc(e))
		if err != nil {
			return nil, err
		}
		k := isoDate(t)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], r)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return []lang.Value{}, nil
	}
	out := []lang.Value{}
	last := keys[len(keys)-1]
	cur, err := toDate(keys[0], loc(e))
	if err != nil {
		return nil, err
	}
	for isoDate(cur) <= last {
		if len(out) >= maxItems(e) {
			return nil, errors.New("Date buckets exceed result budget")
		}
		d := isoDate(cur)
		h, ok := groups[d]
		if ok || agg == "sum" || agg == "count" {
			key := d
			if unit == "month" {
				key = d[:7]
			}
			out = append(out, lang.ObjectFrom("key", key, "value", aggregate(h, field, agg)))
		}
		switch unit {
		case "month":
			cur = cur.AddDate(0, 1, 0)
		case "week":
			cur = cur.AddDate(0, 0, 7)
		default:
			cur = cur.AddDate(0, 0, 1)
		}
	}
	return out, nil
}

func cumulative(a args, _ env) (lang.Value, error) {
	field := strArg(a, "field")
	r := 0.0
	rows := list(a, "rows")
	out := make([]lang.Value, len(rows))
	for i, it := range rows {
		r += numberOrZero(vi(it, field))
		out[i] = withField(it, field, r)
	}
	return out, nil
}

func numberOrZero(v lang.Value) float64 {
	n := relNumber(v)
	if math.IsNaN(n) {
		return 0
	}
	return n
}

func withField(it lang.Value, field string, v float64) lang.Value {
	if field == "" {
		return v
	}
	o := lang.NewObject(4)
	if src, ok := it.(*lang.Object); ok {
		o = src.Clone()
	}
	o.Set(field, v)
	return o
}

func movingAvg(a args, _ env) (lang.Value, error) {
	field := strArg(a, "field")
	w := int(math.Max(1, math.Trunc(num(a, "window"))))
	rows := list(a, "rows")
	out := make([]lang.Value, len(rows))
	n := 0.0
	for i, it := range rows {
		n += numberOrZero(vi(it, field))
		if i >= w {
			n -= numberOrZero(vi(rows[i-w], field))
		}
		cnt := i + 1
		if cnt > w {
			cnt = w
		}
		out[i] = withField(it, field, n/float64(cnt))
	}
	return out, nil
}

func histogram(a args, e env) (lang.Value, error) {
	vals := finiteNumbers(list(a, "values"))
	bins, err := ec(e, math.Max(1, num(a, "bins")))
	if err != nil {
		return nil, err
	}
	lo, hi := 0.0, 1.0
	if len(vals) > 0 {
		lo, hi = vals[0], vals[0]
		for _, v := range vals {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if v, ok := a.Value("min").(float64); ok {
		lo = v
	}
	if v, ok := a.Value("max").(float64); ok {
		hi = v
	}
	width := 1.0
	if hi > lo {
		width = (hi - lo) / float64(bins)
	}
	counts := make([]float64, bins)
	for _, v := range vals {
		if v >= lo && v <= hi {
			idx := int(math.Min(float64(bins-1), math.Floor((v-lo)/width)))
			counts[idx]++
		}
	}
	out := make([]lang.Value, bins)
	ns := lang.NumberToString
	for h := range bins {
		s, en := lo+float64(h)*width, lo+float64(h+1)*width
		out[h] = lang.ObjectFrom("key", ns(s)+"–"+ns(en), "value", counts[h], "start", s, "end", en)
	}
	return out, nil
}

func stats(a args, _ env) (lang.Value, error) {
	t := finiteNumbers(list(a, "values"))
	sort.Float64s(t)
	r := len(t)
	if r == 0 {
		return lang.ObjectFrom("count", 0.0, "min", 0.0, "max", 0.0, "mean", 0.0, "median", 0.0, "stdev", 0.0), nil
	}
	sum := 0.0
	for _, v := range t {
		sum += v
	}
	mean := sum / float64(r)
	n := r / 2
	median := t[n]
	if r%2 == 0 {
		median = (t[n-1] + t[n]) / 2
	}
	sq := 0.0
	for _, v := range t {
		sq += (v - mean) * (v - mean)
	}
	return lang.ObjectFrom("count", float64(r), "min", t[0], "max", t[r-1], "mean", mean, "median", median, "stdev", math.Sqrt(sq/float64(r))), nil
}
