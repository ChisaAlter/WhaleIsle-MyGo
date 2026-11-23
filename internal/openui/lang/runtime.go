package lang

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// Options configure an Instance.
type Options struct {
	Limits Limits
	// Seed seeds Pick, Shuffle and Random. Renders draw from a fresh stream
	// seeded with Seed (stable across re-renders); actions draw from a
	// second stream that advances with every call.
	Seed uint64
	// Now is the host clock for @Now and @Today (default time.Now).
	Now func() time.Time
	// Location is the calendar time zone (default time.Local).
	Location *time.Location
	// Locale is the host locale ("en" or "zh").
	Locale string
}

// Update is one revision of the program delivered by the host.
type Update struct {
	// Revision is the owner-assigned revision (protocol Render.Revision).
	// Zero means unversioned. A revision lower than the last applied one is
	// stale and ignored.
	Revision uint64
	// Program is the full program text so far.
	Program string
	// Streaming is true while the model is still writing.
	Streaming bool
}

// Frame is an evaluated render.
type Frame struct {
	Revision uint64
	// Root is the evaluated element tree, nil when nothing renders.
	Root *Element
	// Errors are parser and runtime diagnostics. While streaming, parser
	// errors are withheld (spec section 8.2) and only budget and runtime
	// errors appear.
	Errors     []Diagnostic
	Streaming  bool
	Incomplete bool
	// StateVersion changes whenever the state store changes.
	StateVersion uint64
}

// Instance is one rendered program: streaming parser, state store, query
// cache and action execution. It performs no side effects itself; host
// effects are returned from ActionRun.Next and PendingFetches.
// An Instance is not safe for concurrent use.
type Instance struct {
	lib      *Library
	limits   Limits
	opts     Options
	sp       *StreamParser
	store    *Store
	result   *ParseResult
	revision uint64
	program  string
	stream   bool
	actRand  *rand.Rand
	queries  *queryManager
	fields   map[string]map[string]*fieldReg
	persist  *Object
	errors   []Diagnostic
}

type fieldReg struct {
	identity, name, target string
	fallback               Value
}

// NewInstance creates an instance for lib.
func NewInstance(lib *Library, opts Options) *Instance {
	lim := opts.Limits.withDefaults()
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	in := &Instance{
		lib:     lib,
		limits:  lim,
		opts:    opts,
		sp:      NewStreamParser(lib, lim),
		store:   newStore(lim.MaxStateKeys),
		actRand: rand.New(rand.NewPCG(opts.Seed, 0x9e3779b97f4a7c15)),
		fields:  map[string]map[string]*fieldReg{},
	}
	in.queries = newQueryManager(in)
	return in
}

// Library returns the instance library.
func (in *Instance) Library() *Library { return in.lib }

// Result returns the latest parse result.
func (in *Instance) Result() *ParseResult { return in.result }

// Store returns the state store.
func (in *Instance) Store() *Store { return in.store }

// Revision returns the last applied revision.
func (in *Instance) Revision() uint64 { return in.revision }

// Update applies a program revision and renders.
func (in *Instance) Update(u Update) *Frame {
	if u.Revision != 0 && u.Revision < in.revision {
		f := in.Render()
		f.Errors = append(f.Errors, Diagnostic{Source: SourceRuntime, Code: CodeStaleRevision, Message: fmt.Sprintf("revision %d is older than %d", u.Revision, in.revision)})
		return f
	}
	if len(u.Program) > in.limits.MaxProgramBytes {
		f := in.Render()
		f.Errors = append(f.Errors, Diagnostic{Source: SourceParser, Code: CodeProgramTooLarge, Message: fmt.Sprintf("program exceeds %d bytes", in.limits.MaxProgramBytes)})
		return f
	}
	if u.Revision != 0 {
		in.revision = u.Revision
	}
	in.program, in.stream = u.Program, u.Streaming
	in.result = in.sp.Set(u.Program, !u.Streaming)
	in.errors = nil
	if err := in.store.Initialize(in.result.StateDefaults, in.persist); err != nil {
		in.errors = append(in.errors, Diagnostic{Source: SourceRuntime, Code: CodeBudgetExceeded, Message: err.Error()})
	}
	in.persist = nil
	if !u.Streaming {
		in.queries.sync()
	}
	return in.Render()
}

