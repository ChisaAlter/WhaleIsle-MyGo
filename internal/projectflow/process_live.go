package projectflow

import (
	"context"
	"errors"
	"fmt"
	"time"
	"whaleisle/internal/dsh"
)

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
	opened := false
	for {
		if err = mux.Dial(ctx); err != nil {
			return err
		}
		stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{"address": address, "maxMessages": 30}}})
		if err != nil {
			return err
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
		if !opened || (!errors.Is(err, dsh.ErrCarrierLost) && !errors.Is(err, dsh.ErrStreamEnd)) {
			return err
		}
		// A logical stream interruption is observational, not worker completion.
		state.Live = false
		state.Error = "Member observation disconnected; reconnecting…"
		if !accept(state) {
			return context.Canceled
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
