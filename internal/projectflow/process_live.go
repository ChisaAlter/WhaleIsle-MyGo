package projectflow

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
	"whaleisle/internal/dsh"
)

func isTransientFollowError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, dsh.ErrCarrierLost) ||
		errors.Is(err, dsh.ErrStreamEnd) ||
		errors.Is(err, dsh.ErrStreamOverflow) ||
		errors.Is(err, dsh.ErrEndpointNotReady) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var transportErr *dsh.TransportError
	if errors.As(err, &transportErr) {
		return true
	}
	msg := err.Error()
	if strings.Contains(msg, "dsh mux: dial:") {
		// Treat transient handshake / dial errors (e.g., 502/503/504, EOF, reset) as retryable,
		// while still surfacing permanent auth/not-found failures.
		if strings.Contains(msg, "got 401") || strings.Contains(msg, "got 403") || strings.Contains(msg, "got 404") {
			return false
		}
		return true
	}
	return strings.Contains(msg, "dsh mux: closed") || strings.Contains(msg, "connection refused") || strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset")
}

// FollowProcess retains the original continuable child Session while its
// read-only process viewer is open. A new carrier's snapshot replaces the old
// window. accept=false releases this stream immediately after navigation.
func FollowProcess(ctx context.Context, mux *dsh.Mux, p Project, d WorkDetail, accept func(ProcessSnapshot) bool) error {
	address, err := d.ProcessAddress(p)
	if err != nil {
		return err
	}
	if mux == nil {
		return fmt.Errorf("member history transport is unavailable")
	}
	const (
		minBackoff = 500 * time.Millisecond
		maxBackoff = 5 * time.Second
	)
	opened := false
	backoff := minBackoff
	waitRetry := func(last ProcessSnapshot) error {
		// A logical stream interruption is observational, not worker completion.
		last.Live = false
		last.Error = "Member observation disconnected; reconnecting…"
		if !accept(last) {
			return context.Canceled
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		return nil
	}
	var lastState ProcessSnapshot
	for {
		if err = mux.Dial(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !opened || !isTransientFollowError(err) {
				return err
			}
			if werr := waitRetry(lastState); werr != nil {
				return werr
			}
			continue
		}
		stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{"address": address, "maxMessages": 30}}})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !opened || !isTransientFollowError(err) {
				return err
			}
			if werr := waitRetry(lastState); werr != nil {
				return werr
			}
			continue
		}
		state := ProcessSnapshot{}
		for {
			raw, recvErr := stream.Recv(ctx)
			if recvErr != nil {
				err = recvErr
				break
			}
			state, err = FoldProcessFrame(state, raw)
			if err != nil {
				break
			}
			if state.Type == "snapshot" {
				opened = true
				backoff = minBackoff
				lastState = state
				if !accept(state) {
					stream.Cancel()
					return context.Canceled
				}
			}
		}
		stream.Cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !opened || !isTransientFollowError(err) {
			return err
		}
		if werr := waitRetry(state); werr != nil {
			return werr
		}
	}
}
