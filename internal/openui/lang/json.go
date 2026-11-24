package lang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

// ParseJSON decodes JSON into language values, keeping object key order
// (objects become *Object, numbers float64).
func ParseJSON(data []byte) (Value, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeJSONValue(dec, 0)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("trailing data after JSON value")
	}
	return v, nil
}

const maxJSONDepth = 200

func decodeJSONValue(dec *json.Decoder, depth int) (Value, error) {
	if depth > maxJSONDepth {
		return nil, errors.New("JSON nesting too deep")
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := NewObject(4)
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k, ok := kt.(string)
				if !ok {
					return nil, errors.New("invalid object key")
				}
				v, err := decodeJSONValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				o.Set(k, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return o, nil
		case '[':
			arr := make([]Value, 0, 4)
			for dec.More() {
				v, err := decodeJSONValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	case json.Number:
		f := StringToNumber(string(t))
		return f, nil
	case string, bool, nil:
		return t, nil
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

// FromGo converts encoding/json style values (map[string]any, []any,
// float64, json.Number, ints) into language values. Map keys are sorted.
func FromGo(v any) Value {
	switch t := v.(type) {
	case nil, bool, string, float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		return StringToNumber(string(t))
	case []any:
		out := make([]Value, len(t))
		for i, x := range t {
			out[i] = FromGo(x)
		}
		return out
	case *Object:
		return t
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sortStrings(keys)
		o := NewObject(len(keys))
		for _, k := range keys {
			o.Set(k, FromGo(t[k]))
		}
		return o
	}
	return v
}

// MarshalJSON encodes the object with its key order.
func (o *Object) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	writeJSON(&b, o, 0)
	return []byte(b.String()), nil
}

// EncodeJSON encodes a value like JSON.stringify. Elements encode as
// {"type","props"}, action plans as {"actions":[...]} and bindings as
// {"$binding": target}; unevaluated nodes encode as null.
func EncodeJSON(v Value) []byte {
	var b strings.Builder
	writeJSON(&b, v, 0)
	return []byte(b.String())
}

func writeJSON(b *strings.Builder, v Value, depth int) {
	if depth > maxJSONDepth {
		b.WriteString("null")
		return
	}
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			b.WriteString("null")
		} else {
			b.WriteString(NumberToString(t))
		}
	case string:
		writeJSONString(b, t)
	case []Value:
		b.WriteByte('[')
		for i, x := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, x, depth+1)
		}
		b.WriteByte(']')
	case *Object:
		b.WriteByte('{')
		for i, k := range t.Keys() {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSONString(b, k)
			b.WriteByte(':')
			writeJSON(b, t.vals[k], depth+1)
		}
		b.WriteByte('}')
	case *Element:
		b.WriteString(`{"type":`)
		writeJSONString(b, t.Type)
		b.WriteString(`,"props":`)
		writeJSON(b, t.Props, depth+1)
		b.WriteByte('}')
	case *ActionPlan:
		b.WriteString(`{"actions":`)
		writeJSON(b, float64(len(t.steps)), depth+1)
		b.WriteByte('}')
	case *Field:
		writeJSON(b, encodeFields(t, false), depth+1)
	case *Binding:
		b.WriteString(`{"$binding":`)
		writeJSONString(b, t.Target)
		b.WriteByte('}')
	default:
		b.WriteString("null")
	}
}

func writeJSONString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch c {
			case '"':
				b.WriteString(`\"`)
			case '\\':
				b.WriteString(`\\`)
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			case '\b':
				b.WriteString(`\b`)
			case '\f':
				b.WriteString(`\f`)
			default:
				if c < 0x20 {
					fmt.Fprintf(b, `\u%04x`, c)
				} else {
					b.WriteByte(c)
				}
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteString(`\ufffd`)
		} else {
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	b.WriteByte('"')
}

// decodeJSONStringLiteral decodes a double-quoted JSON string literal with
// JSON.parse rules (no raw control characters, only JSON escapes).
func decodeJSONStringLiteral(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	body := raw[1 : len(raw)-1]
	if !strings.ContainsAny(body, "\\\"") {
		for i := 0; i < len(body); i++ {
			if body[i] < 0x20 {
				return "", false
			}
		}
		return body, true
	}
	var b strings.Builder
	b.Grow(len(body))
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c < 0x20 || c == '"' {
			return "", false
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(body) {
			return "", false
		}
		switch body[i] {
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case '/':
			b.WriteByte('/')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			r, ok := hex4(body, i+1)
			if !ok {
				return "", false
			}
			i += 4
			if r >= 0xD800 && r < 0xDC00 && i+6 < len(body)+0 && i+2 < len(body) && body[i+1] == '\\' && body[i+2] == 'u' {
				if r2, ok := hex4(body, i+3); ok && r2 >= 0xDC00 && r2 < 0xE000 {
					r = 0x10000 + (r-0xD800)<<10 + (r2 - 0xDC00)
					i += 6
				}
			}
			b.WriteRune(r)
		default:
			return "", false
		}
	}
	return b.String(), true
}

func hex4(s string, at int) (rune, bool) {
	if at+4 > len(s) {
		return 0, false
	}
	var r rune
	for _, c := range s[at : at+4] {
		d := digitVal(c)
		if d < 0 {
			return 0, false
		}
		r = r<<4 | rune(d)
	}
	return r, true
}
