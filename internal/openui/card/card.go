// Package card owns one native Whale UI card: the OpenUI Lang instance of a
// fenced block, render_ui result or conversation panel, its streaming
// coalescing, its persisted interaction state (openui-state-v1) and the
// execution of its actions' host effects.
//
// The native renderer (internal/ui/openui) builds views from Card.Frame and
// calls the Card for writes and clicks. Everything that waits — disk, the
// network, timers — happens here, off the build path, and ends with an
// invalidate. A Card evaluates its program only when the program or its
// state changes; reading the frame costs nothing.
package card

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/openui/lang/whaleui"
)

// StreamCoalesce is the minimum spacing of streaming program updates.
const StreamCoalesce = 40 * time.Millisecond

// persistDelay debounces interaction-state writes.
const persistDelay = 250 * time.Millisecond

// componentStateKey is the store key of renderer-local persisted state
// (Flashcard flip, Timer clock), shared with the Whale viewer.
const componentStateKey = "__whale_component_state__"

// Host executes the host effects of a card's actions. Its methods are called
// from a card goroutine, one effect at a time, only after a real click in
// the native renderer started the action.
type Host interface {
	// Submit sends text to the assistant through the card's owner (the
	// composer queue of the conversation that shows it). The host checks
	// that the owner is still current.
	Submit(ctx context.Context, text string) error
	// OpenURL opens a validated http(s) URL in the system browser.
	OpenURL(url string) error
	// Copy writes text to the clipboard.
	Copy(text string) error
	// Panel publishes a whale-ui-panels-v2 operation of a completed dsh-ui
	// spec that asks to open as the conversation panel.
	Panel(op openui.PanelOperation) error
}

// Config configures a Card.
type Config struct {
	Library *lang.Library
	Locale  string
	// StateDir and StateID locate the persisted interaction state
	// (openui-state-v1/<StateID>.json); empty disables persistence.
	StateDir, StateID string
	Host              Host
	// Invalidate asks the window for a frame; it may run on any goroutine.
	Invalidate func()
	Seed       uint64
	Now        func() time.Time
	// Coalesce overrides StreamCoalesce (tests).
	Coalesce time.Duration
	// Media and Resources load the pictures of Image and SearchImage; nil
	// leaves them as placeholders.
	Media     *media.Service
	Resources *openui.ResourceAPI
}

// Card is one rendered Whale UI program. Its methods are safe for
// concurrent use.
type Card struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc

	mu         sync.Mutex
	in         *lang.Instance
	frame      *lang.Frame
	rev        uint64
	language   string
	program    string
	streaming  bool
	fed        bool
	fedAt      time.Time
	pending    *source
	streamTime *time.Timer
	panel      bool
	panelSent  string
	persistT   *time.Timer
	dirty      bool
	busy       map[string]bool
	failures   map[string]error
	notice     error
	loaded     bool
	closed     bool
	unregister []func()
	owner      *media.Owner

	evals atomic.Uint64
}

type source struct {
	language, program string
	streaming         bool
}

// New creates a card and starts loading its persisted state.
func New(cfg Config) *Card {
	if cfg.Library == nil {
		cfg.Library, _ = whaleui.Library()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Coalesce == 0 {
		cfg.Coalesce = StreamCoalesce
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Card{cfg: cfg, ctx: ctx, cancel: cancel, busy: map[string]bool{}, failures: map[string]error{}}
	c.in = lang.NewInstance(cfg.Library, lang.Options{Seed: cfg.Seed, Now: cfg.Now, Locale: cfg.Locale})
	c.frame = &lang.Frame{}
	if cfg.StateDir == "" || cfg.StateID == "" {
		c.loaded = true
	} else {
		go c.load()
	}
	return c
}

func (c *Card) invalidate() {
	if c.cfg.Invalidate != nil {
		c.cfg.Invalidate()
	}
}

// Host is the host the card was created with.
func (c *Card) Host() Host { return c.cfg.Host }

// Evaluations counts program evaluations (parse and render passes).
func (c *Card) Evaluations() uint64 { return c.evals.Load() }

// Now is the card's clock.
func (c *Card) Now() time.Time { return c.cfg.Now() }

// Frame returns the last evaluated frame. Treat it as read-only.
func (c *Card) Frame() *lang.Frame {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frame
}

// Revision is the revision of the applied program.
func (c *Card) Revision() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rev
}

// Streaming reports whether the applied program is still being written.
func (c *Card) Streaming() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.streaming
}

