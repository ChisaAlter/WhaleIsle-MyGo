package lang

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Profile selects runtime behaviors where the Whale UI viewer runtime
// deliberately differs from upstream lang-core 1.0.0. The zero value is the
// upstream (spec 1.0 conformance) behavior.
type Profile struct {
	// UnicodeIdentifiers accepts Unicode letters in names (Whale lexer).
	UnicodeIdentifiers bool
	// StrictUnions validates literal values against anyOf/oneOf/allOf and
	// JSON Schema type arrays; upstream accepts any data in a composite
	// slot that is not component-only.
	StrictUnions bool
	// LiteralStateDefaults requires literal $variable defaults and reports
	// invalid-state-default (Whale); upstream lowers any expression.
	LiteralStateDefaults bool
	// CompletionDiagnostics adds incomplete-input, unresolved-reference and
	// an empty-program no-root error once the stream is complete (Whale).
	CompletionDiagnostics bool
	// RuntimeValidation checks evaluated props against their schemas at
	// render time and drops invalid values (Whale).
	RuntimeValidation bool
	// AllowData enables Query and Mutation statements. When false they are
	// reported as data-unavailable and never fetch (Whale has no business
	// tool provider).
	AllowData bool
}

// UpstreamProfile is the lang-core 1.0.0 behavior with Query/Mutation enabled.
var UpstreamProfile = Profile{AllowData: true}

// WhaleProfile is the Whale UI viewer runtime behavior.
var WhaleProfile = Profile{
	UnicodeIdentifiers:    true,
	StrictUnions:          true,
	LiteralStateDefaults:  true,
	CompletionDiagnostics: true,
	RuntimeValidation:     true,
}

// Param is a component prop or a function/action argument, positional in
// declaration order.
type Param struct {
	Name       string
	Required   bool
	Default    Value
	HasDefault bool
	// Schema is the JSON Schema fragment of the param (action $refs inlined).
	Schema *Object
	// Bindable marks a $binding<T> prop: a $variable passed to it becomes a
	// two-way Binding.
	Bindable bool
}

// Component is a library component.
type Component struct {
	Name        string
	Description string
	Signature   string
	Params      []*Param
}

// FuncEnv is the deterministic environment of a function call.
type FuncEnv struct {
	// Random returns a float in [0,1). Seeded per instance.
	Random func() float64
	// Now returns the host clock.
	Now func() time.Time
	// Location is the host time zone for calendar dates.
	Location *time.Location
	// Locale is the host locale ("en", "zh").
	Locale string
	// MaxListItems bounds produced arrays.
	MaxListItems int
}

// FuncImpl implements a library function. args holds the validated
// arguments keyed by param name. A returned error reports runtime-error and
// the call evaluates to null.
type FuncImpl func(args *Object, env *FuncEnv) (Value, error)

// Function is a library function called as @Name(...).
type Function struct {
	Name         string
	Description  string
	Signature    string
	Params       []*Param
	ParamsSchema *Object
	Returns      *Object
	Pure         bool
	Impl         FuncImpl
}

// ActionDef is a custom library action step called as @Name(...).
type ActionDef struct {
	Name         string
	Description  string
	Signature    string
	Params       []*Param
	ParamsSchema *Object
}

// Library defines the components, functions and actions a program may use.
type Library struct {
	ID      string
	Version string
	// Root is the root component used for the fallback entry.
	Root    string
	Profile Profile

	Components     map[string]*Component
	ComponentOrder []string
	Functions      map[string]*Function
	FunctionOrder  []string
	Actions        map[string]*ActionDef
	ActionOrder    []string

	// FormComponents maps a form component to the prop holding its form
	// name; "" means the form element's identity names the form.
	FormComponents map[string]string

	// Extra carries loader-specific data (for example the Whale UI genui
	// schemas); the core ignores it.
	Extra map[string]Value

	calls map[string]*callDef
}

type callKind uint8

const (
	callFunction callKind = iota
	callAction
)

type callDef struct {
	kind    callKind
	params  []*Param
	builtin bool
	lazy    bool
	fn      *Function
	action  *ActionDef
}

var bindingSig = regexp.MustCompile(`(\w+)\??: \$binding<`)

// BindableFromSignature returns the param names a prompt signature prints as
// $binding<T>.
func BindableFromSignature(sig string) map[string]bool {
	out := map[string]bool{}
	for _, m := range bindingSig.FindAllStringSubmatch(sig, -1) {
		out[m[1]] = true
	}
	return out
}

