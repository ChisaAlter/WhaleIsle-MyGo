package dsh

import (
	"context"
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

// MaintainMux keeps a mux connected until ctx ends. onReady runs after each
// successful Dial (including the first). onLost runs after carrier failures.
func MaintainMux(ctx context.Context, mux *Mux, backoff Backoff, onReady func(context.Context) error, onLost func(error)) error {
	b := backoff.normalized()
	delay := b.Initial
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		lost := make(chan error, 1)
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

		if err := mux.Dial(ctx); err != nil {
			if onLost != nil {
				onLost(err)
			}
			if err := sleepBackoff(ctx, delay); err != nil {
				return err
			}
			delay = nextDelay(delay, b)
			continue
		}
		delay = b.Initial
		if onReady != nil {
			if err := onReady(ctx); err != nil {
				_ = mux.Close()
				return err
			}
		}
		select {
		case <-ctx.Done():
			_ = mux.Close()
			return ctx.Err()
		case err := <-lost:
			if onLost != nil {
				onLost(err)
			}
			_ = mux.Close()
			mux.Reset()
			if err := sleepBackoff(ctx, delay); err != nil {
				return err
			}
			delay = nextDelay(delay, b)
		}
	}
}

func sleepBackoff(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
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
