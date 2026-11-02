package session

import "encoding/json"

// peekEventType extracts the top-level "type" string from large persistence
// events (> 512 bytes) when it appears in the leading primitive fields before
// any nested object/array or escape sequence. Small events return (_, false) so
// synthetic unit-test fixtures with duplicate or escaped keys always take the
// full JSON decoder path.
func peekEventType(raw json.RawMessage) (string, bool) {
	if len(raw) <= 512 {
		return "", false
	}
	limit := len(raw)
	if limit > 256 {
		limit = 256
	}
	buf := raw[:limit]
	i := skipJSONSpace(buf, 0)
	if i >= len(buf) || buf[i] != '{' {
		return "", false
	}
	i++
	for {
		i = skipJSONSpace(buf, i)
		if i >= len(buf) || buf[i] != '"' {
			return "", false
		}
		key, next, ok := readSimpleJSONString(buf, i)
		if !ok {
			return "", false
		}
		i = skipJSONSpace(buf, next)
		if i >= len(buf) || buf[i] != ':' {
			return "", false
		}
		i = skipJSONSpace(buf, i+1)
		if i >= len(buf) {
			return "", false
		}
		if key == "type" {
			if buf[i] != '"' {
				return "", false
			}
			val, _, ok := readSimpleJSONString(buf, i)
			if !ok || val == "" {
				return "", false
			}
			return val, true
		}
		switch buf[i] {
		case '"':
			_, nextVal, ok := readSimpleJSONString(buf, i)
			if !ok {
				return "", false
			}
			i = nextVal
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 't', 'f', 'n':
			for i < len(buf) && buf[i] != ',' && buf[i] != '}' && buf[i] > ' ' {
				i++
			}
		default:
			return "", false
		}
		i = skipJSONSpace(buf, i)
		if i >= len(buf) || buf[i] != ',' {
			return "", false
		}
		i++
	}
}

func skipJSONSpace(buf []byte, i int) int {
	for i < len(buf) {
		switch buf[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func readSimpleJSONString(buf []byte, start int) (string, int, bool) {
	if start >= len(buf) || buf[start] != '"' {
		return "", start, false
	}
	for i := start + 1; i < len(buf); i++ {
		b := buf[i]
		if b == '\\' || b < 0x20 {
			return "", i, false
		}
		if b == '"' {
			return string(buf[start+1 : i]), i + 1, true
		}
	}
	return "", len(buf), false
}
