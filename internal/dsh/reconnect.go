package dsh

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Backoff controls reconnect delay caps after a carrier loss.
type Backoff struct {
	Initial, Max time.Duration
	Factor       float64
}

func (b Backoff) normalized() Backoff {
	if b.Initial <= 0 {
		b.Initial = 500 * time.Millisecond
	}
	if b.Max <= 0 {
		b.Max = 10 * time.Second
	}
	if b.Factor < 1 {
		b.Factor = 2
	}
	return b
}

var ErrReconnectRequested = errors.New("dsh mux: manual reconnect requested")

type muxAttempt struct {
	ctx        context.Context
	cancel     context.CancelCauseFunc
	ready      chan struct{}
	readySet   bool // guarded by Mux.maintainMu
	generation <-chan struct{}
	watchDone  chan struct{}
	started    time.Time
	deadline   time.Time
}
type muxMaintainer struct {
	ctx      context.Context
	requests chan struct{}
	attempt  *muxAttempt // guarded by Mux.maintainMu
	stopping bool
}

// RequestReconnect interrupts the current attempt or retry delay, retaining
// the same Mux. It returns false without an active MaintainMux owner.
func (m *Mux) RequestReconnect() bool {
	m.maintainMu.Lock()
	defer m.maintainMu.Unlock()
	o := m.maintainer
	if o == nil || o.stopping || o.ctx.Err() != nil {
		return false
	}
	if o.attempt != nil {
		o.attempt.cancel(ErrReconnectRequested)
	}
	select {
	case o.requests <- struct{}{}:
	default:
	}
	return true
}

// BindGeneration registers the existing events pump's completion. A rejected
// caller must not start that pump. Retirement waits for its Retire/join cleanup.
func (m *Mux) BindGeneration(ctx context.Context, done <-chan struct{}) bool {
	m.maintainMu.Lock()
	defer m.maintainMu.Unlock()
	o := m.maintainer
	if o == nil || o.stopping || o.attempt == nil || o.attempt.ctx != ctx || ctx.Err() != nil || done == nil || o.attempt.generation != nil {
		return false
	}
	o.attempt.generation = done
	return true
}

// MarkReady accepts the current generation's first $events Ready only.
func (m *Mux) MarkReady(ctx context.Context) bool {
	m.maintainMu.Lock()
	defer m.maintainMu.Unlock()
	o := m.maintainer
	if o == nil || o.stopping || o.attempt == nil || o.attempt.ctx != ctx || ctx.Err() != nil || o.attempt.readySet {
		return false
	}
	if !o.attempt.deadline.IsZero() && !time.Now().Before(o.attempt.deadline) {
		return false
	}
	o.attempt.readySet = true
	close(o.attempt.ready)
	return true
}

// MaintainMux preserves the socket-ready callback contract for existing users.
// Native $events owners use MaintainMuxWithRecovery for generation readiness.
func MaintainMux(ctx context.Context, mux *Mux, backoff Backoff, onReady func(context.Context) error, onLost func(error)) error {
	return maintainMux(ctx, mux, backoff.normalized(), RecoveryConfig{}, false, onReady, onLost, nil)
}

// MaintainMuxWithRecovery includes Dial and $events readiness in the Host's
// readiness deadline. Warning receipts are logs, not another UI state source.
func MaintainMuxWithRecovery(ctx context.Context, mux *Mux, config RecoveryConfig, onReady func(context.Context) error, onLost func(error), onWarning func(error)) error {
	if err := config.validate(); err != nil {
		return err
	}
	return maintainMux(ctx, mux, config.backoff(), config, true, onReady, onLost, onWarning)
}