// Panel reports whether the program is a dsh-ui spec that opens as the
// conversation panel instead of inline.
func (c *Card) Panel() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.panel
}

// Loaded reports whether the persisted state was read.
func (c *Card) Loaded() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loaded
}

// Notice is the card-level failure to show (state could not be read or
// saved, the panel could not be published), or nil.
func (c *Card) Notice() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.notice
}

// SetSource delivers the program text so far. Calling it every frame with
// an unchanged source does nothing. While streaming, updates are applied at
// most every Coalesce; the final (non-streaming) text applies at once.
func (c *Card) SetSource(language, program string, streaming bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	s := source{language, program, streaming}
	if c.pending != nil && *c.pending == s {
		return
	}
	if c.pending == nil && c.fed && c.language == language && c.program == program && c.streaming == streaming {
		return
	}
	if streaming && c.fed && c.streaming {
		if wait := c.cfg.Coalesce - c.cfg.Now().Sub(c.fedAt); wait > 0 {
			c.pending = &s
			if c.streamTime == nil {
				c.streamTime = time.AfterFunc(wait, c.flushStream)
			}
			return
		}
	}
	c.applyLocked(s)
}

func (c *Card) flushStream() {
	c.mu.Lock()
	c.streamTime = nil
	if c.pending == nil || c.closed {
		c.mu.Unlock()
		return
	}
	c.applyLocked(*c.pending)
	c.mu.Unlock()
	c.invalidate()
}

func (c *Card) applyLocked(s source) {
	c.pending = nil
	if c.streamTime != nil {
		c.streamTime.Stop()
		c.streamTime = nil
	}
	c.language, c.program, c.streaming = s.language, s.program, s.streaming
	c.fed, c.fedAt = true, c.cfg.Now()
	c.rev++
	prepared := whaleui.PrepareProgram(s.language, s.program, s.streaming)
	c.panel = prepared.Panel
	prev := c.frame
	f := c.in.Update(lang.Update{Revision: c.rev, Program: prepared.Program, Streaming: s.streaming})
	c.evals.Add(1)
	if len(prepared.Diagnostics) > 0 {
		f.Errors = append(prepared.Diagnostics, f.Errors...)
	}
	// A streaming revision whose root statement is not complete yet keeps the
	// previous tree on screen instead of flashing empty.
	if s.streaming && f.Root == nil && prev != nil && prev.Root != nil {
		f.Root = prev.Root
	}
	c.setFrameLocked(f)
	// The busy and failure marks belong to the elements of the old revision.
	clear(c.busy)
	clear(c.failures)
	if prepared.Panel && !s.streaming && len(prepared.Diagnostics) == 0 && c.cfg.Host != nil {
		c.publishPanelLocked(s.program, prepared.Append)
	}
}

func (c *Card) setFrameLocked(f *lang.Frame) {
	c.frame = f
	c.registerFieldsLocked(f.Root)
}

func (c *Card) renderLocked() {
	f := c.in.Render()
	c.evals.Add(1)
	if c.streaming && f.Root == nil && c.frame != nil && c.frame.Root != nil {
		f.Root = c.frame.Root
	}
	c.setFrameLocked(f)
}

// publishPanelLocked sends the completed spec (without its panel/append
// flags) to the conversation panel once per spec text.
func (c *Card) publishPanelLocked(program string, appendOp bool) {
	if c.panelSent == program {
		return
	}
	v, err := lang.ParseJSON([]byte(program))
	o, ok := v.(*lang.Object)
	if err != nil || !ok {
		return
	}
	o = o.Clone()
	o.Delete("panel")
	o.Delete("append")
	c.panelSent = program
	op := openui.PanelOperation{Append: appendOp, Spec: json.RawMessage(lang.EncodeJSON(o))}
	host := c.cfg.Host
	go func() {
		if err := host.Panel(op); err != nil {
			c.mu.Lock()
			c.notice = err
			c.mu.Unlock()
			c.invalidate()
		}
	}()
}

// ── State writes from the renderer (UI thread) ──

// SetBinding writes a bound input's value.
func (c *Card) SetBinding(b *lang.Binding, v lang.Value) {
	c.write(func() error { return c.in.SetBinding(b, v) })
}

// SetField writes an unbound input's form field.
func (c *Card) SetField(form, componentType, name string, v lang.Value) {
	c.write(func() error { return c.in.SetField(form, componentType, name, v) })
}

// FieldValue reads an unbound input's form field (nil when never written).
func (c *Card) FieldValue(form, name string) lang.Value {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.in.FieldValue(form, name)
}

