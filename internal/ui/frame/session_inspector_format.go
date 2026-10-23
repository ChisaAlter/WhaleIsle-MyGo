package frame

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

type inspectorField struct {
	name string
	raw  json.RawMessage
}

// Enumerate JSON.parse's own keys: array indexes first, then insertion order.
// Keep raw values so inspecting a large integer never edits the event bytes.
func inspectorFields(raw []byte) []inspectorField {
	d := json.NewDecoder(bytes.NewReader(raw))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return nil
	}
	var fields []inspectorField
	seen := map[string]int{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil
		}
		name, ok := token.(string)
		if !ok {
			return nil
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil
		}
		if at, exists := seen[name]; exists {
			fields[at].raw = value
		} else {
			seen[name] = len(fields)
			fields = append(fields, inspectorField{name, value})
		}
	}
	if _, err := d.Token(); err != nil {
		return nil
	}
	index := func(name string) (uint64, bool) {
		n, err := strconv.ParseUint(name, 10, 32)
		return n, err == nil && n < 4294967295 && strconv.FormatUint(n, 10) == name
	}
	sort.SliceStable(fields, func(i, j int) bool {
		a, ai := index(fields[i].name)
		b, bi := index(fields[j].name)
		if ai != bi {
			return ai
		}
		return ai && a < b
	})
	return fields
}

func inspectorScalar(raw []byte) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return strings.TrimSpace(string(raw))
}

func inspectorHasTime(raw []byte) bool {
	for _, field := range inspectorFields(raw) {
		if field.name == "time" {
			return string(field.raw) != "null"
		}
	}
	return false
}

func inspectorFilterQuery(query string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(query), "*"))
}

func inspectorSlice(text string, limit int) string {
	units := utf16.Encode([]rune(text))
	if len(units) <= limit {
		return text
	}
	return string(utf16.Decode(units[:limit]))
}

func inspectorPreview(raw []byte) string { return inspectorPreviewAt(bytes.TrimSpace(raw), 0) }
func inspectorPreviewAt(raw []byte, depth int) string {
	if len(raw) == 0 {
		return "undefined"
	}
	if raw[0] != '{' && raw[0] != '[' {
		if raw[0] == '"' {
			return inspectorSlice(inspectorScalar(raw), 180)
		}
		return string(raw)
	}
	var fields []inspectorField
	if raw[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return string(raw)
		}
		if depth > 1 {
			return "[" + strconv.Itoa(len(items)) + "]"
		}
		for i, item := range items {
			fields = append(fields, inspectorField{strconv.Itoa(i), item})
		}
	} else {
		if depth > 1 {
			return "{…}"
		}
		fields = inspectorFields(raw)
	}
	kind := ""
	for _, field := range fields {
		if field.name == "type" {
			kind = inspectorScalar(field.raw)
		}
	}
	var shown []inspectorField
	for _, field := range fields {
		key := strings.ToLower(field.name)
		if depth == 0 && key == "time" {
			continue
		}
		if kind == "tool-call-delta" {
			if field.name != "argumentsDelta" {
				continue
			}
		} else if kind == "reasoning-delta" {
			if field.name != "text" {
				continue
			}
		} else if key == "type" || key == "seq" || key == "sequence" {
			continue
		}
		shown = append(shown, field)
	}
	if len(shown) == 1 {
		return inspectorPreviewAt(bytes.TrimSpace(shown[0].raw), depth+1)
	}
	var parts []string
	for _, field := range shown[:min(5, len(shown))] {
		parts = append(parts, field.name+": "+inspectorPreviewAt(bytes.TrimSpace(field.raw), depth+1))
	}
	return inspectorSlice(strings.Join(parts, " · "), 240)
}
