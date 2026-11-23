package dsh

import (
	"context"
	"errors"
	"time"
)

// RetryPolicy is the backoff a transient read follows: Initial doubles up to
// Max between attempts, and no attempt starts once Budget has elapsed.
type RetryPolicy struct {
	Initial time.Duration
	Max     time.Duration
	Budget  time.Duration
}

// ReadRetry is the policy for idempotent Host reads: 200 ms doubling to
// 3.2 s, about 30 s in total. A Host that is still activating a lazy service
// or a pooled socket the server already closed both clear well inside it.
var ReadRetry = RetryPolicy{Initial: 200 * time.Millisecond, Max: 3200 * time.Millisecond, Budget: 30 * time.Second}

func (p RetryPolicy) normalized() RetryPolicy {
	if p.Initial <= 0 {
		p.Initial = ReadRetry.Initial
	}
	if p.Max < p.Initial {
		p.Max = p.Initial
	}
	if p.Budget <= 0 {
		p.Budget = ReadRetry.Budget
	}
	return p
}

// idempotentReads is the explicit allowlist RetryTransient retries. Every
// entry only reads Host state; a mutation must never be listed, because a
// request whose answer was lost may already have been applied.
var idempotentReads = map[string]bool{
	"session/list":                  true,
	"session/projections":           true,
	"session/events":                true,
	"session/page":                  true,
	"session/get":                   true,
	"session/modelCatalog":          true,
	"agentPresets/list":             true,
	"skills/list":                   true,
	"skillInventory/list":           true,
	"permissionPresets/catalog":     true,
	"settings/describe":             true,
	"commands/list":                 true,
	"speech/catalog":                true,
	"pluginInventory/list":          true,
	"llm/listProviders":             true,
	"llm/listConfigurableProviders": true,
}

// IdempotentRead reports whether method is on the retry allowlist.
func IdempotentRead(method string) bool { return idempotentReads[method] }

// Transient reports whether err describes a failure a retry can clear: a
// reset/refused/EOF transport, a lazily activated endpoint that is not
// registered yet, or an HTTP 429/5xx. Cancellation of the caller is never
// transient.
func Transient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, ErrEndpointNotReady) {
		return true
	}
	var transport *TransportError
	if errors.As(err, &transport) {
		return transport.Temporary()
	}
	var status *HTTPError
	if errors.As(err, &status) {
		return status.Temporary()
	}
	return false
}

// RetryTransient runs fn for method and repeats it while the failure is
// Transient, the policy's budget allows another attempt and ctx is live.
// A method outside the idempotent allowlist runs exactly once: retrying a
// mutation could apply it twice.
func RetryTransient(ctx context.Context, policy RetryPolicy, method string, fn func(context.Context) error) error {
	if !IdempotentRead(method) {
		return fn(ctx)
	}
	policy = policy.normalized()
	start := time.Now()
	delay := policy.Initial
	for {
		err := fn(ctx)
		if err == nil || !Transient(err) || ctx.Err() != nil {
			return err
		}
		if time.Since(start)+delay > policy.Budget {
			return err
		}
		if waitErr := sleepCtx(ctx, delay); waitErr != nil {
			return err
		}
		delay = min(delay*2, policy.Max)
	}
}

// CallRead is Call for an idempotent read, retried under ReadRetry. A method
// outside the allowlist behaves exactly like Call.
func (c *Client) CallRead(ctx context.Context, method string, args any, out any) error {
	return RetryTransient(ctx, ReadRetry, method, func(ctx context.Context) error {
		return c.Call(ctx, method, args, out)
	})
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Resubscribable reports whether a follow stream that ended with err should be
// opened again on the same carrier: the Host has not registered the endpoint
// yet, ended the stream, or this client dropped it after an inbox overflow.
// Carrier loss is not listed — the connection owner restarts every follow
// when the next generation is ready.
func Resubscribable(err error) bool {
	return errors.Is(err, ErrEndpointNotReady) || errors.Is(err, ErrStreamEnd) || errors.Is(err, ErrStreamOverflow)
}

// Resubscriber paces re-opening one follow stream: 200 ms doubling to 3.2 s,
// reset once the stream delivered a value again.
type Resubscriber struct {
	Policy RetryPolicy
	next   time.Duration
}

// Wait sleeps the next backoff step; it returns ctx's error when cancelled.
func (r *Resubscriber) Wait(ctx context.Context) error {
	p := r.Policy.normalized()
	if r.next <= 0 {
		r.next = p.Initial
	}
	d := r.next
	r.next = min(r.next*2, p.Max)
	return sleepCtx(ctx, d)
}

// Reset returns the pacing to the first step after a healthy delivery.
func (r *Resubscriber) Reset() { r.next = 0 }