// ComponentState reads renderer-local persisted state of an element.
func (c *Card) ComponentState(identity, key string) lang.Value {
	c.mu.Lock()
	defer c.mu.Unlock()
	o, _ := c.in.Store().Get(componentStateKey).(*lang.Object)
	return o.Value(identity + "/" + key)
}

// SetComponentState writes renderer-local persisted state of an element.
func (c *Card) SetComponentState(identity, key string, v lang.Value) {
	c.write(func() error {
		o, _ := c.in.Store().Get(componentStateKey).(*lang.Object)
		k := identity + "/" + key
		if old, ok := o.Get(k); ok && lang.SameValue(old, v) {
			return nil
		}
		next := o.Clone()
		next.Set(k, v)
		return c.in.SetState(componentStateKey, next)
	})
}

func (c *Card) write(fn func() error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	before := c.in.Store().Version()
	err := fn()
	changed := c.in.Store().Version() != before
	if err != nil {
		c.notice = err
	}
	if changed {
		c.renderLocked()
		c.schedulePersistLocked()
	}
	c.mu.Unlock()
	if changed || err != nil {
		c.invalidate()
	}
}

// ── Persistence ──

func (c *Card) load() {
	raw, err := openui.ReadState(c.cfg.StateDir, c.cfg.StateID)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.loaded = true
	if err != nil {
		c.notice = err
	} else if len(raw) > 2 {
		before := c.in.Store().Version()
		if err := c.in.RestoreState(raw); err != nil {
			c.notice = err
		} else if c.fed && c.in.Store().Version() != before {
			c.renderLocked()
		}
	}
	c.mu.Unlock()
	c.invalidate()
}

func (c *Card) schedulePersistLocked() {
	if c.cfg.StateDir == "" || c.cfg.StateID == "" {
		return
	}
	c.dirty = true
	if c.persistT == nil {
		c.persistT = time.AfterFunc(persistDelay, func() { c.persist() })
	}
}

func (c *Card) persist() {
	c.mu.Lock()
	c.persistT = nil
	if !c.dirty {
		c.mu.Unlock()
		return
	}
	c.dirty = false
	raw, err := c.in.PersistState()
	c.mu.Unlock()
	if err == nil {
		err = openui.SaveState(c.cfg.StateDir, c.cfg.StateID, raw)
	}
	if err != nil {
		c.mu.Lock()
		c.notice = err
		c.mu.Unlock()
		c.invalidate()
	}
}

// Close stops the card: pending state is written, timers stop and running
// actions are cancelled.
func (c *Card) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	for _, t := range []*time.Timer{c.streamTime, c.persistT} {
		if t != nil {
			t.Stop()
		}
	}
	c.streamTime, c.persistT = nil, nil
	for _, u := range c.unregister {
		u()
	}
	c.unregister = nil
	dirty := c.dirty
	c.mu.Unlock()
	if c.owner != nil {
		c.owner.Release()
	}
	c.cancel()
	if dirty {
		c.persist()
	}
}

// Flush writes pending interaction state now (tests, shutdown).
func (c *Card) Flush() {
	c.mu.Lock()
	if c.persistT != nil {
		c.persistT.Stop()
		c.persistT = nil
	}
	c.mu.Unlock()
	c.persist()
}

// ── Form field registration ──

// fieldName is the Whale viewer's field name: the label, else the bound
// prop's name.
var fieldProps = map[string]string{
	"Input": "value", "TextArea": "value", "Select": "value", "SegmentedControl": "value",
	"Slider": "value", "RadioGroup": "value", "DatePicker": "value", "Checkbox": "checked",
	"Timer": "seconds",
}

// FieldName returns the form field name of an input element and its value
// prop, or "" when el is not an input.
func FieldName(el *lang.Element) (name, prop string) {
	prop = fieldProps[el.Type]
	if prop == "" {
		return "", ""
	}
	if l, ok := el.Prop("label").(string); ok {
		return l, prop
	}
	return prop, prop
}

func walkElementProps(v lang.Value, visit func(*lang.Element)) {
	switch t := v.(type) {
	case *lang.Element:
		visit(t)
	case []lang.Value:
		for _, x := range t {
			walkElementProps(x, visit)
		}
	}
}

