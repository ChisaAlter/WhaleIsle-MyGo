package voicecapture

import (
	"context"
	"errors"
	"sync"
)

type macPermissionRequest struct {
	done    chan struct{}
	granted bool // published by closing done
}

// A canceled waiter cannot cancel macOS's dialog. Share its pending request
// instead of allocating an unbounded sequence of callbacks if Record is retried.
// A late completion publishes permission only; it never opens an audio device.
type macPermission struct {
	usage   func() error
	status  func() int64
	request func(func(bool))
	mu      sync.Mutex
	pending *macPermissionRequest
}

func (p *macPermission) authorize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.usage(); err != nil {
		return err
	}
	switch p.status() {
	case 3: // AVAuthorizationStatusAuthorized
		return nil
	case 0: // AVAuthorizationStatusNotDetermined
	default:
		return errors.New("microphone access denied or restricted; review macOS Privacy & Security settings")
	}
	p.mu.Lock()
	pending := p.pending
	first := pending == nil
	if first {
		pending = &macPermissionRequest{done: make(chan struct{})}
		p.pending = pending
	}
	p.mu.Unlock()
	if first {
		p.request(func(granted bool) {
			p.mu.Lock()
			pending.granted = granted
			close(pending.done)
			if p.pending == pending {
				p.pending = nil
			}
			p.mu.Unlock()
		})
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pending.done:
		if err := ctx.Err(); err != nil {
			return err
		}
		if !pending.granted {
			return errors.New("microphone access denied; review macOS Privacy & Security settings")
		}
		return nil
	}
}
