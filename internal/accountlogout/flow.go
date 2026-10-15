// Package accountlogout owns the transient, explicitly reviewed account sign-out.
// It does not stop tasks, terminate the application, or read stored credentials.
package accountlogout

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type Client interface {
	Call(context.Context, string, any, any) error
}
type Impact string

const (
	ImpactIdle               Impact = "idle"
	ImpactRunning            Impact = "running"
	ImpactUnknown            Impact = "unknown"
	ErrorIdentityUnavailable        = "identity-unavailable"
	ErrorAccountChanged             = "account-changed"
	ErrorImpactChanged              = "impact-changed"
	ErrorFailed                     = "failed"
	ErrorOutcomeUnknown             = "outcome-unknown"
)

type State struct {
	Open, Busy, Unknown, Completed bool
	Impact                         Impact
	AccountID, Error               string
	ReviewID                       uint64
}

// Flow binds reviews to a single pointer-backed RPC client and safe account ID.
// Bind must be called when the Host changes; no endpoint/token is retained here.
type Flow struct {
	mu       sync.Mutex
	client   Client
	epoch    uint64
	sequence uint64
	state    State
	cancel   context.CancelFunc
	notify   func()
}

// SetNotify connects transient busy/review changes to the native UI. Callbacks
// run without the flow lock and may safely read Snapshot.
func (f *Flow) SetNotify(notify func()) { f.mu.Lock(); f.notify = notify; f.mu.Unlock() }
func (f *Flow) changed() {
	f.mu.Lock()
	notify := f.notify
	f.mu.Unlock()
	if notify != nil {
		notify()
	}
}

func (f *Flow) Bind(client Client) {
	defer f.changed()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.client == client {
		return
	}
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	f.epoch++
	f.client = client
	f.state = State{}
}
func (f *Flow) Snapshot() State { f.mu.Lock(); defer f.mu.Unlock(); return f.state }
func (f *Flow) Cancel(reviewID uint64) bool {
	defer f.changed()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.Busy || !f.state.Open || f.state.ReviewID != reviewID {
		return false
	}
	f.state = State{}
	return true
}

type operation struct {
	ctx    context.Context
	cancel context.CancelFunc
	client Client
	epoch  uint64
	review State
}

func (f *Flow) begin(ctx context.Context, kind string, reviewID uint64) (operation, bool) {
	defer f.changed()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.client == nil || f.state.Busy {
		return operation{}, false
	}
	if kind == "request" {
		if f.state.Open {
			return operation{}, false
		}
		f.sequence++
		f.state = State{Open: true, Impact: ImpactUnknown, ReviewID: f.sequence}
	} else if !f.state.Open || f.state.ReviewID != reviewID || f.state.AccountID == "" || (kind == "confirm" && f.state.Unknown) || (kind == "check" && !f.state.Unknown) {
		return operation{}, false
	}
	review := f.state
	f.state.Busy = true
	f.state.Error = ""
	opctx, cancel := context.WithCancel(ctx)
	f.cancel = cancel
	return operation{opctx, cancel, f.client, f.epoch, review}, true
}
func (f *Flow) current(o operation) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.epoch == o.epoch && f.client == o.client && o.ctx.Err() == nil
}
func (f *Flow) finish(o operation, state State) {
	defer f.changed()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.epoch != o.epoch || f.client != o.client {
		return
	}
	state.Busy = false
	if state.Open {
		f.sequence++
		state.ReviewID = f.sequence
	}
	f.state = state
	f.cancel = nil
}
func call(o operation, method string, args, out any) error {
	ctx, cancel := context.WithTimeout(o.ctx, 10*time.Second)
	defer cancel()
	return o.client.Call(ctx, method, args, out)
}

type accountView struct {
	Status  string `json:"status"`
	Attempt *struct {
		ID    string `json:"id"`
		Phase string `json:"phase"`
	} `json:"attempt"`
}

