package lang

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EffectKind is the kind of host side effect an action step asks for.
type EffectKind uint8

// Effect kinds.
const (
	// EffectToAssistant sends a message to the assistant
	// (continue_conversation event).
	EffectToAssistant EffectKind = iota + 1
	// EffectOpenURL opens a validated http(s) URL (open_url event).
	EffectOpenURL
	// EffectCopyToClipboard copies Text through the host clipboard bridge
	// (the library's @CopyToClipboard action).
	EffectCopyToClipboard
	// EffectCustom is any other library action; Event.Type is its name.
	EffectCustom
	// EffectRunMutation calls a tool for a Mutation; resolve it with
	// ActionRun.ResolveMutation before calling Next again.
	EffectRunMutation
	// EffectRefetch reports that queries were scheduled again; the fetch
	// itself appears in PendingFetches. It needs no acknowledgement.
	EffectRefetch
)

// HostEvent is the event shape of spec section 6.3.
type HostEvent struct {
	Type                 string  `json:"type"`
	HumanFriendlyMessage string  `json:"humanFriendlyMessage"`
	Params               *Object `json:"params"`
	FormName             string  `json:"formName,omitempty"`
	FormState            *Object `json:"formState,omitempty"`
}

// Effect is one host side effect of an action.
type Effect struct {
	Kind  EffectKind
	Event HostEvent
	// URL is set for EffectOpenURL.
	URL string
	// Text is set for EffectCopyToClipboard.
	Text string
	// Mutation is set for EffectRunMutation.
	Mutation *MutationRequest
	// QueryID is set for EffectRefetch.
	QueryID string
}

// MutationRequest asks the host to call a tool for a Mutation.
type MutationRequest struct {
	ID   string
	Tool string
	Args Value
}

// Event and URL limits shared with the desktop protocol.
const (
	maxEventBytes     = 64 << 10
	maxAssistantBytes = 16 << 10
)

// ErrActionStopped is returned by Next after a step failed.
var ErrActionStopped = errors.New("action stopped")

// ActionRun executes one action plan step by step. Set and Reset are applied
// to the state store inside Next; host effects are returned one at a time.
// Calling Next again acknowledges the previous effect; Fail stops the run.
type ActionRun struct {
	in      *Instance
	label   string
	form    string
	steps   []planStep
	legacy  *Object
	deflt   bool
	pos     int
	done    bool
	pending *MutationRequest
	err     error
	// Diagnostics collects runtime errors of evaluated step arguments.
	Diagnostics []Diagnostic
}

// Dispatch starts an action. action is the action prop value (an
// *ActionPlan, a legacy {type, params} object, or nil for a component's
// default action, which sends label to the assistant). form is the
// triggering element's FormName.
func (in *Instance) Dispatch(label, form string, action Value) *ActionRun {
	run := &ActionRun{in: in, label: label, form: form}
	if in.stream {
		run.done = true
		return run
	}
	switch t := action.(type) {
	case *ActionPlan:
		run.steps = t.steps
	case *Object:
		run.legacy = t
	case nil:
		run.deflt = true
	default:
		run.done = true
		run.err = fmt.Errorf("value is not an action")
	}
	return run
}

// Err returns the error that stopped the run, if any.
func (r *ActionRun) Err() error { return r.err }

// Fail reports that the host could not complete the last effect; the
// remaining steps do not run.
func (r *ActionRun) Fail(err error) {
	if err == nil {
		err = errors.New("action was not acknowledged")
	}
	if r.pending != nil {
		r.ResolveMutation(nil, err)
		return
	}
	r.err = err
	r.done = true
}

