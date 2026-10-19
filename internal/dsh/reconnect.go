package dsh

import (
	"context"
	"errors"
	"time"
)

// Backoff controls reconnect delays after a carrier loss.
type Backoff struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
}

func (b Backoff) normalized() Backoff {
	if b.Initial <= 0 {
		b.Initial = 200 * time.Millisecond
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

type muxMaintainer struct {
	ctx           context.Context
	requests      chan struct{}
	cancelAttempt context.CancelFunc // guarded by Mux.maintainMu
	stopping      bool
}

// RequestReconnect interrupts the current attempt or retry delay. It keeps the
// same Mux and returns false when no active MaintainMux owns it.
func (m *Mux) RequestReconnect() bool {
	m.maintainMu.Lock()
	defer m.maintainMu.Unlock()
	o := m.maintainer
	if o == nil || o.stopping || o.ctx.Err() != nil {
		return false
	}
	if o.cancelAttempt != nil {
		o.cancelAttempt()
	}
	select {
	case o.requests <- struct{}{}:
	default:
	}
	return true
}

// MaintainMux keeps a mux connected until ctx ends. Each onReady receives an
// attempt context canceled before its streams are retired. Manual retries skip
// backoff; onLost also receives ErrReconnectRequested for those retries.
func MaintainMux(ctx context.Context, mux *Mux, backoff Backoff, onReady func(context.Context) error, onLost func(error)) error {
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
		if o.cancelAttempt != nil {
			o.cancelAttempt()
		}
		mux.maintainMu.Unlock()
		_ = mux.closeCarrier(true)
		mux.mu.Lock()
		mux.OnCarrierLost = prev
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
	manualPending := func() bool {
		select {
		case <-o.requests:
			return true
		default:
			return false
		}
	}
	retire := func(cancel context.CancelFunc) {
		cancel()
		mux.maintainMu.Lock()
		o.cancelAttempt = nil
		mux.maintainMu.Unlock()
		_ = mux.closeCarrier(true)
		mux.Reset()
	}
	b := backoff.normalized()
	delay := b.Initial
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-lost:
		default:
		}
		attempt, cancel := context.WithCancel(ctx)
		mux.maintainMu.Lock()
		o.cancelAttempt = cancel
		mux.maintainMu.Unlock()
		if manualPending() {
			retire(cancel)
			delay = b.Initial
			notify(ErrReconnectRequested)
			continue
		}
		err := mux.Dial(attempt)
		if ctx.Err() != nil {
			cancel()
			return ctx.Err()
		}
		if manualPending() {
			retire(cancel)
			delay = b.Initial
			notify(ErrReconnectRequested)
			continue
		}
		if err != nil {
			retire(cancel)
			notify(err)
			manual, waitErr := waitMuxRetry(ctx, o.requests, delay)
			if waitErr != nil {
				return waitErr
			}
			if manual {
				delay = b.Initial
				notify(ErrReconnectRequested)
			} else {
				delay = nextDelay(delay, b)
			}
			continue
		}
		delay = b.Initial
		if onReady != nil {
			err = onReady(attempt)
		}
		if ctx.Err() != nil {
			cancel()
			return ctx.Err()
		}
		if manualPending() {
			retire(cancel)
			notify(ErrReconnectRequested)
			continue
		}
		if err != nil {
			cancel()
			return err
		}
		manual := false
		select {
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case <-o.requests:
			manual = true
			err = ErrReconnectRequested
		case err = <-lost:
		}
		retire(cancel)
		notify(err)
		if manual {
			continue
		}
		manual, err = waitMuxRetry(ctx, o.requests, delay)
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

func sleepBackoff(ctx context.Context, d time.Duration) error {
	_, err := waitMuxRetry(ctx, nil, d)
	return err
}

func waitMuxRetry(ctx context.Context, requests <-chan struct{}, d time.Duration) (bool, error) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-requests:
		return true, nil
	case <-t.C:
		return false, nil
	}
}

func nextDelay(cur time.Duration, b Backoff) time.Duration {
	next := time.Duration(float64(cur) * b.Factor)
	if next > b.Max {
		return b.Max
	}
	if next < b.Initial {
		return b.Initial
	}
	return next
}
