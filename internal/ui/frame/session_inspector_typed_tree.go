package frame

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

const inspectorObjectPageSize = 50

// Expansion and paging belong to paths in the selected reader, not retained
// object values. Each frame reads the current native publication again.
type inspectorObjectTreeState struct {
	expanded map[string]bool
	limits   map[string]int
}
type inspectorObjectTreeRow struct {
	path, name, label, reference, circular string
	depth                                  int
	expandable, expanded, more             bool
}
type inspectorValueIdentity struct {
	typeOf           reflect.Type
	pointer          uintptr
	length, capacity int
}
type inspectorMapEntry struct{ Key, Value any }
type inspectorUnexportedValue struct{}

func (s *inspectorObjectTreeState) isExpanded(path string) bool {
	if expanded, ok := s.expanded[path]; ok {
		return expanded
	}
	return path == "$"
}
func (s *inspectorObjectTreeState) toggle(path string) {
	if s.expanded == nil {
		s.expanded = map[string]bool{}
	}
	s.expanded[path] = !s.isExpanded(path)
}
func (s *inspectorObjectTreeState) pageLimit(path string) int {
	if limit := s.limits[path]; limit > 0 {
		return limit
	}
	return inspectorObjectPageSize
}
func (s *inspectorObjectTreeState) more(path string) {
	if s.limits == nil {
		s.limits = map[string]int{}
	}
	s.limits[path] = s.pageLimit(path) + inspectorObjectPageSize
}

func inspectorReflectValue(value any) reflect.Value {
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}
func inspectorValueID(value any) (inspectorValueIdentity, bool) {
	v := inspectorReflectValue(value)
	if !v.IsValid() {
		return inspectorValueIdentity{}, false
	}
	switch v.Kind() {
	case reflect.Map:
		if !v.IsNil() {
			return inspectorValueIdentity{typeOf: v.Type(), pointer: uintptr(v.UnsafePointer())}, true
		}
	case reflect.Pointer:
		if !v.IsNil() && v.Pointer() != 0 {
			return inspectorValueIdentity{typeOf: v.Type(), pointer: v.Pointer()}, true
		}
	case reflect.Slice:
		if !v.IsNil() && v.Pointer() != 0 {
			return inspectorValueIdentity{typeOf: v.Type(), pointer: v.Pointer(), length: v.Len(), capacity: v.Cap()}, true
		}
	}
	return inspectorValueIdentity{}, false
}

func inspectorValueDescription(value any) (string, bool) {
	if _, unexported := value.(inspectorUnexportedValue); unexported {
		return "<unexported>", false
	}
	if raw, ok := value.(json.RawMessage); ok {
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return "undefined", false
		}
		if !json.Valid(raw) {
			return "Invalid JSON: " + text, false
		}
		switch text[0] {
		case '{':
			return "Object", true
		case '[':
			var items []json.RawMessage
			_ = json.Unmarshal(raw, &items)
			return fmt.Sprintf("Array(%d)", len(items)), true
		default:
			return text, false
		}
	}
	if value, ok := value.(time.Time); ok {
		return "time.Time(" + value.Format(time.RFC3339Nano) + ")", false
	}
	v := inspectorReflectValue(value)
	if !v.IsValid() {
		return "nil", false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return v.Type().String() + "(nil)", false
		}
		label, expandable := inspectorValueDescription(v.Elem().Interface())
		if !expandable {
			label = "&" + label
		}
		return label, expandable
	}
	switch v.Kind() {
	case reflect.Struct:
		return v.Type().String(), true
	case reflect.Map, reflect.Slice:
		if v.IsNil() {
			return v.Type().String() + "(nil)", false
		}
		return fmt.Sprintf("%s(%d)", v.Type(), v.Len()), true
	case reflect.Array:
		return v.Type().String(), true
	case reflect.String:
		return strconv.Quote(v.String()), false
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), false
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), false
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, v.Type().Bits()), false
	case reflect.Complex64, reflect.Complex128:
		return strconv.FormatComplex(v.Complex(), 'g', -1, v.Type().Bits()), false
	case reflect.Func:
		if v.IsNil() {
			return v.Type().String() + "(nil)", false
		}
		return "[" + v.Type().String() + "]", false
	case reflect.Chan, reflect.UnsafePointer:
		return v.Type().String(), false
	}
	return v.Type().String(), false
}

// Enumerate one level only. Map keys remain typed values; bytes remain indexed
// bytes. No methods, getters, user String functions, or JSON marshaling run here.
func inspectorValueFields(value any, limit int) ([]inspectorValueField, bool) {
	var fields []inspectorValueField
	appendField := func(name string, value any) bool {
		if len(fields) > limit {
			return false
		}
		fields = append(fields, inspectorValueField{name, value})
		return len(fields) <= limit
	}
	if raw, ok := value.(json.RawMessage); ok {
		text := strings.TrimSpace(string(raw))
		if strings.HasPrefix(text, "{") {
			for _, field := range inspectorFields(raw) {
				if !appendField(field.name, field.raw) {
					break
				}
			}
		} else if strings.HasPrefix(text, "[") {
			var items []json.RawMessage
			_ = json.Unmarshal(raw, &items)
			for i, item := range items {
				if !appendField(strconv.Itoa(i), item) {
					break
				}
			}
		}
		return fields[:min(len(fields), limit)], len(fields) > limit
	}
	v := inspectorReflectValue(value)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil, false
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			var value any = inspectorUnexportedValue{}
			if field.CanInterface() {
				value = field.Interface()
			}
			if !appendField(v.Type().Field(i).Name, value) {
				break
			}
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !appendField(strconv.Itoa(i), v.Index(i).Interface()) {
				break
			}
		}
	case reflect.Map:
		keys := v.MapKeys()
		// Sorting affects presentation only; it never coerces the actual key.
		sort.SliceStable(keys, func(i, j int) bool {
			a, _ := inspectorValueDescription(keys[i].Interface())
			b, _ := inspectorValueDescription(keys[j].Interface())
			return a < b
		})
		for i, key := range keys {
			if !appendField(strconv.Itoa(i), inspectorMapEntry{key.Interface(), v.MapIndex(key).Interface()}) {
				break
			}
		}
	}
	return fields[:min(len(fields), limit)], len(fields) > limit
}