func (in *Instance) getState(name string) Value {
	if v, ok := in.store.vals.Get(name); ok {
		return unwrapField(v)
	}
	if in.result != nil {
		return in.result.StateDefaults.Value(name)
	}
	return nil
}

func (in *Instance) resolveRef(name string) Value {
	return in.queries.result(name)
}

func (in *Instance) funcEnv(r *rand.Rand) *FuncEnv {
	return &FuncEnv{Random: r.Float64, Now: in.opts.Now, Location: in.opts.Location, Locale: in.opts.Locale, MaxListItems: in.limits.MaxListItems}
}

func (in *Instance) newEnv(r *rand.Rand) *evalEnv {
	return &evalEnv{
		lib:        in.lib,
		getState:   in.getState,
		resolveRef: in.resolveRef,
		budget:     &budget{phase: "evaluation", maxCount: in.limits.MaxEvalSteps, maxDepth: in.limits.MaxDepth},
		fenv:       in.funcEnv(r),
		limits:     &in.limits,
	}
}

// Render evaluates the current program with the current state.
func (in *Instance) Render() (frame *Frame) {
	frame = &Frame{Revision: in.revision, Streaming: in.stream, StateVersion: in.store.version}
	res := in.result
	if res == nil {
		return frame
	}
	frame.Incomplete = res.Incomplete
	if !in.stream {
		frame.Errors = append(frame.Errors, res.Errors...)
	} else {
		for _, d := range res.Errors {
			if d.Code == CodeBudgetExceeded || d.Code == CodeParseException {
				frame.Errors = append(frame.Errors, d)
			}
		}
	}
	frame.Errors = append(frame.Errors, in.errors...)
	frame.Errors = append(frame.Errors, in.queries.errors()...)
	r := &renderer{
		env:    in.newEnv(rand.New(rand.NewPCG(in.opts.Seed, 0))),
		lib:    in.lib,
		budget: &budget{phase: "render", maxCount: in.limits.MaxRenderNodes, maxDepth: in.limits.MaxDepth},
		limits: &in.limits,
	}
	defer func() {
		if rec := recover(); rec != nil {
			frame.Root = nil
			code, msg := CodeRuntimeError, fmt.Sprintf("Prop evaluation failed: %v", rec)
			if be, ok := rec.(*budgetError); ok {
				code, msg = CodeBudgetExceeded, be.Error()
			}
			frame.Errors = append(frame.Errors, Diagnostic{Source: SourceRuntime, Code: code, Message: msg})
		}
	}()
	frame.Root = r.renderRoot(res)
	if !in.stream {
		frame.Errors = append(frame.Errors, r.errors...)
	}
	return frame
}

// SetBinding writes a value through a two-way binding. It returns false
// when the write failed (key budget).
func (in *Instance) SetBinding(b *Binding, value Value) error {
	if b == nil {
		return nil
	}
	next := value
	if b.expr != nil {
		env := in.newEnv(in.actRand)
		env.extra = map[string]Value{"$value": value}
		var err error
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("%v", rec)
				}
			}()
			next = env.eval(b.expr)
		}()
		if err != nil {
			return err
		}
	}
	if err := in.store.Set(b.Target, dataOnly(next), false); err != nil {
		return err
	}
	in.afterStateChange()
	return nil
}

// SetState writes a $variable directly.
func (in *Instance) SetState(name string, value Value) error {
	if err := in.store.Set(name, value, false); err != nil {
		return err
	}
	in.afterStateChange()
	return nil
}