// LoadLibrarySpec builds a library from a LibrarySpec document (spec
// prompt.md section 2): {id, version, root, schema: {properties, $defs,
// functions, actions}, components: {Name: {signature, description}}, ...}.
// impls supplies function implementations by name; functions without one
// evaluate to null with a runtime-error.
func LoadLibrarySpec(data []byte, impls map[string]FuncImpl, profile Profile) (*Library, error) {
	v, err := ParseJSON(data)
	if err != nil {
		return nil, fmt.Errorf("library spec: %w", err)
	}
	doc, ok := v.(*Object)
	if !ok {
		return nil, errors.New("library spec: not an object")
	}
	schema, _ := doc.Value("schema").(*Object)
	if schema == nil {
		return nil, errors.New("library spec: missing schema")
	}
	lib := &Library{
		ID:             str(doc.Value("id")),
		Version:        str(doc.Value("version")),
		Root:           str(doc.Value("root")),
		Profile:        profile,
		Components:     map[string]*Component{},
		Functions:      map[string]*Function{},
		Actions:        map[string]*ActionDef{},
		FormComponents: map[string]string{},
		Extra:          map[string]Value{},
	}
	for _, k := range doc.Keys() {
		switch k {
		case "id", "version", "root", "schema", "components", "functions", "actions":
		default:
			lib.Extra[k] = doc.Value(k)
		}
	}
	sigs, _ := doc.Value("components").(*Object)
	fnSigs, _ := doc.Value("functions").(*Object)
	actSigs, _ := doc.Value("actions").(*Object)
	defs, _ := schema.Value("$defs").(*Object)
	props, _ := schema.Value("properties").(*Object)
	functions, _ := schema.Value("functions").(*Object)
	actions, _ := schema.Value("actions").(*Object)

	// Action $defs are data shapes, never components; inline them in slots.
	actionDefs := map[string]*Object{}
	for _, name := range append([]string{"ActionExpression", "OpenUrl", "ToAssistant"}, actions.Keys()...) {
		if body, ok := defs.Value(name).(*Object); ok && !props.Has(name) {
			actionDefs["#/$defs/"+name] = body
		}
	}
	for _, name := range defs.Keys() {
		if props != nil {
			if !props.Has(name) {
				continue
			}
		} else if _, isAction := actionDefs["#/$defs/"+name]; isAction {
			continue
		}
		def, _ := defs.Value(name).(*Object)
		c := &Component{Name: name, Description: str(def.Value("description"))}
		if s, ok := sigs.Value(name).(*Object); ok {
			c.Signature = str(s.Value("signature"))
			if d := str(s.Value("description")); d != "" && c.Description == "" {
				c.Description = d
			}
		}
		bind := BindableFromSignature(c.Signature)
		if s, ok := sigs.Value(name).(*Object); ok {
			if arr, ok := s.Value("bindable").([]Value); ok {
				for _, b := range arr {
					bind[str(b)] = true
				}
			}
		}
		c.Params = compileParams(def, actionDefs)
		for _, p := range c.Params {
			p.Bindable = bind[p.Name]
		}
		lib.Components[name] = c
		lib.ComponentOrder = append(lib.ComponentOrder, name)
	}
	for _, name := range functions.Keys() {
		f, _ := functions.Value(name).(*Object)
		ps, _ := f.Value("params").(*Object)
		fn := &Function{
			Name:         name,
			Description:  str(f.Value("description")),
			Params:       compileParams(ps, actionDefs),
			ParamsSchema: ps,
			Pure:         f.Value("pure") != false,
			Impl:         impls[name],
		}
		fn.Returns, _ = f.Value("returns").(*Object)
		if s, ok := fnSigs.Value(name).(*Object); ok {
			fn.Signature = str(s.Value("signature"))
		}
		lib.Functions[name] = fn
		lib.FunctionOrder = append(lib.FunctionOrder, name)
	}
	for _, name := range actions.Keys() {
		a, _ := actions.Value(name).(*Object)
		ps, _ := a.Value("params").(*Object)
		ad := &ActionDef{Name: name, Description: str(a.Value("description")), Params: compileParams(ps, actionDefs), ParamsSchema: ps}
		if s, ok := actSigs.Value(name).(*Object); ok {
			ad.Signature = str(s.Value("signature"))
		}
		lib.Actions[name] = ad
		lib.ActionOrder = append(lib.ActionOrder, name)
	}
	if c := lib.Components["Form"]; c != nil {
		prop := ""
		if len(c.Params) > 0 && c.Params[0].Name == "name" {
			prop = "name"
		}
		lib.FormComponents["Form"] = prop
	}
	if err := lib.init(); err != nil {
		return nil, err
	}
	return lib, nil
}