// registerFieldsLocked records the inputs of the tree for the form payload
// of actions (Whale viewer semantics: every mounted input).
func (c *Card) registerFieldsLocked(root *lang.Element) {
	for _, u := range c.unregister {
		u()
	}
	c.unregister = c.unregister[:0]
	var walk func(el *lang.Element)
	walk = func(el *lang.Element) {
		if el == nil {
			return
		}
		if name, prop := FieldName(el); name != "" {
			target := ""
			var fallback lang.Value
			switch v := el.Prop(prop).(type) {
			case *lang.Binding:
				target = v.Target
			default:
				fallback = v
			}
			c.unregister = append(c.unregister, c.in.RegisterField(el.FormName, el.Identity(), name, target, fallback))
		}
		if el.Props != nil {
			for _, k := range el.Props.Keys() {
				walkElementProps(el.Props.Value(k), walk)
			}
		}
	}
	walk(root)
}

// ── Actions ──

// Busy reports whether the action started by element identity is running.
func (c *Card) Busy(identity string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.busy[identity]
}

// Failure is why the last action of element identity stopped, or nil.
func (c *Card) Failure(identity string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures[identity]
}

// ErrOwnerChanged stops an action whose card changed after the click.
var ErrOwnerChanged = errors.New("OpenUI action owner changed before execution")

// Dispatch runs the action of a clicked element. revision is the frame
// revision the click was built on: a click on an older revision is ignored,
// so a click can never act on content the user did not see. One click
// starts one run; clicks on the same element while it runs are ignored.
// Local steps (@Set, @Reset) apply at once; host effects run in order on a
// card goroutine.
func (c *Card) Dispatch(revision uint64, identity, label, form string, action lang.Value) {
	c.mu.Lock()
	if c.closed || revision != c.rev || c.busy[identity] || c.streaming {
		c.mu.Unlock()
		return
	}
	delete(c.failures, identity)
	run := c.in.Dispatch(label, form, action)
	c.mu.Unlock()
	c.drive(run, identity, revision, false)
}

// drive runs local steps until the next host effect, which continues on a
// goroutine.
func (c *Card) drive(run *lang.ActionRun, identity string, revision uint64, async bool) {
	for {
		c.mu.Lock()
		before := c.in.Store().Version()
		eff, err := run.Next()
		if c.in.Store().Version() != before {
			c.renderLocked()
			c.schedulePersistLocked()
		}
		if err == nil && eff != nil && c.rev != revision {
			err = ErrOwnerChanged
			run.Fail(err)
		}
		if err != nil || eff == nil {
			if err != nil {
				c.failures[identity] = err
			}
			delete(c.busy, identity)
			c.mu.Unlock()
			c.invalidate()
			return
		}
		switch eff.Kind {
		case lang.EffectRefetch, lang.EffectCustom:
			c.mu.Unlock()
			continue
		case lang.EffectRunMutation:
			run.Fail(errors.New(lang.CodeDataUnavailable))
			c.mu.Unlock()
			continue
		}
		c.busy[identity] = true
		c.mu.Unlock()
		if !async {
			c.invalidate()
			go func() {
				if c.execute(run, eff) {
					c.drive(run, identity, revision, true)
				} else {
					c.drive(run, identity, revision, true)
				}
			}()
			return
		}
		c.execute(run, eff)
	}
}

// execute performs one host effect; a failure stops the run.
func (c *Card) execute(run *lang.ActionRun, eff *lang.Effect) bool {
	host := c.cfg.Host
	var err error
	switch {
	case host == nil:
		err = errors.New("OpenUI native actions are unavailable on this surface")
	case c.ctx.Err() != nil:
		err = c.ctx.Err()
	default:
		switch eff.Kind {
		case lang.EffectToAssistant:
			err = host.Submit(c.ctx, AssistantText(eff.Event))
		case lang.EffectOpenURL:
			if err = lang.ValidateExternalURL(eff.URL); err == nil {
				err = host.OpenURL(eff.URL)
			}
		case lang.EffectCopyToClipboard:
			err = host.Copy(eff.Text)
		}
	}
	if err != nil {
		c.mu.Lock()
		run.Fail(err)
		c.mu.Unlock()
		return false
	}
	return true
}

// AssistantText is the message a continue_conversation event submits: the
// human-friendly message, then the form state and context as the Whale
// viewer path sent them (openui.Event.ActionText).
func AssistantText(e lang.HostEvent) string {
	ev := openui.Event{Message: e.HumanFriendlyMessage}
	if e.FormState != nil && e.FormState.Len() > 0 {
		ev.FormState = json.RawMessage(lang.EncodeJSON(e.FormState))
	}
	if ctx := e.Params.Value("context"); ctx != nil {
		ev.Context = json.RawMessage(lang.EncodeJSON(ctx))
	}
	return ev.ActionText()
}