// SetField writes a form field. form "" is page level.
func (in *Instance) SetField(form, componentType, name string, value Value) error {
	f := &Field{Value: value, ComponentType: componentType}
	var err error
	if form == "" {
		err = in.store.Set(name, f, false)
	} else {
		cur, _ := in.store.Get(form).(*Object)
		next := cur.Clone()
		next.Set(name, f)
		err = in.store.Set(form, next, false)
	}
	if err == nil {
		in.afterStateChange()
	}
	return err
}

// FieldValue reads a form field value.
func (in *Instance) FieldValue(form, name string) Value {
	if form == "" {
		return unwrapField(in.store.Get(name))
	}
	o, _ := in.store.Get(form).(*Object)
	return unwrapField(o.Value(name))
}

// RegisterField records a mounted input for the Whale form payload: target
// is the bound $variable (if any), fallback the value shown when nothing was
// typed. The returned func unregisters it.
func (in *Instance) RegisterField(form, identity, name, target string, fallback Value) func() {
	m := in.fields[form]
	if m == nil {
		m = map[string]*fieldReg{}
		in.fields[form] = m
	}
	key := identity + "/" + name
	reg := &fieldReg{identity: identity, name: name, target: target, fallback: fallback}
	m[key] = reg
	return func() {
		if m[key] == reg {
			delete(m, key)
		}
	}
}

func (in *Instance) afterStateChange() {
	if !in.stream {
		in.queries.sync()
	}
}

// formState builds the event form payload (spec section 6.3). With
// registered fields it follows the Whale viewer (plain values of the
// registered fields, plus page $variables); otherwise upstream lang-core.
func (in *Instance) formState(form string) *Object {
	if len(in.fields) > 0 {
		return in.whaleFormState(form)
	}
	if form != "" {
		if raw, ok := in.store.Get(form).(*Object); ok {
			return ObjectFrom(form, encodeFields(raw, false))
		}
	}
	return encodeFields(in.store.Snapshot(), false).(*Object)
}

func (in *Instance) whaleFormState(form string) *Object {
	out := NewObject(4)
	forms := []string{form}
	all := form == ""
	if all {
		forms = forms[:0]
		for f := range in.fields {
			forms = append(forms, f)
		}
		sortStrings(forms)
	}
	for _, f := range forms {
		regs := in.fields[f]
		if regs == nil {
			continue
		}
		keys := make([]string, 0, len(regs))
		for k := range regs {
			keys = append(keys, k)
		}
		sortStrings(keys)
		group := NewObject(len(keys))
		for _, k := range keys {
			r := regs[k]
			if r.target != "" {
				out.Set(r.target, in.getState(r.target))
				continue
			}
			v := in.FieldValue(f, r.name)
			if v == nil {
				v = r.fallback
			}
			name := r.name
			if group.Has(name) {
				name = k
			}
			group.Set(name, v)
		}
		if group.Len() > 0 {
			if f != "" {
				out.Set(f, group)
			} else {
				for _, k := range group.Keys() {
					out.Set(k, group.Value(k))
				}
			}
		}
	}
	if all {
		for _, k := range in.store.vals.keys {
			if len(k) > 0 && k[0] == '$' {
				out.Set(k, unwrapField(in.store.vals.vals[k]))
			}
		}
	}
	return out
}

// encodeFields converts *Field values to {value, componentType} objects
// (or the persisted {__whaleField:1,...} form).
func encodeFields(v Value, persisted bool) Value {
	switch t := v.(type) {
	case *Field:
		o := NewObject(3)
		if persisted {
			o.Set("__whaleField", 1.0)
		}
		o.Set("value", t.Value)
		if t.ComponentType != "" || !persisted {
			o.Set("componentType", t.ComponentType)
		}
		return o
	case *Object:
		o := NewObject(t.Len())
		for _, k := range t.keys {
			o.Set(k, encodeFields(t.vals[k], persisted))
		}
		return o
	}
	return v
}