// init validates names and builds the call registry. Call it after
// building a Library by hand.
func (lib *Library) init() error {
	lib.calls = make(map[string]*callDef, len(builtinCalls)+len(lib.Functions)+len(lib.Actions))
	for k, v := range builtinCalls {
		lib.calls[k] = v
	}
	reserved := func(name string) bool {
		_, b := builtinCalls[name]
		return b || name == "Query" || name == "Mutation"
	}
	for name := range lib.Components {
		if reserved(name) {
			return fmt.Errorf("library: component %q uses a reserved name", name)
		}
	}
	for _, name := range lib.FunctionOrder {
		if reserved(name) || lib.Components[name] != nil {
			return fmt.Errorf("library: function %q uses a reserved name", name)
		}
		f := lib.Functions[name]
		lib.calls[name] = &callDef{kind: callFunction, params: f.Params, fn: f}
	}
	for _, name := range lib.ActionOrder {
		if reserved(name) || lib.Components[name] != nil {
			return fmt.Errorf("library: action %q uses a reserved name", name)
		}
		a := lib.Actions[name]
		lib.calls[name] = &callDef{kind: callAction, params: a.Params, action: a}
	}
	return nil
}

// Init finalizes a Library built by hand.
func (lib *Library) Init() error { return lib.init() }

func (lib *Library) callDef(name string) *callDef {
	if lib == nil {
		return builtinCalls[name]
	}
	return lib.calls[name]
}

// isRuntimeCall reports names that stay AST for the runtime: built-ins,
// library functions and actions, Query and Mutation.
func (lib *Library) isRuntimeCall(name string) bool {
	return lib.callDef(name) != nil || isReservedCall(name)
}

func compileParams(def *Object, actionDefs map[string]*Object) []*Param {
	props, _ := def.Value("properties").(*Object)
	req := map[string]bool{}
	if arr, ok := def.Value("required").([]Value); ok {
		for _, r := range arr {
			req[str(r)] = true
		}
	}
	out := make([]*Param, 0, props.Len())
	for _, k := range props.Keys() {
		s, _ := props.Value(k).(*Object)
		p := &Param{Name: k, Required: req[k]}
		if s != nil {
			if d, ok := s.Get("default"); ok {
				p.Default, p.HasDefault = d, true
			}
			if len(actionDefs) > 0 {
				if in, ok := inlineActionRefs(s, actionDefs, 0).(*Object); ok {
					s = in
				}
			}
		}
		p.Schema = s
		out = append(out, p)
	}
	return out
}

func inlineActionRefs(v Value, defs map[string]*Object, depth int) Value {
	if depth > 32 {
		return v
	}
	switch t := v.(type) {
	case *Object:
		if ref, ok := t.Value("$ref").(string); ok {
			if body, ok := defs[ref]; ok {
				return inlineActionRefs(body, defs, depth+1)
			}
			return t
		}
		var out *Object
		for _, k := range t.Keys() {
			x := t.vals[k]
			y := inlineActionRefs(x, defs, depth+1)
			if !sameValueOrNode(x, y) {
				if out == nil {
					out = t.Clone()
				}
				out.Set(k, y)
			}
		}
		if out != nil {
			return out
		}
		return t
	case []Value:
		var out []Value
		for i, x := range t {
			y := inlineActionRefs(x, defs, depth+1)
			if !sameValueOrNode(x, y) {
				if out == nil {
					out = append([]Value(nil), t...)
				}
				out[i] = y
			}
		}
		if out != nil {
			return out
		}
	}
	return v
}

func sameValueOrNode(a, b Value) bool {
	if isObjectLike(a) || isObjectLike(b) {
		return sameRef(a, b)
	}
	return StrictEqual(a, b)
}

func str(v Value) string {
	s, _ := v.(string)
	return s
}

// paramsSignature renders `Name(a*: string, b: number)` for hints.
func paramsSignature(name string, params []*Param) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		marked := p.Name
		if p.Required {
			marked += "*"
		}
		if t := schemaTypeName(p.Schema); t != "" {
			marked += ": " + t
		}
		parts = append(parts, marked)
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
}

// hintSignature is the error-object hint (spec 8.3).
func (lib *Library) hintSignature(component string) string {
	c := lib.Components[component]
	if c == nil {
		return ""
	}
	parts := make([]string, 0, len(c.Params))
	for _, p := range c.Params {
		if p.Required {
			parts = append(parts, p.Name+"*")
		} else {
			parts = append(parts, p.Name)
		}
	}
	return "Signature: " + component + "(" + strings.Join(parts, ", ") + ") — * marks required"
}