func identity(o operation, meta any) (string, bool, error) {
	var view accountView
	if err := call(o, "account/getState", map[string]any{}, &view); err != nil {
		return "", false, err
	}
	if view.Status == "signed-out" {
		return "", true, nil
	}
	if view.Status != "credential-stored" {
		return "", false, errors.New("unknown account state")
	}
	if view.Attempt != nil {
		switch view.Attempt.Phase {
		case "initializing", "waiting-browser", "exchanging", "committing":
			return "", false, errors.New("account sign-in is active")
		}
	}
	var profile struct {
		Status string `json:"status"`
		Value  struct {
			ID *string `json:"id"`
		} `json:"value"`
	}
	if err := call(o, "account/getProfile", map[string]any{"client": meta}, &profile); err != nil {
		return "", false, err
	}
	if profile.Status != "ready" || profile.Value.ID == nil || strings.TrimSpace(*profile.Value.ID) == "" {
		return "", false, errors.New("safe account identity unavailable")
	}
	return *profile.Value.ID, false, nil
}
func impact(o operation) Impact {
	var running bool
	if err := call(o, "account/hasRunningAccountTasks", map[string]any{}, &running); err != nil {
		return ImpactUnknown
	}
	if running {
		return ImpactRunning
	}
	return ImpactIdle
}

// Request reads the exact account task-impact RPC used by AccountMenu. It never
// signs out; failed impact inspection remains an explicit unknown warning.
func (f *Flow) Request(ctx context.Context, meta any) {
	o, ok := f.begin(ctx, "request", 0)
	if !ok {
		return
	}
	defer o.cancel()
	id, signedOut, err := identity(o, meta)
	if err != nil {
		f.finish(o, State{Error: ErrorIdentityUnavailable})
		return
	}
	if signedOut {
		f.finish(o, State{Completed: true})
		return
	}
	i := impact(o)
	f.finish(o, State{Open: true, AccountID: id, Impact: i})
}

// Confirm is exclusively the user's current confirmation button. Re-read safe
// identity and impact before the write; a changed review needs a fresh click.
func (f *Flow) Confirm(ctx context.Context, meta any, reviewID uint64) {
	o, ok := f.begin(ctx, "confirm", reviewID)
	if !ok {
		return
	}
	defer o.cancel()
	id, signedOut, err := identity(o, meta)
	if err != nil {
		f.finish(o, State{Error: ErrorIdentityUnavailable})
		return
	}
	if signedOut {
		f.finish(o, State{Completed: true})
		return
	}
	if id != o.review.AccountID {
		f.finish(o, State{Error: ErrorAccountChanged})
		return
	}
	i := impact(o)
	if i != o.review.Impact {
		s := o.review
		s.Impact = i
		s.Error = ErrorImpactChanged
		f.finish(o, s)
		return
	}
	if !f.current(o) {
		return
	}
	var receipt accountView
	err = call(o, "account/signOut", map[string]any{"client": meta}, &receipt)
	if err == nil && receipt.Status == "signed-out" {
		f.finish(o, State{Completed: true})
		return
	}
	// Errors, lost responses, and malformed success receipts do not establish
	// whether the write happened. Read back; never automatically send it again.
	f.reconcile(o, meta)
}

// Check is the only action admitted while a previous write outcome is unknown.
// It is read-only even when the account is confirmed to remain signed in.
func (f *Flow) Check(ctx context.Context, meta any, reviewID uint64) {
	o, ok := f.begin(ctx, "check", reviewID)
	if !ok {
		return
	}
	defer o.cancel()
	f.reconcile(o, meta)
}
func (f *Flow) reconcile(o operation, meta any) {
	if !f.current(o) {
		return
	}
	id, signedOut, err := identity(o, meta)
	if err != nil {
		s := o.review
		s.Unknown = true
		s.Error = ErrorOutcomeUnknown
		f.finish(o, s)
		return
	}
	if signedOut {
		f.finish(o, State{Completed: true})
		return
	}
	if id != o.review.AccountID {
		f.finish(o, State{Error: ErrorAccountChanged})
		return
	}
	s := o.review
	s.Unknown = false
	s.Error = ErrorFailed
	f.finish(o, s)
}
