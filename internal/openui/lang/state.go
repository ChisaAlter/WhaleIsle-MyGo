package lang

import (
	"errors"
	"fmt"
	"strings"
)

// StateFormat is the persisted interaction-state document written to the
// openui-state-v1 store: {"format":"whale-ui-state","version":1,"values":{}}.
const StateFormat = "whale-ui-state"

// fieldComponents are the input components whose legacy {value,
// componentType} objects are fields (Whale viewer uSe).
var fieldComponents = map[string]bool{"Input": true, "TextArea": true, "Select": true, "Checkbox": true, "RadioGroup": true, "DatePicker": true, "Slider": true, "SegmentedControl": true, "Timer": true, "Map": true}

// MarshalState encodes the persisted values. It fails when the document
// exceeds maxBytes (protocol MaxStateBytes when zero).
func MarshalState(values *Object, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultLimits.MaxStateBytes
	}
	doc := ObjectFrom("format", StateFormat, "version", 1.0, "values", encodeFields(values, true))
	raw := EncodeJSON(doc)
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("%s: %d bytes exceeds %d", CodeStateTooLarge, len(raw), maxBytes)
	}
	return raw, nil
}

// UnmarshalState decodes a persisted state document. It accepts the
// whale-ui-state format and the legacy flat object of older viewers.
func UnmarshalState(raw []byte) (*Object, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return NewObject(0), nil
	}
	v, err := ParseJSON(raw)
	if err != nil {
		return nil, err
	}
	doc, ok := v.(*Object)
	if !ok {
		return nil, errors.New("state is not an object")
	}
	if doc.Value("format") == StateFormat {
		if doc.Value("version") != 1.0 {
			return nil, errors.New("unsupported Whale UI interaction state")
		}
		vals, ok := doc.Value("values").(*Object)
		if !ok {
			return nil, errors.New("unsupported Whale UI interaction state")
		}
		return decodeFields(vals, false).(*Object), nil
	}
	if doc.Has("format") {
		return nil, errors.New("unsupported Whale UI interaction state")
	}
	// Legacy: $keys are plain values; other keys are fields or forms.
	out := NewObject(doc.Len())
	for _, k := range doc.Keys() {
		val := doc.vals[k]
		if strings.HasPrefix(k, "$") {
			out.Set(k, val)
			continue
		}
		out.Set(k, decodeFields(val, true))
	}
	return out, nil
}

func legacyField(o *Object) (*Field, bool) {
	if o.Len() != 2 || !o.Has("value") {
		return nil, false
	}
	ct, ok := o.Value("componentType").(string)
	if !ok || !fieldComponents[ct] {
		return nil, false
	}
	return &Field{Value: o.Value("value"), ComponentType: ct}, true
}

// decodeFields turns {__whaleField:1, value, componentType} objects (or
// legacy {value, componentType}) back into *Field, one form level deep.
func decodeFields(v Value, legacy bool) Value {
	o, ok := v.(*Object)
	if !ok {
		return v
	}
	if f, ok := asField(o, legacy); ok {
		return f
	}
	out := NewObject(o.Len())
	for _, k := range o.keys {
		x := o.vals[k]
		if xo, ok := x.(*Object); ok {
			if f, ok := asField(xo, legacy); ok {
				out.Set(k, f)
				continue
			}
		}
		out.Set(k, x)
	}
	return out
}

func asField(o *Object, legacy bool) (*Field, bool) {
	if o.Value("__whaleField") == 1.0 {
		ct, _ := o.Value("componentType").(string)
		return &Field{Value: o.Value("value"), ComponentType: ct}, true
	}
	if legacy {
		return legacyField(o)
	}
	return nil, false
}

// PersistState returns the persisted state document of the instance (the
// values the user produced, never pristine defaults).
func (in *Instance) PersistState() ([]byte, error) {
	return MarshalState(in.store.Persisted(), in.limits.MaxStateBytes)
}

// RestoreState applies a persisted state document. Before the first Update
// it is applied over the declared defaults at initialization; afterwards it
// is written into the store directly.
func (in *Instance) RestoreState(raw []byte) error {
	if len(raw) > in.limits.MaxStateBytes {
		return fmt.Errorf("%s: %d bytes exceeds %d", CodeStateTooLarge, len(raw), in.limits.MaxStateBytes)
	}
	vals, err := UnmarshalState(raw)
	if err != nil {
		return err
	}
	if !in.store.inited {
		in.persist = vals
		return nil
	}
	for _, k := range vals.Keys() {
		if err := in.store.Set(k, vals.vals[k], false); err != nil {
			return err
		}
	}
	in.afterStateChange()
	return nil
}