// ResolveMutation delivers the result of an EffectRunMutation. A failed
// mutation stops the remaining steps.
func (r *ActionRun) ResolveMutation(data Value, err error) {
	m := r.pending
	if m == nil {
		return
	}
	r.pending = nil
	me := r.in.queries.mutations[m.ID]
	if me == nil {
		return
	}
	r.in.queries.version++
	if err != nil {
		code := "tool-error"
		msg := fmt.Sprintf("Mutation %q failed: %s", m.Tool, err.Error())
		hint := ""
		if tnf, ok := err.(*ToolNotFoundError); ok {
			code = "tool-not-found"
			msg = fmt.Sprintf("Mutation tool %q not found", m.Tool)
			if len(tnf.Available) > 0 {
				hint = "Available tools: " + strings.Join(tnf.Available, ", ")
			}
		}
		me.result = MutationResult{Status: "error", Error: err.Error()}
		me.err = &Diagnostic{Source: SourceMutation, Code: code, Message: msg, StatementID: m.ID, Component: "Mutation", ToolName: m.Tool, Hint: hint}
		r.err = err
		r.done = true
		return
	}
	me.result = MutationResult{Status: "success", Data: dataOnly(data)}
	me.err = nil
	r.in.afterStateChange()
}

func (r *ActionRun) event(typ, msg string, params *Object) HostEvent {
	if params == nil {
		params = NewObject(0)
	}
	return HostEvent{Type: typ, HumanFriendlyMessage: msg, Params: params, FormName: r.form, FormState: r.in.formState(r.form)}
}

func (r *ActionRun) fail(err error) (*Effect, error) {
	r.err = err
	r.done = true
	return nil, err
}

// Next runs local steps up to the next host effect and returns it, or nil
// when the action is finished. The error is non-nil when a step failed.
func (r *ActionRun) Next() (*Effect, error) {
	if r.pending != nil {
		return r.fail(errors.New("mutation result was not delivered"))
	}
	if r.done {
		if r.err != nil {
			return nil, r.err
		}
		return nil, nil
	}
	if r.deflt {
		r.done = true
		return r.assistant(r.label, nil)
	}
	if r.legacy != nil {
		r.done = true
		return r.legacyEffect()
	}
	for r.pos < len(r.steps) {
		st := r.steps[r.pos]
		r.pos++
		eff, err := r.runStep(st)
		if err != nil {
			return r.fail(err)
		}
		if eff != nil {
			return eff, nil
		}
	}
	r.done = true
	return nil, nil
}

func (r *ActionRun) assistant(msg string, ctx Value) (*Effect, error) {
	if strings.TrimSpace(msg) == "" || len(msg) > maxAssistantBytes {
		return r.fail(fmt.Errorf("%s: the assistant message must be 1 to %d bytes", CodeActionInvalid, maxAssistantBytes))
	}
	params := NewObject(1)
	if ctx != nil {
		params.Set("context", dataOnly(ctx))
	}
	return &Effect{Kind: EffectToAssistant, Event: r.event("continue_conversation", msg, params)}, nil
}

// ValidateExternalURL accepts an absolute http(s) URL without credentials.
func ValidateExternalURL(raw string) error {
	if raw == "" || len(raw) > maxEventBytes || !utf8.ValidString(raw) || strings.TrimSpace(raw) != raw || strings.ContainsFunc(raw, unicode.IsControl) {
		return errors.New("invalid external URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return errors.New("links require an absolute HTTP(S) URL without credentials")
	}
	return nil
}

func (r *ActionRun) openURL(raw string) (*Effect, error) {
	if err := ValidateExternalURL(raw); err != nil {
		return r.fail(fmt.Errorf("%s: %w", CodeActionInvalid, err))
	}
	return &Effect{Kind: EffectOpenURL, URL: raw, Event: r.event("open_url", "", ObjectFrom("url", raw))}, nil
}

func (r *ActionRun) legacyEffect() (*Effect, error) {
	typ, _ := r.legacy.Value("type").(string)
	if typ == "" {
		typ = "continue_conversation"
	}
	if r.in.lib.Actions[typ] != nil {
		return nil, nil // only a real call delivers a custom action
	}
	params := NewObject(2)
	if p, ok := r.legacy.Value("params").(*Object); ok {
		params = dataOnly(p).(*Object)
	}
	if u, ok := r.legacy.Value("url").(string); ok && u != "" {
		params.Set("url", u)
	}
	if c := r.legacy.Value("context"); Truthy(c) {
		params.Set("context", dataOnly(c))
	}
	switch typ {
	case "open_url":
		u, _ := params.Value("url").(string)
		return r.openURL(u)
	case "continue_conversation":
		msg := r.label
		if strings.TrimSpace(msg) == "" {
			return r.fail(fmt.Errorf("%s: empty assistant message", CodeActionInvalid))
		}
		return &Effect{Kind: EffectToAssistant, Event: r.event(typ, msg, params)}, nil
	}
	return &Effect{Kind: EffectCustom, Event: r.event(typ, r.label, params)}, nil
}

func (r *ActionRun) env() *evalEnv {
	env := r.in.newEnv(r.in.actRand)
	env.report = func(msg string) {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{Source: SourceRuntime, Code: CodeRuntimeError, Message: msg})
	}
	return env
}

