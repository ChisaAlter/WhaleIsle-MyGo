package settings

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FieldKind selects how the generic settings form renders one key.
type FieldKind string

const (
	FieldString FieldKind = "string"
	FieldNumber FieldKind = "number"
	FieldBool   FieldKind = "bool"
	FieldEnum   FieldKind = "enum"
	FieldJSON   FieldKind = "json"
)

// Field is one editable row derived from a schemastery schema envelope.
type Field struct {
	Key         string
	Label       string
	Description string
	Kind        FieldKind
	Required    bool
	EnumOptions []string
}

type schemaEnvelope struct {
	UID  json.RawMessage            `json:"uid"`
	Refs map[string]json.RawMessage `json:"refs"`
}

type schemaNode struct {
	Type     string                     `json:"type"`
	Dict     map[string]json.RawMessage `json:"dict"`
	List     []json.RawMessage          `json:"list"`
	Inner    json.RawMessage            `json:"inner"`
	Value    json.RawMessage            `json:"value"`
	Meta     *schemaMeta                `json:"meta"`
	SKey     json.RawMessage            `json:"sKey"`
	Callback json.RawMessage            `json:"callback"`

	refs map[string]json.RawMessage `json:"-"`
}

type schemaMeta struct {
	Required    *bool  `json:"required"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// BuildFields walks a schemastery schema envelope and returns top-level object
// fields (one segment keys). Nested or unsupported shapes become JSON rows.
func BuildFields(schema json.RawMessage) ([]Field, error) {
	root, err := resolveRoot(schema)
	if err != nil {
		return nil, err
	}
	var fields []Field
	collectFields(root, "", &fields)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })
	return fields, nil
}

func collectFields(node schemaNode, prefix string, out *[]Field) {
	switch node.Type {
	case "object":
		if node.Dict == nil {
			return
		}
		keys := make([]string, 0, len(node.Dict))
		for k := range node.Dict {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child, err := resolveRef(node, node.Dict[k])
			if err != nil {
				continue
			}
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if child.Type == "object" && len(child.Dict) > 0 {
				if prefix == "" {
					*out = append(*out, fieldForNode(key, child))
				} else {
					collectFields(child, key, out)
				}
				continue
			}
			f := fieldForNode(key, child)
			*out = append(*out, f)
		}
	default:
		if prefix != "" {
			*out = append(*out, Field{Key: prefix, Label: prefix, Kind: FieldJSON})
		}
	}
}

func fieldForNode(key string, node schemaNode) Field {
	required := false
	label := key
	desc := ""
	if node.Meta != nil {
		if node.Meta.Required != nil {
			required = *node.Meta.Required
		}
		if node.Meta.Label != "" {
			label = node.Meta.Label
		}
		desc = node.Meta.Description
	}
	switch node.Type {
	case "string":
		return Field{Key: key, Label: label, Description: desc, Kind: FieldString, Required: required}
	case "number":
		return Field{Key: key, Label: label, Description: desc, Kind: FieldNumber, Required: required}
	case "boolean":
		return Field{Key: key, Label: label, Description: desc, Kind: FieldBool, Required: required}
	case "union":
		if opts := constUnionOptions(node); len(opts) > 0 {
			return Field{Key: key, Label: label, Description: desc, Kind: FieldEnum, Required: required, EnumOptions: opts}
		}
	}
	return Field{Key: key, Label: label, Description: desc, Kind: FieldJSON, Required: required}
}

func constUnionOptions(node schemaNode) []string {
	if len(node.List) == 0 {
		return nil
	}
	var opts []string
	for _, ref := range node.List {
		child, err := resolveRef(node, ref)
		if err != nil || child.Type != "const" {
			return nil
		}
		var v any
		if err := json.Unmarshal(child.Value, &v); err != nil {
			return nil
		}
		s, ok := v.(string)
		if !ok {
			return nil
		}
		opts = append(opts, s)
	}
	return opts
}

func resolveRoot(schema json.RawMessage) (schemaNode, error) {
	var env schemaEnvelope
	if err := json.Unmarshal(schema, &env); err != nil {
		return schemaNode{}, err
	}
	if len(env.UID) > 0 && len(env.Refs) > 0 {
		id := strings.Trim(string(env.UID), `"`)
		raw, ok := env.Refs[id]
		if !ok {
			return schemaNode{}, fmt.Errorf("settings schema: missing ref %q", id)
		}
		var node schemaNode
		if err := json.Unmarshal(raw, &node); err != nil {
			return schemaNode{}, err
		}
		node.refs = env.Refs
		return node, nil
	}
	var node schemaNode
	if err := json.Unmarshal(schema, &node); err != nil {
		return schemaNode{}, err
	}
	return node, nil
}