func maintainMux(ctx context.Context, mux *Mux, b Backoff, config RecoveryConfig, requireReady bool, onReady func(context.Context) error, onLost func(error), onWarning func(error)) error {
	o := &muxMaintainer{ctx: ctx, requests: make(chan struct{}, 1)}
	mux.maintainMu.Lock()
	if mux.maintainer != nil {
		mux.maintainMu.Unlock()
		return errors.New("dsh mux: already maintained")
	}
	mux.maintainer = o
	mux.maintainMu.Unlock()
	lost := make(chan error, 1)
	mux.mu.Lock()
	prev := mux.OnCarrierLost
	previousDialTimeout := mux.DialTimeout
	if requireReady && mux.DialTimeout <= 0 {
		mux.DialTimeout = time.Duration(config.GenerationReadyTimeoutMS) * time.Millisecond
	}
	mux.OnCarrierLost = func(err error) {
		if prev != nil {
			prev(err)
		}
		select {
		case lost <- err:
		default:
		}
	}
	mux.mu.Unlock()
	defer func() {
		mux.maintainMu.Lock()
		o.stopping = true
		mux.maintainMu.Unlock()
		_ = mux.closeCarrier(true)
		mux.mu.Lock()
		mux.OnCarrierLost = prev
		mux.DialTimeout = previousDialTimeout
		mux.mu.Unlock()
		mux.maintainMu.Lock()
		mux.maintainer = nil
		mux.maintainMu.Unlock()
	}()
	notify := func(err error) {
		if onLost != nil {
			onLost(err)
		}
	}
	pending := func() bool {
		select {
		case <-o.requests:
			return true
		default:
			return false
		}
	}
	retire := func(a *muxAttempt) {
		a.cancel(context.Canceled)
		mux.maintainMu.Lock()
		done := a.generation
		if o.attempt == a {
			o.attempt = nil
		}
		mux.maintainMu.Unlock()
		_ = mux.closeCarrier(true)
		if done != nil {
			<-done
		}
		<-a.watchDone
		mux.Reset()
	}
	delay := b.Initial
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-lost:
		default:
		}
		attempt, cancel := context.WithCancelCause(ctx)
		a := &muxAttempt{ctx: attempt, cancel: cancel, ready: make(chan struct{}), watchDone: make(chan struct{}), started: time.Now()}
		if requireReady {
			a.deadline = a.started.Add(time.Duration(config.GenerationReadyTimeoutMS) * time.Millisecond)
		}
		mux.maintainMu.Lock()
		o.attempt = a
		mux.maintainMu.Unlock()
		if requireReady {
			go mux.watchReadiness(o, a, config, onWarning)
		} else {
			close(a.watchDone)
		}
		err, manual, fatal := runMuxAttempt(ctx, mux, a, o.requests, lost, pending, requireReady, onReady)
		if errors.Is(err, ErrReconnectRequested) {
			manual = true
			pending()
		}
		select {
		case <-a.ready:
			delay = b.Initial
		default:
		}
		retire(a)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if fatal {
			return err
		}
		notify(err)
		if manual {
			delay = b.Initial
			continue
		}
		manual, err = waitMuxRetry(ctx, o.requests, jitterDelay(delay))
		if err != nil {
			return err
		}
		if manual {
			delay = b.Initial
			notify(ErrReconnectRequested)
		} else {
			delay = nextDelay(delay, b)
		}
	}
}

func runMuxAttempt(host context.Context, mux *Mux, a *muxAttempt, requests <-chan struct{}, lost <-chan error, pending func() bool, requireReady bool, onReady func(context.Context) error) (err error, manual, fatal bool) {
	if pending() {
		return ErrReconnectRequested, true, false
	}
	err = mux.Dial(a.ctx)
	if host.Err() != nil {
		return host.Err(), false, false
	}
	if pending() {
		return ErrReconnectRequested, true, false
	}
	if a.ctx.Err() != nil {
		return context.Cause(a.ctx), false, false
	}
	if err != nil {
		return err, false, false
	}
	if onReady != nil {
		err = onReady(a.ctx)
	}
	if host.Err() != nil {
		return host.Err(), false, false
	}
	if pending() {
		return ErrReconnectRequested, true, false
	}
	if a.ctx.Err() != nil {
		return context.Cause(a.ctx), false, false
	}
	if err != nil {
		return err, false, !requireReady
	}
	if !requireReady {
		mux.MarkReady(a.ctx)
	}
	select {
	case <-host.Done():
		return host.Err(), false, false
	case <-requests:
		return ErrReconnectRequested, true, false
	case <-a.ctx.Done():
		return context.Cause(a.ctx), false, false
	case err = <-lost:
		return err, false, false
	}
}

func (m *Mux) watchReadiness(o *muxMaintainer, a *muxAttempt, config RecoveryConfig, onWarning func(error)) {
	defer close(a.watchDone)
	timeout := time.NewTimer(time.Until(a.deadline))
	defer timeout.Stop()
	if config.GenerationReadyWarnMS <= config.GenerationReadyTimeoutMS {
		warning := time.NewTimer(time.Until(a.started.Add(time.Duration(config.GenerationReadyWarnMS) * time.Millisecond)))
		defer warning.Stop()
		select {
		case <-a.ctx.Done():
			return
		case <-a.ready:
			return
		case <-warning.C:
		}
		m.maintainMu.Lock()
		active := m.maintainer == o && o.attempt == a && !a.readySet && a.ctx.Err() == nil
		m.maintainMu.Unlock()
		if active && onWarning != nil {
			onWarning(fmt.Errorf("connection generation is still not ready after %dms", config.GenerationReadyWarnMS))
		}
	}
	select {
	case <-a.ctx.Done():
		return
	case <-a.ready:
		return
	case <-timeout.C:
	}
	failure := fmt.Errorf("connection generation was not ready within %dms", config.GenerationReadyTimeoutMS)
	m.maintainMu.Lock()
	active := m.maintainer == o && o.attempt == a && !a.readySet && a.ctx.Err() == nil
	if active {
		a.cancel(failure)
	}
	m.maintainMu.Unlock()
	if active && onWarning != nil {
		onWarning(failure)
	}
}

func jitterDelay(cap time.Duration) time.Duration {
	return time.Duration(float64(cap) * (0.5 + rand.Float64()*0.5))
}
func sleepBackoff(ctx context.Context, d time.Duration) error {
	_, err := waitMuxRetry(ctx, nil, d)
	return err
}
func waitMuxRetry(ctx context.Context, requests <-chan struct{}, d time.Duration) (bool, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-requests:
		return true, nil
	case <-timer.C:
		return false, nil
	}
}
func nextDelay(cur time.Duration, b Backoff) time.Duration {
	next := float64(cur) * b.Factor
	if next >= float64(b.Max) {
		return b.Max
	}
	if next < float64(b.Initial) {
		return b.Initial
	}
	return time.Duration(next)
}