func inspectorRegisteredReference(value any, objects map[string]inspectorObject) string {
	if reference, ok := value.(inspectorReference); ok {
		return reference.Key
	}
	var message *session.Message
	switch value := value.(type) {
	case session.Message:
		message = &value
	case *session.Message:
		message = value
	}
	if message != nil {
		key := "chat:" + message.ID
		if object, ok := objects[key]; ok && object.Kind == "node" {
			return key
		}
	}
	return ""
}

func (s *inspectorObjectTreeState) rows(object inspectorObject, objects map[string]inspectorObject) []inspectorObjectTreeRow {
	var rows []inspectorObjectTreeRow
	ancestors := map[inspectorValueIdentity]string{}
	var visit func(any, string, string, int, bool, []inspectorValueField)
	visit = func(value any, name, path string, depth int, root bool, extra []inspectorValueField) {
		row := inspectorObjectTreeRow{path: path, name: name, depth: depth}
		reference := inspectorRegisteredReference(value, objects)
		if !root && reference != "" {
			row.reference = reference
			if next, ok := objects[reference]; ok {
				row.label = next.Label + " · " + next.Identity
				if next.Kind == "nodeData" {
					value = next.Value
				} else {
					rows = append(rows, row)
					return
				}
			} else {
				row.label = "Unavailable · " + reference
				rows = append(rows, row)
				return
			}
		}
		if identity, ok := inspectorValueID(value); ok {
			if parent, circular := ancestors[identity]; circular {
				row.circular, row.label = parent, "↩ "+parent
				rows = append(rows, row)
				return
			}
			ancestors[identity] = path
			defer delete(ancestors, identity)
		}
		row.label, row.expandable = inspectorValueDescription(value)
		row.expanded = row.expandable && s.isExpanded(path)
		rows = append(rows, row)
		if !row.expanded {
			return
		}
		fields, more := inspectorValueFields(value, s.pageLimit(path))
		fields = append(append([]inspectorValueField(nil), extra...), fields...)
		for _, field := range fields {
			child := path + "/" + strings.ReplaceAll(strings.ReplaceAll(field.Name, "~", "~0"), "/", "~1")
			visit(field.Value, field.Name, child, depth+1, false, nil)
		}
		if len(fields) == 0 {
			rows = append(rows, inspectorObjectTreeRow{path: path + "/empty", label: "(empty)", depth: depth + 1})
		}
		if more {
			rows = append(rows, inspectorObjectTreeRow{path: path, label: paneLabel("更多…", "More…"), depth: depth + 1, more: true})
		}
	}
	visit(object.Value, "$", "$", 0, true, object.Fields)
	return rows
}

func inspectorTypedTree(c *ui.Context, state *inspectorObjectTreeState, object inspectorObject, objects map[string]inspectorObject, navigate func(string), colors theme.Colors) {
	ui.Column(c).FillWidth().Font(inspectorFont()).FontSize(11).Children(func() {
		for _, row := range state.rows(object, objects) {
			ctx := c.Key("object-field:" + row.path + ":" + row.label)
			ui.Row(ctx).FillWidth().Padding(2, 0, 2, float32(row.depth+1)*16).Gap(4).AlignItems(ui.Start).Children(func() {
				if row.more {
					button := inspectorButton(ctx, "More object entries "+row.path)
					button.Children(func() { inspectorText(ctx, row.label).TextColor(chrome.Color(colors.Accent)) })
					if button.Clicked() {
						state.more(row.path)
					}
					return
				}
				if row.reference != "" {
					label := row.label
					if next, ok := objects[row.reference]; ok {
						label = next.Label + " · " + next.Identity
					}
					button := inspectorButton(ctx.Key("reference"), "Open "+label).Tooltip(label)
					button.Children(func() { inspectorText(ctx, row.name+": ↗ "+label).TextColor(chrome.Color(colors.Accent)) })
					if button.Clicked() {
						navigate(row.reference)
					}
					if !row.expandable {
						return
					}
				}
				if row.expandable {
					marker := "▸ "
					if row.expanded {
						marker = "▾ "
					}
					name := row.name + ": "
					if row.reference != "" {
						name = ""
					}
					button := inspectorButton(ctx.Key("expand"), row.name+": "+row.label).Expanded(row.expanded).MinWidth(0)
					button.Children(func() { inspectorText(ctx, marker+name+row.label).TextColor(chrome.Color(colors.FG)) })
					if button.Clicked() {
						state.toggle(row.path)
					}
				} else {
					name := row.name
					if name != "" {
						name += ": "
					}
					ui.Text(ctx, name+row.label).FontSize(11).TextColor(chrome.Color(colors.Secondary)).Selectable()
				}
			})
		}
	})
}