func resolveRef(ctx schemaNode, ref json.RawMessage) (schemaNode, error) {
	refs := ctx.refs
	var node schemaNode
	if err := json.Unmarshal(ref, &node); err == nil && node.Type != "" {
		node.refs = refs
		return node, nil
	}
	var idNum float64
	if err := json.Unmarshal(ref, &idNum); err == nil {
		id := strconv.FormatInt(int64(idNum), 10)
		raw, ok := refs[id]
		if !ok {
			return schemaNode{}, fmt.Errorf("settings schema: missing ref %q", id)
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return schemaNode{}, err
		}
		node.refs = refs
		return node, nil
	}
	var idStr string
	if err := json.Unmarshal(ref, &idStr); err == nil {
		raw, ok := refs[idStr]
		if !ok {
			return schemaNode{}, fmt.Errorf("settings schema: missing ref %q", idStr)
		}
		if err := json.Unmarshal(raw, &node); err != nil {
			return schemaNode{}, err
		}
		node.refs = refs
		return node, nil
	}
	return schemaNode{}, fmt.Errorf("settings schema: invalid ref")
}

// PatchValue parses a draft string for one field into JSON for settings/update.
func PatchValue(kind FieldKind, draft string) (json.RawMessage, error) {
	if kind == FieldString {
		return json.Marshal(draft)
	}
	draft = strings.TrimSpace(draft)
	switch kind {
	case FieldEnum:
		b, err := json.Marshal(draft)
		return b, err
	case FieldNumber:
		if draft == "" {
			return json.RawMessage("null"), nil
		}
		f, err := strconv.ParseFloat(draft, 64)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(f)
		return b, err
	case FieldBool:
		switch strings.ToLower(draft) {
		case "1", "true", "yes", "on":
			return json.RawMessage("true"), nil
		case "0", "false", "no", "off", "":
			return json.RawMessage("false"), nil
		default:
			return nil, fmt.Errorf("invalid boolean %q", draft)
		}
	case FieldJSON:
		if draft == "" {
			return json.RawMessage("null"), nil
		}
		if !json.Valid([]byte(draft)) {
			return nil, fmt.Errorf("invalid JSON")
		}
		return json.RawMessage(draft), nil
	default:
		return nil, fmt.Errorf("unknown field kind")
	}
}

// ValueDraft formats a JSON value at key for editing.
func ValueDraft(value json.RawMessage, key string, kind FieldKind) string {
	root, ok := draftRoot(value)
	return rootDraft(root, ok, key, kind)
}

// draftRoot decodes a namespace value once so every field can be drafted from
// it; ok is false when the value is not a JSON object.
func draftRoot(value json.RawMessage) (map[string]json.RawMessage, bool) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(value, &root); err != nil {
		return nil, false
	}
	return root, true
}

// rootDraft is ValueDraft for a value already decoded by draftRoot.
func rootDraft(root map[string]json.RawMessage, decoded bool, key string, kind FieldKind) string {
	if !decoded {
		return defaultDraft(kind)
	}
	raw, ok := root[key]
	if !ok || len(raw) == 0 {
		return defaultDraft(kind)
	}
	switch kind {
	case FieldString, FieldEnum:
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	case FieldNumber:
		var n float64
		if json.Unmarshal(raw, &n) == nil {
			return strconv.FormatFloat(n, 'f', -1, 64)
		}
	case FieldBool:
		var b bool
		if json.Unmarshal(raw, &b) == nil {
			if b {
				return "true"
			}
			return "false"
		}
	case FieldJSON:
		var v any
		if json.Unmarshal(raw, &v) == nil {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		}
	}
	return strings.TrimSpace(string(raw))
}

func defaultDraft(kind FieldKind) string {
	switch kind {
	case FieldBool:
		return "false"
	default:
		return ""
	}
}