func (r *ActionRun) runStep(st planStep) (eff *Effect, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			eff, err = nil, fmt.Errorf("%v", rec)
		}
	}()
	if st.data != nil {
		switch st.data.Value("type") {
		case "open_url":
			u, _ := st.data.Value("url").(string)
			return r.openURL(u)
		case "continue_conversation":
			msg, _ := st.data.Value("message").(string)
			return r.assistant(msg, st.data.Value("context"))
		}
		return nil, nil
	}
	c := st.call
	env := r.env()
	switch c.Name {
	case stepToAssistant:
		args := env.evalArgs(c, builtinCalls[stepToAssistant].params)
		msg := ""
		if m := args.Value("message"); m != nil {
			msg = ToString(m)
		}
		return r.assistant(msg, args.Value("context"))
	case stepOpenURL:
		args := env.evalArgs(c, builtinCalls[stepOpenURL].params)
		u := ""
		if v := args.Value("url"); v != nil {
			u = ToString(v)
		}
		return r.openURL(u)
	case stepSet:
		if len(c.Args) < 2 {
			return nil, errors.New("Action step is invalid")
		}
		target, ok := c.Args[0].(*StateRefNode)
		if !ok {
			return nil, errors.New("Action step is invalid")
		}
		v := dataOnly(env.eval(c.Args[1]))
		if err := r.in.store.Set(target.N, v, false); err != nil {
			return nil, err
		}
		r.in.afterStateChange()
		return nil, nil
	case stepReset:
		var targets []string
		for _, a := range c.Args {
			if s, ok := a.(*StateRefNode); ok {
				targets = append(targets, s.N)
			}
		}
		if len(targets) == 0 {
			return nil, errors.New("Action step is invalid")
		}
		for _, t := range targets {
			var d Value
			if r.in.result != nil {
				d = r.in.result.StateDefaults.Value(t)
			}
			if err := r.in.store.Set(t, d, true); err != nil {
				return nil, err
			}
		}
		r.in.afterStateChange()
		return nil, nil
	case stepRun:
		var ref *RuntimeRefNode
		if len(c.Args) > 0 {
			ref, _ = c.Args[0].(*RuntimeRefNode)
		}
		if ref == nil {
			return nil, errors.New("Action step is invalid")
		}
		if !ref.Mutation {
			r.in.Refetch(ref.N)
			return &Effect{Kind: EffectRefetch, QueryID: ref.N}, nil
		}
		return r.runMutation(ref.N, env)
	}
	def := r.in.lib.Actions[c.Name]
	if def == nil {
		return nil, errors.New("Action step is invalid")
	}
	args := env.callArgs(c, def.Params)
	if missesRequired(args, def.Params) {
		return nil, fmt.Errorf("@%s is missing required action arguments", c.Name)
	}
	checked := env.checkValue(args, def.ParamsSchema, def.paramsCompiled, c.Name, "", true)
	if isInvalid(checked) {
		return nil, fmt.Errorf("@%s action arguments are invalid", c.Name)
	}
	params, _ := checked.(*Object)
	if params == nil {
		params = args
	}
	params = dataOnly(params).(*Object)
	for _, p := range def.Params {
		if p.HasDefault && !params.Has(p.Name) {
			params.Set(p.Name, p.Default)
		}
	}
	if c.Name == "CopyToClipboard" {
		text, _ := params.Value("text").(string)
		if len(text) > maxEventBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
			return nil, fmt.Errorf("%s: invalid clipboard text", CodeActionInvalid)
		}
		return &Effect{Kind: EffectCopyToClipboard, Text: text, Event: r.event(c.Name, "", params)}, nil
	}
	return &Effect{Kind: EffectCustom, Event: r.event(c.Name, "", params)}, nil
}

func (r *ActionRun) runMutation(id string, env *evalEnv) (*Effect, error) {
	q := r.in.queries
	me := q.mutations[id]
	if me == nil {
		return nil, fmt.Errorf("mutation %q is not available", id)
	}
	if me.result.Status == "loading" {
		return nil, fmt.Errorf("mutation %q is already running", id)
	}
	var args Value = NewObject(0)
	if r.in.result != nil {
		for _, ms := range r.in.result.Mutations {
			if ms.ID == id && ms.Args != nil {
				args = dataOnly(env.eval(ms.Args))
			}
		}
	}
	me.result = MutationResult{Status: "loading"}
	q.version++
	req := &MutationRequest{ID: id, Tool: me.tool, Args: args}
	r.pending = req
	return &Effect{Kind: EffectRunMutation, Mutation: req}, nil
}

// StepInfo describes one action step with its arguments evaluated now.
type StepInfo struct {
	// Action is the step name without @: ToAssistant, OpenUrl, Set, Reset,
	// Run, or a library action name.
	Action string
	Args   []Value
}

// DescribePlan evaluates a plan's step arguments with the current state,
// for display and inspection. Invalid steps are omitted.
func (in *Instance) DescribePlan(p *ActionPlan) []StepInfo {
	if p == nil {
		return nil
	}
	out := make([]StepInfo, 0, len(p.steps))
	for _, st := range p.steps {
		info, ok := in.describeStep(st)
		if ok {
			out = append(out, info)
		}
	}
	return out
}

func (in *Instance) describeStep(st planStep) (info StepInfo, ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			ok = false
		}
	}()
	if st.data != nil {
		switch st.data.Value("type") {
		case "open_url":
			return StepInfo{Action: stepOpenURL, Args: []Value{st.data.Value("url")}}, true
		case "continue_conversation":
			args := []Value{st.data.Value("message")}
			if c := st.data.Value("context"); c != nil {
				args = append(args, c)
			}
			return StepInfo{Action: stepToAssistant, Args: args}, true
		}
		return info, false
	}
	c := st.call
	env := in.newEnv(in.actRand)
	switch c.Name {
	case stepToAssistant:
		a := env.evalArgs(c, builtinCalls[stepToAssistant].params)
		msg := ""
		if m := a.Value("message"); m != nil {
			msg = ToString(m)
		}
		args := []Value{msg}
		if ctx := a.Value("context"); ctx != nil {
			args = append(args, dataOnly(ctx))
		}
		return StepInfo{Action: stepToAssistant, Args: args}, true
	case stepOpenURL:
		a := env.evalArgs(c, builtinCalls[stepOpenURL].params)
		u := ""
		if v := a.Value("url"); v != nil {
			u = ToString(v)
		}
		return StepInfo{Action: stepOpenURL, Args: []Value{u}}, true
	case stepSet:
		if len(c.Args) < 2 {
			return info, false
		}
		t, ok := c.Args[0].(*StateRefNode)
		if !ok {
			return info, false
		}
		return StepInfo{Action: stepSet, Args: []Value{t.N, dataOnly(env.eval(c.Args[1]))}}, true
	case stepReset:
		var args []Value
		for _, a := range c.Args {
			if s, ok := a.(*StateRefNode); ok {
				args = append(args, s.N)
			}
		}
		return StepInfo{Action: stepReset, Args: args}, len(args) > 0
	case stepRun:
		if len(c.Args) > 0 {
			if r, ok := c.Args[0].(*RuntimeRefNode); ok {
				return StepInfo{Action: stepRun, Args: []Value{r.N}}, true
			}
		}
		return info, false
	}
	def := in.lib.Actions[c.Name]
	if def == nil {
		return info, false
	}
	a := env.callArgs(c, def.Params)
	args := make([]Value, len(def.Params))
	for i, p := range def.Params {
		v := a.Value(p.Name)
		if v == nil && p.HasDefault {
			v = p.Default
		}
		args[i] = dataOnly(v)
	}
	return StepInfo{Action: c.Name, Args: args}, true
}
